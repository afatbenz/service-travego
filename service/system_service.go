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

func (s *SystemService) GetDeviceList(status string) ([]model.DeviceListItem, error) {
	return s.repo.GetDeviceList(status)
}

func (s *SystemService) UpdateDevice(account string, action string, enableData *model.DeviceEnableRequest) error {
	return s.repo.UpdateDevice(account, action, enableData)
}
