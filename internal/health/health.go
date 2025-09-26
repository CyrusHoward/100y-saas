package health

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"
	
	"100y-saas/internal/version"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type HealthResponse struct {
	Status    HealthStatus          `json:"status"`
	Version   string               `json:"version"`
	Timestamp time.Time            `json:"timestamp"`
	Uptime    time.Duration        `json:"uptime"`
	Checks    map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
	Status    HealthStatus `json:"status"`
	Message   string      `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time    `json:"timestamp"`
}

type HealthChecker struct {
	db        *sql.DB
	startTime time.Time
}

func NewHealthChecker(db *sql.DB) *HealthChecker {
	return &HealthChecker{
		db:        db,
		startTime: time.Now(),
	}
}

func (h *HealthChecker) Check() *HealthResponse {
	now := time.Now()
	response := &HealthResponse{
		Status:    StatusHealthy,
		Version:   version.Version,
		Timestamp: now,
		Uptime:    now.Sub(h.startTime),
		Checks:    make(map[string]CheckResult),
	}

	// Check database connectivity
	dbCheck := h.checkDatabase()
	response.Checks["database"] = dbCheck

	// Check disk space (basic)
	diskCheck := h.checkDisk()
	response.Checks["disk"] = diskCheck

	// Determine overall status
	overallStatus := StatusHealthy
	for _, check := range response.Checks {
		if check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		} else if check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}
	response.Status = overallStatus

	return response
}

func (h *HealthChecker) checkDatabase() CheckResult {
	start := time.Now()
	
	// Simple ping to check connection
	err := h.db.Ping()
	duration := time.Since(start)
	
	if err != nil {
		return CheckResult{
			Status:    StatusUnhealthy,
			Message:   "Database connection failed: " + err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	// Check if we can execute a simple query
	var count int
	err = h.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		return CheckResult{
			Status:    StatusDegraded,
			Message:   "Database query failed: " + err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	message := "Database is accessible"
	if duration > 100*time.Millisecond {
		return CheckResult{
			Status:    StatusDegraded,
			Message:   "Database is slow to respond",
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return CheckResult{
		Status:    StatusHealthy,
		Message:   message,
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

func (h *HealthChecker) checkDisk() CheckResult {
	start := time.Now()
	
	// Basic check - try to create a temporary file
	// In a real implementation, you might check actual disk usage
	tempFile, err := os.CreateTemp("", "health-check-*")
	duration := time.Since(start)
	
	if err != nil {
		return CheckResult{
			Status:    StatusUnhealthy,
			Message:   "Cannot write to disk: " + err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}
	
	tempFile.Close()
	os.Remove(tempFile.Name())
	
	return CheckResult{
		Status:    StatusHealthy,
		Message:   "Disk is writable",
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	health := h.Check()
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	switch health.Status {
	case StatusDegraded:
		statusCode = http.StatusOK // 200 but with degraded status
	case StatusUnhealthy:
		statusCode = http.StatusServiceUnavailable // 503
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(health)
}

// Simple liveness probe (always returns 200 OK if server is running)
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// Readiness probe (checks if app is ready to serve traffic)
func (h *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Quick database check
	err := h.db.Ping()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "not ready",
			"error":     "database not available",
			"timestamp": time.Now(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}
