package service

import (
	"encoding/json"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/model"
	"service-travego/repository"
	"strconv"

	"github.com/google/uuid"
)

type FleetService struct {
	repo       *repository.FleetRepository
	citiesName map[string]string
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

func (s *FleetService) GetServiceFleets() ([]model.ServiceFleetItem, error) {
	items, err := s.repo.GetServiceFleets()
	if err != nil {
		return nil, err
	}

	for i := range items {
		item := &items[i]
		item.Price = item.OriginalPrice // Default

		if item.DiscountType != nil && item.DiscountValue != nil {
			switch *item.DiscountType {
			case "PERCENT":
				// assuming discount_value is percentage e.g. 10 for 10%
				item.Price = item.OriginalPrice - (item.OriginalPrice * *item.DiscountValue / 100)
			case "AMOUNT":
				item.Price = item.OriginalPrice - *item.DiscountValue
			case "FLAT":
				item.Price = *item.DiscountValue
			}
		}
	}
	return items, nil
}

func (s *FleetService) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	items, err := s.repo.ListFleets(req)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to list fleets")
	}
	return items, nil
}

func (s *FleetService) GetFleetDetail(orgID, fleetID string) (*model.FleetDetailResponse, error) {
	meta, err := s.repo.GetFleetDetailMeta(orgID, fleetID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
	}
	fac, err := s.repo.GetFleetFacilities(fleetID)
	if err != nil {
		fac = []string{}
	}
	pickup, err := s.repo.GetFleetPickup(orgID, fleetID)
	if err != nil {
		pickup = []model.FleetPickupItem{}
	}
	addon, err := s.repo.GetFleetAddon(orgID, fleetID)
	if err != nil {
		addon = []model.FleetAddonItem{}
	}
	prices, err := s.repo.GetFleetPrices(orgID, fleetID)
	if err != nil {
		prices = []model.FleetPriceItem{}
	}
	images, err := s.repo.GetFleetImages(fleetID)
	if err != nil {
		images = []model.FleetImageItem{}
	}

	s.ensureCitiesLoaded()

	for i := range pickup {
		key := intToString(pickup[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			pickup[i].CityName = name
		} else {
			pickup[i].CityName = ""
		}
	}

	for i := range prices {
		prices[i].RentTypeLabel = configs.RentType(prices[i].RentType).String()
	}

	resp := &model.FleetDetailResponse{
		Meta:       *meta,
		Facilities: fac,
		Pickup:     pickup,
		Addon:      addon,
		Pricing:    prices,
		Images:     images,
	}
	return resp, nil
}

func (s *FleetService) ensureCitiesLoaded() {
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

func intToString(n int) string { return strconv.Itoa(n) }
