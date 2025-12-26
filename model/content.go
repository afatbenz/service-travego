package model

import "time"

type Content struct {
	UUID           string    `json:"uuid"`
	SectionTag     string    `json:"section_tag"`
	Parent         string    `json:"parent"`
	Type           string    `json:"type"`
	IsActive       bool      `json:"is_active"`
	Content        string    `json:"content"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type ContentRequest struct {
	SectionTag string                   `json:"section_tag"`
	Parent     string                   `json:"parent"`
	Type       string                   `json:"type"`
	IsActive   *bool                    `json:"is_active"`
	Content    string                   `json:"content"`
	List       []ContentListItemRequest `json:"list"`
}

type ContentResponse struct {
	SectionTag string                    `json:"section_tag"`
	Parent     string                    `json:"parent"`
	Type       string                    `json:"type"`
	IsActive   bool                      `json:"is_active"`
	Content    string                    `json:"content"`
	List       []ContentListItemResponse `json:"list"`
}

type ContentListItem struct {
	UUID      string    `json:"uuid"`
	ContentID string    `json:"content_id"`
	Label     string    `json:"label"`
	Icon      string    `json:"icon"`
	SubLabel  string    `json:"sub_label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContentListItemRequest struct {
	Icon     string `json:"icon"`
	Label    string `json:"label"`
	SubLabel string `json:"sub_label"`
	ListID   string `json:"list_id"`
}

type ContentListItemResponse struct {
	UUID     string `json:"uuid"`
	Icon     string `json:"icon"`
	Label    string `json:"label"`
	SubLabel string `json:"sub_label"`
}
