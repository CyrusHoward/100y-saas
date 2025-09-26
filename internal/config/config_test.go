package config

import (
	"os"
	"testing"
	"time"
)

func TestConfig_Load(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(*Config) error
	}{
		{
			name: "default config",
			envVars: map[string]string{},
			wantErr: false,
			check: func(c *Config) error {
				if c.Environment != "development" {
					t.Errorf("Expected environment 'development', got '%s'", c.Environment)
				}
				if c.Server.Port != 8080 {
					t.Errorf("Expected port 8080, got %d", c.Server.Port)
				}
				if c.Database.Path != "data/app.db" {
					t.Errorf("Expected database path 'data/app.db', got '%s'", c.Database.Path)
				}
				return nil
			},
		},
		{
			name: "production config",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
				"PORT":        "3000",
				"DATABASE_PATH": "/var/lib/app/app.db",
				"BASE_URL":    "https://example.com",
			},
			wantErr: false,
			check: func(c *Config) error {
				if c.Environment != "production" {
					t.Errorf("Expected environment 'production', got '%s'", c.Environment)
				}
				if c.Server.Port != 3000 {
					t.Errorf("Expected port 3000, got %d", c.Server.Port)
				}
				if c.Database.Path != "/var/lib/app/app.db" {
					t.Errorf("Expected database path '/var/lib/app/app.db', got '%s'", c.Database.Path)
				}
				if c.Server.BaseURL != "https://example.com" {
					t.Errorf("Expected base URL 'https://example.com', got '%s'", c.Server.BaseURL)
				}
				return nil
			},
		},
		{
			name: "invalid port",
			envVars: map[string]string{
				"PORT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid read timeout",
			envVars: map[string]string{
				"SERVER_READ_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid database max connections",
			envVars: map[string]string{
				"DATABASE_MAX_OPEN_CONNECTIONS": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			
			config, err := Load()
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if tt.check != nil {
				if err := tt.check(config); err != nil {
					t.Errorf("Config validation failed: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					IdleTimeout:     60 * time.Second,
					ShutdownTimeout: 30 * time.Second,
					BaseURL:         "http://localhost:8080",
				},
				Database: DatabaseConfig{
					Path:                  "data/app.db",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionLifetime:    time.Hour,
				},
				Auth: AuthConfig{
					PasswordMinLength: 8,
					SessionDuration:   24 * time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid environment",
			config: &Config{
				Environment: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port: 70000,
				},
			},
			wantErr: true,
		},
		{
			name: "missing database path",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Path: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid password min length",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Path: "data/app.db",
				},
				Auth: AuthConfig{
					PasswordMinLength: 3, // Too short
				},
			},
			wantErr: true,
		},
		{
			name: "invalid session duration",
			config: &Config{
				Environment: "development",
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					Path: "data/app.db",
				},
				Auth: AuthConfig{
					PasswordMinLength: 8,
					SessionDuration:   time.Minute, // Too short
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected validation error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		environment string
		expected    bool
	}{
		{"production", true},
		{"prod", true},
		{"development", false},
		{"dev", false},
		{"test", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			config := &Config{Environment: tt.environment}
			result := config.IsProduction()
			
			if result != tt.expected {
				t.Errorf("Expected IsProduction() = %v for environment '%s', got %v", 
					tt.expected, tt.environment, result)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		environment string
		expected    bool
	}{
		{"development", true},
		{"dev", true},
		{"production", false},
		{"prod", false},
		{"test", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			config := &Config{Environment: tt.environment}
			result := config.IsDevelopment()
			
			if result != tt.expected {
				t.Errorf("Expected IsDevelopment() = %v for environment '%s', got %v", 
					tt.expected, tt.environment, result)
			}
		})
	}
}

func TestParseEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
		wantErr      bool
	}{
		{
			name:         "valid integer",
			envValue:     "1234",
			defaultValue: 5678,
			expected:     1234,
			wantErr:      false,
		},
		{
			name:         "empty value uses default",
			envValue:     "",
			defaultValue: 5678,
			expected:     5678,
			wantErr:      false,
		},
		{
			name:         "invalid integer",
			envValue:     "not-a-number",
			defaultValue: 5678,
			expected:     0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_ENV_VAR", tt.envValue)
			defer os.Unsetenv("TEST_ENV_VAR")
			
			result, err := parseEnvInt("TEST_ENV_VAR", tt.defaultValue)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestParseEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
		wantErr      bool
	}{
		{
			name:         "valid duration",
			envValue:     "30s",
			defaultValue: 60 * time.Second,
			expected:     30 * time.Second,
			wantErr:      false,
		},
		{
			name:         "empty value uses default",
			envValue:     "",
			defaultValue: 60 * time.Second,
			expected:     60 * time.Second,
			wantErr:      false,
		},
		{
			name:         "invalid duration",
			envValue:     "not-a-duration",
			defaultValue: 60 * time.Second,
			expected:     0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_DURATION_VAR", tt.envValue)
			defer os.Unsetenv("TEST_DURATION_VAR")
			
			result, err := parseEnvDuration("TEST_DURATION_VAR", tt.defaultValue)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

// Helper function to clear relevant environment variables
func clearEnv() {
	envVars := []string{
		"ENVIRONMENT",
		"PORT",
		"DATABASE_PATH",
		"DATABASE_MAX_OPEN_CONNECTIONS",
		"DATABASE_MAX_IDLE_CONNECTIONS",
		"DATABASE_CONNECTION_LIFETIME",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
		"SERVER_SHUTDOWN_TIMEOUT",
		"BASE_URL",
		"PASSWORD_MIN_LENGTH",
		"SESSION_DURATION",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
		"SMTP_FROM",
	}
	
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func BenchmarkConfig_Load(b *testing.B) {
	// Clear environment for consistent benchmarking
	clearEnv()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	config := &Config{
		Environment: "development",
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			BaseURL:         "http://localhost:8080",
		},
		Database: DatabaseConfig{
			Path:                  "data/app.db",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionLifetime:    time.Hour,
		},
		Auth: AuthConfig{
			PasswordMinLength: 8,
			SessionDuration:   24 * time.Hour,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatalf("Config validation failed: %v", err)
		}
	}
}
