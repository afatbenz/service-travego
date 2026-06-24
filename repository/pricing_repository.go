package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/model"
	"sync"
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
