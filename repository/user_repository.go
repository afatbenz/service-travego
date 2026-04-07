package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"
)

type UserRepository struct {
	db     *sql.DB
	driver string
}

func NewUserRepository(db *sql.DB, driver string) *UserRepository {
	return &UserRepository{
		db:     db,
		driver: driver,
	}
}

// getPlaceholder returns query placeholder
func (r *UserRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindAll retrieves users
func (r *UserRepository) FindAll() ([]model.User, error) {
	query := `
		SELECT user_id, fullname, email, password, phone, address, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := database.Query(r.db, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		var deletedAt sql.NullTime
		var fullname, address sql.NullString

		err := rows.Scan(
			&user.UserID,
			&fullname,
			&user.Email,
			&user.Password,
			&user.Phone,
			&address,
			&user.CreatedAt,
			&user.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		if fullname.Valid {
			user.Name = fullname.String
		}
		if address.Valid {
			user.Address = address.String
		}
		if deletedAt.Valid {
			user.DeletedAt = &deletedAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// FindByID retrieves user
// Implementation is in user_repository_uuid.go

// FindByEmail retrieves user
// Implementation is in user_repository_uuid.go

// Create inserts user
// Implementation is in user_repository_uuid.go

// Update updates user
func (r *UserRepository) Update(user *model.User) (*model.User, error) {
	user.UpdatedAt = time.Now()

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
            UPDATE users
            SET fullname = %s, email = %s, phone = %s, address = %s, city = %s, province = %s, postal_code = %s, 
                npwp = %s, gender = %s, date_of_birth = %s, updated_at = %s
            WHERE user_id = %s
            RETURNING created_at
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

		err := database.QueryRow(
			r.db,
			query,
			user.Name,
			user.Email,
			user.Phone,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.UpdatedAt,
			user.UserID,
		).Scan(&user.CreatedAt)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, sql.ErrNoRows
			}
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
            UPDATE users
            SET fullname = %s, email = %s, phone = %s, address = %s, city = %s, province = %s, postal_code = %s,
                npwp = %s, gender = %s, date_of_birth = %s, updated_at = %s
            WHERE user_id = %s
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

		_, err := database.Exec(
			r.db,
			query,
			user.Name,
			user.Email,
			user.Phone,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.UpdatedAt,
			user.UserID,
		)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// Delete soft deletes user
// Implementation is in user_repository_uuid.go
