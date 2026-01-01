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
        SELECT uuid, section_tag, parent, type, is_active, content, organization_id, created_at, created_by, updated_at, updated_by
        FROM content
        WHERE section_tag = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	var content model.Content
	err := r.db.QueryRow(query, sectionTag, orgID).Scan(
		&content.UUID,
		&content.SectionTag,
		&content.Parent,
		&content.Type,
		&content.IsActive,
		&content.Content,
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
func (r *ContentRepository) FindByTagParentAndOrgID(sectionTag, parent, orgID string) (*model.Content, error) {
	query := fmt.Sprintf(`
        SELECT uuid, section_tag, parent, type, is_active, content, organization_id, created_at, created_by, updated_at, updated_by
        FROM content
        WHERE section_tag = %s AND parent = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var content model.Content
	err := r.db.QueryRow(query, sectionTag, parent, orgID).Scan(
		&content.UUID,
		&content.SectionTag,
		&content.Parent,
		&content.Type,
		&content.IsActive,
		&content.Content,
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
        INSERT INTO content (uuid, section_tag, parent, type, is_active, content, organization_id, created_at, created_by, updated_at, updated_by)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11),
	)

	_, err := r.db.Exec(query,
		content.UUID,
		content.SectionTag,
		content.Parent,
		content.Type,
		content.IsActive,
		content.Content,
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
        SET content = %s, type = %s, is_active = %s, updated_at = %s, updated_by = %s
        WHERE section_tag = %s AND parent = %s AND organization_id = %s
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
		r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
	)

	_, err := r.db.Exec(query,
		content.Content,
		content.Type,
		content.IsActive,
		content.UpdatedAt,
		content.UpdatedBy,
		content.SectionTag,
		content.Parent,
		content.OrganizationID,
	)

	return err
}

// FindByParentAndOrgID retrieves content by parent and organization
func (r *ContentRepository) FindByParentAndOrgID(parent, orgID string) ([]model.Content, error) {
	query := fmt.Sprintf(`
        SELECT uuid, section_tag, parent, type, is_active, content
        FROM content
        WHERE parent = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))

	rows, err := r.db.Query(query, parent, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []model.Content
	for rows.Next() {
		var c model.Content
		if err := rows.Scan(&c.UUID, &c.SectionTag, &c.Parent, &c.Type, &c.IsActive, &c.Content); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}

	return contents, nil
}

// FindAllByOrgID retrieves all content for an organization
func (r *ContentRepository) FindAllByOrgID(orgID string) ([]model.Content, error) {
	query := fmt.Sprintf(`
        SELECT uuid, section_tag, parent, type, is_active, content
        FROM content
        WHERE organization_id = %s
    `, r.getPlaceholder(1))

	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []model.Content
	for rows.Next() {
		var c model.Content
		var typeNull sql.NullString
		var isActiveNull sql.NullBool
		if err := rows.Scan(&c.UUID, &c.SectionTag, &c.Parent, &typeNull, &isActiveNull, &c.Content); err != nil {
			return nil, err
		}
		c.Type = typeNull.String
		if isActiveNull.Valid {
			c.IsActive = isActiveNull.Bool
		} else {
			c.IsActive = true
		}
		contents = append(contents, c)
	}

	return contents, nil
}

// FindByUUIDAndTagAndOrgID checks if content exists by uuid, section_tag and organization_id
func (r *ContentRepository) FindByUUIDAndTagAndOrgID(uuid, sectionTag, orgID string) (*model.Content, error) {
	query := fmt.Sprintf(`
        SELECT uuid, section_tag, parent, type, is_active, content, organization_id, created_at, created_by, updated_at, updated_by
        FROM content
        WHERE uuid = %s AND section_tag = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	var content model.Content
	err := r.db.QueryRow(query, uuid, sectionTag, orgID).Scan(
		&content.UUID,
		&content.SectionTag,
		&content.Parent,
		&content.Type,
		&content.IsActive,
		&content.Content,
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

func (r *ContentRepository) InsertContentListItems(items []model.ContentListItem) error {
	if len(items) == 0 {
		return nil
	}
	for _, it := range items {
		query := fmt.Sprintf(`
            INSERT INTO content_list (uuid, content_id, label, icon, sub_label, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7),
		)
		_, err := r.db.Exec(query, it.UUID, it.ContentID, it.Label, it.Icon, it.SubLabel, it.CreatedAt, it.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ContentRepository) UpdateContentListItemByUUID(uuid, label, icon, subLabel string, updatedAt interface{}) error {
	query := fmt.Sprintf(`
        UPDATE content_list
        SET label = %s, icon = %s, sub_label = %s, updated_at = %s
        WHERE uuid = %s
    `,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
	)
	_, err := r.db.Exec(query, label, icon, subLabel, updatedAt, uuid)
	return err
}

func (r *ContentRepository) FindContentListByContentID(contentID string) ([]model.ContentListItem, error) {
	query := fmt.Sprintf(`
        SELECT uuid, content_id, label, icon, sub_label, created_at, updated_at
        FROM content_list
        WHERE content_id = %s
        ORDER BY created_at ASC
    `, r.getPlaceholder(1))
	rows, err := r.db.Query(query, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.ContentListItem
	for rows.Next() {
		var it model.ContentListItem
		var iconNull sql.NullString
		var subLabelNull sql.NullString
		if err := rows.Scan(&it.UUID, &it.ContentID, &it.Label, &iconNull, &subLabelNull, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Icon = iconNull.String
		it.SubLabel = subLabelNull.String
		items = append(items, it)
	}
	return items, nil
}

func (r *ContentRepository) GetContentListByTag(sectionTag, orgID string) ([]model.ContentListItem, error) {
	content, err := r.FindByTagAndOrgID(sectionTag, orgID)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}
	return r.FindContentListByContentID(content.UUID)
}

func (r *ContentRepository) DeleteContentByUUID(uuid, orgID string) error {
	query := fmt.Sprintf(`
        DELETE FROM content
        WHERE uuid = %s AND organization_id = %s
    `, r.getPlaceholder(1), r.getPlaceholder(2))
	result, err := r.db.Exec(query, uuid, orgID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
