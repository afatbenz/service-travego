package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"service-travego/model"
	"service-travego/repository"
)

type GarageService struct {
	garageRepo *repository.GarageRepository
	cityMap    map[string]string
}

func NewGarageService(garageRepo *repository.GarageRepository) *GarageService {
	return &GarageService{
		garageRepo: garageRepo,
	}
}

func (s *GarageService) ensureLocationsLoaded() {
	if s.cityMap != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		s.cityMap = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		s.cityMap = map[string]string{}
		return
	}
	s.cityMap = make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		s.cityMap[c.ID] = c.Name
	}
}

func (s *GarageService) GetGarages(organizationID, itemID string) ([]model.GarageWithLabel, error) {
	garages, err := s.garageRepo.GetAll(organizationID, itemID)
	if err != nil {
		return nil, err
	}

	s.ensureLocationsLoaded()

	result := make([]model.GarageWithLabel, len(garages))
	for i, g := range garages {
		label := g.GarageCity
		if name, ok := s.cityMap[g.GarageCity]; ok {
			label = name
		}
		result[i] = model.GarageWithLabel{
			GarageID:        g.GarageID,
			OrganizationID:  g.OrganizationID,
			GarageName:      g.GarageName,
			GarageAddress:   g.GarageAddress,
			GarageCity:      g.GarageCity,
			GarageCityLabel: label,
			CreatedAt:       g.CreatedAt,
			CreatedBy:       g.CreatedBy,
			UpdatedAt:       g.UpdatedAt,
			UpdatedBy:       g.UpdatedBy,
		}
	}

	return result, nil
}

func (s *GarageService) CreateGarage(organizationID, createdBy string, req *model.CreateGarageRequest) (*model.Garage, error) {
	if req.GarageName == "" {
		return nil, errors.New("garage_name is required")
	}
	if req.GarageCity == "" {
		return nil, errors.New("garage_city is required")
	}

	garage := &model.Garage{
		OrganizationID: organizationID,
		GarageName:     req.GarageName,
		GarageAddress:  req.GarageAddress,
		GarageCity:     req.GarageCity,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}

	if err := s.garageRepo.Create(garage); err != nil {
		return nil, err
	}

	return garage, nil
}

func (s *GarageService) UpdateGarage(garageID, organizationID, updatedBy string, req *model.UpdateGarageRequest) (*model.Garage, error) {
	if req.GarageName == "" {
		return nil, errors.New("garage_name is required")
	}
	if req.GarageCity == "" {
		return nil, errors.New("garage_city is required")
	}

	existing, err := s.garageRepo.GetByID(garageID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("garage not found")
		}
		return nil, err
	}

	updates := map[string]interface{}{
		"garage_name":    req.GarageName,
		"garage_address": req.GarageAddress,
		"garage_city":    req.GarageCity,
		"updated_by":     updatedBy,
	}

	if err := s.garageRepo.Update(garageID, organizationID, updates); err != nil {
		return nil, err
	}

	existing.GarageName = req.GarageName
	existing.GarageAddress = req.GarageAddress
	existing.GarageCity = req.GarageCity
	existing.UpdatedBy = updatedBy

	return existing, nil
}

func (s *GarageService) DeleteGarage(garageID, organizationID string) error {
	return s.garageRepo.Delete(garageID, organizationID)
}
