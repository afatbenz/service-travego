package service

import (
	"service-travego/model"
	"service-travego/repository"
)

type DashboardService struct {
	repo *repository.DashboardRepository
}

func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{
		repo: repo,
	}
}

func (s *DashboardService) GetPartnerSummary(orgID string) (*model.DashboardPartnerSummaryResponse, error) {
	return s.repo.GetPartnerSummary(orgID)
}
