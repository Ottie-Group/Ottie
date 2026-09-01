package main

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

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

// isSameOrigin checks if Origin or Referer header matches the request host
func isSameOrigin(r *http.Request) bool {
	// Check Sec-Fetch-Site (present in all modern browsers)
	fetchSite := r.Header.Get("Sec-Fetch-Site")
	if fetchSite == "cross-site" {
		return false
	}

	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		// Match hostname (ignoring port or protocol differences between dev proxies)
		if strings.EqualFold(u.Host, r.Host) || strings.EqualFold(u.Hostname(), "localhost") || strings.EqualFold(u.Hostname(), "127.0.0.1") {
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
		if strings.EqualFold(u.Host, r.Host) || strings.EqualFold(u.Hostname(), "localhost") || strings.EqualFold(u.Hostname(), "127.0.0.1") {
			return true
		}
		return false
	}

	// If neither Origin nor Referer is present (e.g. server-to-server or direct curl), allow if Sec-Fetch-Site is not cross-site
	return fetchSite != "cross-site"
}
