package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken        = errors.New("email already registered")
	ErrSessionExpired    = errors.New("session expired")
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	IsActive  bool      `json:"is_active"`
}

type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthService struct {
	db *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{db: db}
}

// hashPassword creates a simple SHA256 hash (for production, use bcrypt)
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// generateToken creates a random session token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (a *AuthService) Register(email, password string) (*User, error) {
	passwordHash := hashPassword(password)
	
	result, err := a.db.Exec(
		"INSERT INTO users (email, password_hash) VALUES (?, ?)",
		email, passwordHash,
	)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	userID, _ := result.LastInsertId()
	return a.GetUserByID(userID)
}

func (a *AuthService) Login(email, password string) (*Session, *User, error) {
	passwordHash := hashPassword(password)
	
	var userID int64
	err := a.db.QueryRow(
		"SELECT id FROM users WHERE email = ? AND password_hash = ? AND is_active = 1",
		email, passwordHash,
	).Scan(&userID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Update last login
	a.db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", userID)

	// Create session
	token, err := generateToken()
	if err != nil {
		return nil, nil, err
	}

	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour sessions
	
	_, err = a.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt,
	)
	if err != nil {
		return nil, nil, err
	}

	user, err := a.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}

	return &Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}, user, nil
}

func (a *AuthService) ValidateSession(token string) (*User, error) {
	if token == "" {
		return nil, ErrSessionExpired
	}

	var userID int64
	var expiresAt time.Time
	
	err := a.db.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE token = ?",
		token,
	).Scan(&userID, &expiresAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionExpired
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		a.db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return nil, ErrSessionExpired
	}

	return a.GetUserByID(userID)
}

func (a *AuthService) Logout(token string) error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (a *AuthService) GetUserByID(id int64) (*User, error) {
	var user User
	var lastLogin sql.NullTime
	
	err := a.db.QueryRow(
		"SELECT id, email, created_at, last_login, is_active FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &lastLogin, &user.IsActive)
	
	if err != nil {
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

// CleanupExpiredSessions removes old sessions (call this periodically)
func (a *AuthService) CleanupExpiredSessions() error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}
