package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

func setupTestApp(t *testing.T) *App {
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
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	return &App{
		db:        db,
		store:     store,
		dekStore:  dekStore,
		tmpl:      tmpl,
		serverKey: serverKey,
	}
}

func TestFullWorkflow(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	// 1. Initial State: Count is 0
	n, err := countUsers(app.db)
	if err != nil || n != 0 {
		t.Fatalf("expected 0 users initially, got %d", n)
	}

	// 2. First-time Setup: Create Admin
	setupForm := url.Values{
		"csrf_token":       {""},
		"username":         {"admin_otto"},
		"password":         {"super-secret-password-123"},
		"confirm_password": {"super-secret-password-123"},
	}

	// Get setup page to establish CSRF
	req := httptest.NewRequest("GET", "/setup", nil)
	rec := httptest.NewRecorder()
	app.handleSetupPage(rec, req)

	// Extract cookie and CSRF
	cookie := rec.Header().Get("Set-Cookie")
	reqPost := httptest.NewRequest("POST", "/setup", strings.NewReader(setupForm.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Cookie", cookie)

	sess, _ := app.store.Get(reqPost, sessionName)
	csrf, _ := sess.Values["csrf"].(string)
	setupForm.Set("csrf_token", csrf)

	reqPost = httptest.NewRequest("POST", "/setup", strings.NewReader(setupForm.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Cookie", cookie)
	recPost := httptest.NewRecorder()

	app.handleSetupSubmit(recPost, reqPost)

	if recPost.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after setup, got %d", recPost.Code)
	}

	adminUser, err := getUserByUsername(app.db, "admin_otto")
	if err != nil || adminUser.Role != "admin" {
		t.Fatalf("expected admin user created with role 'admin', got err: %v", err)
	}

	// 3. Admin creates standard user "charlie"
	cookie = recPost.Header().Get("Set-Cookie")
	reqAdmin := httptest.NewRequest("GET", "/admin", nil)
	reqAdmin.Header.Set("Cookie", cookie)
	recAdmin := httptest.NewRecorder()
	app.handleAdminPage(recAdmin, reqAdmin)
	adminCookie := recAdmin.Header().Get("Set-Cookie")
	if adminCookie == "" {
		adminCookie = cookie
	}

	reqAdminSess := httptest.NewRequest("GET", "/admin", nil)
	reqAdminSess.Header.Set("Cookie", adminCookie)
	sess, _ = app.store.Get(reqAdminSess, sessionName)
	csrfToken, _ := sess.Values["csrf"].(string)

	createForm := url.Values{
		"csrf_token": {csrfToken},
		"username":   {"charlie"},
		"role":       {"user"},
		"password":   {"charlies-password-456"},
	}
	reqCreate := httptest.NewRequest("POST", "/admin/users/create", strings.NewReader(createForm.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreate.Header.Set("Cookie", adminCookie)
	recCreate := httptest.NewRecorder()

	app.handleAdminCreateUser(recCreate, reqCreate)
	if recCreate.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after admin user create, got %d, body: %s", recCreate.Code, recCreate.Body.String())
	}

	charlie, err := getUserByUsername(app.db, "charlie")
	if err != nil || charlie.Role != "user" {
		t.Fatalf("expected user charlie created with role 'user', got %v", err)
	}

	// 4. Charlie logs in
	loginForm := url.Values{
		"csrf_token": {"login_csrf"},
		"username":   {"charlie"},
		"password":   {"charlies-password-456"},
	}
	reqLogin := httptest.NewRequest("POST", "/login", strings.NewReader(loginForm.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sessLogin, _ := app.store.Get(reqLogin, sessionName)
	sessLogin.Values["csrf"] = "login_csrf"
	sessLogin.Save(reqLogin, httptest.NewRecorder())

	recLogin := httptest.NewRecorder()
	app.handleLoginSubmit(recLogin, reqLogin)
	if recLogin.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after successful login, got %d", recLogin.Code)
	}

	// 5. Charlie adds a TOTP secret
	charlieCookie := recLogin.Header().Get("Set-Cookie")
	secret := "JBSWY3DPEHPK3PXP"
	code, err := totp.GenerateCodeCustom(secret, testingTime(), totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("GenerateCodeCustom failed: %v", err)
	}

	addForm := url.Values{
		"csrf_token":         {"add_csrf"},
		"otpauth_or_secret":  {secret},
		"issuer":             {"GitHub"},
		"account_name":       {"charlie@github"},
		"category":           {"Work"},
		"confirm_code":       {code},
	}
	reqAdd := httptest.NewRequest("POST", "/add", strings.NewReader(addForm.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAdd.Header.Set("Cookie", charlieCookie)
	sessAdd, _ := app.store.Get(reqAdd, sessionName)
	sessAdd.Values["csrf"] = "add_csrf"
	sessAdd.Save(reqAdd, httptest.NewRecorder())

	recAdd := httptest.NewRecorder()
	app.handleAddSubmit(recAdd, reqAdd)
	if recAdd.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after add TOTP account, got %d", recAdd.Code)
	}

	accounts, err := listAccounts(app.db, charlie.ID)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("expected 1 account for charlie, got %d", len(accounts))
	}

	// 6. Charlie changes password
	pwChangeForm := url.Values{
		"csrf_token":       {"pw_csrf"},
		"current_password": {"charlies-password-456"},
		"new_password":     {"new-charlie-pass-789"},
		"confirm_password": {"new-charlie-pass-789"},
	}
	reqPw := httptest.NewRequest("POST", "/settings/password", strings.NewReader(pwChangeForm.Encode()))
	reqPw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPw.Header.Set("Cookie", charlieCookie)
	sessPw, _ := app.store.Get(reqPw, sessionName)
	sessPw.Values["csrf"] = "pw_csrf"
	sessPw.Save(reqPw, httptest.NewRecorder())

	recPw := httptest.NewRecorder()
	app.handlePasswordChange(recPw, reqPw)
	if recPw.Code != http.StatusOK {
		t.Fatalf("expected 200 after password change, got %d", recPw.Code)
	}

	// 7. Charlie logs in with NEW password
	newLoginForm := url.Values{
		"csrf_token": {"new_login_csrf"},
		"username":   {"charlie"},
		"password":   {"new-charlie-pass-789"},
	}
	reqNewLogin := httptest.NewRequest("POST", "/login", strings.NewReader(newLoginForm.Encode()))
	reqNewLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sessNewLogin, _ := app.store.Get(reqNewLogin, sessionName)
	sessNewLogin.Values["csrf"] = "new_login_csrf"
	sessNewLogin.Save(reqNewLogin, httptest.NewRecorder())

	recNewLogin := httptest.NewRecorder()
	app.handleLoginSubmit(recNewLogin, reqNewLogin)
	if recNewLogin.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after login with new password, got %d", recNewLogin.Code)
	}

	// 8. Charlie exports vault
	newCharlieCookie := recNewLogin.Header().Get("Set-Cookie")
	reqExport := httptest.NewRequest("POST", "/settings/export", strings.NewReader("csrf_token=export_csrf"))
	reqExport.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqExport.Header.Set("Cookie", newCharlieCookie)
	sessExport, _ := app.store.Get(reqExport, sessionName)
	sessExport.Values["csrf"] = "export_csrf"
	sessExport.Save(reqExport, httptest.NewRecorder())

	recExport := httptest.NewRecorder()
	app.handleExportVault(recExport, reqExport)
	if recExport.Code != http.StatusOK {
		t.Fatalf("expected 200 on vault export, got %d", recExport.Code)
	}
	if !strings.Contains(recExport.Body.String(), "GitHub") {
		t.Fatalf("expected export payload to contain 'GitHub', got %s", recExport.Body.String())
	}

	// 8b. Charlie configures Email OTP login verification
	saveOTPForm := url.Values{
		"csrf_token": {"save_otp_csrf"},
		"otp_method": {"email"},
		"email":      {"charlie@gmail.com"},
		"phone":      {""},
	}
	reqSaveOTP := httptest.NewRequest("POST", "/settings/otp", strings.NewReader(saveOTPForm.Encode()))
	reqSaveOTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqSaveOTP.Header.Set("Cookie", newCharlieCookie)
	sessSaveOTP, _ := app.store.Get(reqSaveOTP, sessionName)
	sessSaveOTP.Values["csrf"] = "save_otp_csrf"
	sessSaveOTP.Save(reqSaveOTP, httptest.NewRecorder())

	recSaveOTP := httptest.NewRecorder()
	app.handleSettingsSaveOTP(recSaveOTP, reqSaveOTP)
	if recSaveOTP.Code != http.StatusOK {
		t.Fatalf("expected 200 on save OTP settings, got %d", recSaveOTP.Code)
	}

	// 8c. Test Email OTP login challenge
	reqLoginOTP := httptest.NewRequest("POST", "/login", strings.NewReader(newLoginForm.Encode()))
	reqLoginOTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sessLoginOTP, _ := app.store.Get(reqLoginOTP, sessionName)
	sessLoginOTP.Values["csrf"] = "new_login_csrf"
	sessLoginOTP.Save(reqLoginOTP, httptest.NewRecorder())

	recLoginOTP := httptest.NewRecorder()
	app.handleLoginSubmit(recLoginOTP, reqLoginOTP)
	if recLoginOTP.Code != http.StatusSeeOther || recLoginOTP.Header().Get("Location") != "/login/otp" {
		t.Fatalf("expected redirect to /login/otp, got code %d loc %s", recLoginOTP.Code, recLoginOTP.Header().Get("Location"))
	}

	// Check that an OTP code was generated in session
	charlieOTPCookie := recLoginOTP.Header().Get("Set-Cookie")
	reqVerifyPage := httptest.NewRequest("GET", "/login/otp", nil)
	reqVerifyPage.Header.Set("Cookie", charlieOTPCookie)
	sessVerifyPage, _ := app.store.Get(reqVerifyPage, sessionName)
	otpCode, _ := sessVerifyPage.Values["pending_otp_code"].(string)
	if otpCode == "" {
		t.Fatalf("expected pending_otp_code in session")
	}

	// Submit valid 6-digit OTP code
	verifyOTPForm := url.Values{
		"csrf_token": {"verify_otp_csrf"},
		"code":       {otpCode},
	}
	reqVerifyOTP := httptest.NewRequest("POST", "/login/otp", strings.NewReader(verifyOTPForm.Encode()))
	reqVerifyOTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqVerifyOTP.Header.Set("Cookie", charlieOTPCookie)
	sessVerifyOTP, _ := app.store.Get(reqVerifyOTP, sessionName)
	sessVerifyOTP.Values["csrf"] = "verify_otp_csrf"
	sessVerifyOTP.Save(reqVerifyOTP, httptest.NewRecorder())

	recVerifyOTP := httptest.NewRecorder()
	app.handleLoginOTPSubmit(recVerifyOTP, reqVerifyOTP)
	if recVerifyOTP.Code != http.StatusSeeOther || recVerifyOTP.Header().Get("Location") != "/" {
		t.Fatalf("expected redirect to / after valid OTP code, got %d loc %s", recVerifyOTP.Code, recVerifyOTP.Header().Get("Location"))
	}

	// 8d. Test Regenerate Recovery Phrase
	regenForm := url.Values{
		"csrf_token": {"regen_csrf"},
		"password":   {"new-charlie-pass-789"},
	}
	reqRegen := httptest.NewRequest("POST", "/settings/recovery/regenerate", strings.NewReader(regenForm.Encode()))
	reqRegen.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqRegen.Header.Set("Cookie", recVerifyOTP.Header().Get("Set-Cookie"))
	sessRegen, _ := app.store.Get(reqRegen, sessionName)
	sessRegen.Values["csrf"] = "regen_csrf"
	sessRegen.Save(reqRegen, httptest.NewRecorder())

	recRegen := httptest.NewRecorder()
	app.handleRegenerateRecovery(recRegen, reqRegen)
	if recRegen.Code != http.StatusOK {
		t.Fatalf("expected 200 on regenerate recovery, got %d", recRegen.Code)
	}

	// 8e. Test Zero-Knowledge Account Recovery via /recover
	// Charlie forgot his password and recovers with a known 12-word phrase
	charlieUser, _ := getUserByID(app.db, charlie.ID)
	kekCharlie, _ := DeriveKEK("new-charlie-pass-789", charlieUser.Salt)
	dekCharlie, _ := UnwrapDEK(charlieUser.EncDEK, kekCharlie)

	testPhrase, _ := GenerateMnemonicPhrase(12)
	recSalt, _ := GenerateSalt()
	recKEK, _ := DeriveKEK(NormalizePhrase(testPhrase), recSalt)
	recEncDEK, _ := WrapDEK(dekCharlie, recKEK)
	phraseHash, _ := bcrypt.GenerateFromPassword([]byte(NormalizePhrase(testPhrase)), bcrypt.DefaultCost)
	_ = updateUserRecoveryData(app.db, charlie.ID, recEncDEK, recSalt, string(phraseHash))

	recoverForm := url.Values{
		"csrf_token":       {"recover_csrf"},
		"username":         {"charlie"},
		"recovery_phrase":  {testPhrase},
		"new_password":     {"charlie-recovered-999"},
		"confirm_password": {"charlie-recovered-999"},
	}
	reqRecover := httptest.NewRequest("POST", "/recover", strings.NewReader(recoverForm.Encode()))
	reqRecover.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sessRecover, _ := app.store.Get(reqRecover, sessionName)
	sessRecover.Values["csrf"] = "recover_csrf"
	sessRecover.Save(reqRecover, httptest.NewRecorder())

	recRecover := httptest.NewRecorder()
	app.handleRecoverSubmit(recRecover, reqRecover)
	if recRecover.Code != http.StatusSeeOther || recRecover.Header().Get("Location") != "/" {
		t.Fatalf("expected redirect to / after recovery, got %d loc %s", recRecover.Code, recRecover.Header().Get("Location"))
	}

	// Verify Charlie's vault is fully decrypted with the new password
	recoveredCookie := recRecover.Header().Get("Set-Cookie")
	reqCodesRecovered := httptest.NewRequest("GET", "/api/codes", nil)
	reqCodesRecovered.Header.Set("Cookie", recoveredCookie)
	recCodesRecovered := httptest.NewRecorder()
	app.handleCodesAPI(recCodesRecovered, reqCodesRecovered)
	if !strings.Contains(recCodesRecovered.Body.String(), "GitHub") {
		t.Fatalf("expected recovered vault to contain 'GitHub', got %s", recCodesRecovered.Body.String())
	}

	// 9. Admin deletes Charlie
	delForm := url.Values{
		"csrf_token": {"del_csrf"},
		"user_id":    {"2"},
	}
	reqDel := httptest.NewRequest("POST", "/admin/users/delete", strings.NewReader(delForm.Encode()))
	reqDel.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDel.Header.Set("Cookie", adminCookie)
	sessDel, _ := app.store.Get(reqDel, sessionName)
	sessDel.Values["csrf"] = "del_csrf"
	sessDel.Save(reqDel, httptest.NewRecorder())

	recDel := httptest.NewRecorder()
	app.handleAdminDeleteUser(recDel, reqDel)
	if recDel.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after admin user delete, got %d", recDel.Code)
	}

	// Verify Charlie is deleted and all TOTP accounts are wiped
	deletedAccounts, _ := listAccounts(app.db, charlie.ID)
	if len(deletedAccounts) != 0 {
		t.Fatalf("expected 0 accounts for deleted user charlie, got %d", len(deletedAccounts))
	}
}

func testingTime() (t time.Time) {
	return time.Now()
}
