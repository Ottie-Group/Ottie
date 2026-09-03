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
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

// Version of Ottie (can be set at build time via -ldflags "-X main.Version=...")
var Version = "1.0.3"

//go:embed static/* frontend/dist/*
var embeddedFS embed.FS

var isDebugLogging = false

func initDebugLog() {
	val := os.Getenv("OTTIE_DEBUG_LOGS")
	if val == "" {
		val = os.Getenv("OTTIE_DEBUG")
	}
	if val == "" {
		val = os.Getenv("DEBUG")
	}
	isDebugLogging = val == "1" || strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")
	if isDebugLogging {
		log.Println("[DEBUG] Verbose diagnostics logging enabled via OTTIE_DEBUG_LOGS=1")
	}
}

func debugLog(format string, v ...any) {
	if isDebugLogging {
		log.Printf(format, v...)
	}
}

func init() {
	// Fix Windows registry MIME type bugs (where .css is often mapped to text/plain)
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".html", "text/html; charset=utf-8")
}

func securityHeaders(next http.Handler, secureCookie bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';")
		h.Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")

		// Only emit HSTS when on a secure origin (HTTPS / TLS or behind an HTTPS reverse proxy)
		isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") || secureCookie
		if isHTTPS {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		// Protect mutating API endpoints against CSRF and cross-origin attacks
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodPatch {
				if !isSameOrigin(r) {
					log.Printf("[SECURITY BLOCKED] 403 Forbidden on %s %s from IP=%s (Host=%q, Origin=%q, Referer=%q, Sec-Fetch-Site=%q)",
						r.Method, r.URL.Path, getClientIP(r), r.Host, r.Header.Get("Origin"), r.Header.Get("Referer"), r.Header.Get("Sec-Fetch-Site"))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"Cross-origin request blocked for security"}`))
					return
				}
				// Limit payload size to 1MB max to prevent resource exhaustion attacks
				r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	Version = strings.TrimPrefix(strings.TrimSpace(Version), "v")
	initDebugLog()
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
			log.Println("Generated persistent session secret in data/.session_secret")
		}
	}

	maxAge := 86400 * 7 // Default: 7 days (604800 seconds)
	if envMaxAge := os.Getenv("OTTIE_SESSION_MAX_AGE"); envMaxAge != "" {
		if sec, err := strconv.Atoi(envMaxAge); err == nil && sec > 0 {
			maxAge = sec
		} else if strings.HasSuffix(envMaxAge, "d") {
			daysStr := strings.TrimSuffix(envMaxAge, "d")
			if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
				maxAge = days * 86400
			}
		} else if d, err := time.ParseDuration(envMaxAge); err == nil && d > 0 {
			maxAge = int(d.Seconds())
		}
	} else if envMaxAgeSec := os.Getenv("OTTIE_SESSION_MAX_AGE_SECONDS"); envMaxAgeSec != "" {
		if sec, err := strconv.Atoi(envMaxAgeSec); err == nil && sec > 0 {
			maxAge = sec
		}
	}

	// Cookie security flag: Defaults to false for plain HTTP local dev, set OTTIE_INSECURE_COOKIES=0 or OTTIE_SECURE_COOKIES=1 for HTTPS in production
	secureCookie := false
	if envInsecure := os.Getenv("OTTIE_INSECURE_COOKIES"); envInsecure != "" {
		if envInsecure == "0" || strings.EqualFold(envInsecure, "false") {
			secureCookie = true
		} else if envInsecure == "1" || strings.EqualFold(envInsecure, "true") {
			secureCookie = false
		}
	} else if envSecure := os.Getenv("OTTIE_SECURE_COOKIES"); envSecure != "" {
		if envSecure == "1" || strings.EqualFold(envSecure, "true") {
			secureCookie = true
		} else if envSecure == "0" || strings.EqualFold(envSecure, "false") {
			secureCookie = false
		}
	}

	cookieStore := sessions.NewCookieStore(secretBytes)
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	}

	dekStore := newSessionStore()
	rateLimiter := NewRateLimiter(20, time.Minute)

	serverKeyEnv := os.Getenv("APP_SERVER_KEY")
	if serverKeyEnv == "" {
		serverKeyEnv = os.Getenv("OTTIE_SERVER_KEY")
	}

	var serverKey []byte
	if serverKeyEnv != "" {
		if decoded, err := base64.StdEncoding.DecodeString(serverKeyEnv); err == nil && len(decoded) >= 32 {
			serverKey = decoded[:32]
		} else {
			serverKey = []byte(serverKeyEnv)
			for len(serverKey) < 32 {
				serverKey = append(serverKey, serverKey...)
			}
			serverKey = serverKey[:32]
		}
	} else {
		serverKey = make([]byte, 32)
		copy(serverKey, secretBytes[:32])
	}

	app := &App{
		db:          db,
		store:       cookieStore,
		dekStore:    dekStore,
		serverKey:   serverKey,
		rateLimiter: rateLimiter,
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
	mux.HandleFunc("GET /api/sessions", app.handleApiSessions)
	mux.HandleFunc("POST /api/sessions/revoke", app.handleApiSessionRevoke)
	mux.HandleFunc("POST /api/sessions/revoke-others", app.handleApiSessionsRevokeOthers)
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

	// React Assets (/assets/*)
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

	// React handler
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

		// Serve React index.html
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
	log.Printf("Ottie v%s is swimming at http://%s", Version, addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux, secureCookie)))
}
