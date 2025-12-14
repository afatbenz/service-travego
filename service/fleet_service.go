package service

import (
	"net/http"
	"service-travego/model"
	"service-travego/repository"

	"github.com/google/uuid"
)

type FleetService struct {
	repo *repository.FleetRepository
}

func NewFleetService(repo *repository.FleetRepository) *FleetService {
	return &FleetService{repo: repo}
}

func (s *FleetService) CreateFleet(createdBy, organizationID string, req *model.CreateFleetRequest) (string, error) {
	if req.FleetName == "" || req.FleetType == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleet_name and fleet_type are required")
	}
	id := uuid.New().String()
	err := s.repo.CreateFleetWithDetails(id, createdBy, organizationID, req)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet")
	}
	return id, nil
}
