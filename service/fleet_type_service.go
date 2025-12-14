package service

import "service-travego/repository"

type FleetTypeService struct {
    repo *repository.FleetTypeRepository
}

func NewFleetTypeService(repo *repository.FleetTypeRepository) *FleetTypeService {
    return &FleetTypeService{repo: repo}
}

func (s *FleetTypeService) GetAllFleetTypes() ([]map[string]interface{}, error) {
    return s.repo.FindAll()
}
