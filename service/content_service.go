package service

import (
	"errors"
	"service-travego/model"
	"service-travego/repository"
	"time"

	"github.com/google/uuid"
)

type ContentService struct {
	repo *repository.ContentRepository
}

func NewContentService(repo *repository.ContentRepository) *ContentService {
	return &ContentService{
		repo: repo,
	}
}

// UpsertGeneralContent handles insert or update of general content
func (s *ContentService) UpsertGeneralContent(req model.ContentRequest, orgID, userID string) error {
	if req.SectionTag == "" {
		return errors.New("section_tag is required")
	}

	// Check if content exists
	existingContent, err := s.repo.FindByTagAndOrgID(req.SectionTag, orgID)
	if err != nil {
		return err
	}

	now := time.Now()

	if existingContent != nil {
		// Update
		existingContent.Text = req.Text
		existingContent.UpdatedAt = now
		existingContent.UpdatedBy = userID
		return s.repo.Update(existingContent)
	}

	// Create
	newContent := &model.Content{
		UUID:           uuid.New().String(),
		SectionTag:     req.SectionTag,
		Text:           req.Text,
		OrganizationID: orgID,
		CreatedAt:      now,
		CreatedBy:      userID,
		UpdatedAt:      now,
		UpdatedBy:      userID,
	}

	return s.repo.Create(newContent)
}

// GetGeneralContent retrieves content by section_tag and organization_id
func (s *ContentService) GetGeneralContent(sectionTag, orgID string) (*model.ContentResponse, error) {
	content, err := s.repo.FindByTagAndOrgID(sectionTag, orgID)
	if err != nil {
		return nil, err
	}

	if content == nil {
		return nil, nil
	}

	return &model.ContentResponse{
		SectionTag: content.SectionTag,
		Text:       content.Text,
	}, nil
}
