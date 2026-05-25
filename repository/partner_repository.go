package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/model"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	partnerCitiesOnce sync.Once
	partnerCitiesMap  map[string]string
)

func getPartnerCitiesMap() map[string]string {
	partnerCitiesOnce.Do(func() {
		partnerCitiesMap = map[string]string{}
		f, err := os.Open("config/location.json")
		if err != nil {
			fmt.Printf("Error opening location.json: %v\n", err)
			return
		}
		defer f.Close()
		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			fmt.Printf("Error decoding location.json: %v\n", err)
			return
		}
		for _, c := range loc.Cities {
			partnerCitiesMap[strings.TrimSpace(c.ID)] = c.Name
		}
	})
	return partnerCitiesMap
}

type PartnerRepository struct {
	db     *sql.DB
	driver string
}

func NewPartnerRepository(db *sql.DB, driver string) *PartnerRepository {
	return &PartnerRepository{db: db, driver: driver}
}

func (r *PartnerRepository) GetCityLabel(cityID *int) string {
	if cityID == nil {
		return ""
	}
	m := getPartnerCitiesMap()
	if label, ok := m[fmt.Sprintf("%d", *cityID)]; ok {
		return label
	}
	return ""
}

func (r *PartnerRepository) List(orgID, partnerName string) ([]model.OperationPartner, error) {
	query := `
		SELECT partner_id, partner_name, partner_address, partner_city, partner_phone, pic_name, created_at, created_by, updated_at, updated_by, organization_id
		FROM operation_partner
		WHERE organization_id = $1
	`
	args := []interface{}{orgID}

	if partnerName != "" {
		query += ` AND partner_name ILIKE $2`
		args = append(args, "%"+partnerName+"%")
	}

	query += ` ORDER BY created_at DESC`

	// adjust for mysql
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "ILIKE", "LIKE")
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.OperationPartner
	for rows.Next() {
		var p model.OperationPartner
		err := rows.Scan(
			&p.PartnerID, &p.PartnerName, &p.PartnerAddress, &p.PartnerCity, &p.PartnerPhone, &p.PicName,
			&p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy, &p.OrganizationID,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *PartnerRepository) Create(req model.CreateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	partnerID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO operation_partner (partner_id, partner_name, partner_address, partner_city, partner_phone, pic_name, created_at, created_by, updated_at, updated_by, organization_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
		query = strings.ReplaceAll(query, "$4", "?")
		query = strings.ReplaceAll(query, "$5", "?")
		query = strings.ReplaceAll(query, "$6", "?")
		query = strings.ReplaceAll(query, "$7", "?")
		query = strings.ReplaceAll(query, "$8", "?")
		query = strings.ReplaceAll(query, "$9", "?")
		query = strings.ReplaceAll(query, "$10", "?")
		query = strings.ReplaceAll(query, "$11", "?")
	}

	_, err := r.db.Exec(query, partnerID, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PicName, now, userID, now, userID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(partnerID, orgID)
}

func (r *PartnerRepository) Update(req model.UpdateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	now := time.Now()

	query := `
		UPDATE operation_partner
		SET partner_name = $1, partner_address = $2, partner_city = $3, partner_phone = $4, pic_name = $5, updated_at = $6, updated_by = $7
		WHERE partner_id = $8 AND organization_id = $9
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
		query = strings.ReplaceAll(query, "$4", "?")
		query = strings.ReplaceAll(query, "$5", "?")
		query = strings.ReplaceAll(query, "$6", "?")
		query = strings.ReplaceAll(query, "$7", "?")
		query = strings.ReplaceAll(query, "$8", "?")
		query = strings.ReplaceAll(query, "$9", "?")
	}

	_, err := r.db.Exec(query, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PicName, now, userID, req.PartnerID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(req.PartnerID, orgID)
}

func (r *PartnerRepository) GetByID(partnerID, orgID string) (*model.OperationPartner, error) {
	query := `
		SELECT partner_id, partner_name, partner_address, partner_city, partner_phone, pic_name, created_at, created_by, updated_at, updated_by, organization_id
		FROM operation_partner
		WHERE partner_id = $1 AND organization_id = $2
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
	}

	var p model.OperationPartner
	err := r.db.QueryRow(query, partnerID, orgID).Scan(
		&p.PartnerID, &p.PartnerName, &p.PartnerAddress, &p.PartnerCity, &p.PartnerPhone, &p.PicName,
		&p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy, &p.OrganizationID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PartnerRepository) GetOrCreateByNamePhone(orgID, userID, partnerName, partnerPhone string) (string, error) {
	query := `
		SELECT partner_id
		FROM operation_partner
		WHERE partner_name = $1 AND partner_phone = $2 AND organization_id = $3
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
		query = strings.ReplaceAll(query, "$3", "?")
	}

	var partnerID string
	err := r.db.QueryRow(query, partnerName, partnerPhone, orgID).Scan(&partnerID)
	if err == nil {
		return partnerID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	createReq := model.CreateOperationPartnerRequest{
		PartnerName:  partnerName,
		PartnerPhone: partnerPhone,
		PicName:      partnerName,
	}

	partner, err := r.Create(createReq, orgID, userID)
	if err != nil {
		return "", err
	}

	return partner.PartnerID, nil
}
