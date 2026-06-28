package service

import (
	"database/sql"
	"fmt"
	"service-travego/model"
	"strings"
)

func NormalizeAssistantAccountNumber(accountNumber string) string {
	accountNumber = strings.TrimSpace(accountNumber)
	if strings.HasPrefix(accountNumber, "0") {
		return "62" + accountNumber[1:]
	}
	return accountNumber
}

func (s *OrganizationService) assistantAccountLimitOrZero(organizationID string) (int, error) {
	accountLimit, err := s.orgRepo.GetAssistantAccountLimit(organizationID)
	if err != nil {
		return 0, err
	}
	return accountLimit, nil
}

func (s *OrganizationService) AssistantList(organizationID string) (map[string]interface{}, error) {
	totalAccount, err := s.orgRepo.CountActiveAssistantAccounts(organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, 500, "failed to count assistant accounts")
	}

	accountLimit, err := s.assistantAccountLimitOrZero(organizationID)
	if err != nil {
		return nil, err
	}

	items, err := s.orgRepo.ListAssistantAccounts(organizationID)
	if err != nil {
		items = []model.AssistantAccountListItem{}
	}

	remaining := 0
	if accountLimit > 0 {
		remaining = accountLimit - totalAccount
		if remaining < 0 {
			remaining = 0
		}
	}

	return map[string]interface{}{
		"total_account":   totalAccount,
		"remaining_limit": remaining,
		"users":           items,
	}, nil
}

func (s *OrganizationService) AssistantSubmit(organizationID, userID string, req *model.AssistantSubmitRequest) (map[string]interface{}, error) {
	userType := req.UserType
	if userType != 1 && userType != 2 {
		return nil, NewServiceError(ErrInvalidInput, 400, "user_type harus 1 atau 2")
	}

	accountLimit, err := s.assistantAccountLimitOrZero(organizationID)
	if err != nil {
		return nil, err
	}

	if accountLimit > 0 {
		currentTotal, err := s.orgRepo.CountActiveAssistantAccounts(organizationID)
		fmt.Println(currentTotal, " - currentTotal")
		fmt.Println(accountLimit, " - accountLimit")
		fmt.Println(err, " - err")
		if err != nil {
			return nil, NewServiceError(ErrInternalServer, 500, "failed to count assistant accounts")
		}
		if currentTotal >= accountLimit {
			return nil, NewServiceError(ErrInvalidInput, 400, "assistant account limit reached")
		}
	}

	accountNumber := NormalizeAssistantAccountNumber(req.AccountNumber)
	accountName := strings.TrimSpace(req.AccountName)

	var employee *model.AssistantEmployeeTarget
	var assistantUserID *string

	if userType == 1 {
		if accountName == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "account_name wajib untuk user_type 1")
		}
	} else {
		employeeID := strings.TrimSpace(req.EmployeeID)
		if employeeID == "" {
			return nil, NewServiceError(ErrInvalidInput, 400, "employee_id wajib untuk user_type 2")
		}

		var err error
		employee, err = s.orgRepo.GetAssistantEmployeeTarget(organizationID, employeeID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, NewServiceError(ErrNotFound, 404, "employee not found")
			}
			return nil, NewServiceError(ErrInternalServer, 500, "failed to validate employee")
		}

		assistantUserID = &employee.UUID
		if accountName == "" {
			accountName = employee.Fullname
		}
	}

	assistantID, err := s.orgRepo.CreateAssistantAccount(organizationID, userID, userType, assistantUserID, accountNumber, accountName)
	if err != nil {
		fmt.Println(err, " - err")
		return nil, NewServiceError(ErrInternalServer, 500, "failed to create assistant account")
	}

	response := map[string]interface{}{
		"assistant_id":   assistantID,
		"user_type":      userType,
		"account_number": accountNumber,
	}
	if employee != nil {
		response["employee_id"] = employee.EmployeeID
	}
	if accountName != "" {
		response["account_name"] = accountName
	}

	return response, nil
}

func (s *OrganizationService) GetAssistantAccountByID(organizationID, assistantID string) (*model.AssistantAccountListItem, error) {
	item, err := s.orgRepo.GetAssistantAccountByID(organizationID, assistantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, 404, "assistant account not found")
		}
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get assistant account")
	}
	return item, nil
}

func (s *OrganizationService) AssistantUpdate(organizationID string, req *model.AssistantUpdateRequest) error {
	if req.AccountName == nil && req.AccountNumber == nil {
		return NewServiceError(ErrInvalidInput, 400, "account_name atau account_number wajib diisi")
	}

	var accountNumber *string
	if req.AccountNumber != nil {
		normalized := NormalizeAssistantAccountNumber(*req.AccountNumber)
		accountNumber = &normalized
	}

	if err := s.orgRepo.UpdateAssistantAccount(organizationID, strings.TrimSpace(req.AssistantID), req.AccountName, accountNumber); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "assistant account not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to update assistant account")
	}

	return nil
}

func (s *OrganizationService) AssistantDelete(organizationID, employeeID string) error {
	employee, err := s.orgRepo.GetAssistantEmployeeTarget(organizationID, strings.TrimSpace(employeeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "employee not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to validate employee")
	}

	if err := s.orgRepo.DeleteAssistantAccountByUserID(organizationID, employee.UUID); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "assistant account not found")
		}
		return NewServiceError(ErrInternalServer, 500, "failed to delete assistant account")
	}

	return nil
}

func (s *OrganizationService) EmployeeWhatsApp(organizationID, employeeID string) (*model.EmployeeWhatsAppResponse, error) {
	employee, err := s.orgRepo.GetAssistantEmployeeTarget(organizationID, strings.TrimSpace(employeeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, 404, "employee not found")
		}
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get employee phone")
	}

	phone := strings.TrimSpace(employee.Phone)
	return &model.EmployeeWhatsAppResponse{
		EmployeeID: employee.EmployeeID,
		Phone:      phone,
		HasPhone:   phone != "",
	}, nil
}

func (s *OrganizationService) AssistantWhatsAppBusinessList(organizationID string) (*model.AssistantWhatsAppBusinessListResponse, error) {
	data, err := s.orgRepo.GetAssistantWhatsAppBusinessList(organizationID)
	if err != nil {
		fmt.Println(err, " - err")
		return nil, NewServiceError(ErrInternalServer, 500, "failed to get assistant whatsapp business list")
	}

	// Handle empty case
	if data == nil {
		data = &model.AssistantWhatsAppBusinessListResponse{}
	}

	statusLabel := "Unverified"
	if data.DeviceID != "" {
		statusLabel = "Verified"
	}

	// Default available is false
	available := false

	if s.subscriptionRepo != nil {
		subscriptions, subErr := s.subscriptionRepo.GetSubscriptionDetails(organizationID)
		if subErr == nil && len(subscriptions) > 0 {
			packageID := subscriptions[0].PackageID
			// If package_id != trave01, available is true
			if packageID != "trave01" {
				available = true
			}
		}
	}

	response := model.AssistantWhatsAppBusinessListResponse{
		AccountNumber: data.AccountNumber,
		DeviceID:      data.DeviceID,
		DeviceName:    data.DeviceName,
		DeviceToken:   data.DeviceToken,
		Status:        0,
		StatusLabel:   statusLabel,
		Available:     available,
	}

	// Set status 1 if available true
	if available {
		response.Status = 1
	}

	return &response, nil
}

func (s *OrganizationService) AssistantWhatsAppBusinessUpdate(organizationID, accountNumber string) error {
	existingAccount, err := s.orgRepo.GetAssistantWhatsAppBusinessList(organizationID)
	if err != nil {
		return NewServiceError(ErrInternalServer, 500, "failed to check existing whatsapp business account")
	}

	hasExisting := existingAccount != nil && existingAccount.AccountNumber != ""
	if !hasExisting {
		// No existing record, create new
		err = s.orgRepo.CreateAssistantWhatsappBusiness(organizationID, accountNumber)
		if err != nil {
			return NewServiceError(ErrInternalServer, 500, "failed to create whatsapp business account")
		}
	} else {
		// Existing record, update
		err = s.orgRepo.UpdateAssistantWhatsappBusiness(organizationID, accountNumber)
		if err != nil {
			if err == sql.ErrNoRows {
				// Maybe the record was deleted after check, try creating
				err = s.orgRepo.CreateAssistantWhatsappBusiness(organizationID, accountNumber)
				if err != nil {
					return NewServiceError(ErrInternalServer, 500, "failed to create whatsapp business account")
				}
			} else {
				return NewServiceError(ErrInternalServer, 500, "failed to update whatsapp business account")
			}
		}
	}

	return nil
}
