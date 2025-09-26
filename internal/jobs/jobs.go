package jobs

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

type Job struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Payload     string    `json:"payload"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	ScheduledAt time.Time `json:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type JobHandler func(payload string) error

type JobProcessor struct {
	db       *sql.DB
	handlers map[string]JobHandler
	running  bool
}

func NewJobProcessor(db *sql.DB) *JobProcessor {
	return &JobProcessor{
		db:       db,
		handlers: make(map[string]JobHandler),
	}
}

func (jp *JobProcessor) RegisterHandler(jobType string, handler JobHandler) {
	jp.handlers[jobType] = handler
}

func (jp *JobProcessor) EnqueueJob(jobType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = jp.db.Exec(
		"INSERT INTO jobs (type, payload) VALUES (?, ?)",
		jobType, string(payloadJSON),
	)
	return err
}

func (jp *JobProcessor) EnqueueDelayedJob(jobType string, payload interface{}, delay time.Duration) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	scheduledAt := time.Now().Add(delay)
	_, err = jp.db.Exec(
		"INSERT INTO jobs (type, payload, scheduled_at) VALUES (?, ?, ?)",
		jobType, string(payloadJSON), scheduledAt,
	)
	return err
}

func (jp *JobProcessor) Start() {
	if jp.running {
		return
	}
	jp.running = true

	// Register built-in cleanup jobs
	jp.RegisterHandler("cleanup_sessions", jp.handleCleanupSessions)
	jp.RegisterHandler("cleanup_usage_events", jp.handleCleanupUsageEvents)

	// Start processing jobs
	go jp.processJobs()
	
	// Schedule periodic cleanup jobs
	go jp.scheduleCleanupJobs()
}

func (jp *JobProcessor) Stop() {
	jp.running = false
}

func (jp *JobProcessor) processJobs() {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for jp.running {
		select {
		case <-ticker.C:
			jp.processNextJob()
		}
	}
}

func (jp *JobProcessor) processNextJob() {
	tx, err := jp.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Get next pending job
	var job Job
	var startedAt, completedAt sql.NullTime
	
	err = tx.QueryRow(`
		SELECT id, type, payload, status, attempts, max_attempts, scheduled_at, started_at, completed_at, error
		FROM jobs 
		WHERE status = 'pending' AND scheduled_at <= CURRENT_TIMESTAMP
		ORDER BY scheduled_at ASC 
		LIMIT 1
	`).Scan(
		&job.ID, &job.Type, &job.Payload, &job.Status, &job.Attempts, &job.MaxAttempts,
		&job.ScheduledAt, &startedAt, &completedAt, &job.Error,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return // No jobs to process
		}
		log.Printf("Failed to fetch job: %v", err)
		return
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Mark job as running
	now := time.Now()
	_, err = tx.Exec(
		"UPDATE jobs SET status = 'running', started_at = ?, attempts = attempts + 1 WHERE id = ?",
		now, job.ID,
	)
	if err != nil {
		log.Printf("Failed to update job status: %v", err)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit job update: %v", err)
		return
	}

	// Process the job
	handler, exists := jp.handlers[job.Type]
	if !exists {
		jp.markJobFailed(job.ID, "no handler registered for job type: "+job.Type)
		return
	}

	err = handler(job.Payload)
	if err != nil {
		job.Attempts++
		if job.Attempts >= job.MaxAttempts {
			jp.markJobFailed(job.ID, err.Error())
		} else {
			jp.retryJob(job.ID, err.Error())
		}
		return
	}

	jp.markJobCompleted(job.ID)
}

func (jp *JobProcessor) markJobCompleted(jobID int64) {
	_, err := jp.db.Exec(
		"UPDATE jobs SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?",
		jobID,
	)
	if err != nil {
		log.Printf("Failed to mark job as completed: %v", err)
	}
}

func (jp *JobProcessor) markJobFailed(jobID int64, errorMsg string) {
	_, err := jp.db.Exec(
		"UPDATE jobs SET status = 'failed', completed_at = CURRENT_TIMESTAMP, error = ? WHERE id = ?",
		errorMsg, jobID,
	)
	if err != nil {
		log.Printf("Failed to mark job as failed: %v", err)
	}
}

func (jp *JobProcessor) retryJob(jobID int64, errorMsg string) {
	// Exponential backoff: 1min, 5min, 30min
	backoffMinutes := []int{1, 5, 30}
	var delay time.Duration

	var attempts int
	jp.db.QueryRow("SELECT attempts FROM jobs WHERE id = ?", jobID).Scan(&attempts)
	
	if attempts <= len(backoffMinutes) {
		delay = time.Duration(backoffMinutes[attempts-1]) * time.Minute
	} else {
		delay = 30 * time.Minute
	}

	scheduledAt := time.Now().Add(delay)
	_, err := jp.db.Exec(
		"UPDATE jobs SET status = 'pending', scheduled_at = ?, error = ? WHERE id = ?",
		scheduledAt, errorMsg, jobID,
	)
	if err != nil {
		log.Printf("Failed to reschedule job: %v", err)
	}
}

func (jp *JobProcessor) scheduleCleanupJobs() {
	ticker := time.NewTicker(24 * time.Hour) // Schedule daily
	defer ticker.Stop()

	// Schedule initial cleanup jobs
	jp.EnqueueJob("cleanup_sessions", nil)
	jp.EnqueueJob("cleanup_usage_events", nil)

	for jp.running {
		select {
		case <-ticker.C:
			// Schedule daily cleanup jobs
			jp.EnqueueJob("cleanup_sessions", nil)
			jp.EnqueueJob("cleanup_usage_events", nil)
		}
	}
}

// Built-in job handlers
func (jp *JobProcessor) handleCleanupSessions(payload string) error {
	_, err := jp.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	if err != nil {
		return err
	}
	log.Println("Cleaned up expired sessions")
	return nil
}

func (jp *JobProcessor) handleCleanupUsageEvents(payload string) error {
	// Keep usage events for 90 days
	_, err := jp.db.Exec(
		"DELETE FROM usage_events WHERE created_at < datetime('now', '-90 days')",
	)
	if err != nil {
		return err
	}
	log.Println("Cleaned up old usage events")
	return nil
}
