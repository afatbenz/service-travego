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
			COALESCE(t.invoice_number, '') AS invoice_number,
			t.description,
			t.transaction_type,
			t.transaction_item,
			t.transaction_category,
			t.payment_type,
			t.status,
			COALESCE(t.amount, 0) as amount,
			t.transaction_date,
			t.created_at,
			COALESCE(u.fullname, '') as created_by
		FROM transactions t
		LEFT JOIN users u ON t.created_by = u.user_id
		WHERE t.status = 1 AND %s
		ORDER BY t.created_at DESC
	`, strings.Join(where, " AND "))
	rows, err := database.Query(r.db, query, args...)
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
			&it.PaymentType,
			&it.Status,
			&it.Amount,
			&it.TransactionDate,
			&it.CreatedAt,
			&it.CreatedBy,
		); err != nil {
			return nil, err
		}

		if transactionItem.Valid {
			it.TransactionItem = transactionItem.String
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TransactionRepository) DeleteFleetTripExpense(orgID, scheduleNumber, transactionTripID string) error {
	placeholder := r.getPlaceholder

	transactionTripIDExpr := "transaction_trip_id = " + placeholder(1)
	scheduleNumberExpr := "schedule_number = " + placeholder(2)
	orgExpr := "organization_id = " + placeholder(3)

	if r.driver == "postgres" || r.driver == "pgx" {
		transactionTripIDExpr = "transaction_trip_id::text = " + placeholder(1)
		orgExpr = "organization_id::text = " + placeholder(3)
	}

	query := fmt.Sprintf(`
		DELETE FROM transaction_fleet_trips
		WHERE %s AND %s AND %s
	`, transactionTripIDExpr, scheduleNumberExpr, orgExpr)

	result, err := r.db.Exec(query, transactionTripID, scheduleNumber, orgID)
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

type CreateExpenseTransactionRequest struct {
	Amount              float64
	Description         string
	UnitID              string
	PaymentMethod       int
	PaymentType         int
	TransactionDate     time.Time
	TransactionCategory string
	TransactionItem     string
}

type UpdateExpenseTransactionRequest struct {
	TransactionID       string
	Amount              float64
	UnitID              string
	PaymentMethod       int
	TransactionDate     time.Time
	TransactionCategory string
	TransactionItem     string
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
			payment_type,
			transaction_type,
			transaction_item,
			amount,
			created_by,
			organization_id,
			payment_method,
			bank_account,
			bank_code,
			created_at,
			status
		) VALUES (
			%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 1
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

func (r *TransactionRepository) CreateExpenseTransaction(orgID, userID string, req *CreateExpenseTransactionRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	transactionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	orderType := 4
	if strings.TrimSpace(req.UnitID) != "" {
		orderType = 1
	}

	invoiceNumber, err := utils.GenerateInvoiceNumberTx(tx, r.driver, orgID, orderType, now)
	if err != nil {
		return err
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		INSERT INTO transactions (
			transaction_id,
			transaction_type,
			order_type,
			transaction_category,
			transaction_item,
			invoice_number,
			description,
			transaction_date,
			payment_type,
			organization_id,
			amount,
			created_at,
			created_by,
			payment_method,
			status
		) VALUES (
			%[1]s, %[2]s, %[3]s, %[4]s, %[5]s,
			%[6]s, %[7]s, %[8]s, %[9]s, %[10]s,
			%[11]s, %[12]s, %[13]s, %[14]s, 1
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8), placeholder(9), placeholder(10),
		placeholder(11), placeholder(12), placeholder(13), placeholder(14),
	)

	_, err = tx.Exec(
		query,
		transactionID.String(),
		2,
		orderType,
		req.TransactionCategory,
		req.TransactionItem,
		invoiceNumber,
		req.Description,
		req.TransactionDate,
		req.PaymentType,
		orgID,
		req.Amount,
		now,
		userID,
		req.PaymentMethod,
	)
	if err != nil {
		return err
	}

	if strings.TrimSpace(req.UnitID) != "" {
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
				%[1]s, %[2]s, %[3]s, %[4]s, %[5]s, %[6]s
			)
		`, placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5), placeholder(6))
		_, err = tx.Exec(
			queryFleet,
			transactionFleetID.String(),
			transactionID.String(),
			req.UnitID,
			orgID,
			now,
			userID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TransactionRepository) SoftDeleteExpenseTransaction(orgID, transactionID string) error {
	placeholder := r.getPlaceholder

	transactionIDExpr := "transaction_id = " + placeholder(1)
	orgExpr := "organization_id = " + placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		transactionIDExpr = "transaction_id::text = " + placeholder(1)
		orgExpr = "organization_id::text = " + placeholder(2)
	}

	query := fmt.Sprintf(`
		UPDATE transactions
		SET status = 0
		WHERE %s AND %s AND transaction_type = %s AND COALESCE(status, 1) <> 0
	`, transactionIDExpr, orgExpr, placeholder(3))

	result, err := r.db.Exec(query, transactionID, orgID, 2)
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

func (r *TransactionRepository) UpdateExpenseTransaction(orgID, userID string, req *UpdateExpenseTransactionRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	placeholder := r.getPlaceholder
	orderType := 4
	if strings.TrimSpace(req.UnitID) != "" {
		orderType = 1
	}

	transactionIDExpr := "transaction_id = " + placeholder(7)
	orgExpr := "organization_id = " + placeholder(8)
	if r.driver == "postgres" || r.driver == "pgx" {
		transactionIDExpr = "transaction_id::text = " + placeholder(7)
		orgExpr = "organization_id::text = " + placeholder(8)
	}

	query := fmt.Sprintf(`
		UPDATE transactions
		SET
			order_type = %[1]s,
			transaction_category = %[2]s,
			transaction_item = %[3]s,
			amount = %[4]s,
			transaction_date = %[5]s,
			payment_method = %[6]s
		WHERE %[7]s AND %[8]s AND transaction_type = %[9]s AND COALESCE(status, 1) <> 0
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5), placeholder(6),
		transactionIDExpr, orgExpr, placeholder(9),
	)

	result, err := tx.Exec(
		query,
		orderType,
		req.TransactionCategory,
		req.TransactionItem,
		req.Amount,
		req.TransactionDate,
		req.PaymentMethod,
		req.TransactionID,
		orgID,
		2,
	)
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

	fleetTransactionIDExpr := "transaction_id = " + placeholder(1)
	fleetOrgExpr := "organization_id = " + placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		fleetTransactionIDExpr = "transaction_id::text = " + placeholder(1)
		fleetOrgExpr = "organization_id::text = " + placeholder(2)
	}

	if strings.TrimSpace(req.UnitID) != "" {
		updateFleetQuery := fmt.Sprintf(`
			UPDATE transaction_fleets
			SET fleet_unit_id = %s
			WHERE %s AND %s
		`, placeholder(3), fleetTransactionIDExpr, fleetOrgExpr)

		updateFleetResult, err := tx.Exec(updateFleetQuery, req.TransactionID, orgID, req.UnitID)
		if err != nil {
			return err
		}

		fleetRowsAffected, err := updateFleetResult.RowsAffected()
		if err != nil {
			return err
		}

		if fleetRowsAffected == 0 {
			transactionFleetID, err := uuid.NewV7()
			if err != nil {
				return err
			}

			insertFleetQuery := fmt.Sprintf(`
				INSERT INTO transaction_fleets (
					transaction_fleet_id,
					transaction_id,
					fleet_unit_id,
					organization_id,
					created_at,
					created_by
				) VALUES (
					%[1]s, %[2]s, %[3]s, %[4]s, %[5]s, %[6]s
				)
			`, placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5), placeholder(6))

			_, err = tx.Exec(
				insertFleetQuery,
				transactionFleetID.String(),
				req.TransactionID,
				req.UnitID,
				orgID,
				time.Now(),
				userID,
			)
			if err != nil {
				return err
			}
		}
	} else {
		deleteFleetQuery := fmt.Sprintf(`
			DELETE FROM transaction_fleets
			WHERE %s AND %s
		`, fleetTransactionIDExpr, fleetOrgExpr)

		if _, err := tx.Exec(deleteFleetQuery, req.TransactionID, orgID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TransactionRepository) GetFleetOrderIDByScheduleNumber(scheduleNumber, orgID string) (string, bool, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	orgID = strings.TrimSpace(orgID)
	if scheduleNumber == "" || orgID == "" {
		return "", false, nil
	}

	placeholder := r.getPlaceholder
	orgExpr := "organization_id = " + placeholder(2)
	if r.driver == "postgres" || r.driver == "pgx" {
		orgExpr = "organization_id::text = " + placeholder(2)
	}

	query := fmt.Sprintf(`
		SELECT COALESCE(order_id::text, '')
		FROM schedule_fleets
		WHERE schedule_number = %s AND %s
		LIMIT 1
	`, placeholder(1), orgExpr)

	var orderID string
	err := database.QueryRow(r.db, query, scheduleNumber, orgID).Scan(&orderID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return orderID, true, nil
}

func (r *TransactionRepository) CreateFleetTripOperationalExpenseTransaction(orgID, userID, orderID, scheduleNumber string, amount float64, description string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	transactionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	invoiceNumber, err := utils.GenerateInvoiceNumberTx(tx, r.driver, orgID, 1, now)
	if err != nil {
		return err
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		INSERT INTO transactions (
			transaction_id,
			transaction_type,
			order_type,
			invoice_number,
			transaction_category,
			description,
			transaction_date,
			payment_type,
			organization_id,
			amount,
			transaction_label,
			reference_id,
			created_at,
			created_by,
			payment_method,
			transaction_item,
			status
		) VALUES (
			%[1]s, %[2]s, %[3]s, %[4]s, %[5]s,
			%[6]s, %[7]s, %[8]s, %[9]s, %[10]s,
			%[11]s, %[12]s, %[13]s, %[14]s, %[15]s,
			%[16]s, 1
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8), placeholder(9), placeholder(10),
		placeholder(11), placeholder(12), placeholder(13), placeholder(14), placeholder(15),
		placeholder(16),
	)

	_, err = tx.Exec(
		query,
		transactionID.String(),
		2,
		1,
		invoiceNumber,
		"TRX01",
		description,
		now,
		1004,
		orgID,
		amount,
		orderID,
		scheduleNumber,
		now,
		userID,
		1,
		"TRX-I00",
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TransactionRepository) CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem string, paymentMethod int, status int, amount float64, description string) error {
	now := time.Now()
	transactionTripID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		INSERT INTO transaction_fleet_trips (
			transaction_trip_id,
			schedule_number,
			transaction_type,
			transaction_category,
			transaction_item,
			amount,
			payment_type,
			description,
			created_at,
			created_by,
			reference_id,
			organization_id,
			status
		) VALUES (
			%[1]s, %[2]s, %[3]s, %[4]s, %[5]s,
			%[6]s, %[7]s, %[8]s, %[9]s, %[10]s,
			%[11]s, %[12]s, %[13]s
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8), placeholder(9), placeholder(10),
		placeholder(11), placeholder(12), placeholder(13),
	)

	_, err = r.db.Exec(
		query,
		transactionTripID.String(),
		scheduleNumber,
		2,
		"TRX01",
		transactionItem,
		amount,
		paymentMethod,
		description,
		now,
		userID,
		orderID,
		orgID,
		status,
	)
	return err
}

func (r *TransactionRepository) SumTransactionsAmountByReferenceID(referenceID string) (float64, error) {
	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return 0, nil
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0) AS total_amount
		FROM transactions
		WHERE reference_id = %s
	`, placeholder(1))

	var total float64
	err := database.QueryRow(r.db, query, referenceID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TransactionRepository) SumFleetTripAmountByScheduleNumberAndPaymentMethod(scheduleNumber string) (map[int]float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return map[int]float64{}, nil
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0) AS total_amount, COALESCE(payment_type, 0) AS payment_method
		FROM transaction_fleet_trips
		WHERE schedule_number = %s
		GROUP BY COALESCE(payment_type, 0)
	`, placeholder(1))

	rows, err := database.Query(r.db, query, scheduleNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int]float64{}
	for rows.Next() {
		var total float64
		var paymentMethod int
		if err := rows.Scan(&total, &paymentMethod); err != nil {
			return nil, err
		}
		out[paymentMethod] = total
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TransactionRepository) GetFleetTripAmountSummary(scheduleNumber, orgID string) (model.FleetTripAmountSummary, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	orgID = strings.TrimSpace(orgID)
	result := model.FleetTripAmountSummary{}
	if scheduleNumber == "" || orgID == "" {
		return result, nil
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(CASE 
				WHEN status = 1 AND payment_type = 1 
				THEN amount 
				ELSE 0 
			END), 0) AS total_expenses, 

			COALESCE(SUM(CASE 
				WHEN status = 1 AND payment_type = 2 
				THEN amount 
				ELSE 0 
			END), 0) AS total_claimed, 

			COALESCE(SUM(CASE 
				WHEN payment_type = 2 
				THEN amount 
				ELSE 0 
			END), 0) AS total_reimburse,
			
			COUNT(*) FILTER (
				WHERE status = 0
				AND payment_type = 2
			) AS total_item_reimburse

		FROM transaction_fleet_trips 
		WHERE schedule_number = %s AND organization_id = %s
	`, placeholder(1), placeholder(2))

	err := database.QueryRow(r.db, query, scheduleNumber, orgID).Scan(
		&result.TotalExpenses,
		&result.TotalClaimed,
		&result.TotalReimburse,
		&result.TotalItemReimburse,
	)
	if err != nil {
		return result, err
	}

	result.RemainingClaim = result.TotalReimburse - result.TotalClaimed
	result.TotalExpenses = result.TotalExpenses + result.TotalReimburse

	return result, nil
}

func (r *TransactionRepository) SumFleetTripAmountByScheduleNumber(scheduleNumber string) (float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return 0, nil
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0) AS total_expenses
		FROM transaction_fleet_trips
		WHERE schedule_number = %s
	`, placeholder(1))

	var total float64
	err := database.QueryRow(r.db, query, scheduleNumber).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TransactionRepository) ListFleetTripExpensesByScheduleNumber(scheduleNumber, orgID string) ([]model.FleetTripExpenseRow, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	orgID = strings.TrimSpace(orgID)
	if scheduleNumber == "" || orgID == "" {
		return []model.FleetTripExpenseRow{}, nil
	}

	placeholder := r.getPlaceholder

	orgExpr := "tft.organization_id::text = " + placeholder(2)
	createdByUserJoinExpr := "u.user_id::text = tft.created_by::text"
	createdByEmployeeJoinExpr := "e.employee_id::text = tft.created_by::text"

	query := fmt.Sprintf(`
		SELECT
			COALESCE(transaction_trip_id::text, '') AS transaction_trip_id,
			COALESCE(transaction_category, '') AS transaction_category,
			COALESCE(transaction_item, '') AS transaction_item,
			COALESCE(amount, 0) AS amount,
			COALESCE(payment_type, 0) AS payment_type,
			COALESCE(description, '') AS description,
			tft.created_at,
			COALESCE(tft.status, 0) AS status,
			COALESCE(u.fullname, e.fullname, '') AS created_by
		FROM transaction_fleet_trips tft
		LEFT JOIN users u ON %s
		LEFT JOIN employee e ON %s
		WHERE tft.schedule_number = %s AND %s
		ORDER BY tft.created_at DESC
	`, createdByUserJoinExpr, createdByEmployeeJoinExpr, placeholder(1), orgExpr)

	rows, err := database.Query(r.db, query, scheduleNumber, orgID)
	if err != nil {
		fmt.Printf("failed to query fleet trip expenses: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	out := make([]model.FleetTripExpenseRow, 0)
	for rows.Next() {
		var it model.FleetTripExpenseRow
		if err := rows.Scan(
			&it.TransactionTripID,
			&it.TransactionCategory,
			&it.TransactionItem,
			&it.Amount,
			&it.PaymentMethod,
			&it.Description,
			&it.CreatedAt,
			&it.Status,
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

func (r *TransactionRepository) GetReimbursementAmount(scheduleNumber string) (float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return 0, nil
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0) AS total_amount
		FROM transaction_fleet_trips
		WHERE schedule_number = %s AND status = 0 AND payment_type = 2
	`, placeholder(1))

	var total float64
	err := database.QueryRow(r.db, query, scheduleNumber).Scan(&total)
	if err != nil {
		fmt.Printf("failed to query reimbursement amount: %v\n", err)
		return 0, err
	}
	return total, nil
}

func (r *TransactionRepository) CreateFleetTripReimbursement(orgID, userID string, reimbursement *model.FleetTripReimbursement) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	transactionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	// Parse transaction date
	transactionDate, err := time.Parse("2006-01-02", reimbursement.TransactionDate)
	if err != nil {
		return err
	}

	invoiceNumber, err := utils.GenerateInvoiceNumberTx(tx, r.driver, orgID, 1, now)
	if err != nil {
		return err
	}

	placeholder := r.getPlaceholder
	// Insert into transactions table
	insertTransactionQuery := fmt.Sprintf(`
		INSERT INTO transactions (
			transaction_id,
			transaction_type,
			order_type,
			payment_type,
			transaction_category,
			transaction_item,
			invoice_number,
			description,
			transaction_date,
			payment_method,
			amount,
			reference_id,
			status,
			organization_id,
			created_by,
			created_at
		) VALUES (
			%[1]s, %[2]s, %[3]s, '1004', %[4]s, %[5]s,
			%[6]s, %[7]s, %[8]s, %[9]s, %[10]s,
			%[11]s, %[12]s, %[13]s, %[14]s, %[15]s
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8), placeholder(9), placeholder(10),
		placeholder(11), placeholder(12), placeholder(13), placeholder(14), placeholder(15),
	)

	_, err = tx.Exec(
		insertTransactionQuery,
		transactionID.String(),
		2,
		1,
		"TRX01",
		"TRX-I13",
		invoiceNumber,
		"Biaya Operasional "+reimbursement.ScheduleNumber,
		transactionDate,
		reimbursement.PaymentMethodID,
		reimbursement.Amount,
		reimbursement.ScheduleNumber,
		1,
		orgID,
		userID,
		now,
	)
	if err != nil {
		return err
	}

	// Insert into transaction_reimbursement table
	reimburseID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	insertReimbursementQuery := fmt.Sprintf(`
		INSERT INTO transaction_reimbursement (
			reimburse_id,
			reference_id,
			organization_id,
			amount,
			employee_id,
			status,
			payment_method,
			created_at
		) VALUES (
			%[1]s, %[2]s, %[3]s, %[4]s, %[5]s,
			%[6]s, %[7]s, %[8]s
		)
	`,
		placeholder(1), placeholder(2), placeholder(3), placeholder(4), placeholder(5),
		placeholder(6), placeholder(7), placeholder(8),
	)

	_, err = tx.Exec(
		insertReimbursementQuery,
		reimburseID.String(),
		transactionID.String(),
		orgID,
		reimbursement.Amount,
		reimbursement.RecipientID,
		1,
		reimbursement.PaymentMethodID,
		now,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *TransactionRepository) MarkReimbursementPaid(scheduleNumber string) error {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return fmt.Errorf("schedule number is empty")
	}

	placeholder := r.getPlaceholder
	query := fmt.Sprintf(`
		UPDATE transaction_fleet_trips
		SET status = 1
		WHERE schedule_number = %s AND status = 0 AND payment_type = 2
	`, placeholder(1))

	_, err := r.db.Exec(query, scheduleNumber)
	if err != nil {
		fmt.Printf("failed to mark reimbursement paid: %v\n", err)
		return err
	}
	return nil
}
