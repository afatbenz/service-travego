package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"

	"github.com/google/uuid"
)

type OrganizationUserRepository struct {
	db     *sql.DB
	driver string
}

func NewOrganizationUserRepository(db *sql.DB, driver string) *OrganizationUserRepository {
	return &OrganizationUserRepository{
		db:     db,
		driver: driver,
	}
}

// getPlaceholder returns query placeholder
func (r *OrganizationUserRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// GetOrganizationAndRoleByUserID retrieves organization_id
func (r *OrganizationUserRepository) GetOrganizationAndRoleByUserID(userID string) (organizationID string, roleUser int, err error) {
	query := fmt.Sprintf(`
		SELECT organization_id, organization_role
		FROM organization_users
		WHERE user_id = %s AND is_active = true
		LIMIT 1
	`, r.getPlaceholder(1))

	err = database.QueryRow(r.db, query, userID).Scan(&organizationID, &roleUser)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, sql.ErrNoRows
		}
		return "", 0, err
	}

	return organizationID, roleUser, nil
}

// CheckUserInOrganization checks user existence
func (r *OrganizationUserRepository) CheckUserInOrganization(userID, organizationID string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM organization_users
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var count int
	err := database.QueryRow(r.db, query, userID, organizationID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *OrganizationUserRepository) CreateOrganizationUser(orgUser *model.OrganizationUser) error {
	query := fmt.Sprintf(`
        INSERT INTO organization_users (
            uuid, user_id, organization_id, organization_role, is_active, created_at, created_by, updated_at, updated_by
        ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
	)

	_, err := database.Exec(
		r.db,
		query,
		orgUser.UUID,
		orgUser.UserID,
		orgUser.OrganizationID,
		orgUser.OrganizationRole,
		orgUser.IsActive,
		orgUser.CreatedAt,
		orgUser.CreatedBy,
		orgUser.UpdatedAt,
		orgUser.UpdatedBy,
	)
	return err
}

// CreateSubscription inserts a new subscription record
func (r *OrganizationUserRepository) CreateSubscription(orgID string) error {
	return r.CreateSubscriptionWithDuration(orgID, 30)
}

// CreateSubscriptionWithDuration inserts a new subscription record
func (r *OrganizationUserRepository) CreateSubscriptionWithDuration(orgID string, expiryDays int) error {
	subscriptionID := uuid.New().String()
	now := time.Now()
	activateDate := now.Format("2006-01-02")
	expiryDate := now.AddDate(0, 0, expiryDays).Format("2006-01-02")
	packageID := "trave01"
	subscriptionType := 1
	status := 1

	query := fmt.Sprintf(`
		INSERT INTO _subscription (
			subscription_id, organization_id, package_id, activate_date, expiry_date, created_at, subscription_type, status
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
	)

	_, err := database.Exec(
		r.db,
		query,
		subscriptionID,
		orgID,
		packageID,
		activateDate,
		expiryDate,
		now,
		subscriptionType,
		status,
	)
	return err
}

// UpdateOrganizationUserRole updates role
func (r *OrganizationUserRepository) UpdateOrganizationUserRole(userID, organizationID string, role int) error {
	query := fmt.Sprintf(`
		UPDATE organization_users
		SET organization_role = %s, updated_at = %s
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := database.Exec(r.db, query, role, time.Now(), userID, organizationID)
	return err
}

// GetUsersByOrganizationID retrieves users
func (r *OrganizationUserRepository) GetUsersByOrganizationID(organizationID string) ([]model.OrganizationUser, error) {
	query := fmt.Sprintf(`
		SELECT uuid, user_id, organization_id, organization_role, is_active, created_at, created_by, updated_at, updated_by
		FROM organization_users
		WHERE organization_id = %s
	`, r.getPlaceholder(1))

	rows, err := database.Query(r.db, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgUsers []model.OrganizationUser
	for rows.Next() {
		var orgUser model.OrganizationUser
		err := rows.Scan(
			&orgUser.UUID,
			&orgUser.UserID,
			&orgUser.OrganizationID,
			&orgUser.OrganizationRole,
			&orgUser.IsActive,
			&orgUser.CreatedAt,
			&orgUser.CreatedBy,
			&orgUser.UpdatedAt,
			&orgUser.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		orgUsers = append(orgUsers, orgUser)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orgUsers, nil
}

// GetOrganizationWithJoinDateByUserID retrieves data
func (r *OrganizationUserRepository) GetOrganizationWithJoinDateByUserID(userID string) (organizationCode, organizationName, companyName string, joinDate time.Time, organizationRole int, err error) {
	query := fmt.Sprintf(`
		SELECT o.organization_code, o.organization_name, o.company_name, ou.created_at, ou.organization_role
		FROM organization_users ou
		INNER JOIN organizations o ON ou.organization_id = o.organization_id
		WHERE ou.user_id = %s AND ou.is_active = true
		ORDER BY ou.created_at DESC
		LIMIT 1
	`, r.getPlaceholder(1))

	err = database.QueryRow(r.db, query, userID).Scan(&organizationCode, &organizationName, &companyName, &joinDate, &organizationRole)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", time.Time{}, 0, sql.ErrNoRows
		}
		return "", "", "", time.Time{}, 0, err
	}

	return organizationCode, organizationName, companyName, joinDate, organizationRole, nil
}

// GetUsers retrieves users from an organization with optional status filter
func (r *OrganizationUserRepository) GetUsers(organizationID string, status interface{}) ([]model.User, error) {
	query := fmt.Sprintf(`
		SELECT u.user_id, u.username, u.fullname, u.email, u.phone, u.address, u.city, u.province, u.avatar, u.created_at, ou.is_active
		FROM users u
		INNER JOIN organization_users ou ON u.user_id = ou.user_id
		WHERE ou.organization_id = %s
	`, r.getPlaceholder(1))

	var args []interface{}
	args = append(args, organizationID)

	if status != nil {
		query += fmt.Sprintf(" AND ou.is_active = %s", r.getPlaceholder(len(args)+1))
		args = append(args, status)
	}

	query += " ORDER BY u.fullname"

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		var fullname, address, city, province, avatar sql.NullString

		err := rows.Scan(
			&user.UserID,
			&user.Username,
			&fullname,
			&user.Email,
			&user.Phone,
			&address,
			&city,
			&province,
			&avatar,
			&user.CreatedAt,
			&user.IsActive,
		)
		if err != nil {
			return nil, err
		}

		if fullname.Valid {
			user.Name = fullname.String
		}
		if address.Valid {
			user.Address = address.String
		}
		if city.Valid {
			user.City = city.String
		}
		if province.Valid {
			user.Province = province.String
		}
		if avatar.Valid {
			user.Avatar = avatar.String
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateOrganizationUserActiveByUserID updates is_active on organization_users
func (r *OrganizationUserRepository) UpdateOrganizationUserActiveByUserID(userID, organizationID string, isActive bool) error {
	query := fmt.Sprintf(`
		UPDATE organization_users
		SET is_active = %s, updated_at = %s
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := database.Exec(r.db, query, isActive, time.Now(), userID, organizationID)
	return err
}

// DeleteOrganizationUserByUserID deletes a row from organization_users
func (r *OrganizationUserRepository) DeleteOrganizationUserByUserID(userID, organizationID string) error {
	query := fmt.Sprintf(`
		DELETE FROM organization_users
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	result, err := database.Exec(r.db, query, userID, organizationID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateUserIsActive updates is_active on users table
func (r *OrganizationUserRepository) UpdateUserIsActive(userID, organizationID string, isActive bool) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET is_active = %s, updated_at = %s
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	result, err := database.Exec(r.db, query, isActive, time.Now(), userID, organizationID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetRoleByUserIDAndOrgID retrieves role
func (r *OrganizationUserRepository) GetRoleByUserIDAndOrgID(userID, organizationID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT organization_role
		FROM organization_users
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var role int
	err := database.QueryRow(r.db, query, userID, organizationID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, sql.ErrNoRows
		}
		return 0, err
	}

	return role, nil
}
