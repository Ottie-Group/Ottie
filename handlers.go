package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const sessionName = "ottie_session"

type SessionStore struct {
	mu   sync.RWMutex
	deks map[string][]byte // sessionID -> raw 32-byte DEK
}

func newSessionStore() *SessionStore {
	return &SessionStore{
		deks: make(map[string][]byte),
	}
}

func (s *SessionStore) Set(sessionID string, dek []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deks[sessionID] = dek
}

func (s *SessionStore) Get(sessionID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dek, ok := s.deks[sessionID]
	return dek, ok
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.deks, sessionID)
}

type App struct {
	db          *sql.DB
	store       *sessions.CookieStore
	dekStore    *SessionStore
	serverKey   []byte
	rateLimiter *RateLimiter
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateNumericOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	num := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", num), nil
}

type SessionUser struct {
	ID       int64
	Username string
	Role     string
	DEK      []byte
}

func (a *App) getSession(r *http.Request) (*sessions.Session, error) {
	sess, err := a.store.Get(r, sessionName)
	if sess != nil && sess.Options != nil {
		// If request is over HTTPS (TLS or X-Forwarded-Proto: https), ensure Secure is true.
		// Otherwise, respect the configured environment variable setting on a.store.Options.Secure.
		isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
		if isHTTPS {
			sess.Options.Secure = true
		} else {
			sess.Options.Secure = a.store.Options.Secure
		}
	}
	return sess, err
}

func (a *App) getSessionUser(r *http.Request) (*SessionUser, error) {
	sess, err := a.getSession(r)
	if err != nil || sess == nil {
		log.Printf("[AUTH] getSession error=%v (Cookie header: %q)", err, r.Header.Get("Cookie"))
		return nil, errors.New("no session")
	}

	var uid int64
	switch v := sess.Values["user_id"].(type) {
	case int64:
		uid = v
	case int:
		uid = int64(v)
	case int32:
		uid = int64(v)
	case float64:
		uid = int64(v)
	case string:
		uid, _ = strconv.ParseInt(v, 10, 64)
	}

	if uid == 0 {
		return nil, errors.New("not logged in")
	}

	username, _ := sess.Values["username"].(string)
	role, _ := sess.Values["role"].(string)
	sid, _ := sess.Values["session_token"].(string)

	// Try in-memory store first
	if sid != "" {
		if dek, ok := a.dekStore.Get(sid); ok && len(dek) == 32 {
			return &SessionUser{
				ID:       uid,
				Username: username,
				Role:     role,
				DEK:      dek,
			}, nil
		}
	}

	// Try encrypted backup from session cookie
	if encDEKCookie, ok := sess.Values["enc_dek"].(string); ok && encDEKCookie != "" {
		dek, err := DecryptAESGCM(encDEKCookie, a.serverKey)
		if err == nil && len(dek) == 32 {
			if sid != "" {
				a.dekStore.Set(sid, dek)
			}
			return &SessionUser{
				ID:       uid,
				Username: username,
				Role:     role,
				DEK:      dek,
			}, nil
		}
		log.Printf("[AUTH] DecryptAESGCM on enc_dek failed for user=%s: %v", username, err)
	} else {
		log.Printf("[AUTH] No DEK found for user=%s (sid=%s, enc_dek present=%v)", username, sid, sess.Values["enc_dek"] != nil)
	}

	return nil, errors.New("session key expired")
}

func (a *App) checkCSRF(r *http.Request, sess *sessions.Session) bool {
	want, _ := sess.Values["csrf"].(string)
	got := r.FormValue("csrf_token")
	if got == "" {
		got = r.Header.Get("X-CSRF-Token")
	}
	return want != "" && got != "" && want == got
}

func (a *App) handleCodesAPI(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	accounts, err := listAccounts(a.db, user.ID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	type codeOut struct {
		ID               int64  `json:"id"`
		Issuer           string `json:"issuer"`
		AccountName      string `json:"account_name"`
		Category         string `json:"category"`
		Code             string `json:"code"`
		SecondsRemaining int    `json:"seconds_remaining"`
		Period           int    `json:"period"`
		ProgressPercent  int    `json:"progress_percent"`
	}

	now := time.Now()
	out := make([]codeOut, 0, len(accounts))
	for _, acc := range accounts {
		secret, err := DecryptSecret(acc.EncryptedSecret, user.DEK)
		if err != nil {
			continue
		}
		code, err := totp.GenerateCodeCustom(secret, now, totp.ValidateOpts{
			Period:    uint(acc.Period),
			Digits:    otp.Digits(acc.Digits),
			Algorithm: algoFromString(acc.Algorithm),
		})
		if err != nil {
			continue
		}
		remaining := acc.Period - int(now.Unix())%acc.Period
		progress := int((float64(remaining) / float64(acc.Period)) * 100)
		out = append(out, codeOut{
			ID:               acc.ID,
			Issuer:           acc.Issuer,
			AccountName:      acc.AccountName,
			Category:         acc.Category,
			Code:             code,
			SecondsRemaining: remaining,
			Period:           acc.Period,
			ProgressPercent:  progress,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(out)
}

func (a *App) handleExportVault(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, _ := a.getSession(r)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired token", http.StatusForbidden)
		return
	}

	accounts, err := listAccounts(a.db, user.ID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	type ExportItem struct {
		Issuer      string `json:"issuer"`
		AccountName string `json:"account_name"`
		Category    string `json:"category"`
		Secret      string `json:"secret"`
		Digits      int    `json:"digits"`
		Period      int    `json:"period"`
		Algorithm   string `json:"algorithm"`
		URI         string `json:"uri"`
	}

	type ExportPayload struct {
		OttieVersion string       `json:"ottie_version"`
		ExportDate   time.Time    `json:"export_date"`
		Username     string       `json:"username"`
		Accounts     []ExportItem `json:"accounts"`
	}

	var items []ExportItem
	for _, acc := range accounts {
		secret, err := DecryptSecret(acc.EncryptedSecret, user.DEK)
		if err != nil {
			continue
		}
		uri := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=%s&digits=%d&period=%d",
			acc.Issuer, acc.AccountName, secret, acc.Issuer, acc.Algorithm, acc.Digits, acc.Period)
		items = append(items, ExportItem{
			Issuer:      acc.Issuer,
			AccountName: acc.AccountName,
			Category:    acc.Category,
			Secret:      secret,
			Digits:      acc.Digits,
			Period:      acc.Period,
			Algorithm:   acc.Algorithm,
			URI:         uri,
		})
	}

	payload := ExportPayload{
		OttieVersion: Version,
		ExportDate:   time.Now().UTC(),
		Username:     user.Username,
		Accounts:     items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"ottie-backup-%s-%s.json\"", user.Username, time.Now().Format("20060102-150405")))
	json.NewEncoder(w).Encode(payload)
}

func algoFromString(s string) otp.Algorithm {
	switch strings.ToUpper(s) {
	case "SHA256":
		return otp.AlgorithmSHA256
	case "SHA512":
		return otp.AlgorithmSHA512
	default:
		return otp.AlgorithmSHA1
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"success": false,
		"error":   msg,
	})
}

func isSMTPConfigured() bool {
	return os.Getenv("SMTP_HOST") != "" || os.Getenv("OTTIE_SMTP_HOST") != "" || os.Getenv("SMTP_SERVER") != ""
}

func isSMSGateConfigured() bool {
	return (os.Getenv("SMSGATE_LOGIN") != "" && os.Getenv("SMSGATE_PASSWORD") != "") ||
		(os.Getenv("SMSGATE_USERNAME") != "" && os.Getenv("SMSGATE_PASSWORD") != "") ||
		os.Getenv("SMSGATE_TOKEN") != "" ||
		os.Getenv("SMSGATE_API_KEY") != "" ||
		os.Getenv("SMSGATE_URL") != "" ||
		os.Getenv("SMSGATE_SERVER") != ""
}

func isTwilioConfigured() bool {
	return os.Getenv("TWILIO_ACCOUNT_SID") != "" && os.Getenv("TWILIO_AUTH_TOKEN") != "" && os.Getenv("TWILIO_PHONE_NUMBER") != ""
}

func isSMSConfigured() bool {
	return isSMSGateConfigured() || isTwilioConfigured() ||
		os.Getenv("SMS_API_KEY") != "" ||
		os.Getenv("OTTIE_SMS_GATEWAY") != ""
}

func sendEmailOTP(toEmail, code string) error {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = os.Getenv("OTTIE_SMTP_HOST")
	}
	if host == "" {
		host = os.Getenv("SMTP_SERVER")
	}
	if host == "" {
		return errors.New("no SMTP host configured")
	}

	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}
	if from == "" {
		from = "noreply@ottie.local"
	}

	addr := net.JoinHostPort(host, port)
	subject := "Subject: [Ottie] Your Verification Passcode\r\n"
	fromHeader := fmt.Sprintf("From: %s\r\n", from)
	toHeader := fmt.Sprintf("To: %s\r\n", toEmail)
	mimeHeader := "MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf("Hello,\r\n\r\nYour Ottie login verification passcode is:\r\n\r\n   %s\r\n\r\nThis passcode expires in 5 minutes.\r\n", code)
	msg := []byte(fromHeader + toHeader + subject + mimeHeader + body)

	var auth smtp.Auth
	if user != "" && password != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}

	return smtp.SendMail(addr, auth, from, []string{toEmail}, msg)
}

func sendSMSGateSMS(toPhone, code string) error {
	serverURL := os.Getenv("SMSGATE_URL")
	if serverURL == "" {
		serverURL = os.Getenv("SMSGATE_SERVER")
	}
	if serverURL == "" {
		serverURL = "https://api.sms-gate.app/3rdparty/v1/messages"
	} else if !strings.Contains(serverURL, "/messages") && !strings.Contains(serverURL, "/message") {
		serverURL = strings.TrimSuffix(serverURL, "/") + "/3rdparty/v1/messages"
	}

	login := os.Getenv("SMSGATE_LOGIN")
	if login == "" {
		login = os.Getenv("SMSGATE_USERNAME")
	}
	password := os.Getenv("SMSGATE_PASSWORD")
	token := os.Getenv("SMSGATE_TOKEN")
	if token == "" {
		token = os.Getenv("SMSGATE_API_KEY")
	}

	payload := map[string]any{
		"textMessage": map[string]string{
			"text": fmt.Sprintf("Your Ottie verification passcode is: %s (expires in 5 mins)", code),
		},
		"phoneNumbers": []string{toPhone},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, serverURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if login != "" && password != "" {
		req.SetBasicAuth(login, password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SMSGate returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func sendTwilioSMS(toPhone, code string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromPhone := os.Getenv("TWILIO_PHONE_NUMBER")

	if accountSid == "" || authToken == "" || fromPhone == "" {
		return errors.New("incomplete Twilio SMS credentials")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", accountSid)
	data := url.Values{}
	data.Set("To", toPhone)
	data.Set("From", fromPhone)
	data.Set("Body", fmt.Sprintf("Your Ottie verification passcode is: %s (expires in 5 mins)", code))

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}
	return nil
}

func sendSMSOTP(toPhone, code string) error {
	if isSMSGateConfigured() {
		return sendSMSGateSMS(toPhone, code)
	}
	if isTwilioConfigured() {
		return sendTwilioSMS(toPhone, code)
	}
	return errors.New("no SMS provider configured")
}

func send2FACode(method, destination, code string) {
	if method == "email" && isSMTPConfigured() {
		go func() {
			if err := sendEmailOTP(destination, code); err != nil {
				log.Printf("[Ottie SMTP Warning] Could not deliver email to %s: %v. (Passcode: %s is logged for access)", destination, err, code)
			} else {
				log.Printf("[Ottie SMTP Success] Dispatched 2FA code to %s", destination)
			}
		}()
	} else if method == "sms" && isSMSConfigured() {
		go func() {
			providerName := "SMSGate"
			if !isSMSGateConfigured() && isTwilioConfigured() {
				providerName = "Twilio"
			}
			if err := sendSMSOTP(destination, code); err != nil {
				log.Printf("[Ottie %s SMS Warning] Could not deliver SMS to %s: %v. (Passcode: %s is logged for access)", providerName, destination, err, code)
			} else {
				log.Printf("[Ottie %s SMS Success] Dispatched 2FA SMS to %s", providerName, destination)
			}
		}()
	}
}

func (a *App) handleApiMe(w http.ResponseWriter, r *http.Request) {
	count, err := countUsers(a.db)
	if err == nil && count == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"setupNeeded":   true,
			"authenticated": false,
		})
		return
	}

	sess, _ := a.getSession(r)
	token, _ := sess.Values["csrf"].(string)
	if token == "" {
		token = newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
	}
	w.Header().Set("X-CSRF-Token", token)

	user, err := a.getSessionUser(r)
	if err != nil {
		log.Printf("[SESSION ME] Unauthenticated request from IP=%s (Host=%q, CookiePresent=%v): %v",
			getClientIP(r), r.Host, r.Header.Get("Cookie") != "", err)
		writeJSON(w, http.StatusOK, map[string]any{
			"setupNeeded":   false,
			"authenticated": false,
			"csrfToken":     token,
		})
		return
	}

	log.Printf("[SESSION ME] Authenticated user %q from IP=%s", user.Username, getClientIP(r))

	dbUser, _ := getUserByID(a.db, user.ID)
	has2FA := false
	method := "email"
	dest := ""
	if dbUser != nil {
		has2FA = dbUser.OTPMethod != "" && dbUser.OTPMethod != "none"
		method = dbUser.OTPMethod
		if method == "email" {
			dest = dbUser.Email
		} else if method == "sms" {
			dest = dbUser.Phone
		}
	}

	phrase, _ := sess.Values["new_recovery_phrase"].(string)

	writeJSON(w, http.StatusOK, map[string]any{
		"setupNeeded":   false,
		"authenticated": true,
		"csrfToken":     token,
		"twoFactorConfig": map[string]bool{
			"smtpConfigured": isSMTPConfigured(),
			"smsConfigured":  isSMSConfigured(),
		},
		"user": map[string]any{
			"id":             strconv.FormatInt(user.ID, 10),
			"username":       user.Username,
			"role":           user.Role,
			"has2FA":         has2FA,
			"deliveryMethod": method,
			"deliveryDest":   dest,
			"recoveryKey":    phrase,
		},
	})
}

func (a *App) handleApiAccounts(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	accounts, err := listAccounts(a.db, user.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load accounts")
		return
	}

	type AccountDTO struct {
		ID          string `json:"id"`
		Issuer      string `json:"issuer"`
		AccountName string `json:"accountName"`
		Category    string `json:"category"`
		CreatedAt   string `json:"createdAt"`
	}

	dtos := make([]AccountDTO, 0, len(accounts))
	for _, acc := range accounts {
		cat := acc.Category
		if cat == "" {
			cat = "Personal"
		}
		dtos = append(dtos, AccountDTO{
			ID:          strconv.FormatInt(acc.ID, 10),
			Issuer:      acc.Issuer,
			AccountName: acc.AccountName,
			Category:    cat,
			CreatedAt:   acc.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": dtos,
	})
}

func (a *App) handleApiSetup(w http.ResponseWriter, r *http.Request) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(getClientIP(r)) {
		writeJSONError(w, http.StatusTooManyRequests, "Too many setup attempts. Please wait a minute and try again.")
		return
	}

	count, err := countUsers(a.db)
	if err == nil && count > 0 {
		writeJSONError(w, http.StatusBadRequest, "Setup has already been completed.")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	if len(username) < 3 {
		writeJSONError(w, http.StatusBadRequest, "Username must be at least 3 characters.")
		return
	}
	if len(password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "Password must be at least 8 characters.")
		return
	}

	salt, err := GenerateSalt()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto error generating salt.")
		return
	}

	kek, err := DeriveKEK(password, salt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto error deriving KEK.")
		return
	}

	dek, err := GenerateDEK()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto error generating DEK.")
		return
	}

	encDEK, err := WrapDEK(dek, kek)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto error wrapping DEK.")
		return
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Error hashing password.")
		return
	}

	uid, err := createUserWithDEK(a.db, username, string(pwHash), "admin", encDEK, salt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create admin user.")
		return
	}

	phrase, err := GenerateMnemonicPhrase(12)
	if err == nil {
		recSalt, _ := GenerateSalt()
		recKEK, _ := DeriveKEK(NormalizePhrase(phrase), recSalt)
		recEncDEK, _ := WrapDEK(dek, recKEK)
		_ = updateUserRecoveryData(a.db, uid, recEncDEK, recSalt, "")
	}

	sess, _ := a.getSession(r)
	sess.Values["setup_user_id"] = uid
	sess.Values["setup_dek"] = dek
	sess.Values["new_recovery_phrase"] = phrase
	sess.Save(r, w)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"recoveryPhrase": phrase,
		"recoveryKey":    phrase,
		"words":          strings.Fields(phrase),
	})
}

func (a *App) handleApiSetupConfirm(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.getSession(r)
	uid, _ := sess.Values["setup_user_id"].(int64)
	dek, _ := sess.Values["setup_dek"].([]byte)

	if uid == 0 || len(dek) != 32 {
		writeJSONError(w, http.StatusBadRequest, "Invalid setup session.")
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	delete(sess.Values, "setup_user_id")
	delete(sess.Values, "setup_dek")
	delete(sess.Values, "new_recovery_phrase")

	token, _ := sess.Values["csrf"].(string)
	if token == "" {
		token = newCSRFToken()
		sess.Values["csrf"] = token
	}

	sess.Values["user_id"] = uid
	sess.Values["username"] = "admin"
	sess.Values["role"] = "admin"
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	sess.Save(r, w)

	updateLastLogin(a.db, uid)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiAuthLogin(w http.ResponseWriter, r *http.Request) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(getClientIP(r)) {
		writeJSONError(w, http.StatusTooManyRequests, "Too many login attempts. Please wait a minute and try again.")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	dbUser, err := getUserByUsername(a.db, username)
	if err != nil || dbUser == nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid username or password.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid username or password.")
		return
	}

	kek, err := DeriveKEK(password, dbUser.Salt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto key derivation error.")
		return
	}

	dek, err := UnwrapDEK(dbUser.EncDEK, kek)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Master key decryption failed.")
		return
	}

	sess, _ := a.getSession(r)

	if dbUser.OTPMethod == "email" || dbUser.OTPMethod == "sms" {
		otpCode, err := generateNumericOTP()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "OTP generation error")
			return
		}
		encDEKForPending, _ := EncryptAESGCM(dek, a.serverKey)

		sess.Values["pending_otp_user_id"] = dbUser.ID
		sess.Values["pending_username"] = dbUser.Username
		sess.Values["pending_role"] = dbUser.Role
		sess.Values["pending_otp_enc_dek"] = encDEKForPending
		sess.Values["pending_otp_code"] = otpCode
		sess.Values["pending_otp_expiry"] = time.Now().Add(5 * time.Minute).Unix()
		sess.Save(r, w)

		log.Printf("\n[Ottie 2FA Delivery] User: %s\n   Method: %s -> %s\n   One-Time Passcode: %s (Expires in 5 mins)\n",
			dbUser.Username, dbUser.OTPMethod, dbUser.Email, otpCode)

		dest := dbUser.Email
		if dbUser.OTPMethod == "sms" {
			dest = dbUser.Phone
		}
		send2FACode(dbUser.OTPMethod, dest, otpCode)

		writeJSON(w, http.StatusOK, map[string]any{
			"success":     true,
			"requires2FA": true,
			"method":      dbUser.OTPMethod,
		})
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, err := EncryptAESGCM(dek, a.serverKey)
	if err != nil {
		log.Printf("[LOGIN ERROR] EncryptAESGCM on DEK failed: %v", err)
	}

	token, _ := sess.Values["csrf"].(string)
	if token == "" {
		token = newCSRFToken()
		sess.Values["csrf"] = token
	}

	sess.Values["user_id"] = dbUser.ID
	sess.Values["username"] = dbUser.Username
	sess.Values["role"] = dbUser.Role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	if err := sess.Save(r, w); err != nil {
		log.Printf("[LOGIN ERROR] Failed to save session cookie: %v", err)
	} else {
		log.Printf("[LOGIN SUCCESS] User %q logged in successfully from IP %s (Host=%q, Secure=%v, SameSite=%v)",
			dbUser.Username, getClientIP(r), r.Host, sess.Options.Secure, sess.Options.SameSite)
	}

	updateLastLogin(a.db, dbUser.ID)
	w.Header().Set("X-CSRF-Token", token)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"csrfToken": token,
		"user": map[string]any{
			"id":             strconv.FormatInt(dbUser.ID, 10),
			"username":       dbUser.Username,
			"role":           dbUser.Role,
			"has2FA":         dbUser.OTPMethod != "" && dbUser.OTPMethod != "none",
			"deliveryMethod": dbUser.OTPMethod,
		},
	})
}

func (a *App) handleApiAuthVerify2FA(w http.ResponseWriter, r *http.Request) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(getClientIP(r)) {
		writeJSONError(w, http.StatusTooManyRequests, "Too many passcode attempts. Please wait a minute and try again.")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	sess, _ := a.getSession(r)
	pendingID, _ := sess.Values["pending_otp_user_id"].(int64)
	if pendingID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "No pending login session")
		return
	}

	expectedCode, _ := sess.Values["pending_otp_code"].(string)
	expiry, _ := sess.Values["pending_otp_expiry"].(int64)

	if time.Now().Unix() > expiry || req.Code != expectedCode {
		writeJSONError(w, http.StatusUnauthorized, "Invalid or expired passcode")
		return
	}

	encPendingDEK, _ := sess.Values["pending_otp_enc_dek"].(string)
	dek, err := DecryptAESGCM(encPendingDEK, a.serverKey)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Session decryption error")
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	username, _ := sess.Values["pending_username"].(string)
	role, _ := sess.Values["pending_role"].(string)

	token, _ := sess.Values["csrf"].(string)
	if token == "" {
		token = newCSRFToken()
		sess.Values["csrf"] = token
	}

	delete(sess.Values, "pending_otp_user_id")
	delete(sess.Values, "pending_username")
	delete(sess.Values, "pending_role")
	delete(sess.Values, "pending_otp_enc_dek")
	delete(sess.Values, "pending_otp_code")
	delete(sess.Values, "pending_otp_expiry")

	sess.Values["user_id"] = pendingID
	sess.Values["username"] = username
	sess.Values["role"] = role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	sess.Save(r, w)

	updateLastLogin(a.db, pendingID)
	w.Header().Set("X-CSRF-Token", token)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"csrfToken": token,
		"user": map[string]any{
			"id":       strconv.FormatInt(pendingID, 10),
			"username": username,
			"role":     role,
		},
	})
}

func (a *App) handleApiAuthVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(getClientIP(r)) {
		writeJSONError(w, http.StatusTooManyRequests, "Too many attempts. Please wait a minute and try again.")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	sess, _ := a.getSession(r)
	pendingID, _ := sess.Values["pending_otp_user_id"].(int64)
	if pendingID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "No pending login session")
		return
	}

	dbUser, err := getUserByID(a.db, pendingID)
	if err != nil || dbUser == nil || dbUser.RecoveryEncDEK == "" || dbUser.RecoverySalt == "" {
		writeJSONError(w, http.StatusUnauthorized, "Emergency recovery not configured")
		return
	}

	normCode := NormalizePhrase(req.Code)
	recKEK, err := DeriveKEK(normCode, dbUser.RecoverySalt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto key derivation error")
		return
	}

	dek, err := UnwrapDEK(dbUser.RecoveryEncDEK, recKEK)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid emergency recovery phrase")
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	username, _ := sess.Values["pending_username"].(string)
	role, _ := sess.Values["pending_role"].(string)

	token, _ := sess.Values["csrf"].(string)
	if token == "" {
		token = newCSRFToken()
		sess.Values["csrf"] = token
	}

	delete(sess.Values, "pending_otp_user_id")
	delete(sess.Values, "pending_username")
	delete(sess.Values, "pending_role")
	delete(sess.Values, "pending_otp_enc_dek")

	sess.Values["user_id"] = pendingID
	sess.Values["username"] = username
	sess.Values["role"] = role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	sess.Save(r, w)

	updateLastLogin(a.db, pendingID)
	w.Header().Set("X-CSRF-Token", token)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"csrfToken": token,
		"user": map[string]any{
			"id":       strconv.FormatInt(pendingID, 10),
			"username": username,
			"role":     role,
		},
	})
}

func (a *App) handleApiAuthRecover(w http.ResponseWriter, r *http.Request) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(getClientIP(r)) {
		writeJSONError(w, http.StatusTooManyRequests, "Too many recovery attempts. Please wait a minute and try again.")
		return
	}

	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		RecoveryKey string `json:"recoveryKey"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	phrase := NormalizePhrase(req.RecoveryKey)
	newPassword := req.NewPassword

	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "Otter Name (Username) is required.")
		return
	}

	if phrase == "" || len(strings.Fields(phrase)) != 12 {
		writeJSONError(w, http.StatusBadRequest, "Please provide all 12 recovery words.")
		return
	}

	if len(newPassword) < 8 {
		writeJSONError(w, http.StatusBadRequest, "New master key must be at least 8 characters.")
		return
	}

	dbUser, err := getUserByUsername(a.db, username)
	if err != nil || dbUser == nil || dbUser.RecoveryEncDEK == "" || dbUser.RecoverySalt == "" {
		writeJSONError(w, http.StatusBadRequest, "Invalid username or no recovery key configured.")
		return
	}

	recKEK, err := DeriveKEK(phrase, dbUser.RecoverySalt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Crypto recovery error.")
		return
	}

	dek, err := UnwrapDEK(dbUser.RecoveryEncDEK, recKEK)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid recovery phrase.")
		return
	}

	newSalt, _ := GenerateSalt()
	newKEK, _ := DeriveKEK(newPassword, newSalt)
	newWrappedDEK, _ := WrapDEK(dek, newKEK)
	newPWHash, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

	if err := updateUserPasswordAndDEK(a.db, dbUser.ID, string(newPWHash), newWrappedDEK, newSalt); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update vault password.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiAuthLogout(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.getSession(r)
	if sid, ok := sess.Values["session_token"].(string); ok {
		a.dekStore.Delete(sid)
	}
	sess.Options.MaxAge = -1
	sess.Save(r, w)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiTokens(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Secret      string `json:"secret"`
		Issuer      string `json:"issuer"`
		AccountName string `json:"accountName"`
		Category    string `json:"category"`
		Code        string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	rawInput := strings.TrimSpace(req.Secret)
	var rawSecret, parsedIssuer, parsedAccount string

	if strings.HasPrefix(strings.ToLower(rawInput), "otpauth://") {
		u, err := url.Parse(rawInput)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid otpauth URI format.")
			return
		}
		q := u.Query()
		rawSecret = strings.ToUpper(strings.ReplaceAll(q.Get("secret"), " ", ""))
		parsedIssuer = q.Get("issuer")
		label := strings.TrimPrefix(u.Path, "/")
		label = strings.TrimPrefix(label, "totp/")
		label = strings.TrimPrefix(label, "hotp/")
		label, _ = url.PathUnescape(label)
		if strings.Contains(label, ":") {
			parts := strings.SplitN(label, ":", 2)
			if parsedIssuer == "" {
				parsedIssuer = strings.TrimSpace(parts[0])
			}
			parsedAccount = strings.TrimSpace(parts[1])
		} else if label != "" && parsedAccount == "" {
			parsedAccount = strings.TrimSpace(label)
		}
	} else {
		rawSecret = strings.ToUpper(strings.ReplaceAll(rawInput, " ", ""))
	}

	issuer := strings.TrimSpace(req.Issuer)
	if issuer == "" && parsedIssuer != "" {
		issuer = parsedIssuer
	}

	accountName := strings.TrimSpace(req.AccountName)
	if accountName == "" && parsedAccount != "" {
		accountName = parsedAccount
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Personal"
	}

	if rawSecret == "" || !isValidBase32(rawSecret) {
		writeJSONError(w, http.StatusBadRequest, "Secret key must be a valid Base32 string (letters A-Z and digits 2-7) or a valid otpauth:// URI.")
		return
	}

	if len(rawSecret) < 8 {
		writeJSONError(w, http.StatusBadRequest, "Secret key must be at least 8 characters long.")
		return
	}

	if issuer == "" {
		writeJSONError(w, http.StatusBadRequest, "Service / Issuer name is required.")
		return
	}

	if len(issuer) > 100 {
		writeJSONError(w, http.StatusBadRequest, "Issuer name must not exceed 100 characters.")
		return
	}

	if len(accountName) > 100 {
		writeJSONError(w, http.StatusBadRequest, "Account name must not exceed 100 characters.")
		return
	}

	if len(category) > 50 {
		writeJSONError(w, http.StatusBadRequest, "Category name must not exceed 50 characters.")
		return
	}

	if req.Code != "" {
		valid := totp.Validate(strings.TrimSpace(req.Code), rawSecret)
		if !valid {
			writeJSONError(w, http.StatusBadRequest, "Live confirmation code did not match secret.")
			return
		}
	}

	encSecret, err := EncryptSecret(rawSecret, user.DEK)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to encrypt secret.")
		return
	}

	if err := insertAccount(a.db, user.ID, issuer, accountName, category, encSecret, 6, 30, "SHA1"); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Database error creating token.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiTokensDelete(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	idInt, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid token ID")
		return
	}

	if err := deleteAccount(a.db, idInt, user.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete token.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiSettingsPassword(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(req.NewPassword) < 8 {
		writeJSONError(w, http.StatusBadRequest, "New master key must be at least 8 characters.")
		return
	}

	if req.NewPassword == req.CurrentPassword {
		writeJSONError(w, http.StatusBadRequest, "New master key cannot be the same as your current master key.")
		return
	}

	dbUser, err := getUserByID(a.db, user.ID)
	if err != nil || dbUser == nil {
		writeJSONError(w, http.StatusBadRequest, "User not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Current master key is incorrect.")
		return
	}

	newSalt, _ := GenerateSalt()
	newKEK, _ := DeriveKEK(req.NewPassword, newSalt)
	newWrappedDEK, _ := WrapDEK(user.DEK, newKEK)
	newPWHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

	if err := updateUserPasswordAndDEK(a.db, user.ID, string(newPWHash), newWrappedDEK, newSalt); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update password.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiSettingsOTP(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Enabled     bool   `json:"enabled"`
		Method      string `json:"method"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	method := "none"
	if req.Enabled {
		method = strings.ToLower(strings.TrimSpace(req.Method))
		if method == "" || method == "email" {
			if !isSMTPConfigured() {
				writeJSONError(w, http.StatusBadRequest, "Email (SMTP) is not configured in the server environment.")
				return
			}
			method = "email"
		} else if method == "sms" {
			if !isSMSConfigured() {
				writeJSONError(w, http.StatusBadRequest, "Text/Phone (SMS) is not configured in the server environment.")
				return
			}
			method = "sms"
		} else {
			writeJSONError(w, http.StatusBadRequest, "Invalid 2FA delivery method.")
			return
		}
	}

	email := ""
	phone := ""
	if method == "email" {
		email = strings.TrimSpace(req.Destination)
		if email == "" || !isValidEmail(email) {
			writeJSONError(w, http.StatusBadRequest, "Please provide a valid destination email address.")
			return
		}
	} else if method == "sms" {
		phone = strings.TrimSpace(req.Destination)
		if phone == "" || !isValidPhone(phone) {
			writeJSONError(w, http.StatusBadRequest, "Please provide a valid destination phone number (at least 7 digits).")
			return
		}
	}

	if err := updateUserContactAndOTP(a.db, user.ID, email, phone, method); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update OTP settings.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiSettingsRecoveryRegenerate(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	phrase, err := GenerateMnemonicPhrase(12)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Phrase generation failed")
		return
	}

	recSalt, _ := GenerateSalt()
	recKEK, _ := DeriveKEK(NormalizePhrase(phrase), recSalt)
	recEncDEK, _ := WrapDEK(user.DEK, recKEK)

	if err := updateUserRecoveryData(a.db, user.ID, recEncDEK, recSalt, ""); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update recovery key.")
		return
	}

	sess, _ := a.getSession(r)
	sess.Values["new_recovery_phrase"] = phrase
	sess.Save(r, w)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"recoveryKey": phrase,
	})
}

func (a *App) handleApiAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	users, err := listUsersForAdmin(a.db)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve users.")
		return
	}

	type UserDTO struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		TokenCount int    `json:"tokenCount"`
		CreatedAt  string `json:"createdAt"`
	}

	dtos := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, UserDTO{
			ID:         strconv.FormatInt(u.ID, 10),
			Username:   u.Username,
			Role:       u.Role,
			TokenCount: u.AccountCount,
			CreatedAt:  u.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users": dtos,
	})
}

func (a *App) handleApiAdminUsersCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	role := req.Role
	if role != "admin" && role != "user" {
		role = "user"
	}

	if len(username) < 3 || len(password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "Username min 3 chars, password min 8 chars.")
		return
	}

	salt, _ := GenerateSalt()
	kek, _ := DeriveKEK(password, salt)
	dek, _ := GenerateDEK()
	wrappedDEK, _ := WrapDEK(dek, kek)
	pwHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userID, err := createUserWithDEK(a.db, username, string(pwHash), role, wrappedDEK, salt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to create user: "+err.Error())
		return
	}

	phrase, err := GenerateMnemonicPhrase(12)
	if err == nil {
		recSalt, _ := GenerateSalt()
		recKEK, _ := DeriveKEK(NormalizePhrase(phrase), recSalt)
		recEncDEK, _ := WrapDEK(dek, recKEK)
		_ = updateUserRecoveryData(a.db, userID, recEncDEK, recSalt, "")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (a *App) handleApiAdminUsersDelete(w http.ResponseWriter, r *http.Request) {
	currUser, err := a.getSessionUser(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	idInt, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if idInt == currUser.ID {
		writeJSONError(w, http.StatusBadRequest, "Cannot delete your own account.")
		return
	}

	if err := deleteUser(a.db, idInt); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete user.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}
