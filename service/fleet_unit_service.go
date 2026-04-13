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

	vehicleID := strings.ToUpper(strings.TrimSpace(req.VehicleID))
	plateNumber := strings.ToUpper(strings.TrimSpace(req.PlateNumber))
	if vehicleID != "" {
		existing, err := s.repo.FindExistingVehicleIDs(orgID, []string{vehicleID})
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
		}
		if _, ok := existing[vehicleID]; ok {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
		}
	}
	if plateNumber != "" {
		existing, err := s.repo.FindExistingPlateNumbers(orgID, []string{plateNumber})
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
		}
		if _, ok := existing[plateNumber]; ok {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
		}
	}

	id, err := s.repo.Create(req)
	if err != nil {
		msg := "failed to create fleet unit"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return id, nil
}

func (s *FleetUnitService) CreateBatch(orgID, userID, fleetID string, units []model.FleetUnitCreateUnit) ([]string, error) {
	seenVehicle := map[string]struct{}{}
	seenPlate := map[string]struct{}{}

	vehicleIDs := make([]string, 0, len(units))
	plateNumbers := make([]string, 0, len(units))
	for _, u := range units {
		vid := strings.ToUpper(strings.TrimSpace(u.VehicleID))
		pn := strings.ToUpper(strings.TrimSpace(u.PlateNumber))

		if vid != "" {
			if _, ok := seenVehicle[vid]; ok {
				return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
			}
			seenVehicle[vid] = struct{}{}
			vehicleIDs = append(vehicleIDs, vid)
		}
		if pn != "" {
			if _, ok := seenPlate[pn]; ok {
				return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
			}
			seenPlate[pn] = struct{}{}
			plateNumbers = append(plateNumbers, pn)
		}
	}

	existingVehicles, err := s.repo.FindExistingVehicleIDs(orgID, vehicleIDs)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
	}
	if len(existingVehicles) > 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
	}

	existingPlates, err := s.repo.FindExistingPlateNumbers(orgID, plateNumbers)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
	}
	if len(existingPlates) > 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
	}

	ids := make([]string, 0, len(units))
	for _, u := range units {
		req := &model.FleetUnitCreateRequest{
			VehicleID:      u.VehicleID,
			PlateNumber:    u.PlateNumber,
			FleetID:        fleetID,
			Engine:         u.Engine,
			Transmission:   u.Transmission,
			Capacity:       u.Capacity,
			ProductionYear: u.ProductionYear,
		}
		id, err := s.Create(orgID, userID, req)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
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
