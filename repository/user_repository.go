package repository

import (
	"database/sql"
	"fmt"
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

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *UserRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindAll retrieves all users from database
func (r *UserRepository) FindAll() ([]model.User, error) {
	query := `
		SELECT user_id, fullname, email, password, phone, address, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
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

// FindByID retrieves a user by ID (UUID) from database
// Implementation is in user_repository_uuid.go

// FindByEmail retrieves a user by email from database (UUID with all fields)
// Implementation is in user_repository_uuid.go

// Create inserts a new user into database with UUID
// Implementation is in user_repository_uuid.go

// Update updates an existing user in database
func (r *UserRepository) Update(user *model.User) (*model.User, error) {
	user.UpdatedAt = time.Now()

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
			UPDATE users
			SET fullname = %s, email = %s, phone = %s, address = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
			RETURNING created_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6))

		err := r.db.QueryRow(
			query,
			user.Name,
			user.Email,
			user.Phone,
			user.Address,
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
			SET fullname = %s, email = %s, phone = %s, address = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6))

		result, err := r.db.Exec(
			query,
			user.Name,
			user.Email,
			user.Phone,
			user.Address,
			user.UpdatedAt,
			user.UserID,
		)
		if err != nil {
			return nil, err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rowsAffected == 0 {
			return nil, sql.ErrNoRows
		}

		// For MySQL, we need to fetch the created_at separately
		err = r.db.QueryRow(fmt.Sprintf("SELECT created_at FROM users WHERE user_id = %s", r.getPlaceholder(1)), user.UserID).Scan(&user.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// Delete soft deletes a user from database (UUID)
// Implementation is in user_repository_uuid.go
