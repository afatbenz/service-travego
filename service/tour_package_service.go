package service

import (
	"service-travego/model"
	"service-travego/repository"
)

type TourPackageService struct {
	repo *repository.TourPackageRepository
}

func NewTourPackageService(repo *repository.TourPackageRepository) *TourPackageService {
	return &TourPackageService{
		repo: repo,
	}
}

func (s *TourPackageService) GetTourPackages(orgID string) ([]model.TourPackageListItem, error) {
	return s.repo.GetTourPackagesByOrgID(orgID)
}
