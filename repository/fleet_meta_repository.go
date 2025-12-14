package repository

import (
    "database/sql"
    "fmt"
)

type FleetMetaRepository struct {
    db     *sql.DB
    driver string
}

func NewFleetMetaRepository(db *sql.DB, driver string) *FleetMetaRepository {
    return &FleetMetaRepository{db: db, driver: driver}
}

func (r *FleetMetaRepository) getPlaceholder(pos int) string {
    if r.driver == "mysql" {
        return "?"
    }
    return fmt.Sprintf("$%d", pos)
}

func (r *FleetMetaRepository) FindBodies(organizationID string, search string) ([]string, error) {
    var rows *sql.Rows
    var err error
    if search != "" {
        if r.driver == "mysql" {
            query := fmt.Sprintf(`
                SELECT body FROM fleets
                WHERE organization_id = %s AND body LIKE CONCAT('%%', %s, '%%')
                ORDER BY body
            `, r.getPlaceholder(1), r.getPlaceholder(2))
            rows, err = r.db.Query(query, organizationID, search)
        } else {
            query := fmt.Sprintf(`
                SELECT body FROM fleets
                WHERE organization_id = %s AND body LIKE '%%' || %s || '%%'
                ORDER BY body
            `, r.getPlaceholder(1), r.getPlaceholder(2))
            rows, err = r.db.Query(query, organizationID, search)
        }
    } else {
        query := fmt.Sprintf(`
            SELECT body FROM fleets
            WHERE organization_id = %s
            ORDER BY body
        `, r.getPlaceholder(1))
        rows, err = r.db.Query(query, organizationID)
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var list []string
    for rows.Next() {
        var v sql.NullString
        if err := rows.Scan(&v); err != nil {
            return nil, err
        }
        if v.Valid {
            list = append(list, v.String)
        }
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return list, nil
}

func (r *FleetMetaRepository) FindEngines(organizationID string, search string) ([]string, error) {
    var rows *sql.Rows
    var err error
    if search != "" {
        if r.driver == "mysql" {
            query := fmt.Sprintf(`
                SELECT engine FROM fleets
                WHERE organization_id = %s AND engine LIKE CONCAT('%%', %s, '%%')
                ORDER BY engine
            `, r.getPlaceholder(1), r.getPlaceholder(2))
            rows, err = r.db.Query(query, organizationID, search)
        } else {
            query := fmt.Sprintf(`
                SELECT engine FROM fleets
                WHERE organization_id = %s AND engine LIKE '%%' || %s || '%%'
                ORDER BY engine
            `, r.getPlaceholder(1), r.getPlaceholder(2))
            rows, err = r.db.Query(query, organizationID, search)
        }
    } else {
        query := fmt.Sprintf(`
            SELECT engine FROM fleets
            WHERE organization_id = %s
            ORDER BY engine
        `, r.getPlaceholder(1))
        rows, err = r.db.Query(query, organizationID)
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var list []string
    for rows.Next() {
        var v sql.NullString
        if err := rows.Scan(&v); err != nil {
            return nil, err
        }
        if v.Valid {
            list = append(list, v.String)
        }
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return list, nil
}
