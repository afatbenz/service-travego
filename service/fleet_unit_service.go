package service

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strings"
)

type FleetUnitService struct {
	repo *repository.FleetUnitRepository
}

func NewFleetUnitService(repo *repository.FleetUnitRepository) *FleetUnitService {
	return &FleetUnitService{repo: repo}
}

func (s *FleetUnitService) List(orgID string) ([]model.FleetUnitListItem, error) {
	items, err := s.repo.List(orgID)
	if err != nil {
		msg := "failed to get fleet units"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return items, nil
}

func (s *FleetUnitService) Create(orgID, userID string, req *model.FleetUnitCreateRequest) (string, error) {
	req.OrganizationID = orgID
	req.CreatedBy = userID
	id, err := s.repo.Create(req)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
	}
	return id, nil
}

func (s *FleetUnitService) Update(orgID, userID string, req *model.FleetUnitUpdateRequest) error {
	req.OrganizationID = orgID
	req.UpdatedBy = userID
	if err := s.repo.Update(req); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "fleet unit not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update fleet unit")
	}
	return nil
}

func (s *FleetUnitService) Detail(orgID, uuid string) (*model.FleetUnitDetailResponse, error) {
	res, err := s.repo.Detail(orgID, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet unit not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get fleet unit detail")
	}
	return res, nil
}
