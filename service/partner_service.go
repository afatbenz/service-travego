package service

import (
	"errors"
	"service-travego/model"
	"service-travego/repository"
	"sync"
)

type PartnerService struct {
	repo *repository.PartnerRepository
}

func NewPartnerService(repo *repository.PartnerRepository) *PartnerService {
	return &PartnerService{repo: repo}
}

func (s *PartnerService) List(orgID, partnerName, startDate, endDate string) ([]model.OperationPartner, error) {
	return s.repo.List(orgID, partnerName, startDate, endDate)
}

func (s *PartnerService) Create(req model.CreateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	return s.repo.Create(req, orgID, userID)
}

func (s *PartnerService) Update(req model.UpdateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	partner, err := s.repo.GetByID(req.PartnerID, orgID, nil)
	if err != nil {
		return nil, err
	}
	if partner == nil {
		return nil, errors.New("partner not found")
	}
	return s.repo.Update(req, orgID, userID)
}

func (s *PartnerService) Detail(req *model.OperationPartnerDetailRequest, orgID string) (*model.OperationPartnerDetailResponse, error) {
	if req == nil || req.PartnerID == "" {
		return nil, errors.New("partner not found")
	}

	var (
		partner       *model.OperationPartner
		totalRevenue  float64
		totalExpenses float64
		totalBooking  int64
		fleetUnits    []model.PartnerFleetUnit
	)

	var (
		partnerErr       error
		metricsErr       error
		fleetUnitsErr    error
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		partner, partnerErr = s.repo.GetByID(req.PartnerID, orgID, req)
	}()

	go func() {
		defer wg.Done()
		totalRevenue, totalExpenses, totalBooking, metricsErr = s.repo.GetDetailMetrics(req.PartnerID, orgID, req)
	}()

	go func() {
		defer wg.Done()
		units, err := s.repo.GetPartnerFleetUnits(req.PartnerID, orgID, req)
		if err == nil {
			fleetUnits = units
		} else {
			fleetUnitsErr = err
		}
	}()

	wg.Wait()

	if partnerErr != nil {
		return nil, partnerErr
	}
	if partner == nil {
		return nil, errors.New("partner not found")
	}

	if metricsErr == nil {
		partner.TotalRevenue = totalRevenue
		partner.TotalExpenses = totalExpenses
		partner.TotalBooking = totalBooking
		partner.ProfitEstimate = totalRevenue - totalExpenses
	}

	if fleetUnitsErr != nil {
		fleetUnits = []model.PartnerFleetUnit{}
	} else if fleetUnits == nil {
		fleetUnits = []model.PartnerFleetUnit{}
	}

	label := s.repo.GetCityLabel(partner.PartnerCity)

	return &model.OperationPartnerDetailResponse{
		OperationPartner: *partner,
		PartnerCityLabel: label,
		FleetUnits:       fleetUnits,
	}, nil
}
