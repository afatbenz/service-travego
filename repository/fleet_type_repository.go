package repository

import (
    "database/sql"
)

type FleetTypeRepository struct {
    db *sql.DB
}

func NewFleetTypeRepository(db *sql.DB) *FleetTypeRepository {
    return &FleetTypeRepository{db: db}
}

// FindAll returns all fleet_types rows ordered by label
func (r *FleetTypeRepository) FindAll() ([]map[string]interface{}, error) {
    rows, err := r.db.Query("SELECT * FROM fleet_types ORDER BY label")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    cols, err := rows.Columns()
    if err != nil {
        return nil, err
    }

    var results []map[string]interface{}
    for rows.Next() {
        values := make([]interface{}, len(cols))
        valuePtrs := make([]interface{}, len(cols))
        for i := range values {
            valuePtrs[i] = &values[i]
        }

        if err := rows.Scan(valuePtrs...); err != nil {
            return nil, err
        }

        m := make(map[string]interface{}, len(cols))
        for i, col := range cols {
            v := values[i]
            if b, ok := v.([]byte); ok {
                m[col] = string(b)
            } else {
                m[col] = v
            }
        }
        results = append(results, m)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }
    return results, nil
}
