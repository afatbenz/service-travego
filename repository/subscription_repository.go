package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"service-travego/utils"
)

type SubscriptionRepository struct {
	db     *sql.DB
	driver string
}

func NewSubscriptionRepository(db *sql.DB, driver string) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:     db,
		driver: driver,
	}
}

func (r *SubscriptionRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindAll retrieves users
func (r *SubscriptionRepository) GetSubscriptionDetails(orgID string) ([]model.SubscriptionDetail, error) {
	var subscriptions []model.SubscriptionDetail
	query := fmt.Sprintf("SELECT package_id, activate_date as start_date, expiry_date, package_price FROM _subscription WHERE organization_id = %s AND expiry_date >= now() ORDER BY created_at DESC LIMIT 1", r.getPlaceholder(1))
	rows, err := r.db.Query(query, orgID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var subscription model.SubscriptionDetail
		if err := rows.Scan(&subscription.PackageID, &subscription.StartDate, &subscription.ExpireDate, &subscription.PackagePrice); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetSubscriptionHistory(userID, orgID string) ([]model.SubscriptionHistory, error) {
	query := fmt.Sprintf(`
		SELECT 
			transaction_id, 
			transaction_date, 
			COALESCE(invoice_number, '') as invoice_number, 
			package_id, 
			start_date, 
			expiry_date, 
			COALESCE(payment_method, '') as payment_method, 
			COALESCE(status, 0) as status, 
			created_by, 
			created_at,
			COALESCE(payment_amount, 0) as payment_amount
		FROM travego_transactions 
		WHERE user_id = %s AND organization_id = %s AND status = 1
		ORDER BY created_at DESC
	`, r.getPlaceholder(1), r.getPlaceholder(2))
	fmt.Println(query, userID, orgID)
	rows, err := r.db.Query(query, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subscriptions []model.SubscriptionHistory
	for rows.Next() {
		var subscription model.SubscriptionHistory
		var statusInt int
		var paymentAmount float64
		if err := rows.Scan(&subscription.TransactionID, &subscription.TransactionDate, &subscription.InvoiceNumber, &subscription.PackageID, &subscription.StartDate, &subscription.ExpiryDate, &subscription.PaymentMethod, &statusInt, &subscription.CreatedBy, &subscription.CreatedAt, &paymentAmount); err != nil {
			return nil, err
		}
		// Convert status integer to string
		switch statusInt {
		case 0:
			subscription.Status = "Pending"
		case 1:
			subscription.Status = "Paid"
		case 2:
			subscription.Status = "Processed"
		default:
			subscription.Status = "Unknown"
		}

		if paymentAmount > 0 {
			subscription.PaymentAmount = paymentAmount
		} else {
			subscription.PaymentAmount = 0
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, nil
}

// InsertTravegoTransaction inserts a new subscription transaction
func (r *SubscriptionRepository) InsertTravegoTransaction(transactionID, transactionDate, invoiceNumber, packageID, startDate, expiryDate string, status int, userID, orgID, createdAt, createdBy string) error {
	query := fmt.Sprintf(`
		INSERT INTO travego_transactions 
		(transaction_id, transaction_date, invoice_number, package_id, start_date, expiry_date, status, user_id, organization_id, created_at, created_by) 
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

	_, err := r.db.Exec(query, transactionID, transactionDate, invoiceNumber, packageID, startDate, expiryDate, status, userID, orgID, createdAt, createdBy)
	return err
}

// GenerateSubsInvoiceID generates subscription invoice number
func (r *SubscriptionRepository) GenerateSubsInvoiceID() (string, error) {
	return utils.GenerateSubsInvoiceID(r.db, r.driver)
}
