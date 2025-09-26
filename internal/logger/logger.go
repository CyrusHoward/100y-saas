package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelFatal Level = "FATAL"
)

type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     Level             `json:"level"`
	Message   string            `json:"message"`
	Component string            `json:"component,omitempty"`
	UserID    *int64            `json:"user_id,omitempty"`
	TenantID  *int64            `json:"tenant_id,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Duration  *time.Duration    `json:"duration,omitempty"`
	Error     string            `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type Logger struct {
	component string
	logger    *log.Logger
}

func New(component string) *Logger {
	return &Logger{
		component: component,
		logger:    log.New(os.Stdout, "", 0), // No prefix, we handle formatting
	}
}

func (l *Logger) log(level Level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Component: l.component,
	}

	// Extract common fields
	if fields != nil {
		if userID, ok := fields["user_id"].(int64); ok {
			entry.UserID = &userID
			delete(fields, "user_id")
		}
		if tenantID, ok := fields["tenant_id"].(int64); ok {
			entry.TenantID = &tenantID
			delete(fields, "tenant_id")
		}
		if requestID, ok := fields["request_id"].(string); ok {
			entry.RequestID = requestID
			delete(fields, "request_id")
		}
		if duration, ok := fields["duration"].(time.Duration); ok {
			entry.Duration = &duration
			delete(fields, "duration")
		}
		if err, ok := fields["error"].(string); ok {
			entry.Error = err
			delete(fields, "error")
		}
		if len(fields) > 0 {
			entry.Data = fields
		}
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		l.logger.Printf("LOG_ERROR: Failed to marshal log entry: %v", err)
		l.logger.Printf("%s [%s] %s", entry.Timestamp.Format(time.RFC3339), level, message)
		return
	}

	l.logger.Println(string(jsonBytes))
}

func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, message, f)
}

func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, message, f)
}

func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, message, f)
}

func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, message, f)
}

func (l *Logger) Fatal(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelFatal, message, f)
	os.Exit(1)
}

// Helper methods for common logging scenarios
func (l *Logger) RequestStart(method, path, userAgent, requestID string) {
	l.Info("Request started", map[string]interface{}{
		"request_id": requestID,
		"method":     method,
		"path":       path,
		"user_agent": userAgent,
	})
}

func (l *Logger) RequestEnd(method, path, requestID string, statusCode int, duration time.Duration) {
	level := LevelInfo
	if statusCode >= 500 {
		level = LevelError
	} else if statusCode >= 400 {
		level = LevelWarn
	}

	l.log(level, "Request completed", map[string]interface{}{
		"request_id":  requestID,
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration":    duration,
	})
}

func (l *Logger) DatabaseQuery(query string, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"query":    query,
		"duration": duration,
	}

	if err != nil {
		fields["error"] = err.Error()
		l.Error("Database query failed", fields)
	} else {
		l.Debug("Database query executed", fields)
	}
}

func (l *Logger) UserAction(userID, tenantID int64, action string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"user_id":   userID,
		"tenant_id": tenantID,
		"action":    action,
	}
	for k, v := range details {
		fields[k] = v
	}
	l.Info("User action", fields)
}

func (l *Logger) JobProcessed(jobType string, jobID int64, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"job_type": jobType,
		"job_id":   jobID,
		"duration": duration,
	}

	if err != nil {
		fields["error"] = err.Error()
		l.Error("Job processing failed", fields)
	} else {
		l.Info("Job processed successfully", fields)
	}
}
