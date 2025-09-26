package auth

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Create tables
	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME,
			is_active BOOLEAN DEFAULT 1
		);

		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestAuthService_Register(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	authService := NewAuthService(db)

	// Test successful registration
	user, err := authService.Register("test@example.com", "password123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if user == nil {
		t.Error("Expected user to be returned")
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email to be test@example.com, got %s", user.Email)
	}

	if !user.IsActive {
		t.Error("Expected user to be active")
	}

	// Test duplicate email registration
	_, err = authService.Register("test@example.com", "password456")
	if err != ErrEmailTaken {
		t.Errorf("Expected ErrEmailTaken, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	authService := NewAuthService(db)

	// Register a user first
	_, err := authService.Register("test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	// Test successful login
	session, user, err := authService.Login("test@example.com", "password123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if session == nil {
		t.Error("Expected session to be returned")
	}

	if user == nil {
		t.Error("Expected user to be returned")
	}

	if session.Token == "" {
		t.Error("Expected session token to be set")
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("Expected session to not be expired")
	}

	// Test invalid credentials
	_, _, err = authService.Login("test@example.com", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}

	// Test non-existent user
	_, _, err = authService.Login("nonexistent@example.com", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ValidateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	authService := NewAuthService(db)

	// Register and login a user
	_, err := authService.Register("test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	session, _, err := authService.Login("test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	// Test valid session
	user, err := authService.ValidateSession(session.Token)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if user == nil {
		t.Error("Expected user to be returned")
	}

	// Test invalid token
	_, err = authService.ValidateSession("invalid-token")
	if err != ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired, got %v", err)
	}

	// Test empty token
	_, err = authService.ValidateSession("")
	if err != ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired, got %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	authService := NewAuthService(db)

	// Register and login a user
	_, err := authService.Register("test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	session, _, err := authService.Login("test@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	// Logout
	err = authService.Logout(session.Token)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Try to validate session after logout
	_, err = authService.ValidateSession(session.Token)
	if err != ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired after logout, got %v", err)
	}
}

func TestAuthService_CleanupExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	authService := NewAuthService(db)

	// Insert an expired session manually
	expiredTime := time.Now().Add(-1 * time.Hour)
	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		"expired-token", 1, expiredTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a valid session
	validTime := time.Now().Add(1 * time.Hour)
	_, err = db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		"valid-token", 1, validTime,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Check initial count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("Expected 2 sessions, got %d", count)
	}

	// Cleanup expired sessions
	err = authService.CleanupExpiredSessions()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check count after cleanup
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session after cleanup, got %d", count)
	}

	// Verify the remaining session is the valid one
	var token string
	err = db.QueryRow("SELECT token FROM sessions").Scan(&token)
	if err != nil {
		t.Fatal(err)
	}
	if token != "valid-token" {
		t.Errorf("Expected valid-token to remain, got %s", token)
	}
}

func TestHashPassword(t *testing.T) {
	password := "test123"
	hash1 := hashPassword(password)
	hash2 := hashPassword(password)

	// Same password should produce same hash
	if hash1 != hash2 {
		t.Error("Same password should produce same hash")
	}

	// Different password should produce different hash
	hash3 := hashPassword("different")
	if hash1 == hash3 {
		t.Error("Different passwords should produce different hashes")
	}

	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	token2, err := generateToken()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Tokens should have expected length (64 hex chars)
	if len(token1) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token1))
	}

	if len(token2) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token2))
	}
}
