package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func setupTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_app.db")
	db, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}

	sessionKey := []byte("12345678901234567890123456789012")
	serverKey := []byte("abcdefghijklmnopqrstuvwxyz123456")
	store := sessions.NewCookieStore(sessionKey)
	dekStore := newSessionStore()

	return &App{
		db:          db,
		store:       store,
		dekStore:    dekStore,
		serverKey:   serverKey,
		rateLimiter: NewRateLimiter(100, time.Minute),
	}
}

func testingTime() time.Time {
	return time.Unix(1700000000, 0)
}

func TestFullWorkflow(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.example.com")
	defer os.Unsetenv("SMTP_HOST")

	app := setupTestApp(t)
	defer app.db.Close()

	// Initial State: 0 users, setup needed
	reqMe := httptest.NewRequest("GET", "/api/me", nil)
	recMe := httptest.NewRecorder()
	app.handleApiMe(recMe, reqMe)
	if recMe.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/me, got %d", recMe.Code)
	}
	var meRes map[string]any
	json.NewDecoder(recMe.Body).Decode(&meRes)
	if meRes["setupNeeded"] != true {
		t.Fatalf("expected setupNeeded == true, got %v", meRes["setupNeeded"])
	}

	// Setup Admin Account via REST API
	setupPayload := map[string]string{
		"username": "admin_otto",
		"password": "super-secret-password-123",
	}
	setupBytes, _ := json.Marshal(setupPayload)
	reqSetup := httptest.NewRequest("POST", "/api/setup", bytes.NewReader(setupBytes))
	reqSetup.Header.Set("Content-Type", "application/json")
	recSetup := httptest.NewRecorder()
	app.handleApiSetup(recSetup, reqSetup)

	if recSetup.Code != http.StatusOK {
		t.Fatalf("expected 200 on setup, got %d, body: %s", recSetup.Code, recSetup.Body.String())
	}
	var setupRes map[string]any
	json.NewDecoder(recSetup.Body).Decode(&setupRes)
	if setupRes["success"] != true {
		t.Fatalf("expected setup success == true")
	}
	words := setupRes["words"].([]any)
	if len(words) != 12 {
		t.Fatalf("expected 12 recovery words, got %d", len(words))
	}

	// Confirm setup
	adminCookie := recSetup.Header().Get("Set-Cookie")
	reqConfirm := httptest.NewRequest("POST", "/api/setup/confirm", strings.NewReader(`{"acknowledged":true}`))
	reqConfirm.Header.Set("Content-Type", "application/json")
	reqConfirm.Header.Set("Cookie", adminCookie)
	recConfirm := httptest.NewRecorder()
	app.handleApiSetupConfirm(recConfirm, reqConfirm)
	if recConfirm.Code != http.StatusOK {
		t.Fatalf("expected 200 on setup confirm, got %d", recConfirm.Code)
	}

	// Admin creates standard user "charlie"
	createUserPayload := map[string]string{
		"username": "charlie",
		"password": "charlies-password-456",
		"role":     "user",
	}
	createBytes, _ := json.Marshal(createUserPayload)
	reqCreate := httptest.NewRequest("POST", "/api/admin/users/create", bytes.NewReader(createBytes))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Cookie", adminCookie)
	recCreate := httptest.NewRecorder()
	app.handleApiAdminUsersCreate(recCreate, reqCreate)

	if recCreate.Code != http.StatusOK {
		t.Fatalf("expected 200 on create user, got %d, body: %s", recCreate.Code, recCreate.Body.String())
	}

	charlie, err := getUserByUsername(app.db, "charlie")
	if err != nil || charlie.Role != "user" {
		t.Fatalf("expected user charlie created with role 'user', got %v", err)
	}

	// Charlie logs in via REST API
	loginPayload := map[string]string{
		"username": "charlie",
		"password": "charlies-password-456",
	}
	loginBytes, _ := json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBytes))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	app.handleApiAuthLogin(recLogin, reqLogin)

	if recLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 on charlie login, got %d, body: %s", recLogin.Code, recLogin.Body.String())
	}
	charlieCookie := recLogin.Header().Get("Set-Cookie")

	// Charlie adds a TOTP secret via REST API
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("GenerateCodeCustom failed: %v", err)
	}

	addPayload := map[string]string{
		"secret":      secret,
		"issuer":      "GitHub",
		"accountName": "charlie@github",
		"category":    "Work",
		"code":        code,
	}
	addBytes, _ := json.Marshal(addPayload)
	reqAdd := httptest.NewRequest("POST", "/api/tokens", bytes.NewReader(addBytes))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.Header.Set("Cookie", charlieCookie)
	recAdd := httptest.NewRecorder()
	app.handleApiTokens(recAdd, reqAdd)

	if recAdd.Code != http.StatusOK {
		t.Fatalf("expected 200 on add token, got %d, body: %s", recAdd.Code, recAdd.Body.String())
	}

	// Test TOTP Codes API
	reqCodes := httptest.NewRequest("GET", "/api/codes", nil)
	reqCodes.Header.Set("Cookie", charlieCookie)
	recCodes := httptest.NewRecorder()
	app.handleCodesAPI(recCodes, reqCodes)

	if recCodes.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/codes, got %d", recCodes.Code)
	}
	var codesRes []map[string]any
	json.NewDecoder(recCodes.Body).Decode(&codesRes)
	if len(codesRes) != 1 || codesRes[0]["issuer"] != "GitHub" {
		t.Fatalf("expected 1 token for GitHub, got %v", codesRes)
	}

	// Charlie changes password via REST API
	pwPayload := map[string]string{
		"currentPassword": "charlies-password-456",
		"newPassword":     "new-charlie-pass-789",
	}
	pwBytes, _ := json.Marshal(pwPayload)
	reqPw := httptest.NewRequest("POST", "/api/settings/password", bytes.NewReader(pwBytes))
	reqPw.Header.Set("Content-Type", "application/json")
	reqPw.Header.Set("Cookie", charlieCookie)
	recPw := httptest.NewRecorder()
	app.handleApiSettingsPassword(recPw, reqPw)

	if recPw.Code != http.StatusOK {
		t.Fatalf("expected 200 on password update, got %d", recPw.Code)
	}

	// Charlie logs in with NEW password
	newLoginPayload := map[string]string{
		"username": "charlie",
		"password": "new-charlie-pass-789",
	}
	newLoginBytes, _ := json.Marshal(newLoginPayload)
	reqNewLogin := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(newLoginBytes))
	reqNewLogin.Header.Set("Content-Type", "application/json")
	recNewLogin := httptest.NewRecorder()
	app.handleApiAuthLogin(recNewLogin, reqNewLogin)

	if recNewLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 on login with new password, got %d", recNewLogin.Code)
	}
	newCharlieCookie := recNewLogin.Header().Get("Set-Cookie")

	// Charlie exports vault
	reqExportSess := httptest.NewRequest("GET", "/api/me", nil)
	reqExportSess.Header.Set("Cookie", newCharlieCookie)
	recExportSess := httptest.NewRecorder()
	app.handleApiMe(recExportSess, reqExportSess)
	exportCookie := recExportSess.Header().Get("Set-Cookie")
	if exportCookie == "" {
		exportCookie = newCharlieCookie
	}

	sessExport, _ := app.store.Get(reqExportSess, sessionName)
	csrfToken, _ := sessExport.Values["csrf"].(string)

	reqExport := httptest.NewRequest("POST", "/api/settings/export", nil)
	reqExport.Header.Set("Cookie", exportCookie)
	reqExport.Header.Set("X-CSRF-Token", csrfToken)
	recExport := httptest.NewRecorder()
	app.handleExportVault(recExport, reqExport)

	if recExport.Code != http.StatusOK {
		t.Fatalf("expected 200 on vault export, got %d", recExport.Code)
	}
	if !strings.Contains(recExport.Body.String(), "GitHub") {
		t.Fatalf("expected export to contain GitHub, got %s", recExport.Body.String())
	}

	// Charlie configures Email 2FA
	otpPayload := map[string]any{
		"enabled":     true,
		"method":      "email",
		"destination": "charlie@gmail.com",
	}
	otpBytes, _ := json.Marshal(otpPayload)
	reqOTP := httptest.NewRequest("POST", "/api/settings/otp", bytes.NewReader(otpBytes))
	reqOTP.Header.Set("Content-Type", "application/json")
	reqOTP.Header.Set("Cookie", exportCookie)
	recOTP := httptest.NewRecorder()
	app.handleApiSettingsOTP(recOTP, reqOTP)

	if recOTP.Code != http.StatusOK {
		t.Fatalf("expected 200 on settings OTP, got %d", recOTP.Code)
	}

	// Charlie logs in -> triggers 2FA
	req2FALogin := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(newLoginBytes))
	req2FALogin.Header.Set("Content-Type", "application/json")
	rec2FALogin := httptest.NewRecorder()
	app.handleApiAuthLogin(rec2FALogin, req2FALogin)

	var res2FA map[string]any
	json.NewDecoder(rec2FALogin.Body).Decode(&res2FA)
	if res2FA["requires2FA"] != true {
		t.Fatalf("expected requires2FA == true, got %v", res2FA)
	}

	pendingCookie := rec2FALogin.Header().Get("Set-Cookie")
	reqSess := httptest.NewRequest("GET", "/api/me", nil)
	reqSess.Header.Set("Cookie", pendingCookie)
	sess, _ := app.store.Get(reqSess, sessionName)
	otpCode, _ := sess.Values["pending_otp_code"].(string)

	verifyPayload := map[string]string{"code": otpCode}
	verifyBytes, _ := json.Marshal(verifyPayload)
	reqVerify := httptest.NewRequest("POST", "/api/auth/verify-2fa", bytes.NewReader(verifyBytes))
	reqVerify.Header.Set("Content-Type", "application/json")
	reqVerify.Header.Set("Cookie", pendingCookie)
	recVerify := httptest.NewRecorder()
	app.handleApiAuthVerify2FA(recVerify, reqVerify)

	if recVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2FA verify, got %d", recVerify.Code)
	}

	// Regenerate Recovery Phrase
	finalCookie := recVerify.Header().Get("Set-Cookie")
	reqRegen := httptest.NewRequest("POST", "/api/settings/recovery/regenerate", nil)
	reqRegen.Header.Set("Cookie", finalCookie)
	recRegen := httptest.NewRecorder()
	app.handleApiSettingsRecoveryRegenerate(recRegen, reqRegen)

	if recRegen.Code != http.StatusOK {
		t.Fatalf("expected 200 on recovery regenerate, got %d", recRegen.Code)
	}
	var regenRes map[string]any
	json.NewDecoder(recRegen.Body).Decode(&regenRes)
	recoveryPhrase, _ := regenRes["recoveryKey"].(string)
	if len(strings.Fields(recoveryPhrase)) != 12 {
		t.Fatalf("expected 12 recovery words from regenerate, got: %q", recoveryPhrase)
	}

	// Recover Account with 12 words
	recoverPayload := map[string]string{
		"username":    "charlie",
		"recoveryKey": recoveryPhrase,
		"newPassword": "recovered-password-999",
	}
	recoverBytes, _ := json.Marshal(recoverPayload)
	reqRecover := httptest.NewRequest("POST", "/api/auth/recover", bytes.NewReader(recoverBytes))
	reqRecover.Header.Set("Content-Type", "application/json")
	recRecover := httptest.NewRecorder()
	app.handleApiAuthRecover(recRecover, reqRecover)

	if recRecover.Code != http.StatusOK {
		t.Fatalf("expected 200 on recover, got %d, body: %s", recRecover.Code, recRecover.Body.String())
	}

	// Delete Token
	recoveredCookie := recRecover.Header().Get("Set-Cookie")
	if recoveredCookie == "" {
		recoveredCookie = finalCookie
	}
	tokenID := int64(codesRes[0]["id"].(float64))
	deleteTokPayload := map[string]string{"id": strconv.FormatInt(tokenID, 10)}
	delTokBytes, _ := json.Marshal(deleteTokPayload)
	reqDelTok := httptest.NewRequest("POST", "/api/tokens/delete", bytes.NewReader(delTokBytes))
	reqDelTok.Header.Set("Content-Type", "application/json")
	reqDelTok.Header.Set("Cookie", recoveredCookie)
	recDelTok := httptest.NewRecorder()
	app.handleApiTokensDelete(recDelTok, reqDelTok)

	if recDelTok.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete token, got %d, body: %s", recDelTok.Code, recDelTok.Body.String())
	}

	charlieUser, _ := getUserByUsername(app.db, "charlie")
	remainingAccounts, err := listAccounts(app.db, charlieUser.ID)
	if err != nil {
		t.Fatalf("failed to list accounts after delete: %v", err)
	}
	if len(remainingAccounts) != 0 {
		t.Fatalf("expected 0 accounts after deletion, found %d", len(remainingAccounts))
	}
}
