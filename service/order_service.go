package service

import (
	"encoding/json"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
)

type OrderService struct {
	fleetRepo  *repository.FleetRepository
	citiesName map[string]string
}

func NewOrderService(fleetRepo *repository.FleetRepository) *OrderService {
	return &OrderService{
		fleetRepo: fleetRepo,
	}
}

func (s *OrderService) GetFleetOrderSummary(req *model.OrderFleetSummaryRequest) (*model.OrderFleetSummaryResponse, error) {
	res, err := s.fleetRepo.GetFleetOrderSummary(req.FleetID, req.PriceID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet or price not found")
	}

	s.ensureCitiesLoaded()
	for i := range res.PickupPoints {
		key := strconv.Itoa(res.PickupPoints[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.PickupPoints[i].CityName = name
		}
	}

	return res, nil
}

func (s *OrderService) ensureCitiesLoaded() {
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
