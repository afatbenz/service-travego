package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TourPackageRepository struct {
	db     *sql.DB
	driver string
}

func NewTourPackageRepository(db *sql.DB, driver string) *TourPackageRepository {
	return &TourPackageRepository{
		db:     db,
		driver: driver,
	}
}

func (r *TourPackageRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (r *TourPackageRepository) GetTourPackagesByOrgID(orgID string) ([]model.TourPackageListItem, error) {
	query := `
		SELECT 
			tp.uuid AS package_id,
			tp.package_name,
			tp.thumbnail,
			tp.package_description,
			tp.status,
			tp.active,
			MIN(tpp.min_pax) AS min_pax,
			MIN(tpp.price) AS min_price,
			MAX(tpp.min_pax) AS max_pax,
			MAX(tpp.price) AS max_price
		FROM tour_packages tp
		LEFT JOIN tour_package_prices tpp 
			ON tpp.package_id = tp.uuid
		WHERE tp.organization_id = %s 
		  AND tp.status = 1 AND tp.active = true
		GROUP BY tp.uuid, tp.package_name, tp.thumbnail, tp.package_description, tp.status, tp.active
	`

	// Adjust query placeholder
	query = fmt.Sprintf(query, r.getPlaceholder(1))

	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.TourPackageListItem{} // Initialize as empty slice
	for rows.Next() {
		var item model.TourPackageListItem

		var thumbnail, description sql.NullString
		var status sql.NullInt64
		var active sql.NullBool
		var minPax, maxPax sql.NullInt64
		var minPrice, maxPrice sql.NullFloat64

		err := rows.Scan(
			&item.PackageID,
			&item.PackageName,
			&thumbnail,
			&description,
			&status,
			&active,
			&minPax,
			&minPrice,
			&maxPax,
			&maxPrice,
		)
		if err != nil {
			return nil, err
		}

		if thumbnail.Valid {
			item.Thumbnail = thumbnail.String
		}
		if description.Valid {
			item.PackageDescription = description.String
		}
		if minPax.Valid {
			item.MinPax = int(minPax.Int64)
		}
		if maxPax.Valid {
			item.MaxPax = int(maxPax.Int64)
		}
		if minPrice.Valid {
			item.MinPrice = minPrice.Float64
			item.Price = minPrice.Float64
		}
		if maxPrice.Valid {
			item.MaxPrice = maxPrice.Float64
		}
		if status.Valid {
			item.Status = int(status.Int64)
		}
		if active.Valid {
			item.Active = active.Bool
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *TourPackageRepository) CreateTourPackage(ctx context.Context, req *model.CreateTourPackageRequest, packageID, orgID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Insert into tour_packages
	query := `INSERT INTO tour_packages (uuid, package_name, package_type, package_description, active, thumbnail, organization_id, created_by, created_at, status) VALUES `
	if r.driver == "postgres" || r.driver == "pgx" {
		query += `($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	} else {
		query += `(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err = tx.ExecContext(ctx, query,
		packageID,
		req.PackageName,
		req.PackageType,
		req.PackageDescription,
		req.Active,
		req.Thumbnail,
		orgID,
		userID,
		now,
		1, // Status default 1
	)
	if err != nil {
		log.Printf("[ERROR] CreateTourPackage failed - Path: %s, Error: %v", ctx.Value("path"), err)
		return err
	}

	// 2. Addons
	if len(req.Addons) > 0 {
		addonQuery := `INSERT INTO tour_package_addons (uuid, package_id, organization_id, description, price, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			addonQuery += `($1, $2, $3, $4, $5, $6, $7)`
		} else {
			addonQuery += `(?, ?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, addonQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, addon := range req.Addons {
			_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, addon.Description, addon.Price, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 3. Facilities
	if len(req.Facilities) > 0 {
		facilityQuery := `INSERT INTO tour_package_facilities (uuid, package_id, organization_id, facility, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			facilityQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			facilityQuery += `(?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, facilityQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, facility := range req.Facilities {
			_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, facility, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 4. Itineraries
	if len(req.Itineraries) > 0 {
		itinQuery := `INSERT INTO tour_package_itineraries (uuid, package_id, organization_id, day, activity, location, city_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			itinQuery += `($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		} else {
			itinQuery += `(?, ?, ?, ?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, itinQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, day := range req.Itineraries {
			for _, act := range day.Activities {
				activityTime := act.Time
				if activityTime == "" {
					activityTime = "00:00:00"
				}
				_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, activityTime, act.Description, act.Location, act.City.ID, now, userID)
				if err != nil {
					return err
				}
			}
		}
	}

	// 5. Pickup Areas
	if len(req.PickupAreas) > 0 {
		pickupQuery := `INSERT INTO tour_package_pickup (uuid, package_id, organization_id, city_id, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			pickupQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			pickupQuery += `(?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, pickupQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, area := range req.PickupAreas {
			_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, area.ID, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 6. Prices
	if len(req.Pricing) > 0 {
		priceQuery := `INSERT INTO tour_package_prices (uuid, package_id, organization_id, min_pax, max_pax, price, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			priceQuery += `($1, $2, $3, $4, $5, $6, $7, $8)`
		} else {
			priceQuery += `(?, ?, ?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, priceQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, price := range req.Pricing {
			_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, price.MinPax, price.MaxPax, price.Price, now, userID)
			if err != nil {
				return err
			}
		}
	}

	// 7. Images
	if len(req.Images) > 0 {
		imageQuery := `INSERT INTO tour_package_images (uuid, package_id, organization_id, image_path, created_at, created_by) VALUES `
		if r.driver == "postgres" || r.driver == "pgx" {
			imageQuery += `($1, $2, $3, $4, $5, $6)`
		} else {
			imageQuery += `(?, ?, ?, ?, ?, ?)`
		}

		stmt, err := tx.PrepareContext(ctx, imageQuery)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, img := range req.Images {
			_, err = stmt.ExecContext(ctx, uuid.New().String(), packageID, orgID, img, now, userID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) UpdateTourPackage(ctx context.Context, req *model.UpdateTourPackageRequest, orgID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	updateQuery := `UPDATE tour_packages SET package_name = %s, package_type = %s, package_description = %s, active = %s, thumbnail = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND organization_id = %s`
	updateQuery = fmt.Sprintf(
		updateQuery,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		r.getPlaceholder(9),
	)

	res, err := tx.ExecContext(
		ctx,
		updateQuery,
		req.PackageName,
		req.PackageType,
		req.PackageDescription,
		req.Active,
		req.Thumbnail,
		now,
		userID,
		req.PackageID,
		orgID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}

	if req.Addons != nil {
		keep := make([]string, 0, len(req.Addons))
		for _, it := range req.Addons {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_addons (uuid, package_id, organization_id, description, price, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6, $7)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?, ?)`
				}
				if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, it.Description, it.Price, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}

			upd := `UPDATE tour_package_addons SET description = %s, price = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5))
			if _, err := tx.ExecContext(ctx, upd, it.Description, it.Price, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_addons WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_addons WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Facilities != nil {
		keep := make([]string, 0, len(req.Facilities))
		for _, it := range req.Facilities {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_facilities (uuid, package_id, organization_id, facility, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, it.Facility, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_facilities SET facility = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
			if _, err := tx.ExecContext(ctx, upd, it.Facility, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_facilities WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_facilities WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.PickupAreas != nil {
		keep := make([]string, 0, len(req.PickupAreas))
		for _, it := range req.PickupAreas {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_pickup (uuid, package_id, organization_id, city_id, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, it.ID, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_pickup SET city_id = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
			if _, err := tx.ExecContext(ctx, upd, it.ID, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_pickup WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_pickup WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Pricing != nil {
		keep := make([]string, 0, len(req.Pricing))
		for _, it := range req.Pricing {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_prices (uuid, package_id, organization_id, min_pax, max_pax, price, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6, $7, $8)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?, ?, ?)`
				}
				if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, it.MinPax, it.MaxPax, it.Price, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_prices SET min_pax = %s, max_pax = %s, price = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8))
			if _, err := tx.ExecContext(ctx, upd, it.MinPax, it.MaxPax, it.Price, now, userID, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_prices WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_prices WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Images != nil {
		keep := make([]string, 0, len(req.Images))
		for _, it := range req.Images {
			if it.UUID == "" {
				newID := uuid.New().String()
				ins := `INSERT INTO tour_package_images (uuid, package_id, organization_id, image_path, created_at, created_by) VALUES `
				if r.driver == "postgres" || r.driver == "pgx" {
					ins += `($1, $2, $3, $4, $5, $6)`
				} else {
					ins += `(?, ?, ?, ?, ?, ?)`
				}
				if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, it.ImagePath, now, userID); err != nil {
					return err
				}
				keep = append(keep, newID)
				continue
			}
			upd := `UPDATE tour_package_images SET image_path = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
			upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4))
			if _, err := tx.ExecContext(ctx, upd, it.ImagePath, it.UUID, req.PackageID, orgID); err != nil {
				return err
			}
			keep = append(keep, it.UUID)
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_images WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_images WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	if req.Itineraries != nil {
		keep := make([]string, 0)
		for _, day := range req.Itineraries {
			for _, act := range day.Activities {
				activityTime := act.Time
				if activityTime == "" {
					activityTime = "00:00:00"
				}

				if act.UUID == "" {
					newID := uuid.New().String()
					ins := `INSERT INTO tour_package_itineraries (uuid, package_id, organization_id, day, activity, location, city_id, created_at, created_by) VALUES `
					if r.driver == "postgres" || r.driver == "pgx" {
						ins += `($1, $2, $3, $4, $5, $6, $7, $8, $9)`
					} else {
						ins += `(?, ?, ?, ?, ?, ?, ?, ?, ?)`
					}
					if _, err := tx.ExecContext(ctx, ins, newID, req.PackageID, orgID, activityTime, act.Description, act.Location, act.City.ID, now, userID); err != nil {
						return err
					}
					keep = append(keep, newID)
					continue
				}

				upd := `UPDATE tour_package_itineraries SET day = %s, activity = %s, location = %s, city_id = %s, updated_at = %s, updated_by = %s WHERE uuid = %s AND package_id = %s AND organization_id = %s`
				upd = fmt.Sprintf(upd, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9))
				if _, err := tx.ExecContext(ctx, upd, activityTime, act.Description, act.Location, act.City.ID, now, userID, act.UUID, req.PackageID, orgID); err != nil {
					return err
				}
				keep = append(keep, act.UUID)
			}
		}

		if len(keep) == 0 {
			del := fmt.Sprintf("DELETE FROM tour_package_itineraries WHERE package_id = %s AND organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2))
			if _, err := tx.ExecContext(ctx, del, req.PackageID, orgID); err != nil {
				return err
			}
		} else {
			ph := make([]string, 0, len(keep))
			args := make([]interface{}, 0, 2+len(keep))
			args = append(args, req.PackageID, orgID)
			for i, id := range keep {
				ph = append(ph, r.getPlaceholder(i+3))
				args = append(args, id)
			}
			del := fmt.Sprintf("DELETE FROM tour_package_itineraries WHERE package_id = %s AND organization_id = %s AND uuid NOT IN (%s)", r.getPlaceholder(1), r.getPlaceholder(2), strings.Join(ph, ","))
			if _, err := tx.ExecContext(ctx, del, args...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *TourPackageRepository) GetTourPackageDetail(ctx context.Context, orgID, packageID string) (*model.TourPackageDetailResponse, error) {
	detail := &model.TourPackageDetailResponse{
		Schedules:    []model.TourPackageScheduleItem{},
		Pricing:      []model.TourPackagePricing{},
		PickupAreas:  []model.TourPackagePickupAreaItem{},
		Images:       []string{},
		Itineraries:  []model.TourPackageItineraryDetailItem{},
		Facilities:   []string{},
		Destinations: []model.TourPackageDestinationItem{},
		Addons:       []model.TourPackageAddon{},
	}

	metaQuery := `
		SELECT uuid, package_name, package_type, package_description, thumbnail, duration, min_pax, max_pax, active, status
		FROM tour_packages
		WHERE uuid = %s AND organization_id = %s
		LIMIT 1
	`
	metaQuery = fmt.Sprintf(metaQuery, r.getPlaceholder(1), r.getPlaceholder(2))

	var (
		metaPackageID string
		packageName   sql.NullString
		packageType   sql.NullInt64
		packageDesc   sql.NullString
		thumbnail     sql.NullString
		duration      sql.NullInt64
		minPax        sql.NullInt64
		maxPax        sql.NullInt64
		active        sql.NullBool
		status        sql.NullInt64
	)

	err := r.db.QueryRowContext(ctx, metaQuery, packageID, orgID).Scan(
		&metaPackageID,
		&packageName,
		&packageType,
		&packageDesc,
		&thumbnail,
		&duration,
		&minPax,
		&maxPax,
		&active,
		&status,
	)
	if err != nil {
		return nil, err
	}

	detail.Meta = model.TourPackageDetailMeta{
		PackageID:          metaPackageID,
		PackageName:        packageName.String,
		PackageType:        int(packageType.Int64),
		PackageDescription: packageDesc.String,
		Thumbnail:          thumbnail.String,
		Duration:           int(duration.Int64),
		MinPax:             int(minPax.Int64),
		MaxPax:             int(maxPax.Int64),
		Active:             active.Bool,
		Status:             int(status.Int64),
	}

	scheduleQuery := `
		SELECT date_start, date_end
		FROM tour_package_schedules
		WHERE package_id = %s AND organization_id = %s
		ORDER BY date_start ASC
	`
	scheduleQuery = fmt.Sprintf(scheduleQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	scheduleRows, err := r.db.QueryContext(ctx, scheduleQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer scheduleRows.Close()
	for scheduleRows.Next() {
		var ds, de time.Time
		if err := scheduleRows.Scan(&ds, &de); err != nil {
			return nil, err
		}
		detail.Schedules = append(detail.Schedules, model.TourPackageScheduleItem{
			DateStart: ds.Format("2006-01-02"),
			DateEnd:   de.Format("2006-01-02"),
		})
	}

	priceQuery := `
		SELECT min_pax, max_pax, price
		FROM tour_package_prices
		WHERE package_id = %s AND organization_id = %s
		ORDER BY min_pax ASC, max_pax ASC
	`
	priceQuery = fmt.Sprintf(priceQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	priceRows, err := r.db.QueryContext(ctx, priceQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer priceRows.Close()
	for priceRows.Next() {
		var minPax, maxPax sql.NullInt64
		var priceVal sql.NullFloat64
		if err := priceRows.Scan(&minPax, &maxPax, &priceVal); err != nil {
			return nil, err
		}
		detail.Pricing = append(detail.Pricing, model.TourPackagePricing{
			MinPax: int(minPax.Int64),
			MaxPax: int(maxPax.Int64),
			Price:  priceVal.Float64,
		})
	}

	pickupQuery := `
		SELECT city_id
		FROM tour_package_pickup
		WHERE package_id = %s AND organization_id = %s
		ORDER BY city_id ASC
	`
	pickupQuery = fmt.Sprintf(pickupQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	pickupRows, err := r.db.QueryContext(ctx, pickupQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer pickupRows.Close()
	for pickupRows.Next() {
		var cityID sql.NullInt64
		if err := pickupRows.Scan(&cityID); err != nil {
			return nil, err
		}
		detail.PickupAreas = append(detail.PickupAreas, model.TourPackagePickupAreaItem{CityID: int(cityID.Int64)})
	}

	imageQuery := `
		SELECT image_path
		FROM tour_package_images
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	imageQuery = fmt.Sprintf(imageQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	imageRows, err := r.db.QueryContext(ctx, imageQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer imageRows.Close()
	for imageRows.Next() {
		var img sql.NullString
		if err := imageRows.Scan(&img); err != nil {
			return nil, err
		}
		if img.Valid {
			detail.Images = append(detail.Images, img.String)
		}
	}

	itinQuery := `
		SELECT day, activity, location, city_id
		FROM tour_package_itineraries
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	itinQuery = fmt.Sprintf(itinQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	itinRows, err := r.db.QueryContext(ctx, itinQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer itinRows.Close()
	for itinRows.Next() {
		var (
			tm       time.Time
			act      sql.NullString
			location sql.NullString
			cityID   sql.NullInt64
		)
		if err := itinRows.Scan(&tm, &act, &location, &cityID); err != nil {
			return nil, err
		}
		detail.Itineraries = append(detail.Itineraries, model.TourPackageItineraryDetailItem{
			Time:        tm.Format("15:04:05"),
			Description: act.String,
			Location:    location.String,
			CityID:      int(cityID.Int64),
		})
	}

	facilityQuery := `
		SELECT facility
		FROM tour_package_facilities
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	facilityQuery = fmt.Sprintf(facilityQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	facilityRows, err := r.db.QueryContext(ctx, facilityQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer facilityRows.Close()
	for facilityRows.Next() {
		var facility sql.NullString
		if err := facilityRows.Scan(&facility); err != nil {
			return nil, err
		}
		if facility.Valid && facility.String != "" {
			detail.Facilities = append(detail.Facilities, facility.String)
		}
	}

	destQuery := `
		SELECT city_id, destination
		FROM tour_package_destinations
		WHERE package_id = %s AND organization_id = %s
		ORDER BY city_id ASC
	`
	destQuery = fmt.Sprintf(destQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	destRows, err := r.db.QueryContext(ctx, destQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer destRows.Close()
	for destRows.Next() {
		var cityID sql.NullInt64
		var destination sql.NullString
		if err := destRows.Scan(&cityID, &destination); err != nil {
			return nil, err
		}
		detail.Destinations = append(detail.Destinations, model.TourPackageDestinationItem{
			CityID:      int(cityID.Int64),
			Destination: destination.String,
		})
	}

	addonQuery := `
		SELECT description, price
		FROM tour_package_addons
		WHERE package_id = %s AND organization_id = %s
		ORDER BY created_at ASC
	`
	addonQuery = fmt.Sprintf(addonQuery, r.getPlaceholder(1), r.getPlaceholder(2))
	addonRows, err := r.db.QueryContext(ctx, addonQuery, packageID, orgID)
	if err != nil {
		return nil, err
	}
	defer addonRows.Close()
	for addonRows.Next() {
		var description sql.NullString
		var priceVal sql.NullFloat64
		if err := addonRows.Scan(&description, &priceVal); err != nil {
			return nil, err
		}
		detail.Addons = append(detail.Addons, model.TourPackageAddon{
			Description: description.String,
			Price:       priceVal.Float64,
		})
	}

	return detail, nil
}
