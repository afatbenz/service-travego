package waai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TenantRepository handles database queries for tenant information.
type TenantRepository struct {
	db      *sql.DB
	driver  string
	authMgr *AuthManager
}

// NewTenantRepository creates a new tenant repository.
func NewTenantRepository(db *sql.DB, driver string, authMgr *AuthManager) *TenantRepository {
	return &TenantRepository{
		db:      db,
		driver:  driver,
		authMgr: authMgr,
	}
}

func (tr *TenantRepository) getPlaceholder(pos int) string {
	if tr.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (tr *TenantRepository) assistantIDColumnExpr() string {
	if tr.driver == "mysql" {
		return "aa.assistant_id"
	}
	return "aa.assistant_id::text"
}

func (tr *TenantRepository) textCompareExpr(column string, pos int) string {
	if tr.driver == "mysql" {
		return fmt.Sprintf("%s = %s", column, tr.getPlaceholder(pos))
	}
	return fmt.Sprintf("%s::text = %s", column, tr.getPlaceholder(pos))
}

// GetTenantByPhone retrieves tenant information by WhatsApp phone number
func (tr *TenantRepository) GetTenantByPhone(ctx context.Context, phone string) (*TenantInfo, error) {
	// Normalize phone: remove @s.whatsapp.net suffix if present
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	// Remove any leading +
	phone = strings.TrimPrefix(phone, "+")

	if tr.authMgr != nil {
		authData, err := tr.authMgr.GetTenantAuth(ctx, phone)
		if err != nil {
			fmt.Println("Failed to get tenant auth from redis for phone", phone, ":", err)
		}
		if authData != nil {
			_ = tr.authMgr.RefreshTenantAuthTTL(ctx, phone)
			tenant := &TenantInfo{
				Phone:            phone,
				Name:             authData.FullName,
				FullName:         authData.FullName,
				Role:             authData.RoleName,
				OrganizationID:   authData.OrganizationID,
				OrganizationName: authData.OrganizationName,
				IsActive:         true,
				UserID:           authData.UserID,
				RoleName:         authData.RoleName,
				AccountNumber:    phone,
			}
			return tenant, nil
		}
	}

	query := fmt.Sprintf(`
		SELECT
			assistant_id,
			user_id,
			created_at,
			organization_id,
			avatar,
			fullname,
			role_name,
			division_name,
			account_number,
			user_type
		FROM (
			SELECT
				COALESCE(%[1]s, '') AS assistant_id,
				e.uuid AS user_id,
				aa.created_at,
				aa.organization_id,
				COALESCE(e.avatar, '') AS avatar,
				COALESCE(e.fullname, '') AS fullname,
				COALESCE(orl.role_name, '') AS role_name,
				COALESCE(od.division_name, '') AS division_name,
				COALESCE(aa.account_number, '') AS account_number,
				aa.user_type AS user_type
			FROM assistant_accounts aa
			INNER JOIN employee e ON e.uuid = aa.user_id
			LEFT JOIN organization_roles orl ON e.role_id = orl.role_id
			LEFT JOIN organization_divisions od ON orl.division_id = od.division_id
			WHERE aa.user_type = 2
			  AND aa.status = 1
			  AND COALESCE(e.status, 0) > 0

			UNION ALL

			SELECT
				COALESCE(%[1]s, '') AS assistant_id,
				u.user_id AS user_id,
				aa.created_at,
				aa.organization_id,
				'' AS avatar,
				COALESCE(aa.account_name, '') AS fullname,
				'Admin' AS role_name,
				'Admin' AS division_name,
				COALESCE(aa.account_number, '') AS account_number,
				aa.user_type AS user_type
			FROM assistant_accounts aa
			INNER JOIN organization_users ou ON aa.organization_id = ou.organization_id
			INNER JOIN users u ON u.user_id = ou.user_id
			WHERE aa.user_type = 1
			  AND aa.status = 1
		) assistants
		WHERE assistants.account_number = %[2]s
		ORDER BY user_type ASC, created_at DESC
		LIMIT 1
	`, tr.assistantIDColumnExpr(), tr.getPlaceholder(1))

	var tenant TenantInfo
	var createdAt sql.NullTime
	err := tr.db.QueryRowContext(ctx, query, phone).Scan(
		&tenant.AssistantID,
		&tenant.UserID,
		&createdAt,
		&tenant.OrganizationID,
		&tenant.Avatar,
		&tenant.FullName,
		&tenant.RoleName,
		&tenant.DivisionName,
		&tenant.AccountNumber,
		&tenant.UserType,
	)

	if err != nil {
		fmt.Println("query:", query)
		fmt.Println("Failed to query tenant for phone", phone, ":", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found for phone: %s", phone)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	tenant.Phone = tenant.AccountNumber
	if tenant.Name == "" {
		tenant.Name = tenant.FullName
	}
	if tenant.FullName == "" {
		tenant.FullName = tenant.Name
	}
	tenant.Role = tenant.RoleName
	tenant.IsActive = true

	if tr.authMgr != nil {
		if err := tr.authMgr.SaveTenantAuth(ctx, phone, &tenant); err != nil {
			fmt.Println("Failed to save tenant auth to redis for phone", phone, ":", err)
		}
	}

	return &tenant, nil
}

// GetOrganizationSnapshot retrieves business snapshot for an organization
func (tr *TenantRepository) GetOrganizationSnapshot(ctx context.Context, orgID string) (map[string]interface{}, error) {
	// Query organization basic info
	orgQuery := fmt.Sprintf("SELECT id, name FROM organizations WHERE %s", tr.textCompareExpr("id", 1))
	var org struct {
		ID   string
		Name string
	}

	err := tr.db.QueryRowContext(ctx, orgQuery, orgID).Scan(&org.ID, &org.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found: %s", orgID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Query fleet count
	fleetQuery := fmt.Sprintf("SELECT COUNT(*) FROM fleets WHERE %s", tr.textCompareExpr("organization_id", 1))
	var fleetCount int
	_ = tr.db.QueryRowContext(ctx, fleetQuery, orgID).Scan(&fleetCount)

	// Query fleet units count
	unitQuery := fmt.Sprintf("SELECT COUNT(*) FROM fleet_units WHERE fleet_id IN (SELECT id FROM fleets WHERE %s)", tr.textCompareExpr("organization_id", 1))
	var unitCount int
	_ = tr.db.QueryRowContext(ctx, unitQuery, orgID).Scan(&unitCount)

	// Query today's bookings count
	bookingQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE ` + tr.textCompareExpr("organization_id", 1) + `
		AND DATE(created_at) = CURRENT_DATE
	`
	var bookingCount int
	_ = tr.db.QueryRowContext(ctx, bookingQuery, orgID).Scan(&bookingCount)

	snapshot := map[string]interface{}{
		"organization_name": org.Name,
		"fleet_count":       fleetCount,
		"unit_count":        unitCount,
		"today_bookings":    bookingCount,
	}

	return snapshot, nil
}
