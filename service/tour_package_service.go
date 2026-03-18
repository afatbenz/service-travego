package service

import (
	"context"
	"encoding/json"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
)

type TourPackageService struct {
	repo       *repository.TourPackageRepository
	citiesName map[string]string
}

func NewTourPackageService(repo *repository.TourPackageRepository) *TourPackageService {
	return &TourPackageService{
		repo: repo,
	}
}

func (s *TourPackageService) GetTourPackages(orgID string) ([]model.TourPackageListItem, error) {
	return s.repo.GetTourPackagesByOrgID(orgID)
}

func (s *TourPackageService) CreateTourPackage(ctx context.Context, req *model.CreateTourPackageRequest, orgID, userID string) error {
	packageID := helper.GenerateUUID()
	return s.repo.CreateTourPackage(ctx, req, packageID, orgID, userID)
}

func (s *TourPackageService) UpdateTourPackage(ctx context.Context, req *model.UpdateTourPackageRequest, orgID, userID string) error {
	return s.repo.UpdateTourPackage(ctx, req, orgID, userID)
}

func (s *TourPackageService) GetTourPackageDetail(ctx context.Context, orgID, packageID string) (*model.TourPackageDetailResponse, error) {
	res, err := s.repo.GetTourPackageDetail(ctx, orgID, packageID)
	if err != nil {
		return nil, err
	}

	switch res.Meta.PackageType {
	case 1:
		res.Meta.PackageTypeLabel = "Private Trip"
	case 2:
		res.Meta.PackageTypeLabel = "Open Trip"
	default:
		res.Meta.PackageTypeLabel = "Unknown"
	}

	s.ensureCitiesLoaded()
	for i := range res.PickupAreas {
		key := strconv.Itoa(res.PickupAreas[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.PickupAreas[i].CityName = name
		}
	}
	for i := range res.Itineraries {
		key := strconv.Itoa(res.Itineraries[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.Itineraries[i].CityName = name
		}
	}

	return res, nil
}

func (s *TourPackageService) ensureCitiesLoaded() {
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
