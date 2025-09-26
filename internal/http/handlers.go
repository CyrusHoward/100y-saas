package http

import (
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"100y-saas/internal/analytics"
	"100y-saas/internal/auth"
	"100y-saas/internal/config"
	"100y-saas/internal/logger"
	"100y-saas/internal/saas"
)

type Handlers struct {
	db        *sql.DB
	config    *config.Config
	logger    *logger.Logger
	auth      *auth.AuthService
	saas      *saas.SaaSService
	analytics *analytics.AnalyticsService
	rateLimiter *RateLimiter
	csrf      *CSRFProtection
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TenantRequest struct {
	Name string `json:"name"`
}

type UserContext struct {
	User     *auth.User
	TenantID int64
	Role     string
}

func NewHandlers(db *sql.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		db:          db,
		config:      cfg,
		logger:      logger.New("handlers"),
		auth:        auth.NewAuthService(db),
		saas:        saas.NewSaaSService(db),
		analytics:   analytics.NewAnalyticsService(db),
		rateLimiter: NewRateLimiter(100, time.Hour), // 100 requests per hour for auth
		csrf:        NewCSRFProtection(),
	}
}

// Middleware

func (h *Handlers) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Allow specific origins in production, all in development
		if h.config.IsDevelopment() || origin == h.config.Server.BaseURL {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (h *Handlers) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)
		
		h.logger.RequestStart(r.Method, r.URL.Path, r.UserAgent(), requestID)
		
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		
		// Extract status code (would need response writer wrapper for real implementation)
		h.logger.RequestEnd(r.Method, r.URL.Path, requestID, 200, duration)
	})
}

func (h *Handlers) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			h.writeError(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		user, err := h.auth.ValidateSession(token)
		if err != nil {
			h.writeError(w, "Invalid or expired session", http.StatusUnauthorized)
			return
		}

		// Add user to request context (simplified - in real app use context.Context)
		r.Header.Set("X-User-ID", strconv.FormatInt(user.ID, 10))
		r.Header.Set("X-User-Email", user.Email)

		next(w, r)
	}
}

func (h *Handlers) RequireTenant(next http.HandlerFunc) http.HandlerFunc {
	return h.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)

		if tenantID == 0 {
			h.writeError(w, "Tenant ID required", http.StatusBadRequest)
			return
		}

		// Check access
		hasAccess, role := h.saas.HasAccess(userID, tenantID)
		if !hasAccess {
			h.writeError(w, "Access denied to tenant", http.StatusForbidden)
			return
		}

		r.Header.Set("X-User-Role", role)
		next(w, r)
	})
}

// Auth Handlers

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	if !h.rateLimiter.Allow(IPBasedKey(r)) {
		h.writeError(w, "Too many registration attempts", http.StatusTooManyRequests)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.writeError(w, "Email and password required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < h.config.Auth.PasswordMinLength {
		h.writeError(w, fmt.Sprintf("Password must be at least %d characters", h.config.Auth.PasswordMinLength), http.StatusBadRequest)
		return
	}

	// Register user
	user, err := h.auth.Register(req.Email, req.Password)
	if err != nil {
		if err == auth.ErrEmailTaken {
			h.writeError(w, "Email already registered", http.StatusConflict)
			return
		}
		h.logger.Error("Registration failed", map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
		})
		h.writeError(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	// Create default tenant for user
	tenant, err := h.saas.CreateTenant(req.Email+"'s Workspace", user.ID)
	if err != nil {
		h.logger.Error("Failed to create default tenant", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		// Don't fail registration if tenant creation fails
	}

	// Track registration event
	if tenant != nil {
		h.analytics.TrackEvent(tenant.ID, user.ID, "user_registered", map[string]interface{}{
			"email": req.Email,
		})
	}

	h.writeSuccess(w, map[string]interface{}{
		"user":   user,
		"tenant": tenant,
	}, "Registration successful")
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting
	if !h.rateLimiter.Allow(IPBasedKey(r)) {
		h.writeError(w, "Too many login attempts", http.StatusTooManyRequests)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	session, user, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			h.writeError(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		h.logger.Error("Login failed", map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
		})
		h.writeError(w, "Login failed", http.StatusInternalServerError)
		return
	}

	// Get user's tenants
	tenants, err := h.saas.GetUserTenants(user.ID)
	if err != nil {
		h.logger.Error("Failed to get user tenants", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	// Track login event
	if len(tenants) > 0 {
		h.analytics.TrackEvent(tenants[0].ID, user.ID, "user_login", map[string]interface{}{
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		})
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   h.config.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	h.writeSuccess(w, map[string]interface{}{
		"user":    user,
		"tenants": tenants,
		"session": session,
	}, "Login successful")
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token != "" {
		h.auth.Logout(token)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Secure:   h.config.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	h.writeSuccess(w, nil, "Logged out successfully")
}

// Tenant Handlers

func (h *Handlers) GetTenants(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	
	tenants, err := h.saas.GetUserTenants(userID)
	if err != nil {
		h.logger.Error("Failed to get tenants", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		h.writeError(w, "Failed to get tenants", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, tenants, "")
}

func (h *Handlers) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)

	var req TenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		h.writeError(w, "Tenant name required", http.StatusBadRequest)
		return
	}

	tenant, err := h.saas.CreateTenant(req.Name, userID)
	if err != nil {
		h.logger.Error("Failed to create tenant", map[string]interface{}{
			"user_id":     userID,
			"tenant_name": req.Name,
			"error":       err.Error(),
		})
		h.writeError(w, "Failed to create tenant", http.StatusInternalServerError)
		return
	}

	// Track tenant creation
	h.analytics.TrackEvent(tenant.ID, userID, "tenant_created", map[string]interface{}{
		"tenant_name": req.Name,
	})

	h.writeSuccess(w, tenant, "Tenant created successfully")
}

// Analytics Handlers

func (h *Handlers) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)

	stats, err := h.analytics.GetRealtimeStats(tenantID)
	if err != nil {
		h.logger.Error("Failed to get analytics", map[string]interface{}{
			"tenant_id": tenantID,
			"error":     err.Error(),
		})
		h.writeError(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, stats, "")
}

// Export Handlers

func (h *Handlers) ExportAll(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	role := r.Header.Get("X-User-Role")

	// Only owners can export all data
	if role != "owner" {
		h.writeError(w, "Only tenant owners can export all data", http.StatusForbidden)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	dataType := r.URL.Query().Get("type")
	if dataType == "" {
		dataType = "all"
	}

	// Validate format
	if format != "json" && format != "csv" {
		h.writeError(w, "Format must be 'json' or 'csv'", http.StatusBadRequest)
		return
	}

	// Validate type
	validTypes := []string{"profile", "tenants", "analytics", "items", "all"}
	valid := false
	for _, vt := range validTypes {
		if dataType == vt {
			valid = true
			break
		}
	}
	if !valid {
		h.writeError(w, "Type must be one of: profile, tenants, analytics, items, all", http.StatusBadRequest)
		return
	}

	// Export data based on type and format
	switch format {
	case "json":
		h.exportJSON(w, tenantID, userID, dataType)
	case "csv":
		h.exportCSV(w, tenantID, userID, dataType)
	}

	// Track export event
	h.analytics.TrackEvent(tenantID, userID, "data_exported", map[string]interface{}{
		"format": format,
		"type":   dataType,
	})
}

func (h *Handlers) exportJSON(w http.ResponseWriter, tenantID, userID int64, dataType string) {
	data := map[string]interface{}{
		"tenant_id":   tenantID,
		"exported_at": time.Now(),
		"format":      "json",
		"type":        dataType,
	}

	// Export based on type
	switch dataType {
	case "profile":
		profile, err := h.getUserProfile(userID)
		if err == nil {
			data["profile"] = profile
		}
	
	case "tenants":
		tenants, err := h.getUserTenants(userID)
		if err == nil {
			data["tenants"] = tenants
		}
	
	case "analytics":
		analytics, err := h.getAnalyticsData(tenantID)
		if err == nil {
			data["analytics"] = analytics
		}
	
	case "items":
		items, err := h.getItems(tenantID)
		if err == nil {
			data["items"] = items
		}
	
	case "all":
		// Export all data types
		if profile, err := h.getUserProfile(userID); err == nil {
			data["profile"] = profile
		}
		if tenants, err := h.getUserTenants(userID); err == nil {
			data["tenants"] = tenants
		}
		if analytics, err := h.getAnalyticsData(tenantID); err == nil {
			data["analytics"] = analytics
		}
		if items, err := h.getItems(tenantID); err == nil {
			data["items"] = items
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=tenant_%d_%s_export.json", tenantID, dataType))
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) exportCSV(w http.ResponseWriter, tenantID, userID int64, dataType string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=tenant_%d_%s_export.csv", tenantID, dataType))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch dataType {
	case "profile":
		h.exportProfileCSV(cw, userID)
	case "tenants":
		h.exportTenantsCSV(cw, userID)
	case "analytics":
		h.exportAnalyticsCSV(cw, tenantID)
	case "items":
		h.exportItemsCSV(cw, tenantID)
	case "all":
		// Export all data types in separate sections
		cw.Write([]string{"=== USER PROFILE ==="})
		h.exportProfileCSV(cw, userID)
		cw.Write([]string{""}) // Empty row
		cw.Write([]string{"=== TENANTS ==="})
		h.exportTenantsCSV(cw, userID)
		cw.Write([]string{""}) // Empty row
		cw.Write([]string{"=== ANALYTICS ==="})
		h.exportAnalyticsCSV(cw, tenantID)
		cw.Write([]string{""}) // Empty row
		cw.Write([]string{"=== ITEMS ==="})
		h.exportItemsCSV(cw, tenantID)
	}
}

// Helper functions for data retrieval

func (h *Handlers) getUserProfile(userID int64) (map[string]interface{}, error) {
	var email, name string
	var createdAt time.Time
	err := h.db.QueryRow("SELECT email, COALESCE(name, ''), created_at FROM users WHERE id = ?", userID).Scan(&email, &name, &createdAt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":         userID,
		"email":      email,
		"name":       name,
		"created_at": createdAt,
	}, nil
}

func (h *Handlers) getUserTenants(userID int64) ([]map[string]interface{}, error) {
	rows, err := h.db.Query(`
		SELECT t.id, t.name, t.plan, t.created_at, tu.role
		FROM tenants t
		JOIN tenant_users tu ON t.id = tu.tenant_id
		WHERE tu.user_id = ?
		ORDER BY t.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, plan, role string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &plan, &createdAt, &role); err == nil {
			tenants = append(tenants, map[string]interface{}{
				"id":         id,
				"name":       name,
				"plan":       plan,
				"role":       role,
				"created_at": createdAt,
			})
		}
	}
	return tenants, nil
}

func (h *Handlers) getAnalyticsData(tenantID int64) (map[string]interface{}, error) {
	// Get event counts by type
	rows, err := h.db.Query(`
		SELECT event_type, COUNT(*) as count
		FROM analytics_events
		WHERE tenant_id = ? AND created_at > datetime('now', '-30 days')
		GROUP BY event_type
		ORDER BY count DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	totalEvents := 0
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err == nil {
			events = append(events, map[string]interface{}{
				"event_type": eventType,
				"count":      count,
			})
			totalEvents += count
		}
	}

	// Get unique users count
	var uniqueUsers int
	h.db.QueryRow("SELECT COUNT(DISTINCT user_id) FROM analytics_events WHERE tenant_id = ? AND created_at > datetime('now', '-30 days')", tenantID).Scan(&uniqueUsers)

	return map[string]interface{}{
		"period":        "30_days",
		"total_events":  totalEvents,
		"unique_users":  uniqueUsers,
		"event_breakdown": events,
	}, nil
}

func (h *Handlers) getItems(tenantID int64) ([]map[string]interface{}, error) {
	rows, err := h.db.Query("SELECT id, title, note, created_at FROM items WHERE tenant_id = ? ORDER BY created_at DESC", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, note string
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &note, &createdAt); err == nil {
			items = append(items, map[string]interface{}{
				"id":         id,
				"title":      title,
				"note":       note,
				"created_at": createdAt,
			})
		}
	}
	return items, nil
}

// CSV export helper functions

func (h *Handlers) exportProfileCSV(cw *csv.Writer, userID int64) {
	profile, err := h.getUserProfile(userID)
	if err != nil {
		return
	}

	cw.Write([]string{"Field", "Value"})
	cw.Write([]string{"ID", fmt.Sprintf("%d", userID)})
	cw.Write([]string{"Email", profile["email"].(string)})
	cw.Write([]string{"Name", profile["name"].(string)})
	cw.Write([]string{"Created At", profile["created_at"].(time.Time).Format(time.RFC3339)})
}

func (h *Handlers) exportTenantsCSV(cw *csv.Writer, userID int64) {
	tenants, err := h.getUserTenants(userID)
	if err != nil {
		return
	}

	cw.Write([]string{"ID", "Name", "Plan", "Role", "Created At"})
	for _, tenant := range tenants {
		cw.Write([]string{
			fmt.Sprintf("%d", int64(tenant["id"].(int64))),
			tenant["name"].(string),
			tenant["plan"].(string),
			tenant["role"].(string),
			tenant["created_at"].(time.Time).Format(time.RFC3339),
		})
	}
}

func (h *Handlers) exportAnalyticsCSV(cw *csv.Writer, tenantID int64) {
	analytics, err := h.getAnalyticsData(tenantID)
	if err != nil {
		return
	}

	cw.Write([]string{"Metric", "Value"})
	cw.Write([]string{"Period", analytics["period"].(string)})
	cw.Write([]string{"Total Events", fmt.Sprintf("%d", analytics["total_events"].(int))})
	cw.Write([]string{"Unique Users", fmt.Sprintf("%d", analytics["unique_users"].(int))})
	cw.Write([]string{""}) // Empty row
	cw.Write([]string{"Event Type", "Count"})
	
	if events, ok := analytics["event_breakdown"].([]map[string]interface{}); ok {
		for _, event := range events {
			cw.Write([]string{
				event["event_type"].(string),
				fmt.Sprintf("%d", event["count"].(int)),
			})
		}
	}
}

func (h *Handlers) exportItemsCSV(cw *csv.Writer, tenantID int64) {
	items, err := h.getItems(tenantID)
	if err != nil {
		return
	}

	cw.Write([]string{"ID", "Title", "Note", "Created At"})
	for _, item := range items {
		cw.Write([]string{
			fmt.Sprintf("%d", int64(item["id"].(int64))),
			item["title"].(string),
			item["note"].(string),
			item["created_at"].(time.Time).Format(time.RFC3339),
		})
	}
}

// Utility functions

func (h *Handlers) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func (h *Handlers) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

func extractToken(r *http.Request) string {
	// Try Authorization header first
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Try cookie
	if cookie, err := r.Cookie("session"); err == nil {
		return cookie.Value
	}

	return ""
}

func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
