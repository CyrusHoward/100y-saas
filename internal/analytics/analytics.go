package analytics

import (
	"database/sql"
	"encoding/json"
	"time"
)

type AnalyticsService struct {
	db *sql.DB
}

type UsageSummary struct {
	TenantID    int64            `json:"tenant_id"`
	Period      string           `json:"period"`
	TotalEvents int              `json:"total_events"`
	EventCounts map[string]int   `json:"event_counts"`
	ActiveUsers int              `json:"active_users"`
	StartDate   time.Time        `json:"start_date"`
	EndDate     time.Time        `json:"end_date"`
}

type TopUser struct {
	UserID     int64  `json:"user_id"`
	Email      string `json:"email"`
	EventCount int    `json:"event_count"`
}

func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (a *AnalyticsService) TrackEvent(tenantID, userID int64, eventType string, data map[string]interface{}) error {
	var eventData string
	if data != nil {
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return err
		}
		eventData = string(dataJSON)
	}

	_, err := a.db.Exec(
		"INSERT INTO usage_events (tenant_id, user_id, event_type, event_data) VALUES (?, ?, ?, ?)",
		tenantID, userID, eventType, eventData,
	)
	return err
}

func (a *AnalyticsService) GetDailySummary(tenantID int64, date time.Time) (*UsageSummary, error) {
	startDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endDate := startDate.Add(24 * time.Hour)

	return a.getSummary(tenantID, startDate, endDate, "daily")
}

func (a *AnalyticsService) GetMonthlySummary(tenantID int64, year int, month time.Month) (*UsageSummary, error) {
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	return a.getSummary(tenantID, startDate, endDate, "monthly")
}

func (a *AnalyticsService) getSummary(tenantID int64, startDate, endDate time.Time, period string) (*UsageSummary, error) {
	summary := &UsageSummary{
		TenantID:    tenantID,
		Period:      period,
		EventCounts: make(map[string]int),
		StartDate:   startDate,
		EndDate:     endDate,
	}

	// Get total events and event type counts
	rows, err := a.db.Query(`
		SELECT event_type, COUNT(*) as count
		FROM usage_events 
		WHERE tenant_id = ? AND created_at >= ? AND created_at < ?
		GROUP BY event_type
	`, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalEvents := 0
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		summary.EventCounts[eventType] = count
		totalEvents += count
	}
	summary.TotalEvents = totalEvents

	// Get active users count
	err = a.db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) 
		FROM usage_events 
		WHERE tenant_id = ? AND created_at >= ? AND created_at < ?
	`, tenantID, startDate, endDate).Scan(&summary.ActiveUsers)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (a *AnalyticsService) GetTopUsers(tenantID int64, startDate, endDate time.Time, limit int) ([]*TopUser, error) {
	rows, err := a.db.Query(`
		SELECT ue.user_id, u.email, COUNT(*) as event_count
		FROM usage_events ue
		JOIN users u ON ue.user_id = u.id
		WHERE ue.tenant_id = ? AND ue.created_at >= ? AND ue.created_at < ?
		GROUP BY ue.user_id, u.email
		ORDER BY event_count DESC
		LIMIT ?
	`, tenantID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*TopUser
	for rows.Next() {
		var user TopUser
		if err := rows.Scan(&user.UserID, &user.Email, &user.EventCount); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

func (a *AnalyticsService) GetEventTimeline(tenantID int64, eventType string, startDate, endDate time.Time) (map[string]int, error) {
	rows, err := a.db.Query(`
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM usage_events 
		WHERE tenant_id = ? AND event_type = ? AND created_at >= ? AND created_at < ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, tenantID, eventType, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeline := make(map[string]int)
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		timeline[date] = count
	}

	return timeline, nil
}

// Simple real-time stats for dashboard
func (a *AnalyticsService) GetRealtimeStats(tenantID int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Today's events
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	
	var todayEvents int
	err := a.db.QueryRow(`
		SELECT COUNT(*) FROM usage_events 
		WHERE tenant_id = ? AND created_at >= ? AND created_at < ?
	`, tenantID, today, tomorrow).Scan(&todayEvents)
	if err != nil {
		return nil, err
	}
	stats["today_events"] = todayEvents

	// Active users last 24h
	var activeUsers24h int
	past24h := time.Now().Add(-24 * time.Hour)
	err = a.db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) FROM usage_events 
		WHERE tenant_id = ? AND created_at >= ?
	`, tenantID, past24h).Scan(&activeUsers24h)
	if err != nil {
		return nil, err
	}
	stats["active_users_24h"] = activeUsers24h

	// Total items
	var totalItems int
	err = a.db.QueryRow("SELECT COUNT(*) FROM items WHERE tenant_id = ?", tenantID).Scan(&totalItems)
	if err != nil {
		return nil, err
	}
	stats["total_items"] = totalItems

	// Recent activity (last 10 events)
	rows, err := a.db.Query(`
		SELECT event_type, created_at FROM usage_events 
		WHERE tenant_id = ? 
		ORDER BY created_at DESC LIMIT 10
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recentActivity []map[string]interface{}
	for rows.Next() {
		var eventType string
		var createdAt time.Time
		if err := rows.Scan(&eventType, &createdAt); err != nil {
			return nil, err
		}
		recentActivity = append(recentActivity, map[string]interface{}{
			"event_type": eventType,
			"created_at": createdAt,
		})
	}
	stats["recent_activity"] = recentActivity

	return stats, nil
}
