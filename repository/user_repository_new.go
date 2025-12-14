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
        SELECT user_id, username, fullname, email, password, phone, address, city, province, postal_code, 
               npwp, gender, date_of_birth, is_active, is_verified, is_admin, created_at, updated_at, deleted_at
        FROM users
        WHERE username = %s AND deleted_at IS NULL
    `, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime
	var fullname, address, city, province, postalCode, npwp, gender sql.NullString
	var isAdmin sql.NullBool

	err := r.db.QueryRow(query, username).Scan(
		&user.UserID,
		&user.Username,
		&fullname,
		&user.Email,
		&user.Password,
		&user.Phone,
		&address,
		&city,
		&province,
		&postalCode,
		&npwp,
		&gender,
		&dateOfBirth,
		&user.IsActive,
		&user.IsVerified,
		&isAdmin,
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

	if fullname.Valid {
		user.Name = fullname.String
	}
	if address.Valid {
		user.Address = address.String
	}
	if city.Valid {
		user.City = city.String
	}
	if province.Valid {
		user.Province = province.String
	}
	if postalCode.Valid {
		user.PostalCode = postalCode.String
	}
	if npwp.Valid {
		user.NPWP = npwp.String
	}
	if gender.Valid {
		user.Gender = gender.String
	}
	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	if dateOfBirth.Valid {
		user.DateOfBirth = &dateOfBirth.Time
	}

	return &user, nil
}

// FindByPhone retrieves a user by phone number from database
func (r *UserRepository) FindByPhone(phone string) (*model.User, error) {
	query := fmt.Sprintf(`
        SELECT user_id, username, fullname, email, password, phone, address, city, province, postal_code,
               npwp, gender, date_of_birth, is_active, is_verified, is_admin, created_at, updated_at, deleted_at
        FROM users
        WHERE phone = %s AND deleted_at IS NULL
    `, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime
	var fullname, address, city, province, postalCode, npwp, gender sql.NullString
	var isAdmin sql.NullBool

	err := r.db.QueryRow(query, phone).Scan(
		&user.UserID,
		&user.Username,
		&fullname,
		&user.Email,
		&user.Password,
		&user.Phone,
		&address,
		&city,
		&province,
		&postalCode,
		&npwp,
		&gender,
		&dateOfBirth,
		&user.IsActive,
		&user.IsVerified,
		&isAdmin,
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

	if fullname.Valid {
		user.Name = fullname.String
	}
	if address.Valid {
		user.Address = address.String
	}
	if city.Valid {
		user.City = city.String
	}
	if province.Valid {
		user.Province = province.String
	}
	if postalCode.Valid {
		user.PostalCode = postalCode.String
	}
	if npwp.Valid {
		user.NPWP = npwp.String
	}
	if gender.Valid {
		user.Gender = gender.String
	}
	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Time
	}
	if dateOfBirth.Valid {
		user.DateOfBirth = &dateOfBirth.Time
	}

	return &user, nil
}

// VerifyUser sets is_verified to true for a user
func (r *UserRepository) VerifyUser(id string) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET is_verified = %s, verified_at = %s, updated_at = %s
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))

	now := time.Now()
	result, err := r.db.Exec(query, true, now, now, id)
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
			SET fullname = %s, phone = %s, npwp = %s, gender = %s, date_of_birth = %s,
			    address = %s, city = %s, province = %s, postal_code = %s, avatar = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
			RETURNING username, email, is_active, is_verified, created_at
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

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
			user.Avatar,
			user.UpdatedAt,
			user.UserID,
		).Scan(&user.Username, &user.Email, &user.IsActive, &user.IsVerified, &user.CreatedAt)

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
			SET fullname = %s, phone = %s, npwp = %s, gender = %s, date_of_birth = %s,
			    address = %s, city = %s, province = %s, postal_code = %s, avatar = %s, updated_at = %s
			WHERE user_id = %s AND deleted_at IS NULL
		`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12))

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
			user.Avatar,
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
			SELECT username, email, is_active, is_verified, created_at 
			FROM users WHERE user_id = %s
		`, r.getPlaceholder(1)), user.UserID).Scan(&user.Username, &user.Email, &user.IsActive, &user.IsVerified, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(userID, hashedPassword string) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET password = %s, updated_at = %s
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	now := time.Now()
	result, err := r.db.Exec(query, hashedPassword, now, userID)
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
