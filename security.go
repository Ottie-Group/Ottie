package main

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	emailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex  = regexp.MustCompile(`^\+?[0-9\s\-()]{7,25}$`)
	base32Regex = regexp.MustCompile(`^[A-Z2-7=]+$`)
)

// isValidEmail checks standard email syntax
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// isValidPhone validates reasonable phone number strings
func isValidPhone(phone string) bool {
	clean := strings.ReplaceAll(phone, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, "(", "")
	clean = strings.ReplaceAll(clean, ")", "")
	if len(clean) < 7 || len(clean) > 20 {
		return false
	}
	return phoneRegex.MatchString(phone)
}

// isValidBase32 checks if string contains only valid RFC 4648 Base32 characters
func isValidBase32(s string) bool {
	clean := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
	if len(clean) < 8 || len(clean) > 256 {
		return false
	}
	return base32Regex.MatchString(clean)
}

// RateLimiter tracks recent attempts by IP and key with automatic cleanup
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates an in-memory rate limiter with sliding window
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Periodic cleanup of stale records
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for key, timestamps := range rl.attempts {
				var valid []time.Time
				for _, t := range timestamps {
					if now.Sub(t) < rl.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.attempts, key)
				} else {
					rl.attempts[key] = valid
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Allow checks if the given key (IP or identifier) is within rate limits
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	timestamps := rl.attempts[key]

	var recent []time.Time
	for _, t := range timestamps {
		if now.Sub(t) < rl.window {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		return false
	}

	rl.attempts[key] = append(recent, now)
	return true
}

// getClientIP extracts client IP address, handling proxies and remote addresses
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isLocalOrPrivateIP(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") || strings.EqualFold(hostname, "127.0.0.1") || hostname == "0.0.0.0" || hostname == "::" || strings.HasSuffix(strings.ToLower(hostname), ".local") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// isSameOrigin checks if Origin or Referer header matches the request host or is from the same site
func isSameOrigin(r *http.Request) bool {
	// Check Sec-Fetch-Site (present in all modern browsers)
	fetchSite := r.Header.Get("Sec-Fetch-Site")
	if fetchSite == "cross-site" {
		return false
	}
	if fetchSite == "same-origin" || fetchSite == "same-site" || fetchSite == "none" {
		return true
	}

	reqHost := r.Header.Get("X-Forwarded-Host")
	if reqHost == "" {
		reqHost = r.Host
	}
	reqHostname, _, err := net.SplitHostPort(reqHost)
	if err != nil {
		reqHostname = reqHost
	}

	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if strings.EqualFold(u.Host, reqHost) || strings.EqualFold(u.Hostname(), reqHostname) {
			return true
		}
		if isLocalOrPrivateIP(u.Hostname()) && isLocalOrPrivateIP(reqHostname) {
			return true
		}
		return false
	}

	referer := r.Header.Get("Referer")
	if referer != "" {
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		if strings.EqualFold(u.Host, reqHost) || strings.EqualFold(u.Hostname(), reqHostname) {
			return true
		}
		if isLocalOrPrivateIP(u.Hostname()) && isLocalOrPrivateIP(reqHostname) {
			return true
		}
		return false
	}

	// If neither Origin nor Referer is present, allow if not explicitly marked cross-site
	return fetchSite != "cross-site"
}
