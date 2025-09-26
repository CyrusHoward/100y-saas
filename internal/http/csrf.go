package http

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

// CSRFProtection provides CSRF token generation and validation
type CSRFProtection struct {
	tokens sync.Map // token -> expiry time
	secret []byte
}

// NewCSRFProtection creates a new CSRF protection instance
func NewCSRFProtection() *CSRFProtection {
	secret := make([]byte, 32)
	rand.Read(secret)
	
	csrf := &CSRFProtection{
		secret: secret,
	}
	
	// Clean up expired tokens every 15 minutes
	go csrf.cleanupExpiredTokens()
	
	return csrf
}

// GenerateToken creates a new CSRF token
func (c *CSRFProtection) GenerateToken() string {
	token := make([]byte, 32)
	rand.Read(token)
	
	tokenStr := base64.URLEncoding.EncodeToString(token)
	
	// Store token with 1 hour expiry
	c.tokens.Store(tokenStr, time.Now().Add(time.Hour))
	
	return tokenStr
}

// ValidateToken checks if a CSRF token is valid
func (c *CSRFProtection) ValidateToken(token string) bool {
	if token == "" {
		return false
	}
	
	expiry, exists := c.tokens.Load(token)
	if !exists {
		return false
	}
	
	expiryTime := expiry.(time.Time)
	if time.Now().After(expiryTime) {
		c.tokens.Delete(token)
		return false
	}
	
	return true
}

// ConsumeToken validates and removes a CSRF token (single-use)
func (c *CSRFProtection) ConsumeToken(token string) bool {
	if !c.ValidateToken(token) {
		return false
	}
	
	c.tokens.Delete(token)
	return true
}

// cleanupExpiredTokens removes expired tokens from memory
func (c *CSRFProtection) cleanupExpiredTokens() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.tokens.Range(func(key, value interface{}) bool {
				expiry := value.(time.Time)
				if now.After(expiry) {
					c.tokens.Delete(key)
				}
				return true
			})
		}
	}
}

// CSRFProtectionMiddleware provides CSRF protection for HTTP handlers
func (h *Handlers) CSRFProtectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only protect state-changing HTTP methods
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			// Get CSRF token from header or form
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				token = r.FormValue("csrf_token")
			}
			
			// Validate CSRF token
			if !h.csrf.ConsumeToken(token) {
				h.writeError(w, "Invalid or missing CSRF token", http.StatusForbidden)
				return
			}
		}
		
		// For GET requests, optionally add a new CSRF token to response headers
		if r.Method == "GET" {
			newToken := h.csrf.GenerateToken()
			w.Header().Set("X-CSRF-Token", newToken)
			
			// Also set as cookie for JavaScript access
			http.SetCookie(w, &http.Cookie{
				Name:     "csrf_token",
				Value:    newToken,
				Path:     "/",
				HttpOnly: false, // Allow JavaScript access
				Secure:   h.config.IsProduction(),
				SameSite: http.SameSiteStrictMode,
				MaxAge:   3600, // 1 hour
			})
		}
		
		next.ServeHTTP(w, r)
	})
}

// CSRFTokenHandler provides an endpoint to get a fresh CSRF token
func (h *Handlers) CSRFTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	token := h.csrf.GenerateToken()
	
	h.writeSuccess(w, map[string]interface{}{
		"csrf_token": token,
		"expires_in": 3600, // seconds
	}, "CSRF token generated")
}
