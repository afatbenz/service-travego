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

func (r *PartnerRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
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
		SELECT op.partner_id, op.partner_name, op.partner_address, op.partner_city, op.partner_phone, op.partner_email, op.pic_name, op.created_at, op.organization_id, COUNT(fuo.unit_id) as total_unit
		FROM operation_partner op
		LEFT JOIN fleet_unit_ownership fuo ON fuo.partner_id = op.partner_id
		WHERE op.organization_id = $1
		GROUP BY op.partner_id, op.partner_name, op.partner_address, op.partner_city, op.partner_phone, op.partner_email, op.pic_name, op.created_at, op.created_by, op.updated_at, op.updated_by, op.organization_id
	`
	args := []interface{}{orgID}

	if partnerName != "" {
		query += ` AND op.partner_name ILIKE $2`
		args = append(args, "%"+partnerName+"%")
	}

	query += ` ORDER BY op.created_at DESC`

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
			&p.PartnerID, &p.PartnerName, &p.PartnerAddress, &p.PartnerCity, &p.PartnerPhone, &p.PartnerEmail, &p.PicName,
			&p.CreatedAt,

			&p.OrganizationID, &p.TotalUnit,
		)
		if err != nil {
			return nil, err
		}
		p.PartnerCityLabel = r.GetCityLabel(p.PartnerCity)
		result = append(result, p)
	}
	return result, nil
}

func (r *PartnerRepository) Create(req model.CreateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	partnerID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO operation_partner (partner_id, partner_name, partner_address, partner_city, partner_phone, partner_email, pic_name, created_at, created_by, updated_at, updated_by, organization_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
		query = strings.ReplaceAll(query, "$12", "?")
	}

	_, err := r.db.Exec(query, partnerID, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PartnerEmail, req.PicName, now, userID, now, userID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(partnerID, orgID, nil)
}

func (r *PartnerRepository) Update(req model.UpdateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	now := time.Now()

	query := `
		UPDATE operation_partner
		SET partner_name = $1, partner_address = $2, partner_city = $3, partner_phone = $4, partner_email = $5, pic_name = $6, updated_at = $7, updated_by = $8
		WHERE partner_id = $9 AND organization_id = $10
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
	}

	_, err := r.db.Exec(query, req.PartnerName, req.PartnerAddress, req.PartnerCity, req.PartnerPhone, req.PartnerEmail, req.PicName, now, userID, req.PartnerID, orgID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(req.PartnerID, orgID, nil)
}

func (r *PartnerRepository) GetByID(partnerID, orgID string, filter *model.OperationPartnerDetailRequest) (*model.OperationPartner, error) {
	args := make([]interface{}, 0, 6)
	args = append(args, partnerID, orgID)

	tripCond := ""
	transactionCond := ""

	if filter != nil {
		if v := strings.TrimSpace(filter.TripStartDate); v != "" {
			tripCond += fmt.Sprintf(" AND fo.end_date >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
		if v := strings.TrimSpace(filter.TripEndDate); v != "" {
			tripCond += fmt.Sprintf(" AND fo.start_date <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
		if v := strings.TrimSpace(filter.TransactionStartDate); v != "" {
			transactionCond += fmt.Sprintf(" AND t.transaction_date >= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
		if v := strings.TrimSpace(filter.TransactionEndDate); v != "" {
			transactionCond += fmt.Sprintf(" AND t.transaction_date <= %s", r.getPlaceholder(len(args)+1))
			args = append(args, v)
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			op.partner_name, 
			op.partner_address, 
			op.partner_city, 
			op.partner_phone, 
			op.pic_name, 
			op.partner_email, 
			op.created_at AS join_date, 
			
			(SELECT COUNT(fuo.unit_id) 
			 FROM fleet_unit_ownership fuo 
			 WHERE fuo.partner_id = op.partner_id AND fuo.organization_id = op.organization_id) AS total_units, 
			  
			(SELECT COUNT(sf.uuid) 
			 FROM fleet_units fu 
			 INNER JOIN schedule_fleets sf ON sf.unit_id = fu.unit_id 
			 INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = fu.unit_id 
			 INNER JOIN fleet_orders fo ON fo.order_id = sf.order_id
			 WHERE fuo.partner_id = op.partner_id
			   AND fuo.organization_id = op.organization_id
			   AND sf.organization_id = op.organization_id
			   %s) AS total_schedule, 

			COALESCE(finance.total_revenue, 0) AS total_revenue, 
			COALESCE(finance.total_expenses, 0) AS total_expenses
		FROM operation_partner op 
		LEFT JOIN ( 
			SELECT 
				fuo.partner_id, 
				SUM(CASE WHEN t.transaction_type = 1 THEN t.amount ELSE 0 END) AS total_revenue, 
				SUM(CASE WHEN t.transaction_type = 2 THEN t.amount ELSE 0 END) AS total_expenses 
			FROM fleet_unit_ownership fuo 
			INNER JOIN fleet_units fu ON fu.unit_id = fuo.unit_id 
			INNER JOIN schedule_fleets sf ON sf.unit_id = fu.unit_id 
			INNER JOIN fleet_orders fo ON fo.order_id = sf.order_id
			INNER JOIN transactions t ON t.reference_id = sf.schedule_number 
			WHERE fuo.partner_id = %s
			  AND fuo.organization_id = %s
			  AND sf.organization_id = %s
			  AND t.organization_id = %s
			  %s
			  %s
			GROUP BY fuo.partner_id 
		) finance ON finance.partner_id = op.partner_id 
		WHERE op.partner_id = %s AND op.organization_id = %s
		LIMIT 1
	`,
		tripCond,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(2), r.getPlaceholder(2),
		tripCond, transactionCond,
		r.getPlaceholder(1), r.getPlaceholder(2),
	)

	var p model.OperationPartner
	var joinDate time.Time
	err := r.db.QueryRow(query, args...).Scan(
		&p.PartnerName,
		&p.PartnerAddress,
		&p.PartnerCity,
		&p.PartnerPhone,
		&p.PicName,
		&p.PartnerEmail,
		&joinDate,
		&p.TotalUnits,
		&p.TotalSchedule,
		&p.TotalRevenue,
		&p.TotalExpenses,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	p.PartnerID = partnerID
	p.OrganizationID = &orgID
	p.JoinDate = &joinDate
	p.CreatedAt = &joinDate
	p.TotalUnit = p.TotalUnits
	p.PartnerCityLabel = r.GetCityLabel(p.PartnerCity)
	return &p, nil
}

func (r *PartnerRepository) GetOrCreateByNamePhone(orgID, userID, partnerName, partnerPhone string, partnerEmail *string) (string, error) {
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
		PartnerEmail: partnerEmail,
		PicName:      partnerName,
	}

	partner, err := r.Create(createReq, orgID, userID)
	if err != nil {
		return "", err
	}

	return partner.PartnerID, nil
}

func (r *PartnerRepository) GetPartnerFleetUnits(partnerID, orgID string) ([]model.PartnerFleetUnit, error) {
	query := `
		SELECT f.fleet_name, fu.plate_number, fu.vehicle_id, fu.unit_id 
		FROM fleets f 
		INNER JOIN fleet_units fu ON fu.fleet_id = f.uuid 
		INNER JOIN fleet_unit_ownership fuo ON fuo.unit_id = fu.unit_id 
		WHERE fuo.partner_id = $1 AND fuo.organization_id = $2
	`
	if r.driver == "mysql" {
		query = strings.ReplaceAll(query, "$1", "?")
		query = strings.ReplaceAll(query, "$2", "?")
	}

	rows, err := r.db.Query(query, partnerID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.PartnerFleetUnit
	for rows.Next() {
		var fu model.PartnerFleetUnit
		err := rows.Scan(&fu.FleetName, &fu.PlateNumber, &fu.VehicleID, &fu.UnitID)
		if err != nil {
			return nil, err
		}
		result = append(result, fu)
	}
	return result, nil
}
