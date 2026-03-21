package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"strings"
)

type CustomersRepository struct {
	db     *sql.DB
	driver string
}

func NewCustomersRepository(db *sql.DB, driver string) *CustomersRepository {
	return &CustomersRepository{db: db, driver: driver}
}

func (r *CustomersRepository) ListCustomers(orgID, customerName string) ([]model.CustomerListItem, error) {
	where := make([]string, 0, 2)
	args := make([]interface{}, 0, 2)
	pos := 1

	if orgID != "" {
		where = append(where, fmt.Sprintf("organization_id = %s", r.getPlaceholder(pos)))
		args = append(args, orgID)
		pos++
	}

	if customerName != "" {
		op := "LIKE"
		if r.driver == "postgres" || r.driver == "pgx" {
			op = "ILIKE"
		}
		where = append(where, fmt.Sprintf("customer_name %s %s", op, r.getPlaceholder(pos)))
		args = append(args, "%"+customerName+"%")
		pos++
	}

	query := `
		SELECT customer_id, customer_name, customer_phone, customer_email, customer_address, organization_id
		FROM customers
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += `
		GROUP BY customer_id, customer_name, customer_phone, customer_email, customer_address, organization_id
		ORDER BY customer_name
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CustomerListItem, 0)
	for rows.Next() {
		var it model.CustomerListItem
		if err := rows.Scan(&it.CustomerID, &it.CustomerName, &it.CustomerPhone, &it.CustomerEmail, &it.CustomerAddress, &it.OrganizationID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *CustomersRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}
