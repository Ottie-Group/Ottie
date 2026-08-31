package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

//go:embed templates/* static/*
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

func getOrGenerateKey(envName string, byteLen int) []byte {
	v := os.Getenv(envName)
	if v != "" {
		key, err := base64.StdEncoding.DecodeString(v)
		if err == nil && len(key) >= byteLen {
			return key
		}
	}
	// Auto-generate high entropy key if none provided
	b := make([]byte, byteLen)
	rand.Read(b)
	return b
}

func main() {
	dbPath := os.Getenv("OTTIE_DB_PATH")
	if dbPath == "" {
		dbPath = "./data/ottie.db"
	}
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		log.Printf("warning creating db dir: %v", err)
	}

	sessionKey := getOrGenerateKey("APP_SESSION_KEY", 32)
	serverKey := getOrGenerateKey("APP_SERVER_KEY", 32)

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store := sessions.NewCookieStore(sessionKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int((24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   os.Getenv("OTTIE_INSECURE_COOKIES") == "", // set OTTIE_INSECURE_COOKIES=1 only for local http-only testing
		SameSite: http.SameSiteStrictMode,
	}

	// Try reading templates from disk for live development, or fallback to embeddedFS
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	var tmpl *template.Template
	if tmplDirect, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html"); err == nil {
		tmpl = tmplDirect
	} else {
		tmpl = template.Must(template.New("").Funcs(funcMap).ParseFS(embeddedFS, "templates/*.html"))
	}

	dekStore := newSessionStore()
	app := &App{
		db:        db,
		store:     store,
		dekStore:  dekStore,
		tmpl:      tmpl,
		serverKey: serverKey,
	}

	mux := http.NewServeMux()

	// Public / On-ramp routes
	mux.HandleFunc("GET /setup", app.handleSetupPage)
	mux.HandleFunc("POST /setup", app.handleSetupSubmit)
	mux.HandleFunc("GET /setup/recovery", app.handleSetupRecoveryPage)
	mux.HandleFunc("POST /setup/recovery", app.handleSetupRecoverySubmit)
	mux.HandleFunc("GET /login", app.handleLoginPage)
	mux.HandleFunc("POST /login", app.handleLoginSubmit)
	mux.HandleFunc("GET /login/otp", app.handleLoginOTPPage)
	mux.HandleFunc("POST /login/otp", app.handleLoginOTPSubmit)
	mux.HandleFunc("GET /recover", app.handleRecoverPage)
	mux.HandleFunc("POST /recover", app.handleRecoverSubmit)
	mux.HandleFunc("POST /logout", app.handleLogout)

	// User dashboard & TOTP codes
	mux.HandleFunc("GET /", app.requireAuth(app.handleDashboard))
	mux.HandleFunc("GET /api/codes", app.requireAuth(app.handleCodesAPI))
	mux.HandleFunc("GET /add", app.requireAuth(app.handleAddPage))
	mux.HandleFunc("POST /add", app.requireAuth(app.handleAddSubmit))
	mux.HandleFunc("POST /delete", app.requireAuth(app.handleDelete))
	mux.HandleFunc("GET /settings", app.requireAuth(app.handleSettingsPage))
	mux.HandleFunc("POST /settings/password", app.requireAuth(app.handlePasswordChange))
	mux.HandleFunc("POST /settings/export", app.requireAuth(app.handleExportVault))
	mux.HandleFunc("POST /settings/otp", app.requireAuth(app.handleSettingsSaveOTP))
	mux.HandleFunc("POST /settings/recovery/regenerate", app.requireAuth(app.handleRegenerateRecovery))

	// Admin Panel
	mux.HandleFunc("GET /admin", app.requireAdmin(app.handleAdminPage))
	mux.HandleFunc("POST /admin/users/create", app.requireAdmin(app.handleAdminCreateUser))
	mux.HandleFunc("POST /admin/users/delete", app.requireAdmin(app.handleAdminDeleteUser))

	// Static files (disk with embedded fallback + guaranteed MIME types)
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

	addr := os.Getenv("OTTIE_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	log.Printf("🦦 Ottie TOTP Manager is swimming at http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}
