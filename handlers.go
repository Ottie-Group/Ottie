package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
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
	db        *sql.DB
	store     *sessions.CookieStore
	dekStore  *SessionStore
	tmpl      *template.Template
	serverKey []byte
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

type SessionUser struct {
	ID       int64
	Username string
	Role     string
	DEK      []byte
}

func (a *App) getSessionUser(r *http.Request) (*SessionUser, error) {
	sess, err := a.store.Get(r, sessionName)
	if err != nil || sess == nil {
		return nil, errors.New("no session")
	}

	uid, ok := sess.Values["user_id"].(int64)
	if !ok || uid == 0 {
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
	}

	return nil, errors.New("session key expired")
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCount, _ := countUsers(a.db)
		if userCount == 0 {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}

		user, err := a.getSessionUser(r)
		if err != nil || user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := a.getSessionUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Access forbidden: Admins only", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func (a *App) checkCSRF(r *http.Request, sess *sessions.Session) bool {
	want, _ := sess.Values["csrf"].(string)
	got := r.FormValue("csrf_token")
	if got == "" {
		got = r.Header.Get("X-CSRF-Token")
	}
	return want != "" && got != "" && want == got
}

// --- First-time On-Ramp / Setup ---

func (a *App) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	n, _ := countUsers(a.db)
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "setup.html", map[string]any{
		"CSRF": token,
	})
}

func (a *App) handleSetupSubmit(w http.ResponseWriter, r *http.Request) {
	n, _ := countUsers(a.db)
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form token, please retry", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if len(username) < 3 {
		a.renderSetupError(w, "Username must be at least 3 characters.")
		return
	}
	if len(password) < 8 {
		a.renderSetupError(w, "Password must be at least 8 characters long.")
		return
	}
	if password != confirmPassword {
		a.renderSetupError(w, "Passwords do not match.")
		return
	}

	// 1. Generate salt and derive KEK
	salt, err := GenerateSalt()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	kek, err := DeriveKEK(password, salt)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 2. Generate user DEK and wrap with KEK
	dek, err := GenerateDEK()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	wrappedDEK, err := WrapDEK(dek, kek)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 3. Hash password for login auth
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 4. Create admin user
	userID, err := createUserWithDEK(a.db, username, string(pwHash), "admin", wrappedDEK, salt)
	if err != nil {
		a.renderSetupError(w, "Failed to create user: "+err.Error())
		return
	}

	// 5. Generate 12-word Cryptographic Recovery Phrase & zero-knowledge backup
	phrase, err := GenerateMnemonicPhrase(12)
	if err == nil {
		recSalt, _ := GenerateSalt()
		recKEK, _ := DeriveKEK(NormalizePhrase(phrase), recSalt)
		recEncDEK, _ := WrapDEK(dek, recKEK)
		phraseHash, _ := bcrypt.GenerateFromPassword([]byte(NormalizePhrase(phrase)), bcrypt.DefaultCost)
		_ = updateUserRecoveryData(a.db, userID, recEncDEK, recSalt, string(phraseHash))
	}

	// 6. Establish session and present recovery phrase
	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	sess.Values["user_id"] = userID
	sess.Values["username"] = username
	sess.Values["role"] = "admin"
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	sess.Values["new_recovery_phrase"] = phrase
	delete(sess.Values, "csrf")
	sess.Save(r, w)

	updateLastLogin(a.db, userID)
	http.Redirect(w, r, "/setup/recovery", http.StatusSeeOther)
}

func (a *App) handleSetupRecoveryPage(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	phrase, _ := sess.Values["new_recovery_phrase"].(string)
	if phrase == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	words := strings.Fields(phrase)
	a.tmpl.ExecuteTemplate(w, "setup_recovery.html", map[string]any{
		"Username": user.Username,
		"Phrase":   phrase,
		"Words":    words,
		"CSRF":     token,
	})
}

func (a *App) handleSetupRecoverySubmit(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	delete(sess.Values, "new_recovery_phrase")
	sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) renderSetupError(w http.ResponseWriter, msg string) {
	token := newCSRFToken()
	a.tmpl.ExecuteTemplate(w, "setup.html", map[string]any{
		"CSRF":  token,
		"Error": msg,
	})
}

// --- Login & Logout ---

func (a *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	n, _ := countUsers(a.db)
	if n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	if user, err := a.getSessionUser(r); err == nil && user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)
	a.tmpl.ExecuteTemplate(w, "login.html", map[string]any{"CSRF": token})
}

func generateNumericOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	num := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", num), nil
}

func (a *App) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	ip := clientIP(r)
	failures, lockedUntil, _ := loginFailures(a.db, ip)
	if lockedUntil != nil && time.Now().Before(*lockedUntil) {
		a.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"CSRF":  newCSRFToken(),
			"Error": "Too many failed attempts. Try again in a few minutes.",
		})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := getUserByUsername(a.db, username)
	dummyHash := "$2a$10$CwTycUXWue0Thq9StjUM0uJ8Pg7YvVN3Xz.KM8XvL9nq7DkTX8/9K"
	hashToCheck := dummyHash
	if err == nil {
		hashToCheck = user.PasswordHash
	}
	pwErr := bcrypt.CompareHashAndPassword([]byte(hashToCheck), []byte(password))

	if err != nil || pwErr != nil {
		recordLoginFailure(a.db, ip)
		_ = failures
		a.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"CSRF":  newCSRFToken(),
			"Error": "Invalid username or password.",
		})
		return
	}

	// Derive KEK and unwrap user's DEK
	kek, err := DeriveKEK(password, user.Salt)
	if err != nil {
		http.Error(w, "key derivation failure", http.StatusInternalServerError)
		return
	}

	dek, err := UnwrapDEK(user.EncDEK, kek)
	if err != nil {
		recordLoginFailure(a.db, ip)
		a.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"CSRF":  newCSRFToken(),
			"Error": "Invalid credentials or corrupted vault.",
		})
		return
	}

	clearLoginFailures(a.db, ip)

	// If user has Email or SMS OTP login verification enabled:
	if user.OTPMethod == "email" || user.OTPMethod == "sms" {
		otpCode, err := generateNumericOTP()
		if err != nil {
			http.Error(w, "otp generation error", http.StatusInternalServerError)
			return
		}

		encPendingDEK, _ := EncryptAESGCM(dek, a.serverKey)
		sess.Values["pending_otp_user_id"] = user.ID
		sess.Values["pending_otp_code"] = otpCode
		sess.Values["pending_otp_enc_dek"] = encPendingDEK
		sess.Values["pending_otp_expiry"] = time.Now().Add(5 * time.Minute).Unix()

		dest := user.Email
		if user.OTPMethod == "sms" {
			dest = user.Phone
		}
		sess.Values["pending_otp_dest"] = dest
		sess.Values["pending_otp_method"] = user.OTPMethod

		// Log OTP delivery for self-hosted visibility
		fmt.Printf("\n========================================\n")
		fmt.Printf("📬 [Ottie 2FA Delivery] User: %s\n", user.Username)
		fmt.Printf("   Method: %s -> %s\n", strings.ToUpper(user.OTPMethod), dest)
		fmt.Printf("   One-Time Passcode: %s (Expires in 5 mins)\n", otpCode)
		fmt.Printf("========================================\n\n")

		delete(sess.Values, "csrf")
		sess.Save(r, w)
		http.Redirect(w, r, "/login/otp", http.StatusSeeOther)
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	sess.Values["user_id"] = user.ID
	sess.Values["username"] = user.Username
	sess.Values["role"] = user.Role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	delete(sess.Values, "csrf")
	sess.Save(r, w)

	updateLastLogin(a.db, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLoginOTPPage(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	pendingUID, ok := sess.Values["pending_otp_user_id"].(int64)
	if !ok || pendingUID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := getUserByID(a.db, pendingUID)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	dest, _ := sess.Values["pending_otp_dest"].(string)
	method, _ := sess.Values["pending_otp_method"].(string)

	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "login_otp.html", map[string]any{
		"Username": user.Username,
		"Method":   method,
		"Dest":     dest,
		"CSRF":     token,
	})
}

func (a *App) handleLoginOTPSubmit(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	pendingUID, ok := sess.Values["pending_otp_user_id"].(int64)
	encPendingDEK, okDek := sess.Values["pending_otp_enc_dek"].(string)
	expectedOTP, _ := sess.Values["pending_otp_code"].(string)
	expiry, _ := sess.Values["pending_otp_expiry"].(int64)

	if !ok || !okDek || pendingUID == 0 || encPendingDEK == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := getUserByID(a.db, pendingUID)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))
	recoveryKey := strings.TrimSpace(r.FormValue("recovery_code"))

	valid := false
	if code != "" {
		if time.Now().Unix() <= expiry && code == expectedOTP {
			valid = true
		}
	} else if recoveryKey != "" {
		if VerifyRecoveryCode(recoveryKey, user.RecoveryCodeHash) {
			valid = true
		}
	}

	if !valid {
		ip := clientIP(r)
		recordLoginFailure(a.db, ip)
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)

		dest, _ := sess.Values["pending_otp_dest"].(string)
		method, _ := sess.Values["pending_otp_method"].(string)

		errMsg := "Invalid verification code or expired session."
		if recoveryKey != "" {
			errMsg = "Invalid Emergency Recovery Key."
		}

		a.tmpl.ExecuteTemplate(w, "login_otp.html", map[string]any{
			"Username": user.Username,
			"Method":   method,
			"Dest":     dest,
			"CSRF":     token,
			"Error":    errMsg,
		})
		return
	}

	dek, err := DecryptAESGCM(encPendingDEK, a.serverKey)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)

	sess.Values["user_id"] = user.ID
	sess.Values["username"] = user.Username
	sess.Values["role"] = user.Role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encPendingDEK
	delete(sess.Values, "pending_otp_user_id")
	delete(sess.Values, "pending_otp_code")
	delete(sess.Values, "pending_otp_enc_dek")
	delete(sess.Values, "pending_otp_expiry")
	delete(sess.Values, "pending_otp_dest")
	delete(sess.Values, "pending_otp_method")
	delete(sess.Values, "csrf")
	sess.Save(r, w)

	updateLastLogin(a.db, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleRecoverPage(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
		"CSRF": token,
	})
}

func (a *App) handleRecoverSubmit(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	rawPhrase := strings.TrimSpace(r.FormValue("recovery_phrase"))
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if username == "" || rawPhrase == "" {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "Please provide your username and 12-word recovery phrase.",
		})
		return
	}

	if len(newPassword) < 8 {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "New password must be at least 8 characters long.",
		})
		return
	}

	if newPassword != confirmPassword {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "Passwords do not match.",
		})
		return
	}

	user, err := getUserByUsername(a.db, username)
	if err != nil || user == nil || user.RecoveryEncDEK == "" || user.RecoverySalt == "" {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "Invalid username or recovery phrase.",
		})
		return
	}

	normPhrase := NormalizePhrase(rawPhrase)
	recKEK, err := DeriveKEK(normPhrase, user.RecoverySalt)
	if err != nil {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "Cryptographic error during recovery derivation.",
		})
		return
	}

	// Zero-knowledge recovery: unwrap user's DEK with recovery phrase key
	dek, err := UnwrapDEK(user.RecoveryEncDEK, recKEK)
	if err != nil {
		token := newCSRFToken()
		sess.Values["csrf"] = token
		sess.Save(r, w)
		a.tmpl.ExecuteTemplate(w, "recover.html", map[string]any{
			"CSRF":     token,
			"Username": username,
			"Error":    "Invalid recovery phrase. Please verify all 12 words and order.",
		})
		return
	}

	// Re-encrypt DEK with the new password
	newSalt, err := GenerateSalt()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	newKEK, err := DeriveKEK(newPassword, newSalt)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	newWrappedDEK, err := WrapDEK(dek, newKEK)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	newPwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	if err := updateUserPasswordAndDEK(a.db, user.ID, string(newPwHash), newWrappedDEK, newSalt); err != nil {
		http.Error(w, "database update error", http.StatusInternalServerError)
		return
	}

	// Successfully recovered! Establish active session and enter den
	sid := newCSRFToken()
	a.dekStore.Set(sid, dek)
	encCookieDEK, _ := EncryptAESGCM(dek, a.serverKey)

	sess.Values["user_id"] = user.ID
	sess.Values["username"] = user.Username
	sess.Values["role"] = user.Role
	sess.Values["session_token"] = sid
	sess.Values["enc_dek"] = encCookieDEK
	delete(sess.Values, "csrf")
	sess.Save(r, w)

	updateLastLogin(a.db, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.store.Get(r, sessionName)
	if sid, ok := sess.Values["session_token"].(string); ok && sid != "" {
		a.dekStore.Delete(sid)
	}
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- Dashboard & Codes API ---

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	accounts, err := listAccounts(a.db, user.ID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	categories, _ := listCategories(a.db, user.ID)

	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "dashboard.html", map[string]any{
		"User":       user,
		"Accounts":   accounts,
		"Categories": categories,
		"CSRF":       token,
		"IsAdmin":    user.Role == "admin",
	})
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

// --- Add Account ---

func (a *App) handleAddPage(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)
	a.tmpl.ExecuteTemplate(w, "add.html", map[string]any{
		"User":    user,
		"CSRF":    token,
		"IsAdmin": user.Role == "admin",
	})
}

func (a *App) handleAddSubmit(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	raw := strings.TrimSpace(r.FormValue("otpauth_or_secret"))
	issuerOverride := strings.TrimSpace(r.FormValue("issuer"))
	nameOverride := strings.TrimSpace(r.FormValue("account_name"))
	category := strings.TrimSpace(r.FormValue("category"))
	confirmCode := strings.TrimSpace(r.FormValue("confirm_code"))

	if category == "" {
		category = "Personal"
	}

	var key *otp.Key
	if strings.HasPrefix(raw, "otpauth://") {
		key, err = otp.NewKeyFromURL(raw)
		if err != nil {
			a.renderAddError(w, user, "Could not parse that otpauth:// URI.")
			return
		}
	} else {
		secret := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
		if secret == "" || issuerOverride == "" || nameOverride == "" {
			a.renderAddError(w, user, "Provide an otpauth:// URI, or a secret plus issuer and account name.")
			return
		}
		key, err = otp.NewKeyFromURL("otpauth://totp/" + issuerOverride + ":" + nameOverride +
			"?secret=" + secret + "&issuer=" + issuerOverride)
		if err != nil {
			a.renderAddError(w, user, "Invalid secret key.")
			return
		}
	}

	issuer := key.Issuer()
	if issuerOverride != "" {
		issuer = issuerOverride
	}
	name := key.AccountName()
	if nameOverride != "" {
		name = nameOverride
	}
	if issuer == "" {
		issuer = "Account"
	}
	if name == "" {
		name = "TOTP"
	}

	digits := 6
	if d := key.Digits(); d != 0 {
		digits = int(d)
	}
	period := 30
	if p := key.Period(); p != 0 {
		period = int(p)
	}
	algorithm := "SHA1"
	if alg := key.Algorithm(); alg.String() != "" {
		algorithm = alg.String()
	}

	// Verify confirmation code matches current live code (skew of 1 for clock drift)
	valid, _ := totp.ValidateCustom(confirmCode, key.Secret(), time.Now(), totp.ValidateOpts{
		Period: uint(period), Digits: otp.Digits(digits), Algorithm: algoFromString(algorithm), Skew: 1,
	})
	if !valid {
		a.renderAddError(w, user, "That confirmation code didn't match. Check the secret and try again.")
		return
	}

	// Encrypt secret with user's private DEK
	encSecret, err := EncryptSecret(key.Secret(), user.DEK)
	if err != nil {
		http.Error(w, "encryption failure", http.StatusInternalServerError)
		return
	}

	if err := insertAccount(a.db, user.ID, issuer, name, category, encSecret, digits, period, algorithm); err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) renderAddError(w http.ResponseWriter, user *SessionUser, msg string) {
	token := newCSRFToken()
	a.tmpl.ExecuteTemplate(w, "add.html", map[string]any{
		"User":    user,
		"CSRF":    token,
		"Error":   msg,
		"IsAdmin": user.Role == "admin",
	})
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	deleteAccount(a.db, user.ID, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- Admin Panel (User Management) ---

func (a *App) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	user, _ := a.getSessionUser(r)
	users, err := listUsersForAdmin(a.db)
	if err != nil {
		http.Error(w, "server error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	totalUsers, totalAccounts, _ := getSystemStats(a.db)

	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "admin.html", map[string]any{
		"User":          user,
		"Users":         users,
		"TotalUsers":    totalUsers,
		"TotalAccounts": totalAccounts,
		"CSRF":          token,
		"IsAdmin":       true,
	})
}

func (a *App) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	user, _ := a.getSessionUser(r)
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	role := r.FormValue("role")
	initialPassword := r.FormValue("password")

	if role != "admin" {
		role = "user"
	}
	if len(username) < 3 {
		a.renderAdminError(w, user, "Username must be at least 3 characters.")
		return
	}
	if len(initialPassword) < 8 {
		a.renderAdminError(w, user, "Password must be at least 8 characters.")
		return
	}

	// 1. Generate salt and derive KEK for new user
	salt, err := GenerateSalt()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	kek, err := DeriveKEK(initialPassword, salt)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 2. Generate unique DEK for the new user and wrap with KEK
	dek, err := GenerateDEK()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	wrappedDEK, err := WrapDEK(dek, kek)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 3. Hash password
	pwHash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// 4. Create user
	newUserID, err := createUserWithDEK(a.db, username, string(pwHash), role, wrappedDEK, salt)
	if err != nil {
		a.renderAdminError(w, user, "Failed to create user (username may already exist): "+err.Error())
		return
	}

	// 5. Generate 12-word Cryptographic Recovery Phrase for new user
	phrase, err := GenerateMnemonicPhrase(12)
	if err == nil {
		recSalt, _ := GenerateSalt()
		recKEK, _ := DeriveKEK(NormalizePhrase(phrase), recSalt)
		recEncDEK, _ := WrapDEK(dek, recKEK)
		phraseHash, _ := bcrypt.GenerateFromPassword([]byte(NormalizePhrase(phrase)), bcrypt.DefaultCost)
		_ = updateUserRecoveryData(a.db, newUserID, recEncDEK, recSalt, string(phraseHash))
	}

	http.Redirect(w, r, "/admin?success=user_created", http.StatusSeeOther)
}

func (a *App) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := a.getSessionUser(r)
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	targetIDStr := r.FormValue("user_id")
	targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if targetID == currentUser.ID {
		a.renderAdminError(w, currentUser, "You cannot delete your own admin account from here.")
		return
	}

	// Delete user and cascade all their accounts
	if err := deleteUser(a.db, targetID); err != nil {
		a.renderAdminError(w, currentUser, "Error deleting user: "+err.Error())
		return
	}

	http.Redirect(w, r, "/admin?success=user_deleted", http.StatusSeeOther)
}

func (a *App) renderAdminError(w http.ResponseWriter, user *SessionUser, msg string) {
	users, _ := listUsersForAdmin(a.db)
	totalUsers, totalAccounts, _ := getSystemStats(a.db)
	token := newCSRFToken()
	a.tmpl.ExecuteTemplate(w, "admin.html", map[string]any{
		"User":          user,
		"Users":         users,
		"TotalUsers":    totalUsers,
		"TotalAccounts": totalAccounts,
		"CSRF":          token,
		"Error":         msg,
		"IsAdmin":       true,
	})
}

// --- Settings & Password Change & OTP / Recovery Keys ---

func (a *App) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	dbUser, err := getUserByID(a.db, user.ID)
	if err != nil {
		http.Error(w, "user error", http.StatusInternalServerError)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	token := newCSRFToken()
	sess.Values["csrf"] = token
	sess.Save(r, w)

	a.tmpl.ExecuteTemplate(w, "settings.html", map[string]any{
		"User":              user,
		"RecoveryKeyActive": dbUser.RecoveryEncDEK != "",
		"Email":             dbUser.Email,
		"Phone":             dbUser.Phone,
		"OTPMethod":         dbUser.OTPMethod,
		"CSRF":              token,
		"IsAdmin":           user.Role == "admin",
	})
}

func (a *App) handleSettingsSaveOTP(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	method := strings.TrimSpace(r.FormValue("otp_method"))

	if method == "email" && email == "" {
		a.renderSettingsMessage(w, user, "Please provide an email address for Email OTP delivery.", "")
		return
	}
	if method == "sms" && phone == "" {
		a.renderSettingsMessage(w, user, "Please provide a phone number for SMS OTP delivery.", "")
		return
	}

	if err := updateUserContactAndOTP(a.db, user.ID, email, phone, method); err != nil {
		a.renderSettingsMessage(w, user, "Failed to update login verification settings: "+err.Error(), "")
		return
	}

	msg := "Login verification settings updated successfully."
	if method == "email" {
		msg = "Email OTP login verification is now enabled for " + email + "."
	} else if method == "sms" {
		msg = "SMS OTP login verification is now enabled for " + phone + "."
	}
	a.renderSettingsMessage(w, user, "", msg)
}

func (a *App) handleRegenerateRecovery(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	password := r.FormValue("password")
	dbUser, err := getUserByID(a.db, user.ID)
	if err != nil {
		http.Error(w, "user error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		a.renderSettingsMessage(w, user, "Incorrect password. Recovery phrase was not changed.", "")
		return
	}

	phrase, err := GenerateMnemonicPhrase(12)
	if err != nil {
		a.renderSettingsMessage(w, user, "Failed to generate new recovery phrase.", "")
		return
	}

	recSalt, err := GenerateSalt()
	if err != nil {
		a.renderSettingsMessage(w, user, "Crypto failure.", "")
		return
	}
	recKEK, err := DeriveKEK(NormalizePhrase(phrase), recSalt)
	if err != nil {
		a.renderSettingsMessage(w, user, "Crypto failure.", "")
		return
	}
	recEncDEK, err := WrapDEK(user.DEK, recKEK)
	if err != nil {
		a.renderSettingsMessage(w, user, "Crypto failure.", "")
		return
	}
	phraseHash, err := bcrypt.GenerateFromPassword([]byte(NormalizePhrase(phrase)), bcrypt.DefaultCost)
	if err != nil {
		a.renderSettingsMessage(w, user, "Crypto failure.", "")
		return
	}

	if err := updateUserRecoveryData(a.db, user.ID, recEncDEK, recSalt, string(phraseHash)); err != nil {
		a.renderSettingsMessage(w, user, "Database update error: "+err.Error(), "")
		return
	}

	token := newCSRFToken()
	words := strings.Fields(phrase)
	a.tmpl.ExecuteTemplate(w, "settings.html", map[string]any{
		"User":                user,
		"RecoveryKeyActive":   true,
		"NewRecoveryPhrase":   phrase,
		"Words":               words,
		"Email":               dbUser.Email,
		"Phone":               dbUser.Phone,
		"OTPMethod":           dbUser.OTPMethod,
		"CSRF":                token,
		"Success":             "New 12-Word Recovery Phrase generated! Make sure to write it down safely.",
		"IsAdmin":             user.Role == "admin",
	})
}

func (a *App) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
	if !a.checkCSRF(r, sess) {
		http.Error(w, "invalid or expired form, please retry", http.StatusForbidden)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	dbUser, err := getUserByID(a.db, user.ID)
	if err != nil {
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(currentPassword)); err != nil {
		a.renderSettingsMessage(w, user, "Current password is incorrect.", "")
		return
	}

	if len(newPassword) < 8 {
		a.renderSettingsMessage(w, user, "New password must be at least 8 characters.", "")
		return
	}
	if newPassword != confirmPassword {
		a.renderSettingsMessage(w, user, "New passwords do not match.", "")
		return
	}

	// Derive new KEK with fresh salt
	newSalt, err := GenerateSalt()
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}
	newKEK, err := DeriveKEK(newPassword, newSalt)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// Re-wrap the existing DEK with the new KEK
	newWrappedDEK, err := WrapDEK(user.DEK, newKEK)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "crypto failure", http.StatusInternalServerError)
		return
	}

	if err := updateUserPasswordAndDEK(a.db, user.ID, string(newHash), newWrappedDEK, newSalt); err != nil {
		http.Error(w, "database update error", http.StatusInternalServerError)
		return
	}

	a.renderSettingsMessage(w, user, "", "Password updated successfully!")
}

func (a *App) handleExportVault(w http.ResponseWriter, r *http.Request) {
	user, err := a.getSessionUser(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, _ := a.store.Get(r, sessionName)
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
		OttieVersion: "1.0.0",
		ExportDate:   time.Now().UTC(),
		Username:     user.Username,
		Accounts:     items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"ottie-backup-%s-%s.json\"", user.Username, time.Now().Format("20060102-150405")))
	json.NewEncoder(w).Encode(payload)
}

func (a *App) renderSettingsMessage(w http.ResponseWriter, user *SessionUser, errMsg, successMsg string) {
	dbUser, _ := getUserByID(a.db, user.ID)
	email, phone, otpMethod := "", "", "none"
	hasRec := false
	if dbUser != nil {
		hasRec = dbUser.RecoveryCodeHash != ""
		email = dbUser.Email
		phone = dbUser.Phone
		otpMethod = dbUser.OTPMethod
	}

	token := newCSRFToken()
	a.tmpl.ExecuteTemplate(w, "settings.html", map[string]any{
		"User":              user,
		"RecoveryKeyActive": hasRec,
		"Email":             email,
		"Phone":             phone,
		"OTPMethod":         otpMethod,
		"CSRF":              token,
		"Error":             errMsg,
		"Success":           successMsg,
		"IsAdmin":           user.Role == "admin",
	})
}
