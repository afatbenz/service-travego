package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"strings"
	"time"
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
		SELECT customer_id, customer_name, customer_phone, customer_email, customer_address, customer_company, customer_city, organization_id
		FROM customers
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += `
		GROUP BY customer_id, customer_name, customer_phone, customer_email, customer_address, customer_company, customer_city, organization_id
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
		if err := rows.Scan(&it.CustomerID, &it.CustomerName, &it.CustomerPhone, &it.CustomerEmail, &it.CustomerAddress, &it.CustomerCompany, &it.CustomerCityID, &it.OrganizationID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *CustomersRepository) CreateCustomer(orgID string, req *model.CustomerCreateRequest, customerID string) error {
	query := fmt.Sprintf(`
		INSERT INTO customers
			(customer_id, organization_id, customer_name, customer_phone, customer_telephone, customer_address, customer_city, customer_email, customer_company, customer_bod, created_at)
		VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

	_, err := r.db.Exec(
		query,
		customerID,
		orgID,
		req.CustomerName,
		req.CustomerPhone,
		req.CustomerTelephone,
		req.CustomerAddress,
		req.CustomerCity,
		req.CustomerEmail,
		req.CustomerCompany,
		req.CustomerBOD,
		time.Now(),
	)
	return err
}

func (r *CustomersRepository) GetCustomerDetail(orgID, customerID string) (map[string]interface{}, error) {
	query := fmt.Sprintf(
		"SELECT * FROM customers WHERE organization_id = %s AND customer_id = %s LIMIT 1",
		r.getPlaceholder(1),
		r.getPlaceholder(2),
	)
	rows, err := r.db.Query(query, orgID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(cols))
	for i, col := range cols {
		v := values[i]
		if b, ok := v.([]byte); ok {
			out[col] = string(b)
		} else {
			out[col] = v
		}
	}
	return out, nil
}

func (r *CustomersRepository) UpdateCustomer(orgID, customerID string, req *model.CustomerCreateRequest) error {
	sets := make([]string, 0, 8)
	args := make([]interface{}, 0, 10)
	pos := 1

	if req.CustomerName != "" {
		sets = append(sets, fmt.Sprintf("customer_name = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerName)
		pos++
	}
	if req.CustomerPhone != "" {
		sets = append(sets, fmt.Sprintf("customer_phone = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerPhone)
		pos++
	}
	if req.CustomerTelephone != "" {
		sets = append(sets, fmt.Sprintf("customer_telephone = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerTelephone)
		pos++
	}
	if req.CustomerAddress != "" {
		sets = append(sets, fmt.Sprintf("customer_address = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerAddress)
		pos++
	}
	if req.CustomerCity != "" {
		sets = append(sets, fmt.Sprintf("customer_city = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerCity)
		pos++
	}
	if req.CustomerEmail != "" {
		sets = append(sets, fmt.Sprintf("customer_email = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerEmail)
		pos++
	}
	if req.CustomerCompany != "" {
		sets = append(sets, fmt.Sprintf("customer_company = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerCompany)
		pos++
	}
	if req.CustomerBOD != "" {
		sets = append(sets, fmt.Sprintf("customer_bod = %s", r.getPlaceholder(pos)))
		args = append(args, req.CustomerBOD)
		pos++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = %s", r.getPlaceholder(pos)))
	args = append(args, time.Now())
	pos++

	query := fmt.Sprintf(
		"UPDATE customers SET %s WHERE organization_id = %s AND customer_id = %s",
		strings.Join(sets, ", "),
		r.getPlaceholder(pos),
		r.getPlaceholder(pos+1),
	)
	args = append(args, orgID, customerID)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CustomersRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}
