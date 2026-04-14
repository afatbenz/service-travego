package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
)

type FleetUnitService struct {
	repo              *repository.FleetUnitRepository
	citiesName        map[string]string
	transmissionLabel map[string]string
}

func NewFleetUnitService(repo *repository.FleetUnitRepository) *FleetUnitService {
	return &FleetUnitService{repo: repo}
}

func (s *FleetUnitService) ensureCitiesLoaded() {
	if s.citiesName != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		s.citiesName = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		s.citiesName = map[string]string{}
		return
	}
	m := make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		m[c.ID] = c.Name
	}
	s.citiesName = m
}

func (s *FleetUnitService) ensureTransmissionLoaded() {
	if s.transmissionLabel != nil {
		return
	}
	f, err := os.Open("config/fleet-config.json")
	if err != nil {
		s.transmissionLabel = map[string]string{}
		return
	}
	defer f.Close()
	var cfg model.FleetConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		s.transmissionLabel = map[string]string{}
		return
	}
	m := make(map[string]string, len(cfg.FleetTransmission))
	for _, it := range cfg.FleetTransmission {
		if it.ID != "" && it.Label != "" {
			m[it.ID] = it.Label
		}
	}
	s.transmissionLabel = m
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

	s.ensureTransmissionLoaded()
	res.TransmissionID = res.Transmission
	if label, ok := s.transmissionLabel[strings.TrimSpace(res.Transmission)]; ok && label != "" {
		res.Transmission = label
	}

	if strings.TrimSpace(res.FleetID) != "" {
		cityIDs, err := s.repo.GetFleetPickupCityIDs(orgID, res.FleetID)
		if err == nil && len(cityIDs) > 0 {
			s.ensureCitiesLoaded()
			out := make([]string, 0, len(cityIDs))
			seen := map[string]struct{}{}
			for _, id := range cityIDs {
				key := strconv.Itoa(id)
				name := s.citiesName[key]
				if name == "" {
					continue
				}
				if _, ok := seen[name]; ok {
					continue
				}
				seen[name] = struct{}{}
				out = append(out, name)
			}
			res.PickupPoint = out
		}
	}

	return res, nil
}
