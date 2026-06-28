package cron

import (
	"database/sql"
	"fmt"
	"log"
	"service-travego/internal/wagy"
	"service-travego/model"
	"service-travego/repository"
	"service-travego/service"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type FleetAvailabilityCron struct {
	db          *sql.DB
	driver      string
	wagyClient  *wagy.WagyClient
	scheduleSvc *service.ScheduleService
}

func NewFleetAvailabilityCron(db *sql.DB, driver string, wagyClient *wagy.WagyClient) *FleetAvailabilityCron {
	scheduleRepo := repository.NewScheduleRepository(db, driver)
	scheduleSvc := service.NewScheduleService(scheduleRepo)

	return &FleetAvailabilityCron{
		db:          db,
		driver:      driver,
		wagyClient:  wagyClient,
		scheduleSvc: scheduleSvc,
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
		GROUP BY ac.organization_id, ac.account_number, o.organization_name
	`

	rows, err := c.db.Query(query)
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

	// 2. Get fleet availability
	items, err := c.scheduleSvc.GetFleetAvailability(model.ScheduleFleetAvailabilityServiceInput{
		OrganizationID: org.OrganizationID,
		Filter: model.ScheduleFleetAvailabilityFilter{
			StartDate: startDate,
			EndDate:   endDate,
		},
	})
	if err != nil {
		log.Printf("[FleetAvailabilityCron] GetFleetAvailability error for org %s: %v", org.OrganizationID, err)
		return
	}

	// 3. Format message
	message := c.formatMessage(org.OrganizationName, items)

	// 4. Send via Wagy
	_, err = c.wagyClient.SendMessage(org.AccountNumber, message)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] SendMessage error to %s: %v", org.AccountNumber, err)
		return
	}

	log.Printf("[FleetAvailabilityCron] Message sent to %s (%s)", org.AccountNumber, org.OrganizationName)
}

func (c *FleetAvailabilityCron) formatMessage(orgName string, items []model.ScheduleFleetAvailabilityItem) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Selamat Pagi %s\n\n", orgName))
	b.WriteString("Berikut adalah daftar armada yang tersedia:\n\n")

	if len(items) == 0 {
		b.WriteString("Tidak ada armada yang tersedia untuk 7 hari ke depan.\n")
		return b.String()
	}

	// Group by fleet name
	fleetCount := make(map[string]int)
	for _, item := range items {
		fleetName := item.FleetName
		if fleetName == "" {
			fleetName = item.PlateNumber
		}
		fleetCount[fleetName]++
	}

	for name, count := range fleetCount {
		b.WriteString(fmt.Sprintf("• %s: %d unit tersedia\n", name, count))
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
	_, err := c.AddFunc("0 9 * * 1,3,5", cronJob.Run)
	if err != nil {
		log.Printf("[FleetAvailabilityCron] Failed to register cron: %v", err)
		return nil
	}

	c.Start()
	log.Println("[FleetAvailabilityCron] Scheduled: Mon, Wed, Fri at 09:00")

	return c
}
