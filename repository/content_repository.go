package repository

import (
	"database/sql"
	"fmt"
	"service-travego/model"
)

type ContentRepository struct {
	db     *sql.DB
	driver string
}

func NewContentRepository(db *sql.DB, driver string) *ContentRepository {
	return &ContentRepository{
		db:     db,
		driver: driver,
	}
}

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *ContentRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindByTagAndOrgID checks if content exists
func (r *ContentRepository) FindByTagAndOrgID(sectionTag, orgID string) (*model.Content, error) {
	query := fmt.Sprintf(`
		SELECT uuid, section_tag, text, organization_id, created_at, created_by, updated_at, updated_by
		FROM content
		WHERE section_tag = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var content model.Content
	err := r.db.QueryRow(query, sectionTag, orgID).Scan(
		&content.UUID,
		&content.SectionTag,
		&content.Text,
		&content.OrganizationID,
		&content.CreatedAt,
		&content.CreatedBy,
		&content.UpdatedAt,
		&content.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &content, nil
}

// Create inserts new content
func (r *ContentRepository) Create(content *model.Content) error {
	query := fmt.Sprintf(`
		INSERT INTO content (uuid, section_tag, text, organization_id, created_at, created_by, updated_at, updated_by)
		VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
	)

	_, err := r.db.Exec(query,
		content.UUID,
		content.SectionTag,
		content.Text,
		content.OrganizationID,
		content.CreatedAt,
		content.CreatedBy,
		content.UpdatedAt,
		content.UpdatedBy,
	)

	return err
}

// Update updates existing content
func (r *ContentRepository) Update(content *model.Content) error {
	query := fmt.Sprintf(`
		UPDATE content
		SET text = %s, updated_at = %s, updated_by = %s
		WHERE section_tag = %s AND organization_id = %s
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3),
		r.getPlaceholder(4), r.getPlaceholder(5),
	)

	_, err := r.db.Exec(query,
		content.Text,
		content.UpdatedAt,
		content.UpdatedBy,
		content.SectionTag,
		content.OrganizationID,
	)

	return err
}

// FindByUUIDAndTagAndOrgID checks if content exists by uuid, section_tag and organization_id
func (r *ContentRepository) FindByUUIDAndTagAndOrgID(uuid, sectionTag, orgID string) (*model.Content, error) {
	query := fmt.Sprintf(`
		SELECT uuid, section_tag, text, organization_id, created_at, created_by, updated_at, updated_by
		FROM content
		WHERE uuid = %s AND section_tag = %s AND organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var content model.Content
	err := r.db.QueryRow(query, uuid, sectionTag, orgID).Scan(
		&content.UUID,
		&content.SectionTag,
		&content.Text,
		&content.OrganizationID,
		&content.CreatedAt,
		&content.CreatedBy,
		&content.UpdatedAt,
		&content.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &content, nil
}
