package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"time"
)

// CreateWithUUID creates a new user with UUID (updated method)
func (r *UserRepository) Create(user *model.User) (*model.User, error) {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
			INSERT INTO users (user_id, username, name, email, password, phone, address, city, province, 
			                    postal_code, npwp, gender, date_of_birth, status, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
			RETURNING created_at, updated_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
			r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16))

		var dateOfBirth sql.NullTime
		err := r.db.QueryRow(
			query,
			user.UserID,
			user.Username,
			user.Name,
			user.Email,
			user.Password,
			user.Phone,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.Status,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			return nil, err
		}
		if user.DateOfBirth != nil {
			dateOfBirth.Valid = true
			dateOfBirth.Time = *user.DateOfBirth
		}
	} else {
		query := fmt.Sprintf(`
			INSERT INTO users (user_id, username, name, email, password, phone, address, city, province,
			                    postal_code, npwp, gender, date_of_birth, status, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
			r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15), r.getPlaceholder(16))

		_, err := r.db.Exec(
			query,
			user.UserID,
			user.Username,
			user.Name,
			user.Email,
			user.Password,
			user.Phone,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.Status,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// FindByID retrieves a user by ID (UUID) from database
func (r *UserRepository) FindByID(id string) (*model.User, error) {
	query := fmt.Sprintf(`
		SELECT user_id, username, name, email, password, phone, address, city, province, postal_code,
		       npwp, gender, date_of_birth, status, created_at, updated_at, deleted_at
		FROM users
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&user.UserID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Phone,
		&user.Address,
		&user.City,
		&user.Province,
		&user.PostalCode,
		&user.NPWP,
		&user.Gender,
		&dateOfBirth,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	if dateOfBirth.Valid {
		user.DateOfBirth = &dateOfBirth.Time
	}

	return &user, nil
}

// FindByEmail retrieves a user by email from database (updated for UUID)
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	query := fmt.Sprintf(`
		SELECT user_id, username, name, email, password, phone, address, city, province, postal_code,
		       npwp, gender, date_of_birth, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime

	err := r.db.QueryRow(query, email).Scan(
		&user.UserID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Phone,
		&user.Address,
		&user.City,
		&user.Province,
		&user.PostalCode,
		&user.NPWP,
		&user.Gender,
		&dateOfBirth,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	if dateOfBirth.Valid {
		user.DateOfBirth = &dateOfBirth.Time
	}

	return &user, nil
}

// Delete soft deletes a user from database (UUID)
func (r *UserRepository) Delete(id string) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET deleted_at = %s
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	now := time.Now()
	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
