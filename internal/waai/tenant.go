package waai

import (
	"database/sql"
	"fmt"
	"strings"
)

// TenantInfo holds tenant/organization information for a WhatsApp contact
type TenantInfo struct {
	Phone            string
	Name             string
	Role             string
	OrganizationID   int64
	OrganizationName string
	IsActive         bool
}

// TenantRepository handles database queries for tenant information
type TenantRepository struct {
	db     *sql.DB
	driver string
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sql.DB, driver string) *TenantRepository {
	return &TenantRepository{
		db:     db,
		driver: driver,
	}
}

// GetTenantByPhone retrieves tenant information by WhatsApp phone number
func (tr *TenantRepository) GetTenantByPhone(phone string) (*TenantInfo, error) {
	// Normalize phone: remove @s.whatsapp.net suffix if present
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	// Remove any leading +
	phone = strings.TrimPrefix(phone, "+")

	query := `
		SELECT
			wc.phone,
			wc.name,
			wc.role,
			wc.organization_id,
			o.name as organization_name,
			wc.is_active
		FROM wa_contacts wc
		LEFT JOIN organizations o ON wc.organization_id = o.id
		WHERE wc.phone = $1 AND wc.is_active = true
	`

	var tenant TenantInfo
	err := tr.db.QueryRow(query, phone).Scan(
		&tenant.Phone,
		&tenant.Name,
		&tenant.Role,
		&tenant.OrganizationID,
		&tenant.OrganizationName,
		&tenant.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found for phone: %s", phone)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &tenant, nil
}

// GetOrganizationSnapshot retrieves business snapshot for an organization
func (tr *TenantRepository) GetOrganizationSnapshot(orgID int64) (map[string]interface{}, error) {
	// Query organization basic info
	orgQuery := `SELECT id, name FROM organizations WHERE id = $1`
	var org struct {
		ID   int64
		Name string
	}

	err := tr.db.QueryRow(orgQuery, orgID).Scan(&org.ID, &org.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found: %d", orgID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Query fleet count
	fleetQuery := `SELECT COUNT(*) FROM fleets WHERE organization_id = $1`
	var fleetCount int
	_ = tr.db.QueryRow(fleetQuery, orgID).Scan(&fleetCount)

	// Query fleet units count
	unitQuery := `SELECT COUNT(*) FROM fleet_units WHERE fleet_id IN (SELECT id FROM fleets WHERE organization_id = $1)`
	var unitCount int
	_ = tr.db.QueryRow(unitQuery, orgID).Scan(&unitCount)

	// Query today's bookings count
	bookingQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE organization_id = $1
		AND DATE(created_at) = CURRENT_DATE
	`
	var bookingCount int
	_ = tr.db.QueryRow(bookingQuery, orgID).Scan(&bookingCount)

	snapshot := map[string]interface{}{
		"organization_name": org.Name,
		"fleet_count":       fleetCount,
		"unit_count":        unitCount,
		"today_bookings":    bookingCount,
	}

	return snapshot, nil
}
