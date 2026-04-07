package repository

import (
	"database/sql"
	"service-travego/database"
	"service-travego/model"
)

type GeneralRepository struct {
	db     *sql.DB
	driver string
}

func NewGeneralRepository(db *sql.DB, driver string) *GeneralRepository {
	return &GeneralRepository{
		db:     db,
		driver: driver,
	}
}

// GetBankList retrieves bank list
func (r *GeneralRepository) GetBankList() ([]model.Bank, error) {
	query := `
        SELECT code, name
        FROM bank_list
        ORDER BY name ASC
    `

	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var banks []model.Bank
	for rows.Next() {
		var bank model.Bank
		if err := rows.Scan(&bank.Code, &bank.Name); err != nil {
			return nil, err
		}
		banks = append(banks, bank)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return banks, nil
}
