package repository

import (
	"database/sql"
	"fmt"
	"log"
	"service-travego/database"
	"service-travego/model"
	"time"
)

// Create creates new user
func (r *UserRepository) Create(user *model.User) (*model.User, error) {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
            INSERT INTO users (user_id, username, fullname, email, password, phone, is_active, is_verified, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING created_at, updated_at
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

		err := database.QueryRow(
			r.db,
			query,
			user.UserID,
			user.Username,
			user.Name,
			user.Email,
			user.Password,
			user.Phone,
			user.IsActive,
			user.IsVerified,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(&user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
            INSERT INTO users (user_id, username, fullname, email, password, phone, is_active, is_verified, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10))

		_, err := database.Exec(
			r.db,
			query,
			user.UserID,
			user.Username,
			user.Name,
			user.Email,
			user.Password,
			user.Phone,
			user.IsActive,
			user.IsVerified,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// FindByID retrieves user
func (r *UserRepository) FindByID(id string) (*model.User, error) {
	query := fmt.Sprintf(`
        SELECT user_id, username, fullname, email, password, phone, address, city, province, postal_code,
               npwp, gender, date_of_birth, is_active, is_verified, is_admin, created_at, updated_at, deleted_at
        FROM users
        WHERE user_id = %s AND deleted_at IS NULL
    `, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime
	var fullname, address, city, province, postalCode, npwp, gender sql.NullString
	var isAdmin sql.NullBool

	err := database.QueryRow(r.db, query, id).Scan(
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
			log.Printf("[DEBUG] FindByID no rows - Query: %s | Param: %s", query, id)
			return nil, sql.ErrNoRows
		}
		log.Printf("[ERROR] FindByID query error - Query: %s | Param: %s | Error: %v", query, id, err)
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

// FindByEmail retrieves user
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	query := fmt.Sprintf(`
        SELECT user_id, username, fullname, email, password, phone, address, city, province, postal_code,
               npwp, gender, date_of_birth, is_active, is_verified, is_admin, created_at, updated_at, deleted_at
        FROM users
        WHERE LOWER(email) = LOWER(%s) AND deleted_at IS NULL
    `, r.getPlaceholder(1))

	var user model.User
	var deletedAt, dateOfBirth sql.NullTime
	var fullname, address, city, province, postalCode, npwp, gender sql.NullString
	var isAdmin sql.NullBool

	err := database.QueryRow(r.db, query, email).Scan(
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

// Delete soft deletes user
func (r *UserRepository) Delete(userID string) error {
	query := fmt.Sprintf(`
		UPDATE users
		SET deleted_at = %s
		WHERE user_id = %s AND deleted_at IS NULL
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	now := time.Now()
	result, err := database.Exec(r.db, query, now, userID)
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
