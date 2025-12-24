package model

import "time"

type Content struct {
	UUID           string    `json:"uuid"`
	SectionTag     string    `json:"section_tag"`
	Text           string    `json:"text"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

type ContentRequest struct {
	SectionTag string `json:"section_tag"`
	Text       string `json:"text"`
}

type ContentResponse struct {
	SectionTag string `json:"section_tag"`
	Text       string `json:"text"`
}
