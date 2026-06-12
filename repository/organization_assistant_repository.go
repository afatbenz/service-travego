package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *OrganizationRepository) assistantIDColumn() string {
	if r.driver == "mysql" {
		return "aa.assistant_id"
	}
	return "aa.assistant_id::text"
}

func (r *OrganizationRepository) assistantIDWhere(pos int) string {
	if r.driver == "mysql" {
		return "assistant_id = " + r.getPlaceholder(pos)
	}
	return "assistant_id::text = " + r.getPlaceholder(pos)
}

func (r *OrganizationRepository) assistantOrgWhere(alias string, pos int) string {
	column := "organization_id"
	if alias != "" {
		column = alias + ".organization_id"
	}
	if r.driver == "mysql" {
		return column + " = " + r.getPlaceholder(pos)
	}
	return column + "::text = " + r.getPlaceholder(pos)
}

func (r *OrganizationRepository) assistantUserWhere(pos int) string {
	if r.driver == "mysql" {
		return "user_id = " + r.getPlaceholder(pos)
	}
	return "user_id::text = " + r.getPlaceholder(pos)
}

func (r *OrganizationRepository) employeeUUIDColumn() string {
	if r.driver == "mysql" {
		return "uuid"
	}
	return "uuid::text"
}

func (r *OrganizationRepository) CountActiveAssistantAccounts(organizationID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM assistant_accounts
		WHERE %s
		  AND status = 1
	`, r.assistantOrgWhere("", 1))

	var total int
	if err := database.QueryRow(r.db, query, organizationID).Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func (r *OrganizationRepository) GetAssistantAccountLimit(organizationID string) (int, error) {
	query := fmt.Sprintf(`
		SELECT p.assistant_account_limit
		FROM _subscription s
		INNER JOIN _packages p ON s.package_id = p.package_id
		WHERE %s
		LIMIT 1
	`, r.assistantOrgWhere("s", 1))

	var limit sql.NullInt64
	if err := database.QueryRow(r.db, query, organizationID).Scan(&limit); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !limit.Valid {
		return 0, nil
	}

	return int(limit.Int64), nil
}

func (r *OrganizationRepository) ListAssistantAccounts(organizationID string) ([]model.AssistantAccountListItem, error) {
	employeeJoinExpr := "e.uuid = aa.user_id"
	roleJoinExpr := "e.role_id = orl.role_id"
	divisionJoinExpr := "orl.division_id = od.division_id"

	query := fmt.Sprintf(`
		SELECT
			assistant_id,
			employee_id,
			created_at,
			avatar,
			fullname,
			role_name,
			division_name,
			account_number,
			user_type
		FROM (
			SELECT
				COALESCE(%s, '') AS assistant_id,
				COALESCE(e.employee_id, '') AS employee_id,
				aa.created_at,
				COALESCE(e.avatar, '') AS avatar,
				COALESCE(e.fullname, '') AS fullname,
				COALESCE(orl.role_name, '') AS role_name,
				COALESCE(od.division_name, '') AS division_name,
				COALESCE(aa.account_number, '') AS account_number,
				aa.user_type AS user_type
			FROM assistant_accounts aa
			INNER JOIN employee e ON %s
			LEFT JOIN organization_roles orl ON %s
			LEFT JOIN organization_divisions od ON %s
			WHERE aa.user_type = 2
			  AND aa.status = 1
			  AND %s
			  AND %s
			  AND COALESCE(e.status, 0) > 0

			UNION ALL

			SELECT
				COALESCE(%s, '') AS assistant_id,
				'' AS employee_id,
				aa.created_at,
				'' AS avatar,
				COALESCE(aa.account_name, '') AS fullname,
				'Admin' AS role_name,
				'Admin' AS division_name,
				COALESCE(aa.account_number, '') AS account_number,
				aa.user_type AS user_type
			FROM assistant_accounts aa
			WHERE aa.user_type = 1
			  AND aa.status = 1
			  AND %s
		) assistants
		ORDER BY user_type ASC, created_at DESC
	`,
		r.assistantIDColumn(),
		employeeJoinExpr,
		roleJoinExpr,
		divisionJoinExpr,
		r.assistantOrgWhere("aa", 1),
		r.assistantOrgWhere("e", 2),
		r.assistantIDColumn(),
		r.assistantOrgWhere("aa", 3),
	)
	fmt.Println(query)

	rows, err := database.Query(r.db, query, organizationID, organizationID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.AssistantAccountListItem, 0)
	for rows.Next() {
		var item model.AssistantAccountListItem
		var createdAt sql.NullTime

		if err := rows.Scan(
			&item.AssistantID,
			&item.EmployeeID,
			&createdAt,
			&item.Avatar,
			&item.Fullname,
			&item.RoleName,
			&item.DivisionName,
			&item.AccountNumber,
			&item.UserType,
		); err != nil {
			return nil, err
		}

		if createdAt.Valid {
			item.CreatedAt = createdAt.Time.Format("2006-01-02 15:04:05")
		}

		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *OrganizationRepository) GetAssistantEmployeeTarget(organizationID, employeeID string) (*model.AssistantEmployeeTarget, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(%s, ''),
			COALESCE(employee_id, ''),
			COALESCE(fullname, ''),
			COALESCE(phone, '')
		FROM employee
		WHERE uuid = %s
		  AND %s
		  AND COALESCE(status, 0) > 0
		LIMIT 1
	`, r.employeeUUIDColumn(), r.getPlaceholder(1), r.assistantOrgWhere("", 2))
	fmt.Println(query)

	var item model.AssistantEmployeeTarget
	if err := database.QueryRow(r.db, query, employeeID, organizationID).Scan(
		&item.UUID,
		&item.EmployeeID,
		&item.Fullname,
		&item.Phone,
	); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *OrganizationRepository) CreateAssistantAccount(organizationID, createdBy string, userType int, userID *string, accountNumber, accountName string) (string, error) {
	assistantID := uuid.New().String()
	now := time.Now()

	var userRef interface{}
	if userID != nil && strings.TrimSpace(*userID) != "" {
		userRef = strings.TrimSpace(*userID)
	}

	query := fmt.Sprintf(`
		INSERT INTO assistant_accounts (
			assistant_id,
			organization_id,
			user_type,
			user_id,
			account_number,
			account_name,
			created_at,
			created_by,
			status
		) VALUES (
			%s, %s, %s, %s, %s, %s, %s, %s, 1
		)
	`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
	)

	_, err := database.Exec(
		r.db,
		query,
		assistantID,
		organizationID,
		userType,
		userRef,
		sql.NullString{String: accountNumber, Valid: accountNumber != ""},
		sql.NullString{String: accountName, Valid: accountName != ""},
		now,
		createdBy,
	)
	if err != nil {
		return "", err
	}

	return assistantID, nil
}

func (r *OrganizationRepository) UpdateAssistantAccount(organizationID, assistantID string, accountName, accountNumber *string) error {
	setParts := make([]string, 0, 2)
	args := make([]interface{}, 0, 4)
	pos := 1

	if accountNumber != nil {
		value := strings.TrimSpace(*accountNumber)
		setParts = append(setParts, "account_number = "+r.getPlaceholder(pos))
		args = append(args, sql.NullString{String: value, Valid: value != ""})
		pos++
	}

	if accountName != nil {
		value := strings.TrimSpace(*accountName)
		setParts = append(setParts, "account_name = "+r.getPlaceholder(pos))
		args = append(args, sql.NullString{String: value, Valid: value != ""})
		pos++
	}

	if len(setParts) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE assistant_accounts
		SET %s
		WHERE %s
		  AND %s
		  AND status = 1
	`, strings.Join(setParts, ", "), r.assistantIDWhere(pos), r.assistantOrgWhere("", pos+1))

	args = append(args, assistantID, organizationID)

	result, err := database.Exec(r.db, query, args...)
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

func (r *OrganizationRepository) DeleteAssistantAccountByUserID(organizationID, userID string) error {
	query := fmt.Sprintf(`
		UPDATE assistant_accounts
		SET status = 0
		WHERE %s
		  AND %s
		  AND status = 1
	`, r.assistantUserWhere(1), r.assistantOrgWhere("", 2))

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
