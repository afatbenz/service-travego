package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strings"
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
	if req.Type == "" {
		return errors.New("type is required")
	}

	// Check if content exists
	existingContent, err := s.repo.FindByTagParentAndOrgID(req.SectionTag, req.Parent, orgID)
	if err != nil {
		return err
	}

	now := time.Now()
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if existingContent != nil {
		existingContent.Content = req.Content
		existingContent.Type = req.Type
		existingContent.IsActive = isActive
		existingContent.UpdatedAt = now
		existingContent.UpdatedBy = userID
		if err := s.repo.Update(existingContent); err != nil {
			return err
		}
		if req.Type == "list" && len(req.List) > 0 {
			var toInsert []model.ContentListItem
			for _, li := range req.List {
				if li.Label == "" {
					return errors.New("list.label is required")
				}
				if li.ListID != "" {
					if err := s.repo.UpdateContentListItemByUUID(li.ListID, li.Label, li.Icon, li.SubLabel, now); err != nil {
						return err
					}
					continue
				}
				toInsert = append(toInsert, model.ContentListItem{
					UUID:      uuid.New().String(),
					ContentID: existingContent.UUID,
					Label:     li.Label,
					Icon:      li.Icon,
					SubLabel:  li.SubLabel,
					CreatedAt: now,
					UpdatedAt: now,
				})
			}
			if len(toInsert) > 0 {
				if err := s.repo.InsertContentListItems(toInsert); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Create
	newContent := &model.Content{
		UUID:           uuid.New().String(),
		SectionTag:     req.SectionTag,
		Parent:         req.Parent,
		Type:           req.Type,
		IsActive:       isActive,
		Content:        req.Content,
		OrganizationID: orgID,
		CreatedAt:      now,
		CreatedBy:      userID,
		UpdatedAt:      now,
		UpdatedBy:      userID,
	}
	if err := s.repo.Create(newContent); err != nil {
		return err
	}
	if req.Type == "list" && len(req.List) > 0 {
		var items []model.ContentListItem
		for _, li := range req.List {
			if li.Label == "" {
				return errors.New("list.label is required")
			}
			items = append(items, model.ContentListItem{
				UUID:      uuid.New().String(),
				ContentID: newContent.UUID,
				Label:     li.Label,
				Icon:      li.Icon,
				SubLabel:  li.SubLabel,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := s.repo.InsertContentListItems(items); err != nil {
			return err
		}
	}
	return nil
}

// GetGeneralContent retrieves content by section_tag and organization_id
func (s *ContentService) GetGeneralContent(sectionTag, orgID string) (*model.ContentResponse, error) {
	// For backward compatibility or specific logic, parent might be empty or handled differently
	// Assuming empty parent for now if not provided in signature.
	// To fully support parent in Get, we'd need to update the signature or method.
	// Based on user request "tambahkan payload parent di api /content/create",
	// we focused on Upsert.
	// However, FindByTagAndOrgID now requires parent.
	// Let's assume for GetGeneralContent we might need to adjust or create a new method if parent is needed.
	// For now, let's pass empty string as parent to satisfy the repo method signature change,
	// OR if the user intends to fetch by parent as well, we should update this method signature.
	// Given the prompt "tambahkan payload parent di api /content/create", we focus on write path.
	// But since we changed the repo signature, we must update this call.
	// Assuming empty parent for legacy/simple calls or we need to update the handler too.
	// Let's pass "" for now, but ideally we should update the Get endpoint too if needed.

	// WAIT: The previous GetGeneralContent didn't have parent.
	// If the repo now filters by parent, passing "" will only find rows with empty parent.
	// This might break existing functionality if rows have NULL parent (which we handle as string/empty?).
	// Let's assume we pass "" for parent.

	content, err := s.repo.FindByTagAndOrgID(sectionTag, orgID)
	if err != nil {
		return nil, err
	}

	if content == nil {
		return nil, nil
	}

	return &model.ContentResponse{
		SectionTag: content.SectionTag,
		Parent:     content.Parent,
		Type:       content.Type,
		IsActive:   content.IsActive,
		Content:    content.Content,
	}, nil
}

// GetContentByParent retrieves content by parent and organization_id
func (s *ContentService) GetContentByParent(parent, orgID string) ([]model.ContentResponse, error) {
	contents, err := s.repo.FindByParentAndOrgID(parent, orgID)
	if err != nil {
		return nil, err
	}

	var response []model.ContentResponse
	for _, c := range contents {
		item := model.ContentResponse{
			SectionTag: c.SectionTag,
			Parent:     c.Parent,
			Type:       c.Type,
			IsActive:   c.IsActive,
			Content:    c.Content,
		}
		if c.Type == "list" {
			listItems, err := s.repo.FindContentListByContentID(c.UUID)
			if err != nil {
				return nil, err
			}
			var listRes []model.ContentListItemResponse
			for _, li := range listItems {
				listRes = append(listRes, model.ContentListItemResponse{UUID: li.UUID, Icon: li.Icon, Label: li.Label, SubLabel: li.SubLabel})
			}
			item.List = listRes
		}
		response = append(response, item)
	}
	return response, nil
}

func (s *ContentService) GetContentDetail(parent, sectionTag, orgID string) (*model.ContentResponse, error) {
	content, err := s.repo.FindByTagParentAndOrgID(sectionTag, parent, orgID)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}
	res := &model.ContentResponse{
		SectionTag: content.SectionTag,
		Parent:     content.Parent,
		Type:       content.Type,
		IsActive:   content.IsActive,
		Content:    content.Content,
	}
	if content.Type == "list" {
		listItems, err := s.repo.FindContentListByContentID(content.UUID)
		if err != nil {
			return nil, err
		}
		var listRes []model.ContentListItemResponse
		for _, li := range listItems {
			listRes = append(listRes, model.ContentListItemResponse{UUID: li.UUID, Icon: li.Icon, Label: li.Label, SubLabel: li.SubLabel})
		}
		res.List = listRes
	}
	return res, nil
}

func (s *ContentService) UploadContent(fileHeader *multipart.FileHeader, parent, sectionTag, orgID, userID string) (string, error) {
	// 1. Check existing content
	existingContent, err := s.repo.FindByTagParentAndOrgID(sectionTag, parent, orgID)
	if err != nil {
		return "", err
	}

	// 2. Prepare destination
	uploadDir := "assets/logo"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	// 3. Generate filename: orgID[:5] + "_" + timestamp + ext
	orgPrefix := orgID
	if len(orgID) > 5 {
		orgPrefix = orgID[:5]
	}
	ext := filepath.Ext(fileHeader.Filename)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d%s", orgPrefix, timestamp, ext)
	destPath := filepath.Join(uploadDir, filename)

	// 4. Save file
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	// 5. Delete old file if exists
	if existingContent != nil {
		oldPath := existingContent.Content
		appHost := os.Getenv("APP_HOST")
		stripped := oldPath
		if appHost != "" && strings.HasPrefix(stripped, appHost) {
			stripped = strings.TrimPrefix(stripped, appHost)
		}
		stripped = strings.TrimPrefix(stripped, "/")
		stripped = filepath.FromSlash(stripped)

		if (strings.HasPrefix(stripped, "assets/logo") || strings.HasPrefix(stripped, "assets\\logo")) && stripped != destPath {
			os.Remove(stripped)
		}
	}

	// 6. Upsert Content
	pathForDB := "/" + filepath.ToSlash(destPath)
	fullURL := helper.GetAssetURL(pathForDB)

	now := time.Now()
	if existingContent != nil {
		existingContent.Content = fullURL
		existingContent.UpdatedAt = now
		existingContent.UpdatedBy = userID
		existingContent.Type = "image"
		if err := s.repo.Update(existingContent); err != nil {
			return "", err
		}
		return fullURL, nil
	}

	newContent := &model.Content{
		UUID:           uuid.New().String(),
		SectionTag:     sectionTag,
		Parent:         parent,
		Type:           "image",
		IsActive:       true,
		Content:        fullURL,
		OrganizationID: orgID,
		CreatedAt:      now,
		CreatedBy:      userID,
		UpdatedAt:      now,
		UpdatedBy:      userID,
	}
	if err := s.repo.Create(newContent); err != nil {
		return "", err
	}
	return fullURL, nil
}

// GetAllGeneralContent retrieves all content for an organization grouped by parent
func (s *ContentService) GetAllGeneralContent(orgID string) (map[string][]model.ContentResponse, error) {
	contents, err := s.repo.FindAllByOrgID(orgID)
	if err != nil {
		return nil, err
	}

	response := make(map[string][]model.ContentResponse)
	for _, c := range contents {
		item := model.ContentResponse{
			SectionTag: c.SectionTag,
			Parent:     c.Parent,
			Type:       c.Type,
			IsActive:   c.IsActive,
			Content:    c.Content,
		}

		if c.Type == "list" {
			listItems, err := s.repo.FindContentListByContentID(c.UUID)
			if err != nil {
				return nil, err
			}
			var listRes []model.ContentListItemResponse
			for _, li := range listItems {
				listRes = append(listRes, model.ContentListItemResponse{UUID: li.UUID, Icon: li.Icon, Label: li.Label, SubLabel: li.SubLabel})
			}
			item.List = listRes
		}

		parentKey := c.Parent
		if parentKey == "" {
			parentKey = "ungrouped"
		}

		response[parentKey] = append(response[parentKey], item)
	}
	return response, nil
}

func (s *ContentService) DeleteContentByUUID(uuid, orgID string) error {
	return s.repo.DeleteContentByUUID(uuid, orgID)
}
