package service

import (
	"service-travego/model"
	"service-travego/repository"
)

type SystemService struct {
	repo *repository.SystemRepository
}

func NewSystemService(repo *repository.SystemRepository) *SystemService {
	return &SystemService{
		repo: repo,
	}
}

func (s *SystemService) GetSystemSummarize(period string) (*model.SystemSummarymarizeResponse, error) {
	return s.repo.GetSummarize(period)
}
