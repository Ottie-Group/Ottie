package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"golang.org/x/crypto/bcrypt"
)

func TestSessionManagement(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	// Create user with genuine bcrypt hash
	salt, _ := GenerateSalt()
	kek, _ := DeriveKEK("password123", salt)
	dek, _ := GenerateDEK()
	wrappedDEK, _ := WrapDEK(dek, kek)
	pwHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userID, err := createUserWithDEK(app.db, "alice", string(pwHash), "user", wrappedDEK, salt)
	if err != nil {
		t.Fatalf("createUserWithDEK: %v", err)
	}

	// 1. First login (Session A)
	loginPayload := map[string]string{"username": "alice", "password": "password123"}
	loginBytes, _ := json.Marshal(loginPayload)
	reqLoginA := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBytes))
	reqLoginA.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0")
	recLoginA := httptest.NewRecorder()
	app.handleApiAuthLogin(recLoginA, reqLoginA)

	if recLoginA.Code != http.StatusOK {
		t.Fatalf("Login A failed: %d, body: %s", recLoginA.Code, recLoginA.Body.String())
	}
	cookieA := recLoginA.Header().Get("Set-Cookie")

	// 2. Second login (Session B - mobile)
	reqLoginB := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBytes))
	reqLoginB.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 Chrome/120.0.0.0 Mobile Safari/537.36")
	reqLoginB.Header.Set("X-Ottie-Client", "companion")
	reqLoginB.Header.Set("X-Ottie-Platform", "android")
	recLoginB := httptest.NewRecorder()
	app.handleApiAuthLogin(recLoginB, reqLoginB)

	if recLoginB.Code != http.StatusOK {
		t.Fatalf("Login B failed: %d", recLoginB.Code)
	}
	cookieB := recLoginB.Header().Get("Set-Cookie")

	// 3. Query /api/sessions from Session A
	reqList := httptest.NewRequest("GET", "/api/sessions", nil)
	reqList.Header.Set("Cookie", cookieA)
	recList := httptest.NewRecorder()
	app.handleApiSessions(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("List sessions failed: %d, body: %s", recList.Code, recList.Body.String())
	}

	var listRes struct {
		Success  bool          `json:"success"`
		Sessions []UserSession `json:"sessions"`
	}
	json.NewDecoder(recList.Body).Decode(&listRes)
	if len(listRes.Sessions) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(listRes.Sessions))
	}

	var sessionB_ID string
	var sessionA_isCurrent bool
	for _, s := range listRes.Sessions {
		if s.IsCurrent {
			sessionA_isCurrent = true
		} else {
			sessionB_ID = s.ID
			if s.DeviceName != "Ottie Companion (Android)" {
				t.Fatalf("expected device name 'Ottie Companion (Android)', got %q", s.DeviceName)
			}
		}
	}
	if !sessionA_isCurrent {
		t.Fatal("expected session A to be current")
	}
	if sessionB_ID == "" {
		t.Fatal("expected session B to be found")
	}

	// 4. Session B can access /api/me before revoke
	reqMeB := httptest.NewRequest("GET", "/api/me", nil)
	reqMeB.Header.Set("Cookie", cookieB)
	recMeB := httptest.NewRecorder()
	app.handleApiMe(recMeB, reqMeB)
	if recMeB.Code != http.StatusOK {
		t.Fatalf("expected 200 for session B before revoke, got %d", recMeB.Code)
	}
	var meBPre map[string]any
	json.NewDecoder(recMeB.Body).Decode(&meBPre)
	if meBPre["authenticated"] != true {
		t.Fatal("expected authenticated: true before revoke")
	}

	// 5. Session A revokes Session B
	revokePayload, _ := json.Marshal(map[string]string{"sessionId": sessionB_ID})
	reqRevoke := httptest.NewRequest("POST", "/api/sessions/revoke", bytes.NewReader(revokePayload))
	reqRevoke.Header.Set("Cookie", cookieA)
	recRevoke := httptest.NewRecorder()
	app.handleApiSessionRevoke(recRevoke, reqRevoke)

	if recRevoke.Code != http.StatusOK {
		t.Fatalf("expected 200 on revoke, got %d", recRevoke.Code)
	}

	// 6. Session B is now revoked -> /api/me returns authenticated: false
	reqMeB2 := httptest.NewRequest("GET", "/api/me", nil)
	reqMeB2.Header.Set("Cookie", cookieB)
	recMeB2 := httptest.NewRecorder()
	app.handleApiMe(recMeB2, reqMeB2)
	var meBPost map[string]any
	json.NewDecoder(recMeB2.Body).Decode(&meBPost)
	if meBPost["authenticated"] == true {
		t.Fatal("expected authenticated: false for revoked session B")
	}

	// Protected endpoint /api/accounts must return 401
	reqAccB := httptest.NewRequest("GET", "/api/accounts", nil)
	reqAccB.Header.Set("Cookie", cookieB)
	recAccB := httptest.NewRecorder()
	app.handleApiAccounts(recAccB, reqAccB)
	if recAccB.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on /api/accounts for revoked session B, got %d", recAccB.Code)
	}

	// 7. Session A is still valid
	reqAccA := httptest.NewRequest("GET", "/api/accounts", nil)
	reqAccA.Header.Set("Cookie", cookieA)
	recAccA := httptest.NewRecorder()
	app.handleApiAccounts(recAccA, reqAccA)
	if recAccA.Code != http.StatusOK {
		t.Fatalf("expected 200 for session A on /api/accounts, got %d", recAccA.Code)
	}

	// 8. Create Session C and test Revoke Others
	reqLoginC := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBytes))
	recLoginC := httptest.NewRecorder()
	app.handleApiAuthLogin(recLoginC, reqLoginC)
	cookieC := recLoginC.Header().Get("Set-Cookie")

	reqRevokeOthers := httptest.NewRequest("POST", "/api/sessions/revoke-others", nil)
	reqRevokeOthers.Header.Set("Cookie", cookieA)
	recRevokeOthers := httptest.NewRecorder()
	app.handleApiSessionsRevokeOthers(recRevokeOthers, reqRevokeOthers)

	if recRevokeOthers.Code != http.StatusOK {
		t.Fatalf("expected 200 on revoke others, got %d", recRevokeOthers.Code)
	}

	// Session C is now revoked
	reqAccC := httptest.NewRequest("GET", "/api/accounts", nil)
	reqAccC.Header.Set("Cookie", cookieC)
	recAccC := httptest.NewRecorder()
	app.handleApiAccounts(recAccC, reqAccC)
	if recAccC.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked session C on /api/accounts, got %d", recAccC.Code)
	}

	// Session A is still alive
	recAccA2 := httptest.NewRecorder()
	app.handleApiAccounts(recAccA2, reqAccA)
	if recAccA2.Code != http.StatusOK {
		t.Fatalf("expected 200 for session A after revoke-others, got %d", recAccA2.Code)
	}
	_ = userID
}