package http

import (
	"net/http"
	"sync"
	"time"
)

// Simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	rate    int           // requests per window
	window  time.Duration // time window
}

type bucket struct {
	tokens    int
	lastRefill time.Time
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		window:  window,
	}
	
	// Cleanup old buckets periodically
	go rl.cleanup()
	
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{
			tokens:    rl.rate - 1, // consume one token
			lastRefill: now,
		}
		rl.buckets[key] = b
		return true
	}
	
	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill)
	if elapsed >= rl.window {
		b.tokens = rl.rate
		b.lastRefill = now
	}
	
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	
	return false
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			if now.Sub(bucket.lastRefill) > 2*rl.window {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware for HTTP rate limiting
func (rl *RateLimiter) Middleware(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !rl.Allow(key) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Common key functions
func IPBasedKey(r *http.Request) string {
	// Get real IP from headers (for reverse proxy setups)
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return "ip:" + ip
}

func UserBasedKey(r *http.Request) string {
	// Assumes user ID is available in context (set by auth middleware)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return "user:" + userID
	}
	return IPBasedKey(r) // fallback to IP
}

func TenantBasedKey(r *http.Request) string {
	// Assumes tenant ID is available in context
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		return "tenant:" + tenantID
	}
	return IPBasedKey(r) // fallback to IP
}
