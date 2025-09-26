package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	SMTP      SMTPConfig
	Logging   LoggingConfig
	Analytics AnalyticsConfig
	Jobs      JobsConfig
}

type ServerConfig struct {
	Port            int           `json:"port"`
	Host            string        `json:"host"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	BaseURL         string        `json:"base_url"`
	Environment     string        `json:"environment"`
}

type DatabaseConfig struct {
	Path               string        `json:"path"`
	MaxOpenConnections int           `json:"max_open_connections"`
	MaxIdleConnections int           `json:"max_idle_connections"`
	ConnectionLifetime time.Duration `json:"connection_lifetime"`
	BusyTimeout        time.Duration `json:"busy_timeout"`
	WALMode            bool          `json:"wal_mode"`
}

type AuthConfig struct {
	Secret             string        `json:"-"` // Hidden from JSON
	SessionExpiry      time.Duration `json:"session_expiry"`
	PasswordMinLength  int           `json:"password_min_length"`
	MaxLoginAttempts   int           `json:"max_login_attempts"`
	LockoutDuration    time.Duration `json:"lockout_duration"`
	RequireEmailVerify bool          `json:"require_email_verify"`
}

type SMTPConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"-"` // Hidden from JSON
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	SkipVerify   bool   `json:"skip_verify"`
	Timeout      time.Duration `json:"timeout"`
}

type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"` // json or text
	Output     string `json:"output"` // stdout, stderr, file
	File       string `json:"file,omitempty"`
	MaxSize    int    `json:"max_size"`    // MB
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`     // days
}

type AnalyticsConfig struct {
	RetentionDays    int  `json:"retention_days"`
	BatchSize        int  `json:"batch_size"`
	FlushInterval    time.Duration `json:"flush_interval"`
	EnableRealtime   bool `json:"enable_realtime"`
	TrackAnonymous   bool `json:"track_anonymous"`
}

type JobsConfig struct {
	WorkerCount      int           `json:"worker_count"`
	PollInterval     time.Duration `json:"poll_interval"`
	RetryDelay       time.Duration `json:"retry_delay"`
	MaxRetries       int           `json:"max_retries"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	JobTimeout       time.Duration `json:"job_timeout"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvInt("PORT", 8080),
			Host:            getEnv("HOST", ""),
			ReadTimeout:     getEnvDuration("READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
			Environment:     getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			Path:               getEnv("DB_PATH", "data/app.db"),
			MaxOpenConnections: getEnvInt("DB_MAX_OPEN_CONNECTIONS", 25),
			MaxIdleConnections: getEnvInt("DB_MAX_IDLE_CONNECTIONS", 5),
			ConnectionLifetime: getEnvDuration("DB_CONNECTION_LIFETIME", 30*time.Minute),
			BusyTimeout:        getEnvDuration("DB_BUSY_TIMEOUT", 5*time.Second),
			WALMode:            getEnvBool("DB_WAL_MODE", true),
		},
		Auth: AuthConfig{
			Secret:             getEnv("APP_SECRET", "change-me-in-production"),
			SessionExpiry:      getEnvDuration("SESSION_EXPIRY", 24*time.Hour),
			PasswordMinLength:  getEnvInt("PASSWORD_MIN_LENGTH", 8),
			MaxLoginAttempts:   getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:    getEnvDuration("LOCKOUT_DURATION", 15*time.Minute),
			RequireEmailVerify: getEnvBool("REQUIRE_EMAIL_VERIFY", false),
		},
		SMTP: SMTPConfig{
			Host:        getEnv("SMTP_HOST", "localhost"),
			Port:        getEnvInt("SMTP_PORT", 587),
			Username:    getEnv("SMTP_USERNAME", ""),
			Password:    getEnv("SMTP_PASSWORD", ""),
			FromAddress: getEnv("SMTP_FROM_ADDRESS", "noreply@example.com"),
			FromName:    getEnv("SMTP_FROM_NAME", "100y SaaS"),
			UseTLS:      getEnvBool("SMTP_USE_TLS", true),
			SkipVerify:  getEnvBool("SMTP_SKIP_VERIFY", false),
			Timeout:     getEnvDuration("SMTP_TIMEOUT", 10*time.Second),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "INFO"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			File:       getEnv("LOG_FILE", ""),
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 28),
		},
		Analytics: AnalyticsConfig{
			RetentionDays:  getEnvInt("ANALYTICS_RETENTION_DAYS", 90),
			BatchSize:      getEnvInt("ANALYTICS_BATCH_SIZE", 1000),
			FlushInterval:  getEnvDuration("ANALYTICS_FLUSH_INTERVAL", 5*time.Minute),
			EnableRealtime: getEnvBool("ANALYTICS_ENABLE_REALTIME", true),
			TrackAnonymous: getEnvBool("ANALYTICS_TRACK_ANONYMOUS", false),
		},
		Jobs: JobsConfig{
			WorkerCount:     getEnvInt("JOBS_WORKER_COUNT", 2),
			PollInterval:    getEnvDuration("JOBS_POLL_INTERVAL", 5*time.Second),
			RetryDelay:      getEnvDuration("JOBS_RETRY_DELAY", 1*time.Minute),
			MaxRetries:      getEnvInt("JOBS_MAX_RETRIES", 3),
			CleanupInterval: getEnvDuration("JOBS_CLEANUP_INTERVAL", 1*time.Hour),
			JobTimeout:      getEnvDuration("JOBS_TIMEOUT", 10*time.Minute),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Auth validation
	if c.Auth.Secret == "change-me-in-production" && c.Server.Environment == "production" {
		return fmt.Errorf("APP_SECRET must be changed in production")
	}

	if c.Auth.PasswordMinLength < 4 {
		return fmt.Errorf("password minimum length must be at least 4")
	}

	// Database validation
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Logging validation
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	if !contains(validLogLevels, strings.ToUpper(c.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, strings.ToLower(c.Logging.Format)) {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development" || c.Server.Environment == "dev"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production" || c.Server.Environment == "prod"
}

func (c *Config) IsTest() bool {
	return c.Server.Environment == "test"
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
