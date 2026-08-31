package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/sessions"
)

//go:embed static/* frontend/dist/*
var embeddedFS embed.FS

func init() {
	// Fix Windows registry MIME type bugs (where .css is often mapped to text/plain)
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".html", "text/html; charset=utf-8")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self';")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func main() {
	dbPath := os.Getenv("OTTIE_DB_PATH")
	if dbPath == "" {
		if err := os.MkdirAll("data", 0700); err != nil {
			log.Fatalf("failed to create data directory: %v", err)
		}
		dbPath = filepath.Join("data", "ottie.db")
	} else {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
			log.Fatalf("failed to create db directory: %v", err)
		}
	}

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize SQLite database: %v", err)
	}
	defer db.Close()

	sessionSecret := os.Getenv("OTTIE_SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = os.Getenv("APP_SESSION_KEY")
	}

	var secretBytes []byte
	if sessionSecret != "" {
		var err error
		secretBytes, err = base64.StdEncoding.DecodeString(sessionSecret)
		if err != nil || len(secretBytes) < 32 {
			secretBytes = []byte(sessionSecret)
			for len(secretBytes) < 32 {
				secretBytes = append(secretBytes, secretBytes...)
			}
			secretBytes = secretBytes[:32]
		}
	} else {
		// Persist generated secret to data/.session_secret so restarts don't invalidate active sessions
		secretFilePath := filepath.Join(filepath.Dir(dbPath), ".session_secret")
		if data, err := os.ReadFile(secretFilePath); err == nil && len(data) >= 32 {
			secretBytes = data[:32]
		} else {
			secretBytes = make([]byte, 32)
			if _, err := rand.Read(secretBytes); err != nil {
				log.Fatalf("failed to generate random session secret: %v", err)
			}
			_ = os.WriteFile(secretFilePath, secretBytes, 0600)
			log.Println("🔑 Generated persistent session secret in data/.session_secret")
		}
	}

	cookieStore := sessions.NewCookieStore(secretBytes)
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true if TLS terminated at Ottie level
		SameSite: http.SameSiteLaxMode,
	}

	dekStore := newSessionStore()

	serverKey := make([]byte, 32)
	copy(serverKey, secretBytes[:32])

	app := &App{
		db:        db,
		store:     cookieStore,
		dekStore:  dekStore,
		serverKey: serverKey,
	}

	mux := http.NewServeMux()

	// REST JSON APIs
	mux.HandleFunc("GET /api/me", app.handleApiMe)
	mux.HandleFunc("GET /api/accounts", app.handleApiAccounts)
	mux.HandleFunc("GET /api/codes", app.handleCodesAPI)
	mux.HandleFunc("POST /api/setup", app.handleApiSetup)
	mux.HandleFunc("POST /api/setup/confirm", app.handleApiSetupConfirm)
	mux.HandleFunc("POST /api/auth/login", app.handleApiAuthLogin)
	mux.HandleFunc("POST /api/auth/verify-2fa", app.handleApiAuthVerify2FA)
	mux.HandleFunc("POST /api/auth/verify-otp", app.handleApiAuthVerifyOTP)
	mux.HandleFunc("POST /api/auth/recover", app.handleApiAuthRecover)
	mux.HandleFunc("POST /api/auth/logout", app.handleApiAuthLogout)
	mux.HandleFunc("POST /api/tokens", app.handleApiTokens)
	mux.HandleFunc("POST /api/tokens/delete", app.handleApiTokensDelete)
	mux.HandleFunc("POST /api/settings/password", app.handleApiSettingsPassword)
	mux.HandleFunc("POST /api/settings/otp", app.handleApiSettingsOTP)
	mux.HandleFunc("POST /api/settings/recovery/regenerate", app.handleApiSettingsRecoveryRegenerate)
	mux.HandleFunc("POST /api/settings/export", app.handleExportVault)
	mux.HandleFunc("GET /api/admin/users", app.handleApiAdminUsers)
	mux.HandleFunc("POST /api/admin/users/create", app.handleApiAdminUsersCreate)
	mux.HandleFunc("POST /api/admin/users/delete", app.handleApiAdminUsersDelete)

	// Static Assets & Icons (/static/*)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/static/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		ext := filepath.Ext(path)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		}

		// Try disk first
		if data, err := os.ReadFile(filepath.Join("static", path)); err == nil {
			w.Write(data)
			return
		}

		// Fallback to embedded
		if data, err := embeddedFS.ReadFile("static/" + path); err == nil {
			w.Write(data)
			return
		}

		http.NotFound(w, r)
	})

	// React SPA Vite Assets (/assets/*)
	mux.HandleFunc("GET /assets/", func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/")
		ext := filepath.Ext(relPath)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		}

		// Try disk in frontend/dist first
		if data, err := os.ReadFile(filepath.Join("frontend", "dist", relPath)); err == nil {
			w.Write(data)
			return
		}

		// Fallback to embedded frontend/dist
		if data, err := embeddedFS.ReadFile("frontend/dist/" + relPath); err == nil {
			w.Write(data)
			return
		}

		http.NotFound(w, r)
	})

	// React Single Page App handler (serving index.html for all client routes)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// If request is for an existing static file at root like favicon.svg or ottie.svg
		if r.URL.Path == "/ottie.svg" || r.URL.Path == "/favicon.svg" {
			w.Header().Set("Content-Type", "image/svg+xml")
			if data, err := os.ReadFile(filepath.Join("static", "ottie.svg")); err == nil {
				w.Write(data)
				return
			}
			if data, err := embeddedFS.ReadFile("static/ottie.svg"); err == nil {
				w.Write(data)
				return
			}
		}

		// Serve React SPA index.html
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if data, err := os.ReadFile(filepath.Join("frontend", "dist", "index.html")); err == nil {
			w.Write(data)
			return
		}

		if data, err := embeddedFS.ReadFile("frontend/dist/index.html"); err == nil {
			w.Write(data)
			return
		}

		http.Error(w, "Ottie React frontend not found. Please compile the frontend via 'bun run build'.", http.StatusNotFound)
	})

	addr := os.Getenv("OTTIE_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	log.Printf("🦦 Ottie TOTP Manager (React + TypeScript + Emotion) is swimming at http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}
