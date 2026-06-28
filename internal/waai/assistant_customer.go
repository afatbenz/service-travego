package waai

import (
	"context"
	"database/sql"
	"fmt"
)

// AssistantCustomer merepresentasikan baris assistant_customers
type AssistantCustomer struct {
	DeviceID          string // nomor WA yang menerima pesan customer (owner WA)
	DeviceName        string // nama tampil assistant
	AssistantDeviceID string // device ID Wagy untuk kirim balasan
	Account           string
	OrganizationID    string
	DeviceToken       string // token Wagy untuk device ini
}

// AssistantCustomerRepository queries assistant_customers table
type AssistantCustomerRepository struct {
	db     *sql.DB
	driver string
}

func NewAssistantCustomerRepository(db *sql.DB, driver string) *AssistantCustomerRepository {
	return &AssistantCustomerRepository{db: db, driver: driver}
}

func (r *AssistantCustomerRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindByDeviceID mencari perusahaan assistant berdasarkan nomor WA yang menerima pesan
// phone adalah nomor WA owner (diekstrak dari OwnerJID payload Wagy)
func (r *AssistantCustomerRepository) FindByDeviceID(ctx context.Context, phone string) (*AssistantCustomer, bool, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(device_id, '')             as device_id,
			COALESCE(device_name, '')           as device_name,
			COALESCE(assistant_device_id, '')   as assistant_device_id,
			COALESCE(account, '')               as account,
			COALESCE(organization_id::text, '') as organization_id,
			COALESCE(device_token, '')          as device_token
		FROM assistant_customers
		WHERE account = %s
		LIMIT 1
	`, r.getPlaceholder(1))

	var ac AssistantCustomer
	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&ac.DeviceID,
		&ac.DeviceName,
		&ac.AssistantDeviceID,
		&ac.Account,
		&ac.OrganizationID,
		&ac.DeviceToken,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("query assistant_customers: %w", err)
	}

	if ac.DeviceName == "" {
		ac.DeviceName = "Asisten " + ac.DeviceID
	}

	return &ac, true, nil
}
