package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/model"
	"sync"
	"time"
)

type PricingRepository struct {
	packages []model.Package
	once     sync.Once
	loadErr  error
	db       *sql.DB
	driver   string
}

func NewPricingRepository(db *sql.DB, driver string) *PricingRepository {
	return &PricingRepository{
		db:     db,
		driver: driver,
	}
}

func (r *PricingRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

func (r *PricingRepository) loadPackages() error {
	r.once.Do(func() {
		f, err := os.Open("config/packages.json")
		if err != nil {
			r.loadErr = err
			return
		}
		defer f.Close()

		var data struct {
			Packages []model.Package `json:"packages"`
		}
		if err := json.NewDecoder(f).Decode(&data); err != nil {
			r.loadErr = err
			return
		}
		r.packages = data.Packages
	})
	return r.loadErr
}

func (r *PricingRepository) GetPackages() ([]model.Package, error) {
	if err := r.loadPackages(); err != nil {
		return nil, err
	}
	return r.packages, nil
}

func (r *PricingRepository) GetReviews() ([]model.Review, error) {
	query := `SELECT r.review_id, r.user_id, r.stars, r.review, r.created_at, u.fullname as created_by, o.organization_name 
	FROM travego_reviews r INNER JOIN users u ON r.user_id = u.user_id
	INNER JOIN organization_users ou ON ou.user_id = u.user_id
	INNER JOIN organizations o ON o.organization_id = ou.organization_id
	ORDER BY r.stars, r.created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reviews []model.Review
	for rows.Next() {
		var r model.Review
		if err := rows.Scan(&r.ReviewID, &r.UserID, &r.Stars, &r.Review, &r.CreatedAt, &r.CreatedBy, &r.CompanyName); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

func (r *PricingRepository) SubmitContact(contact model.ContactSubmission) error {
	query := fmt.Sprintf(`INSERT INTO travego_messages (topic_id, fullname, company_name, email, whatsapp, scale, messages, created_at, is_read)
    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))

	_, err := r.db.Exec(query, contact.TopicID, contact.FullName, contact.BusinessName,
		contact.Email, contact.Phone, contact.BusinessScale,
		contact.Message, time.Now(), false)

	if err != nil {
		return fmt.Errorf("gagal insert ke travego_messages: %w", err)
	}
	return nil
}

func (r *PricingRepository) GetSubscriptionByOrgID(orgID string) (*model.Subscription, error) {
	query := fmt.Sprintf(`SELECT package_id, activate_date, expiry_date FROM _subscription WHERE organization_id = %s`,
		r.getPlaceholder(1))
	row := r.db.QueryRow(query, orgID)

	var sub model.Subscription
	err := row.Scan(&sub.PackageID, &sub.ActivateDate, &sub.ExpiryDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *PricingRepository) InsertLog() error {
	query := `INSERT INTO travego_visitors (period, count) 
	VALUES (CURRENT_DATE, 1) 
	ON CONFLICT (period) 
	DO UPDATE SET count = travego_visitors.count + 1;`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("gagal insert ke travego_visitors: %w", err)
	}
	return nil
}
