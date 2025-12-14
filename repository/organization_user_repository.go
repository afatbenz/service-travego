package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"time"
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

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *OrganizationUserRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// GetOrganizationAndRoleByUserID retrieves organization_id and organization_role for a user
// Returns organization_id and organization_role from organization_users table where user_id matches and is_active = true
func (r *OrganizationUserRepository) GetOrganizationAndRoleByUserID(userID string) (organizationID string, roleUser int, err error) {
	query := fmt.Sprintf(`
		SELECT organization_id, organization_role
		FROM organization_users
		WHERE user_id = %s AND is_active = true
		LIMIT 1
	`, r.getPlaceholder(1))

	err = r.db.QueryRow(query, userID).Scan(&organizationID, &roleUser)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, sql.ErrNoRows
		}
		return "", 0, err
	}

	return organizationID, roleUser, nil
}

// CheckUserInOrganization checks if a user exists in organization_users for a given organization_id
func (r *OrganizationUserRepository) CheckUserInOrganization(userID, organizationID string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM organization_users
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var count int
	err := r.db.QueryRow(query, userID, organizationID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateOrganizationUser inserts a new organization_user record
func (r *OrganizationUserRepository) CreateOrganizationUser(orgUser *model.OrganizationUser) error {
    query := fmt.Sprintf(`
        INSERT INTO organization_users (
            uuid, user_id, organization_id, organization_role, is_active, created_at, created_by, updated_at, updated_by
        )
        SELECT %s, u.user_id, o.organization_id, %s, %s, %s, %s, %s, %s
        FROM users u, organizations o
        WHERE u.user_id = %s AND o.organization_id = %s
    `,
        r.getPlaceholder(1), // uuid
        r.getPlaceholder(4), // organization_role
        r.getPlaceholder(5), // is_active
        r.getPlaceholder(6), // created_at
        r.getPlaceholder(7), // created_by
        r.getPlaceholder(8), // updated_at
        r.getPlaceholder(9), // updated_by
        r.getPlaceholder(2), // filter users.user_id
        r.getPlaceholder(3), // filter organizations.organization_id
    )

    _, err := r.db.Exec(
        query,
        orgUser.UUID,
        orgUser.OrganizationRole,
        orgUser.IsActive,
        orgUser.CreatedAt,
        orgUser.CreatedBy,
        orgUser.UpdatedAt,
        orgUser.UpdatedBy,
        orgUser.UserID,
        orgUser.OrganizationID,
    )

    return err
}

// UpdateOrganizationUserRole updates the organization_role for an existing organization_user
func (r *OrganizationUserRepository) UpdateOrganizationUserRole(userID, organizationID string, roleUser int) error {
	query := fmt.Sprintf(`
		UPDATE organization_users
		SET organization_role = %s, updated_at = %s
		WHERE user_id = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	_, err := r.db.Exec(query, roleUser, time.Now(), userID, organizationID)
	return err
}

// GetUsersByOrganizationID retrieves all users in an organization
func (r *OrganizationUserRepository) GetUsersByOrganizationID(organizationID string) ([]model.OrganizationUser, error) {
	query := fmt.Sprintf(`
		SELECT uuid, user_id, organization_id, organization_role, is_active, created_at, created_by, updated_at, updated_by
		FROM organization_users
		WHERE organization_id = %s
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, organizationID)
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

// GetOrganizationWithJoinDateByUserID retrieves organization data with join date (created_at from organization_users)
// Returns organization_code, organization_name, company_name, join_date (created_at), and organization_role
func (r *OrganizationUserRepository) GetOrganizationWithJoinDateByUserID(userID string) (organizationCode, organizationName, companyName string, joinDate time.Time, organizationRole int, err error) {
	query := fmt.Sprintf(`
		SELECT o.organization_code, o.organization_name, o.company_name, ou.created_at, ou.organization_role
		FROM organization_users ou
		INNER JOIN organizations o ON ou.organization_id = o.organization_id
		WHERE ou.user_id = %s AND ou.is_active = true
		ORDER BY ou.created_at DESC
		LIMIT 1
	`, r.getPlaceholder(1))

	err = r.db.QueryRow(query, userID).Scan(&organizationCode, &organizationName, &companyName, &joinDate, &organizationRole)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", time.Time{}, 0, sql.ErrNoRows
		}
		return "", "", "", time.Time{}, 0, err
	}

	return organizationCode, organizationName, companyName, joinDate, organizationRole, nil
}
