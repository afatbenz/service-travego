package service

import (
	"database/sql"
	"service-travego/model"
	"strings"
)

func isZeroOrganizationID(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	if s == "0" || s == "00" || s == "000" {
		return true
	}
	s = strings.ReplaceAll(s, "-", "")
	allZero := true
	for _, r := range s {
		if r != '0' {
			allZero = false
			break
		}
	}
	return allZero
}

func (s *OrganizationService) ListDivisions(organizationID string) ([]model.OrganizationDivision, error) {
	items, err := s.orgRepo.ListDivisions(organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get divisions")
	}
	return items, nil
}

func (s *OrganizationService) CreateDivision(organizationID, userID string, req *model.CreateOrganizationDivisionRequest) (string, error) {
	if strings.TrimSpace(req.DivisionName) == "" {
		return "", NewServiceError(ErrInvalidInput, 400, "division_name wajib")
	}
	id, err := s.orgRepo.CreateDivision(organizationID, userID, strings.TrimSpace(req.DivisionName), strings.TrimSpace(req.Description))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to create division")
	}
	return id, nil
}

func (s *OrganizationService) UpdateDivision(organizationID, userID string, req *model.UpdateOrganizationDivisionRequest) error {
	if strings.TrimSpace(req.DivisionID) == "" {
		return NewServiceError(ErrInvalidInput, 400, "division_id wajib")
	}
	if strings.TrimSpace(req.DivisionName) == "" {
		return NewServiceError(ErrInvalidInput, 400, "division_name wajib")
	}

	targetOrgID, err := s.orgRepo.GetDivisionOrganizationID(strings.TrimSpace(req.DivisionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "division not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update division")
	}
	if isZeroOrganizationID(targetOrgID) || isZeroOrganizationID(organizationID) {
		return NewServiceError(ErrInvalidInput, 400, "UPDATED_ENIED")
	}
	if strings.TrimSpace(targetOrgID) != strings.TrimSpace(organizationID) {
		return NewServiceError(ErrNotFound, 404, "division not found")
	}

	err = s.orgRepo.UpdateDivision(organizationID, userID, strings.TrimSpace(req.DivisionID), strings.TrimSpace(req.DivisionName), strings.TrimSpace(req.Description))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "division not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update division")
	}
	return nil
}

func (s *OrganizationService) DeleteDivision(organizationID, userID, divisionID string) error {
	if strings.TrimSpace(divisionID) == "" {
		return NewServiceError(ErrInvalidInput, 400, "division_id wajib")
	}

	targetOrgID, err := s.orgRepo.GetDivisionOrganizationID(strings.TrimSpace(divisionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "division not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete division")
	}
	if isZeroOrganizationID(targetOrgID) || isZeroOrganizationID(organizationID) {
		return NewServiceError(ErrInvalidInput, 400, "UPDATED_ENIED")
	}
	if strings.TrimSpace(targetOrgID) != strings.TrimSpace(organizationID) {
		return NewServiceError(ErrNotFound, 404, "division not found")
	}

	err = s.orgRepo.DeleteDivision(organizationID, userID, strings.TrimSpace(divisionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "division not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete division")
	}
	return nil
}

func (s *OrganizationService) ListRoles(organizationID string) ([]model.OrganizationRole, error) {
	items, err := s.orgRepo.ListRoles(organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get roles")
	}
	return items, nil
}

func (s *OrganizationService) CreateRole(organizationID, userID string, req *model.CreateOrganizationRoleRequest) (string, error) {
	if strings.TrimSpace(req.RoleName) == "" {
		return "", NewServiceError(ErrInvalidInput, 400, "role_name wajib")
	}
	if strings.TrimSpace(req.DivisionID) == "" {
		return "", NewServiceError(ErrInvalidInput, 400, "division_id wajib")
	}
	ok, err := s.orgRepo.DivisionExists(organizationID, strings.TrimSpace(req.DivisionID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to validate division_id")
	}
	if !ok {
		return "", NewServiceError(ErrInvalidInput, 400, "division_id tidak ditemukan")
	}

	id, err := s.orgRepo.CreateRole(organizationID, userID, strings.TrimSpace(req.RoleName), strings.TrimSpace(req.Description), strings.TrimSpace(req.DivisionID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, 500, "failed to create role")
	}
	return id, nil
}

func (s *OrganizationService) UpdateRole(organizationID, userID string, req *model.UpdateOrganizationRoleRequest) error {
	if strings.TrimSpace(req.RoleID) == "" {
		return NewServiceError(ErrInvalidInput, 400, "role_id wajib")
	}
	if strings.TrimSpace(req.RoleName) == "" {
		return NewServiceError(ErrInvalidInput, 400, "role_name wajib")
	}
	if strings.TrimSpace(req.DivisionID) == "" {
		return NewServiceError(ErrInvalidInput, 400, "division_id wajib")
	}

	targetOrgID, err := s.orgRepo.GetRoleOrganizationID(strings.TrimSpace(req.RoleID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "role not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update role")
	}
	if isZeroOrganizationID(targetOrgID) || isZeroOrganizationID(organizationID) {
		return NewServiceError(ErrInvalidInput, 400, "UPDATED_ENIED")
	}
	if strings.TrimSpace(targetOrgID) != strings.TrimSpace(organizationID) {
		return NewServiceError(ErrNotFound, 404, "role not found")
	}

	ok, err := s.orgRepo.DivisionExists(organizationID, strings.TrimSpace(req.DivisionID))
	if err != nil {
		return NewServiceError(ErrInternalServer, 500, "failed to validate division_id")
	}
	if !ok {
		return NewServiceError(ErrInvalidInput, 400, "division_id tidak ditemukan")
	}

	err = s.orgRepo.UpdateRole(organizationID, userID, strings.TrimSpace(req.RoleID), strings.TrimSpace(req.RoleName), strings.TrimSpace(req.Description), strings.TrimSpace(req.DivisionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "role not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update role")
	}
	return nil
}

func (s *OrganizationService) DeleteRole(organizationID, userID, roleID string) error {
	if strings.TrimSpace(roleID) == "" {
		return NewServiceError(ErrInvalidInput, 400, "role_id wajib")
	}

	targetOrgID, err := s.orgRepo.GetRoleOrganizationID(strings.TrimSpace(roleID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "role not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete role")
	}
	if isZeroOrganizationID(targetOrgID) || isZeroOrganizationID(organizationID) {
		return NewServiceError(ErrInvalidInput, 400, "UPDATED_ENIED")
	}
	if strings.TrimSpace(targetOrgID) != strings.TrimSpace(organizationID) {
		return NewServiceError(ErrNotFound, 404, "role not found")
	}

	err = s.orgRepo.DeleteRole(organizationID, userID, strings.TrimSpace(roleID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "role not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete role")
	}
	return nil
}
