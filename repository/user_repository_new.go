package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"time"
)

// This file contains additional methods for UUID support

// FindByUsername retrieves a user by username from database
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	query := fmt.Sprintf(`
		SELECT user_id, username, name, email, password, phone, address, city, province, postal_code, 
		       npwp, gender, date_of_birth, status, created_at, updated_at, deleted_at
		FROM users
		WHERE username = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime

	err := r.db.QueryRow(query, username).Scan(
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

// UpdateStatus updates user status
func (r *UserRepository) UpdateStatus(id string, status int) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET status = %s, updated_at = %s
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	now := time.Now()
	result, err := r.db.Exec(query, status, now, id)
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

// UpdateProfile updates user profile data
func (r *UserRepository) UpdateProfile(user *model.User) (*model.User, error) {
	user.UpdatedAt = time.Now()

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
			UPDATE users
			SET name = %s, phone = %s, npwp = %s, gender = %s, date_of_birth = %s,
			    address = %s, city = %s, province = %s, postal_code = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
			RETURNING username, email, status, created_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

		var dateOfBirth sql.NullTime
		err := r.db.QueryRow(
			query,
			user.Name,
			user.Phone,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
			user.UpdatedAt,
			user.UserID,
		).Scan(&user.Username, &user.Email, &user.Status, &user.CreatedAt)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, sql.ErrNoRows
			}
			return nil, err
		}
		if user.DateOfBirth != nil {
			dateOfBirth.Valid = true
			dateOfBirth.Time = *user.DateOfBirth
		}
	} else {
		query := fmt.Sprintf(`
			UPDATE users
			SET name = %s, phone = %s, npwp = %s, gender = %s, date_of_birth = %s,
			    address = %s, city = %s, province = %s, postal_code = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11))

		result, err := r.db.Exec(
			query,
			user.Name,
			user.Phone,
			user.NPWP,
			user.Gender,
			user.DateOfBirth,
			user.Address,
			user.City,
			user.Province,
			user.PostalCode,
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

		// Fetch updated data
		err = r.db.QueryRow(fmt.Sprintf(`
			SELECT username, email, status, created_at 
			FROM users WHERE user_id = %s
		`, r.getPlaceholder(1)), user.UserID).Scan(&user.Username, &user.Email, &user.Status, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}
