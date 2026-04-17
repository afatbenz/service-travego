package service

import (
	"database/sql"
	"encoding/json"
	"os"
	"service-travego/model"
	"strconv"
	"strings"
)

func (s *OrganizationService) ensureContractTypesLoaded() {
	if s.contractTypeLabels != nil {
		return
	}
	f, err := os.Open("config/common.json")
	if err != nil {
		s.contractTypeLabels = map[int]string{}
		return
	}
	defer f.Close()

	var cfg model.CommonConfig
	d := json.NewDecoder(f)
	if err := d.Decode(&cfg); err != nil {
		s.contractTypeLabels = map[int]string{}
		return
	}

	out := make(map[int]string, len(cfg.ContractType))
	for _, it := range cfg.ContractType {
		out[it.ID] = it.Label
	}
	s.contractTypeLabels = out
}

func (s *OrganizationService) EmployeeAll(organizationID, divisionName string) ([]model.EmployeeListItem, error) {
	items, err := s.orgRepo.ListEmployees(organizationID, divisionName)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employees")
	}
	s.ensureLocationsLoaded()
	s.ensureContractTypesLoaded()
	for i := range items {
		items[i].AddressCityName = s.citiesName[strconv.Itoa(items[i].AddressCity)]
		if items[i].ContractStatus != nil {
			items[i].ContractStatusLabel = s.contractTypeLabels[*items[i].ContractStatus]
		}
	}
	return items, nil
}

func (s *OrganizationService) EmployeeDetail(organizationID, uuid string) (*model.EmployeeDetailResponse, error) {
	it, err := s.orgRepo.EmployeeDetail(organizationID, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, 404, "employee not found")
		}
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employee detail")
	}
	s.ensureLocationsLoaded()
	it.AddressCityName = s.citiesName[strconv.Itoa(it.AddressCity)]
	return it, nil
}

func (s *OrganizationService) EmployeeCreate(organizationID, userID string, req *model.CreateEmployeeRequest) (string, error) {
	ok, err := s.orgRepo.RoleExistsForOrgOrDefault(organizationID, strings.TrimSpace(req.RoleID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to validate role_id")
	}
	if !ok {
		return "", NewServiceError(ErrInvalidInput, 400, "role_id tidak ditemukan")
	}

	if ok, err := s.orgRepo.EmployeeIDExists(organizationID, strings.TrimSpace(req.EmployeeID)); err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to validate employee_id")
	} else if ok {
		return "", NewServiceError(ErrInvalidInput, 400, "DUPLICATE_EMPLOYEE_ID")
	}

	if nik := strings.TrimSpace(req.NIK); nik != "" {
		if ok, err := s.orgRepo.NIKExists(organizationID, nik); err != nil {
			return "", NewServiceError(ErrInternalServer, 500, "failed to validate nik")
		} else if ok {
			return "", NewServiceError(ErrInvalidInput, 400, "DUPLICATE_NIK")
		}
	}

	id, err := s.orgRepo.CreateEmployee(organizationID, userID, req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid birth_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid birth_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid join_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid join_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid resign_date") {
			return "", NewServiceError(ErrInvalidInput, 400, "invalid resign_date")
		}
		return "", NewServiceError(ErrInternalServer, 500, "failed to create employee")
	}
	return id, nil
}

func (s *OrganizationService) EmployeeUpdate(organizationID, userID string, req *model.UpdateEmployeeRequest) error {
	ok, err := s.orgRepo.RoleExistsForOrgOrDefault(organizationID, strings.TrimSpace(req.RoleID))
	if err != nil {
		return NewServiceError(ErrInternalServer, 500, "failed to validate role_id")
	}
	if !ok {
		return NewServiceError(ErrInvalidInput, 400, "role_id tidak ditemukan")
	}

	err = s.orgRepo.UpdateEmployee(organizationID, userID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "employee not found")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid birth_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid birth_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid join_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid join_date")
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid resign_date") {
			return NewServiceError(ErrInvalidInput, 400, "invalid resign_date")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update employee")
	}
	return nil
}
