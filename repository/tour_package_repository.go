package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
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
			MIN(tpp.min_pax) AS min_pax,
			MIN(tpp.price) AS min_price,
			MAX(tpp.min_pax) AS max_pax,
			MAX(tpp.price) AS max_price
		FROM tour_packages tp
		LEFT JOIN tour_package_prices tpp 
			ON tpp.package_id = tp.uuid
		WHERE tp.organization_id = %s 
		  AND tp.status = 1 AND tp.active = true
		GROUP BY tp.uuid, tp.package_name, tp.thumbnail, tp.package_description
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
		var minPax, maxPax sql.NullInt64
		var minPrice, maxPrice sql.NullFloat64
		
		err := rows.Scan(
			&item.PackageID,
			&item.PackageName,
			&thumbnail,
			&description,
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
		
		items = append(items, item)
	}
	
	return items, nil
}
