package helper

import (
	"database/sql"
	"fmt"
	"service-travego/configs"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// InitDatabase initializes database connection using database/sql
func InitDatabase(cfg *configs.Config) (*sql.DB, error) {
	var dsn string
	dbConfig := cfg.Database

	switch dbConfig.Driver {
	case "postgres":
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.Username,
			dbConfig.Password,
			dbConfig.Database,
			dbConfig.SSLMode,
		)
	case "mysql":
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConfig.Username,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.Database,
		)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", dbConfig.Driver)
	}

	db, err := sql.Open(dbConfig.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
