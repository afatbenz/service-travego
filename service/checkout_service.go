package service

import (
	"net/http"
	"service-travego/model"
	"service-travego/repository"
)

type CheckoutService struct {
	fleetRepo *repository.FleetRepository
}

func NewCheckoutService(fleetRepo *repository.FleetRepository) *CheckoutService {
	return &CheckoutService{
		fleetRepo: fleetRepo,
	}
}

func (s *CheckoutService) GetFleetCheckoutSummary(req *model.CheckoutFleetSummaryRequest) (*model.CheckoutFleetSummaryResponse, error) {
	res, err := s.fleetRepo.GetFleetCheckoutSummary(req.FleetID, req.PriceID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet or price not found")
	}
	return res, nil
}
