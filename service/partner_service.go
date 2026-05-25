package service

import (
	"errors"
	"service-travego/model"
	"service-travego/repository"
)

type PartnerService struct {
	repo *repository.PartnerRepository
}

func NewPartnerService(repo *repository.PartnerRepository) *PartnerService {
	return &PartnerService{repo: repo}
}

func (s *PartnerService) List(orgID, partnerName string) ([]model.OperationPartner, error) {
	return s.repo.List(orgID, partnerName)
}

func (s *PartnerService) Create(req model.CreateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	return s.repo.Create(req, orgID, userID)
}

func (s *PartnerService) Update(req model.UpdateOperationPartnerRequest, orgID, userID string) (*model.OperationPartner, error) {
	partner, err := s.repo.GetByID(req.PartnerID, orgID)
	if err != nil {
		return nil, err
	}
	if partner == nil {
		return nil, errors.New("partner not found")
	}
	return s.repo.Update(req, orgID, userID)
}

func (s *PartnerService) Detail(partnerID, orgID string) (*model.OperationPartnerDetailResponse, error) {
	partner, err := s.repo.GetByID(partnerID, orgID)
	if err != nil {
		return nil, err
	}
	if partner == nil {
		return nil, errors.New("partner not found")
	}

	label := s.repo.GetCityLabel(partner.PartnerCity)

	return &model.OperationPartnerDetailResponse{
		OperationPartner: *partner,
		PartnerCityLabel: label,
	}, nil
}
