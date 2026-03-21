package service

import (
	"service-travego/model"
	"service-travego/repository"
)

type CustomersService struct {
	repo *repository.CustomersRepository
}

func NewCustomersService(repo *repository.CustomersRepository) *CustomersService {
	return &CustomersService{repo: repo}
}

func (s *CustomersService) ListCustomers(orgID, customerName string) ([]model.CustomerListItem, error) {
	return s.repo.ListCustomers(orgID, customerName)
}

