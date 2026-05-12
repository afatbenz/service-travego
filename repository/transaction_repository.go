package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"service-travego/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TransactionRepository struct {
	db     *sql.DB
	driver string
}

func NewTransactionRepository(db *sql.DB, driver string) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		driver: driver,
	}
}

func (r *TransactionRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (r *TransactionRepository) ListAllIncome(orgID string, req *model.TransactionListRequest) ([]model.TransactionListRow, error) {
	where := make([]string, 0, 8)
	args := make([]interface{}, 0, 8)

	where = append(where, "t.transaction_mark = 1")

	if r.driver == "postgres" || r.driver == "pgx" {
		where = append(where, "t.organization_id::text = "+r.getPlaceholder(len(args)+1))
	} else {
		where = append(where, "t.organization_id = "+r.getPlaceholder(len(args)+1))
	}
	args = append(args, orgID)

	if req.Month > 0 {
		if r.driver == "postgres" || r.driver == "pgx" {
			where = append(where, "EXTRACT(MONTH FROM t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		} else {
			where = append(where, "MONTH(t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		}
		args = append(args, req.Month)
	}
	if req.Year > 0 {
		if r.driver == "postgres" || r.driver == "pgx" {
			where = append(where, "EXTRACT(YEAR FROM t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		} else {
			where = append(where, "YEAR(t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		}
		args = append(args, req.Year)
	}
	if strings.TrimSpace(req.NoInvoice) != "" {
		if r.driver == "postgres" || r.driver == "pgx" {
			where = append(where, "t.invoice_number ILIKE "+r.getPlaceholder(len(args)+1))
		} else {
			where = append(where, "t.invoice_number LIKE "+r.getPlaceholder(len(args)+1))
		}
		args = append(args, "%"+strings.TrimSpace(req.NoInvoice)+"%")
	}
	if req.Source > 0 {
		where = append(where, "t.order_type = "+r.getPlaceholder(len(args)+1))
		args = append(args, req.Source)
	}
	if req.TransactionType > 0 {
		where = append(where, "t.transaction_type = "+r.getPlaceholder(len(args)+1))
		args = append(args, req.TransactionType)
	}

	query := fmt.Sprintf(`
		SELECT
			t.transaction_id,
			t.order_type,
			t.invoice_number,
			t.description,
			t.transaction_type,
			t.transaction_mark,
			t.status,
			COALESCE(t.amount, 0) as amount,
			t.transaction_date,
			t.created_at,
			COALESCE(u.fullname, '') as created_by
		FROM transactions t
		INNER JOIN users u ON t.created_by = u.user_id
		WHERE %s
		ORDER BY t.created_at DESC
	`, strings.Join(where, " AND "))

	rows, err := database.Query(r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.TransactionListRow, 0)
	for rows.Next() {
		var it model.TransactionListRow
		if err := rows.Scan(
			&it.TransactionID,
			&it.OrderType,
			&it.InvoiceNumber,
			&it.Description,
			&it.TransactionType,
			&it.TransactionMark,
			&it.Status,
			&it.Amount,
			&it.TransactionDate,
			&it.CreatedAt,
			&it.CreatedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type CreateManualTransactionRequest struct {
	Description     string
	TransactionDate string
	Status          int
	TransactionType int
	Amount          float64
	PaymentMethod   int
	BankAccount     string
	BankCode        string
}

func (r *TransactionRepository) CreateManualTransaction(orgID, userID string, req *CreateManualTransactionRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	invoiceNumber, err := utils.GenerateInvoiceNumberTx(tx, r.driver, orgID, 3, time.Now())
	if err != nil {
		return err
	}

	placeholder := r.getPlaceholder
	args := make([]interface{}, 0, 15)

	query := fmt.Sprintf(`
		INSERT INTO transactions (
			transaction_id,
			order_type,
			invoice_number,
			description,
			transaction_date,
			status,
			transaction_type,
			transaction_mark,
			amount,
			created_by,
			organization_id,
			payment_method,
			bank_account,
			bank_code,
			created_at
		) VALUES (
			%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8), placeholder(9), placeholder(10),
		placeholder(11), placeholder(12), placeholder(13), placeholder(14), placeholder(15),
	)

	args = append(args,
		transactionID.String(),
		3,
		invoiceNumber,
		req.Description,
		req.TransactionDate,
		req.Status,
		req.TransactionType,
		1,
		req.Amount,
		userID,
		orgID,
		req.PaymentMethod,
		req.BankAccount,
		req.BankCode,
		time.Now(),
	)

	_, err = tx.Exec(query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}
