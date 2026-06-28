package cron

import (
	"database/sql"
	"log"
	"time"
)

func insertAssistantAccountStat(db *sql.DB, driver string, organizationID string, status int) {
	period := time.Now().Format("2006-01-02")
	typeVal := 1

	var query string
	if driver == "postgres" || driver == "pgx" {
		query = `
			INSERT INTO assistant_account_stats (period, count, organization_id, type, status)
			VALUES ($1, 1, $2, $3, $4)
			ON CONFLICT (period, type, status, organization_id)
			DO UPDATE SET count = assistant_account_stats.count + 1
		`
	} else {
		query = `
			INSERT INTO assistant_account_stats (period, count, organization_id, type, status)
			VALUES (?, 1, ?, ?, ?)
			ON DUPLICATE KEY UPDATE count = count + 1
		`
	}

	_, err := db.Exec(query, period, organizationID, typeVal, status)
	if err != nil {
		log.Printf("[AssistantAccountStat] Failed to insert stat for org %s, status %d: %v", organizationID, status, err)
	} else {
		log.Printf("[AssistantAccountStat] Stat recorded for org %s, status %d", organizationID, status)
	}
}
