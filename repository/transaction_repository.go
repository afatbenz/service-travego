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

func (r *TransactionRepository) ListAllRevenue(orgID string, req *model.TransactionListRequest) ([]model.TransactionListRow, error) {
	return r.listTransactions(orgID, 1, req)
}

func (r *TransactionRepository) ListAllExpenses(orgID string, req *model.TransactionListRequest) ([]model.TransactionListRow, error) {
	return r.listTransactions(orgID, 2, req)
}

func (r *TransactionRepository) listTransactions(orgID string, TransactionItem int, req *model.TransactionListRequest) ([]model.TransactionListRow, error) {
	where := make([]string, 0, 8)
	args := make([]interface{}, 0, 8)

	where = append(where, "t.transaction_type = "+r.getPlaceholder(len(args)+1))
	args = append(args, TransactionItem)

	where = append(where, "t.organization_id::text = "+r.getPlaceholder(len(args)+1))
	args = append(args, orgID)

	if req.Month > 0 {
		where = append(where, "EXTRACT(MONTH FROM t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		args = append(args, req.Month)
	}
	if req.Year > 0 {
		where = append(where, "EXTRACT(YEAR FROM t.transaction_date) = "+r.getPlaceholder(len(args)+1))
		args = append(args, req.Year)
	}
	if strings.TrimSpace(req.NoInvoice) != "" {
		where = append(where, "t.invoice_number ILIKE "+r.getPlaceholder(len(args)+1))
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
			t.transaction_item,
			t.transaction_category,
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
	fmt.Println(query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactionItem sql.NullString

	out := make([]model.TransactionListRow, 0)
	for rows.Next() {
		var it model.TransactionListRow
		if err := rows.Scan(
			&it.TransactionID,
			&it.OrderType,
			&it.InvoiceNumber,
			&it.Description,
			&it.TransactionType,
			&transactionItem,
			&it.TransactionCategory,
			&it.Status,
			&it.Amount,
			&it.TransactionDate,
			&it.CreatedAt,
			&it.CreatedBy,
		); err != nil {
			return nil, err
		}

		if !transactionItem.Valid {
			it.TransactionItem = transactionItem.String
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type CreateManualTransactionRequest struct {
	OrderType       int
	OrderID         string
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
			transaction_item,
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

	orderType := req.OrderType
	if orderType == 0 {
		orderType = 3 // Default to Other if not provided
	}

	TransactionItem := 1 // Default to Income
	if req.TransactionType > 100 {
		TransactionItem = 2 // Expense
	}

	args = append(args,
		transactionID.String(),
		orderType,
		invoiceNumber,
		req.Description,
		req.TransactionDate,
		req.Status,
		req.TransactionType,
		TransactionItem,
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

	// Logic for transaction_orders and transaction_fleets
	if req.OrderID != "" {
		switch req.OrderType {
		case 1, 2:
			transactionOrderID, err := uuid.NewV7()
			if err != nil {
				return err
			}

			queryOrder := fmt.Sprintf(`
				INSERT INTO transaction_orders (
					transaction_order_id,
					transaction_id,
					order_id,
					organization_id,
					created_at,
					created_by
				) VALUES (
					%s, %s, %s, %s, %s, %s
				)
			`, placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5), placeholder(6))

			_, err = tx.Exec(queryOrder,
				transactionOrderID.String(),
				transactionID.String(),
				req.OrderID,
				orgID,
				time.Now(),
				userID,
			)
			if err != nil {
				return err
			}
		case 4:
			transactionFleetID, err := uuid.NewV7()
			if err != nil {
				return err
			}

			queryFleet := fmt.Sprintf(`
				INSERT INTO transaction_fleets (
					transaction_fleet_id,
					transaction_id,
					fleet_unit_id,
					organization_id,
					created_at,
					created_by
				) VALUES (
					%s, %s, %s, %s, %s, %s
				)
			`, placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5), placeholder(6))

			_, err = tx.Exec(queryFleet,
				transactionFleetID.String(),
				transactionID.String(),
				req.OrderID, // Assuming req.OrderID contains fleet_unit_id for order_type 4
				orgID,
				time.Now(),
				userID,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
