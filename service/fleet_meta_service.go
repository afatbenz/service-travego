package service

import "service-travego/repository"

type FleetMetaService struct {
    repo *repository.FleetMetaRepository
}

func NewFleetMetaService(repo *repository.FleetMetaRepository) *FleetMetaService {
    return &FleetMetaService{repo: repo}
}

func (s *FleetMetaService) GetBodies(orgID string, search string) ([]string, error) {
    return s.repo.FindBodies(orgID, search)
}

func (s *FleetMetaService) GetEngines(orgID string, search string) ([]string, error) {
    return s.repo.FindEngines(orgID, search)
}
