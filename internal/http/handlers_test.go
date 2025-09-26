package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"100y-saas/internal/config"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);

	CREATE TABLE tenants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		plan TEXT DEFAULT 'free',
		owner_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users (id)
	);

	CREATE TABLE tenant_users (
		tenant_id INTEGER,
		user_id INTEGER,
		role TEXT DEFAULT 'member',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (tenant_id, user_id),
		FOREIGN KEY (tenant_id) REFERENCES tenants (id),
		FOREIGN KEY (user_id) REFERENCES users (id)
	);

	CREATE TABLE analytics_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id INTEGER NOT NULL,
		user_id INTEGER,
		event_type TEXT NOT NULL,
		properties TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tenant_id) REFERENCES tenants (id),
		FOREIGN KEY (user_id) REFERENCES users (id)
	);

	CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		note TEXT,
		tenant_id INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func setupTestConfig() *config.Config {
	return &config.Config{
		Environment: "test",
		Auth: config.AuthConfig{
			PasswordMinLength: 8,
			SessionDuration:   time.Hour * 24,
		},
		Database: config.DatabaseConfig{
			Path:                  ":memory:",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionLifetime:    time.Hour,
		},
		Server: config.ServerConfig{
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			BaseURL:         "http://localhost:8080",
		},
	}
}

func TestHandlers_Register(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid registration",
			requestBody: AuthRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: 200,
		},
		{
			name: "missing email",
			requestBody: AuthRequest{
				Password: "password123",
			},
			expectedStatus: 400,
			expectedError:  "Email and password required",
		},
		{
			name: "missing password",
			requestBody: AuthRequest{
				Email: "test@example.com",
			},
			expectedStatus: 400,
			expectedError:  "Email and password required",
		},
		{
			name: "password too short",
			requestBody: AuthRequest{
				Email:    "test@example.com",
				Password: "short",
			},
			expectedStatus: 400,
			expectedError:  "Password must be at least",
		},
		{
			name: "invalid json",
			requestBody: "invalid json",
			expectedStatus: 400,
			expectedError:  "Invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				json.NewEncoder(&body).Encode(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/api/auth/register", &body)
			req.Header.Set("Content-Type", "application/json")
			// Add mock CSRF token for test
			req.Header.Set("X-CSRF-Token", "test-token")

			// Mock CSRF validation for tests
			oldCSRF := handlers.csrf
			handlers.csrf = &CSRFProtection{}
			handlers.csrf.tokens.Store("test-token", time.Now().Add(time.Hour))

			w := httptest.NewRecorder()
			handlers.Register(w, req)

			handlers.csrf = oldCSRF

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response Response
				json.NewDecoder(w.Body).Decode(&response)
				if !strings.Contains(response.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, response.Error)
				}
			}
		})
	}
}

func TestHandlers_Login(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// First register a user
	_, err := handlers.auth.Register("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to register test user: %v", err)
	}

	tests := []struct {
		name           string
		requestBody    AuthRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid login",
			requestBody: AuthRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: 200,
		},
		{
			name: "invalid email",
			requestBody: AuthRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			expectedStatus: 401,
			expectedError:  "Invalid email or password",
		},
		{
			name: "invalid password",
			requestBody: AuthRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			expectedStatus: 401,
			expectedError:  "Invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", "test-token")

			// Mock CSRF validation
			oldCSRF := handlers.csrf
			handlers.csrf = &CSRFProtection{}
			handlers.csrf.tokens.Store("test-token", time.Now().Add(time.Hour))

			w := httptest.NewRecorder()
			handlers.Login(w, req)

			handlers.csrf = oldCSRF

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response Response
				json.NewDecoder(w.Body).Decode(&response)
				if !strings.Contains(response.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, response.Error)
				}
			}
		})
	}
}

func TestHandlers_RequireAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// Register and login a user to get a session
	user, err := handlers.auth.Register("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to register test user: %v", err)
	}

	session, _, err := handlers.auth.Login("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to login test user: %v", err)
	}

	testHandler := handlers.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	tests := []struct {
		name           string
		authHeader     string
		cookie         *http.Cookie
		expectedStatus int
	}{
		{
			name:           "valid session token in header",
			authHeader:     "Bearer " + session.Token,
			expectedStatus: 200,
		},
		{
			name: "valid session token in cookie",
			cookie: &http.Cookie{
				Name:  "session",
				Value: session.Token,
			},
			expectedStatus: 200,
		},
		{
			name:           "no auth token",
			expectedStatus: 401,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			w := httptest.NewRecorder()
			testHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandlers_GetTenants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// Register and login a user
	user, err := handlers.auth.Register("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to register test user: %v", err)
	}

	// Create a tenant
	tenant, err := handlers.saas.CreateTenant("Test Tenant", user.ID)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/tenants", nil)
	req.Header.Set("X-User-ID", "1")

	w := httptest.NewRecorder()
	handlers.GetTenants(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response Response
	json.NewDecoder(w.Body).Decode(&response)

	if !response.Success {
		t.Errorf("Expected success response, got error: %s", response.Error)
	}
}

func TestHandlers_CreateTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	tests := []struct {
		name           string
		requestBody    TenantRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid tenant creation",
			requestBody: TenantRequest{
				Name: "New Tenant",
			},
			expectedStatus: 200,
		},
		{
			name: "missing name",
			requestBody: TenantRequest{
				Name: "",
			},
			expectedStatus: 400,
			expectedError:  "Tenant name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/tenants/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "1")
			req.Header.Set("X-CSRF-Token", "test-token")

			// Mock CSRF validation
			oldCSRF := handlers.csrf
			handlers.csrf = &CSRFProtection{}
			handlers.csrf.tokens.Store("test-token", time.Now().Add(time.Hour))

			w := httptest.NewRecorder()
			handlers.CreateTenant(w, req)

			handlers.csrf = oldCSRF

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response Response
				json.NewDecoder(w.Body).Decode(&response)
				if !strings.Contains(response.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, response.Error)
				}
			}
		})
	}
}

func TestHandlers_ExportAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// Insert test data
	_, err := db.Exec("INSERT INTO items (title, note, tenant_id) VALUES (?, ?, ?)", "Test Item", "Test Note", 1)
	if err != nil {
		t.Fatalf("Failed to insert test item: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/export-all?format=json", nil)
	req.Header.Set("X-User-ID", "1")
	req.Header.Set("X-Tenant-ID", "1")
	req.Header.Set("X-User-Role", "owner")

	w := httptest.NewRecorder()
	handlers.ExportAll(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var exportData map[string]interface{}
	json.NewDecoder(w.Body).Decode(&exportData)

	if exportData["tenant_id"] != float64(1) {
		t.Errorf("Expected tenant_id 1, got %v", exportData["tenant_id"])
	}

	if exportData["format"] != "json" {
		t.Errorf("Expected format json, got %v", exportData["format"])
	}
}

func TestHandlers_CSRFProtection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// Test GET request generates CSRF token
	req := httptest.NewRequest("GET", "/api/csrf-token", nil)
	w := httptest.NewRecorder()
	handlers.CSRFTokenHandler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response Response
	json.NewDecoder(w.Body).Decode(&response)

	if !response.Success {
		t.Errorf("Expected success response, got error: %s", response.Error)
	}

	token, ok := response.Data.(map[string]interface{})["csrf_token"].(string)
	if !ok || token == "" {
		t.Errorf("Expected CSRF token in response, got %v", response.Data)
	}

	// Test CSRF validation middleware
	testHandler := handlers.CSRFProtectionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Test POST without CSRF token (should fail)
	req = httptest.NewRequest("POST", "/test", strings.NewReader("test"))
	w = httptest.NewRecorder()
	testHandler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for missing CSRF token, got %d", w.Code)
	}

	// Test POST with valid CSRF token (should succeed)
	req = httptest.NewRequest("POST", "/test", strings.NewReader("test"))
	req.Header.Set("X-CSRF-Token", token)
	w = httptest.NewRecorder()
	testHandler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for valid CSRF token, got %d", w.Code)
	}
}

// Benchmark tests
func BenchmarkHandlers_Register(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	cfg := setupTestConfig()
	handlers := NewHandlers(db, cfg)

	// Mock CSRF
	handlers.csrf = &CSRFProtection{}
	handlers.csrf.tokens.Store("test-token", time.Now().Add(time.Hour))

	requestBody := AuthRequest{
		Email:    "bench@example.com",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "test-token")

		w := httptest.NewRecorder()
		handlers.Register(w, req)

		// Clean up for next iteration
		db.Exec("DELETE FROM users WHERE email = ?", requestBody.Email)
	}
}

func TestMain(m *testing.M) {
	// Setup any global test configuration here
	os.Exit(m.Run())
}
