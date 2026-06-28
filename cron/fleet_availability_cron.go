package cron

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"service-travego/internal/wagy"
	"service-travego/repository"
	"service-travego/service"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type FleetAvailabilityCron struct {
	db              *sql.DB
	driver          string
	wagyClient      *wagy.WagyClient
	fleetSvc        *service.FleetService
	organizationIDs []string
}

func NewFleetAvailabilityCron(db *sql.DB, driver string, wagyClient *wagy.WagyClient) *FleetAvailabilityCron {
	fleetRepo := repository.NewFleetRepository(db, driver)
	fleetSvc := service.NewFleetService(fleetRepo)

	// Read organization IDs from environment variable
	var orgIDs []string
	orgIDsStr := os.Getenv("FLEET_AVAILABILITY_CRON_ORGANIZATION_IDS")
	if orgIDsStr != "" {
		// Split by comma and trim whitespace
		for _, id := range strings.Split(orgIDsStr, ",") {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				orgIDs = append(orgIDs, trimmed)
			}
		}
	}

	return &FleetAvailabilityCron{
		db:              db,
		driver:          driver,
		wagyClient:      wagyClient,
		fleetSvc:        fleetSvc,
		organizationIDs: orgIDs,
	}
}

type orgTarget struct {
	OrganizationID   string
	AccountNumber    string
	OrganizationName string
}

func (c *FleetAvailabilityCron) Run() {
	log.Println("[FleetAvailabilityCron] Starting scheduled job...")

	if c.wagyClient == nil {
		log.Println("[FleetAvailabilityCron] Wagy client not configured, skipping")
		return
	}

	// 1. Query active organizations with assistant accounts
	targets, err := c.queryActiveOrganizations()
	if err != nil {
		log.Printf("[FleetAvailabilityCron] Failed to query organizations: %v", err)
		return
	}

	if len(targets) == 0 {
		log.Println("[FleetAvailabilityCron] No active organizations found")
		return
	}

	log.Printf("[FleetAvailabilityCron] Found %d active organizations", len(targets))

	// Date range: today to 7 days from now
	today := time.Now().Format("2006-01-02")
	nextWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	for _, org := range targets {
		c.processOrganization(org, today, nextWeek)
	}

	log.Println("[FleetAvailabilityCron] Job completed")
}

func (c *FleetAvailabilityCron) queryActiveOrganizations() ([]orgTarget, error) {
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
			log.Printf("[FleetAvailabilityCron] Scan row error: %v", err)
			continue
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func (c *FleetAvailabilityCron) processOrganization(org orgTarget, startDate, endDate string) {
	log.Printf("[FleetAvailabilityCron] Processing org: %s (%s)", org.OrganizationName, org.OrganizationID)

	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] Invalid startDate %s for org %s: %v", startDate, org.OrganizationID, err)
		return
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] Invalid endDate %s for org %s: %v", endDate, org.OrganizationID, err)
		return
	}

	// 2. Get fleet availability via FleetService
	_, items, err := c.fleetSvc.GetFleetAvailibility(org.OrganizationID, start, end, "")
	if err != nil {
		log.Printf("[FleetAvailabilityCron] GetFleetAvailibility error for org %s: %v", org.OrganizationID, err)
		return
	}

	// 3. Format message
	message := c.formatMessage(org.OrganizationName, items)

	// 4. Send via Wagy
	_, err = c.wagyClient.SendMessage(org.AccountNumber, message)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] SendMessage error to %s: %v", org.AccountNumber, err)
		insertAssistantAccountStat(c.db, c.driver, org.OrganizationID, 2)
		return
	}

	insertAssistantAccountStat(c.db, c.driver, org.OrganizationID, 1)
	log.Printf("[FleetAvailabilityCron] Message sent to %s (%s)", org.AccountNumber, org.OrganizationName)
}

func (c *FleetAvailabilityCron) formatMessage(orgName string, items []repository.FleetAvailibilityItem) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Selamat Pagi %s\n\n", orgName))
	b.WriteString("Berikut adalah daftar armada yang tersedia:\n\n")

	if len(items) == 0 {
		b.WriteString("Tidak ada armada yang tersedia untuk 7 hari ke depan.\n")
		return b.String()
	}

	for _, item := range items {
		fleetName := item.FleetName
		if fleetName == "" {
			fleetName = item.FleetID
		}
		b.WriteString(fmt.Sprintf("• %s: %d unit tersedia\n", fleetName, item.TotalAvailable))
	}

	b.WriteString(fmt.Sprintf("\nPeriode: %s - %s\n", time.Now().Format("02 Jan 2006"), time.Now().AddDate(0, 0, 7).Format("02 Jan 2006")))
	b.WriteString("Terima kasih.\n")

	return b.String()
}

// Start registers the cron job and starts the scheduler
func StartFleetAvailabilityCron(db *sql.DB, driver string, wagyClient *wagy.WagyClient) *cron.Cron {
	c := cron.New(cron.WithLocation(time.Local))

	cronJob := NewFleetAvailabilityCron(db, driver, wagyClient)

	// Schedule: Monday, Wednesday, Friday at 09:00
	_, err := c.AddFunc("0 09 * * 1,3,5", cronJob.Run)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] Failed to register cron: %v", err)
		return nil
	}

	c.Start()
	log.Println("[FleetAvailabilityCron] Scheduled: Mon, Wed, Fri at 09:00")

	return c
}
