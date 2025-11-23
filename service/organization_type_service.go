package service

import (
	"service-travego/model"
	"service-travego/repository"
)

type OrganizationTypeService struct {
	orgTypeRepo *repository.OrganizationTypeRepository
}

func NewOrganizationTypeService(orgTypeRepo *repository.OrganizationTypeRepository) *OrganizationTypeService {
	return &OrganizationTypeService{
		orgTypeRepo: orgTypeRepo,
	}
}

// GetAllOrganizationTypes retrieves all organization types ordered by name ascending
func (s *OrganizationTypeService) GetAllOrganizationTypes() ([]model.OrganizationType, error) {
	return s.orgTypeRepo.FindAll()
}
