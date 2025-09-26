package saas

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrSubscriptionLimit = errors.New("subscription limit exceeded")
	ErrAccessDenied     = errors.New("access denied")
)

type Tenant struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int64     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

type Subscription struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	MaxItems  int       `json:"max_items"`
	MaxUsers  int       `json:"max_users"`
}

type TenantUser struct {
	TenantID int64     `json:"tenant_id"`
	UserID   int64     `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type SaaSService struct {
	db *sql.DB
}

func NewSaaSService(db *sql.DB) *SaaSService {
	return &SaaSService{db: db}
}

func (s *SaaSService) CreateTenant(name string, ownerID int64) (*Tenant, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create tenant
	result, err := tx.Exec(
		"INSERT INTO tenants (name, owner_id) VALUES (?, ?)",
		name, ownerID,
	)
	if err != nil {
		return nil, err
	}

	tenantID, _ := result.LastInsertId()

	// Add owner to tenant_users
	_, err = tx.Exec(
		"INSERT INTO tenant_users (tenant_id, user_id, role) VALUES (?, ?, 'owner')",
		tenantID, ownerID,
	)
	if err != nil {
		return nil, err
	}

	// Create default subscription
	_, err = tx.Exec(
		"INSERT INTO subscriptions (tenant_id, plan, status) VALUES (?, 'free', 'active')",
		tenantID,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetTenant(tenantID)
}

func (s *SaaSService) GetTenant(tenantID int64) (*Tenant, error) {
	var tenant Tenant
	err := s.db.QueryRow(
		"SELECT id, name, owner_id, created_at, is_active FROM tenants WHERE id = ?",
		tenantID,
	).Scan(&tenant.ID, &tenant.Name, &tenant.OwnerID, &tenant.CreatedAt, &tenant.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return &tenant, nil
}

func (s *SaaSService) GetUserTenants(userID int64) ([]*Tenant, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.owner_id, t.created_at, t.is_active 
		FROM tenants t 
		JOIN tenant_users tu ON t.id = tu.tenant_id 
		WHERE tu.user_id = ? AND t.is_active = 1
		ORDER BY t.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.OwnerID, &t.CreatedAt, &t.IsActive); err != nil {
			return nil, err
		}
		tenants = append(tenants, &t)
	}

	return tenants, nil
}

func (s *SaaSService) HasAccess(userID, tenantID int64) (bool, string) {
	var role string
	err := s.db.QueryRow(
		"SELECT role FROM tenant_users WHERE user_id = ? AND tenant_id = ?",
		userID, tenantID,
	).Scan(&role)

	if err != nil {
		return false, ""
	}

	return true, role
}

func (s *SaaSService) GetSubscription(tenantID int64) (*Subscription, error) {
	var sub Subscription
	var endsAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, tenant_id, plan, status, starts_at, ends_at, max_items, max_users 
		FROM subscriptions 
		WHERE tenant_id = ? AND status = 'active'
		ORDER BY id DESC LIMIT 1
	`, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.Plan, &sub.Status,
		&sub.StartsAt, &endsAt, &sub.MaxItems, &sub.MaxUsers,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no active subscription found")
		}
		return nil, err
	}

	if endsAt.Valid {
		sub.EndsAt = &endsAt.Time
	}

	return &sub, nil
}

func (s *SaaSService) CheckItemLimit(tenantID int64) error {
	sub, err := s.GetSubscription(tenantID)
	if err != nil {
		return err
	}

	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM items WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		return err
	}

	if count >= sub.MaxItems {
		return ErrSubscriptionLimit
	}

	return nil
}

func (s *SaaSService) CheckUserLimit(tenantID int64) error {
	sub, err := s.GetSubscription(tenantID)
	if err != nil {
		return err
	}

	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM tenant_users WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		return err
	}

	if count >= sub.MaxUsers {
		return ErrSubscriptionLimit
	}

	return nil
}

func (s *SaaSService) AddUserToTenant(tenantID, userID int64, role string) error {
	if err := s.CheckUserLimit(tenantID); err != nil {
		return err
	}

	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO tenant_users (tenant_id, user_id, role) VALUES (?, ?, ?)",
		tenantID, userID, role,
	)
	return err
}

// Usage tracking for analytics
func (s *SaaSService) TrackEvent(tenantID, userID int64, eventType, eventData string) error {
	_, err := s.db.Exec(
		"INSERT INTO usage_events (tenant_id, user_id, event_type, event_data) VALUES (?, ?, ?, ?)",
		tenantID, userID, eventType, eventData,
	)
	return err
}
