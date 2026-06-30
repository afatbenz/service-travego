package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"service-travego/internal/wagy"
	"service-travego/model"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type unpaidOrderRow struct {
	OrderID        sql.NullString
	PickupLocation sql.NullString
	UnitQty        sql.NullInt64
	PaymentStatus  sql.NullInt64
	PickupCityID   sql.NullString
	CityID         sql.NullString
	CustomerName   sql.NullString
	CustomerPhone  sql.NullString
}

type UnpaidOrdersCron struct {
	db              *sql.DB
	driver          string
	wagyClient      *wagy.WagyClient
	citiesName      map[string]string
	organizationIDs []string
}

func NewUnpaidOrdersCron(db *sql.DB, driver string, wagyClient *wagy.WagyClient) *UnpaidOrdersCron {
	// Read organization IDs from environment variable
	var orgIDs []string
	orgIDsStr := os.Getenv("UNPAID_ORDERS_CRON_ORGANIZATION_IDS")
	if orgIDsStr != "" {
		// Split by comma and trim whitespace
		for _, id := range strings.Split(orgIDsStr, ",") {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				orgIDs = append(orgIDs, trimmed)
			}
		}
	}

	return &UnpaidOrdersCron{
		db:              db,
		driver:          driver,
		wagyClient:      wagyClient,
		organizationIDs: orgIDs,
	}
}

func (c *UnpaidOrdersCron) ensureLocationLoaded() {
	if c.citiesName != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		c.citiesName = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		c.citiesName = map[string]string{}
		return
	}
	m := make(map[string]string, len(loc.Cities))
	for _, ct := range loc.Cities {
		m[ct.ID] = ct.Name
	}
	c.citiesName = m
}

func (c *UnpaidOrdersCron) Run() {
	log.Println("[UnpaidOrdersCron] Starting scheduled job...")

	if c.wagyClient == nil {
		log.Println("[UnpaidOrdersCron] Wagy client not configured, skipping")
		return
	}

	c.ensureLocationLoaded()

	targets, err := c.queryActiveOrganizations()
	if err != nil {
		log.Printf("[UnpaidOrdersCron] Failed to query organizations: %v", err)
		return
	}

	if len(targets) == 0 {
		log.Println("[UnpaidOrdersCron] No active organizations found")
		return
	}

	log.Printf("[UnpaidOrdersCron] Found %d active organizations", len(targets))

	nextWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	for _, org := range targets {
		c.processOrganization(org, nextWeek)
	}

	log.Println("[UnpaidOrdersCron] Job completed")
}

func (c *UnpaidOrdersCron) queryActiveOrganizations() ([]orgTarget, error) {
	query := `
		SELECT ac.organization_id, ac.account_number, o.organization_name
		FROM assistant_accounts ac
		INNER JOIN organizations o ON ac.organization_id = o.organization_id
		INNER JOIN _subscription s ON s.organization_id = ac.organization_id
		WHERE s.expiry_date >= CURRENT_DATE
	`

	var args []interface{}

	// Add organization ID filter if specified
	if len(c.organizationIDs) > 0 {
		placeholders := make([]string, len(c.organizationIDs))
		for i := range c.organizationIDs {
			if c.driver == "postgres" || c.driver == "pgx" {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			} else {
				placeholders[i] = "?"
			}
			args = append(args, c.organizationIDs[i])
		}
		query += fmt.Sprintf(" AND ac.organization_id IN (%s)", strings.Join(placeholders, ","))
	}

	query += " GROUP BY ac.organization_id, ac.account_number, o.organization_name"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query organizations: %w", err)
	}
	defer rows.Close()

	var targets []orgTarget
	for rows.Next() {
		var t orgTarget
		if err := rows.Scan(&t.OrganizationID, &t.AccountNumber, &t.OrganizationName); err != nil {
			log.Printf("[UnpaidOrdersCron] Scan row error: %v", err)
			continue
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func (c *UnpaidOrdersCron) processOrganization(org orgTarget, nextWeek string) {
	log.Printf("[UnpaidOrdersCron] Processing org: %s (%s)", org.OrganizationName, org.OrganizationID)

	orders, err := c.queryUnpaidOrders(org.OrganizationID, nextWeek)
	if err != nil {
		log.Printf("[UnpaidOrdersCron] Query unpaid orders error for org %s: %v", org.OrganizationID, err)
		return
	}

	if len(orders) == 0 {
		log.Printf("[UnpaidOrdersCron] No unpaid orders for org %s", org.OrganizationID)
		return
	}

	message := c.formatMessage(org.OrganizationName, orders)

	_, err = c.wagyClient.SendMessage(org.AccountNumber, message)
	if err != nil {
		log.Printf("[UnpaidOrdersCron] SendMessage error to %s: %v", org.AccountNumber, err)
		insertAssistantAccountStat(c.db, c.driver, org.OrganizationID, 2)
		return
	}

	insertAssistantAccountStat(c.db, c.driver, org.OrganizationID, 1)
	log.Printf("[UnpaidOrdersCron] Message sent to %s (%s) — %d unpaid orders", org.AccountNumber, org.OrganizationName, len(orders))
}

func (c *UnpaidOrdersCron) queryUnpaidOrders(organizationID string, nextWeek string) ([]unpaidOrderRow, error) {
	query := `
		SELECT fo.order_id, fo.pickup_location, fo.unit_qty, fo.payment_status,
		       fo.pickup_city_id, foi.city_id, c.customer_name, c.customer_phone
		FROM fleet_orders fo
		INNER JOIN customer_orders co ON fo.order_id = co.order_id
		INNER JOIN customers c ON c.customer_id = co.customer_id
		INNER JOIN fleet_order_itinerary foi ON foi.order_id = fo.order_id
		WHERE fo.payment_status > 1
		  AND fo.start_date <= $1
		  AND fo.start_date >= CURRENT_DATE
		  AND fo.organization_id = $2
	`

	rows, err := c.db.Query(query, nextWeek, organizationID)
	if err != nil {
		return nil, fmt.Errorf("query unpaid orders: %w", err)
	}
	defer rows.Close()

	var out []unpaidOrderRow
	for rows.Next() {
		var t unpaidOrderRow
		if err := rows.Scan(
			&t.OrderID, &t.PickupLocation, &t.UnitQty, &t.PaymentStatus,
			&t.PickupCityID, &t.CityID, &t.CustomerName, &t.CustomerPhone,
		); err != nil {
			log.Printf("[UnpaidOrdersCron] Scan row error: %v", err)
			continue
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *UnpaidOrdersCron) formatMessage(orgName string, orders []unpaidOrderRow) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Selamat Pagi %s\n\n", orgName))
	b.WriteString("Berikut adalah daftar pesanan yang belum lunas:\n\n")

	for i, o := range orders {
		pickupCity := c.lookupCity(o.PickupCityID.String)
		destCity := c.lookupCity(o.CityID.String)

		b.WriteString(fmt.Sprintf("%d. Order ID: %s\n", i+1, o.OrderID.String))
		b.WriteString(fmt.Sprintf("   Nama Customer: %s\n", o.CustomerName.String))
		b.WriteString(fmt.Sprintf("   No. Telepon: %s\n", o.CustomerPhone.String))
		b.WriteString(fmt.Sprintf("   Pickup City: %s\n", pickupCity))
		b.WriteString(fmt.Sprintf("   Destination City: %s\n", destCity))
		b.WriteString("\n")
	}

	b.WriteString("Terima kasih.\n")

	return b.String()
}

func (c *UnpaidOrdersCron) lookupCity(cityID string) string {
	if cityID == "" {
		return "-"
	}
	if name, ok := c.citiesName[cityID]; ok {
		return name
	}
	return cityID
}

// Start registers the cron job and starts the scheduler
func StartUnpaidOrdersCron(db *sql.DB, driver string, wagyClient *wagy.WagyClient) *cron.Cron {
	c := cron.New(cron.WithLocation(time.Local))

	cronJob := NewUnpaidOrdersCron(db, driver, wagyClient)

	// Schedule: every day at 07:00
	_, err := c.AddFunc("0 7 * * *", cronJob.Run)
	if err != nil {
		log.Printf("[UnpaidOrdersCron] Failed to register cron: %v", err)
		return nil
	}

	c.Start()
	log.Println("[UnpaidOrdersCron] Scheduled: Every day at 07:00")

	return c
}
