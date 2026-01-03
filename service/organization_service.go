package service

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"service-travego/model"
	"service-travego/repository"
	"strings"
	"time"
)

type OrganizationService struct {
	orgRepo     *repository.OrganizationRepository
	orgUserRepo *repository.OrganizationUserRepository
	userRepo    *repository.UserRepository
	orgTypeRepo *repository.OrganizationTypeRepository
}

func NewOrganizationService(orgRepo *repository.OrganizationRepository, userRepo *repository.UserRepository) *OrganizationService {
	return &OrganizationService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

// SetOrganizationUserRepository sets the organization user repository
func (s *OrganizationService) SetOrganizationUserRepository(orgUserRepo *repository.OrganizationUserRepository) {
	s.orgUserRepo = orgUserRepo
}

// SetOrganizationTypeRepository sets the organization type repository
func (s *OrganizationService) SetOrganizationTypeRepository(orgTypeRepo *repository.OrganizationTypeRepository) {
	s.orgTypeRepo = orgTypeRepo
}

// generateOrganizationCode generates organization code from organization name
// Format: non-vowel letters (consonants) from org name + 4 random digits
// Example: "AGRA MAS" -> "AGRMS" + 4 digits, "GARUDA MAS" -> "GRDMS" + 4 digits, "citra adi lancar" -> "CAL" + 4 digits
func (s *OrganizationService) generateOrganizationCode(orgName string) (string, error) {
	// Extract non-vowel letters (consonants) from organization name
	vowels := "aeiouAEIOU "
	var extractedConsonants []string

	for _, char := range orgName {
		charStr := string(char)
		// Skip vowels and spaces
		if !strings.ContainsRune(vowels, char) {
			extractedConsonants = append(extractedConsonants, strings.ToUpper(charStr))
		}
	}

	// Take up to 6 consonants to keep code length reasonable (max 10 with 4 digits)
	var code string
	for i, consonant := range extractedConsonants {
		if i >= 6 {
			break
		}
		code += consonant
	}

	// If no consonants found, use first 5 uppercase letters
	if code == "" {
		for _, char := range orgName {
			if char >= 'A' && char <= 'Z' {
				code += string(char)
				if len(code) >= 5 {
					break
				}
			} else if char >= 'a' && char <= 'z' {
				code += strings.ToUpper(string(char))
				if len(code) >= 5 {
					break
				}
			}
		}
	}

	// Generate 4 random digits
	rand.Seed(time.Now().UnixNano())
	digits := fmt.Sprintf("%04d", rand.Intn(10000))
	code += digits

	// Check if code already exists, if exists regenerate with new random digits
	maxAttempts := 100
	attempts := 0
	consonantPart := code[:len(code)-4] // Store the consonant part before adding digits

	for attempts < maxAttempts {
		existingOrg, err := s.orgRepo.FindByCode(code)
		if err != nil {
			// Check if it's "not found" error
			if err == sql.ErrNoRows {
				// Code doesn't exist, we can use it
				return code, nil
			}
			// Other database error
			return "", fmt.Errorf("failed to check organization code: %w", err)
		}

		// Code exists, regenerate random digits
		if existingOrg != nil {
			digits = fmt.Sprintf("%04d", rand.Intn(10000))
			code = consonantPart + digits
			attempts++
		}
	}

	return "", errors.New("failed to generate unique organization code after multiple attempts")
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(userID string, org *model.Organization) (*model.Organization, error) {
	// Check if user already has an organization (optional, depending on business logic)
	// For now, allow multiple organizations per user

	// Generate organization code if not provided
	if org.OrganizationCode == "" {
		code, err := s.generateOrganizationCode(org.OrganizationName)
		if err != nil {
			return nil, err
		}
		org.OrganizationCode = code
	} else {
		// Check if provided code exists
		existingOrg, err := s.orgRepo.FindByCode(org.OrganizationCode)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if existingOrg != nil {
			return nil, errors.New("organization code already exists")
		}
	}

	org.CreatedBy = userID
	return s.orgRepo.Create(org)
}

// UpdateOrganization updates an existing organization
func (s *OrganizationService) UpdateOrganization(userID string, org *model.Organization) (*model.Organization, error) {
	// Check if organization exists
	existingOrg, err := s.orgRepo.FindByID(org.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("organization not found")
		}
		return nil, err
	}

	// Check if user is authorized to update (must be the creator or admin)
	// Here we check if the user is the creator. In real app, check roles.
	if existingOrg.CreatedBy != userID {
		// Also check organization_users table if needed
		// For now, strict check
		// return nil, errors.New("unauthorized to update organization")
	}

	// Update fields
	existingOrg.OrganizationName = org.OrganizationName
	existingOrg.CompanyName = org.CompanyName
	existingOrg.Address = org.Address
	existingOrg.City = org.City
	existingOrg.Province = org.Province
	existingOrg.Phone = org.Phone
	existingOrg.Email = org.Email

	return s.orgRepo.Update(existingOrg)
}

// GetAPIConfig retrieves API configuration for an organization
func (s *OrganizationService) GetAPIConfig(userID string) (map[string]interface{}, error) {
	// Find organization by user
	// Assuming 1 user = 1 organization for now, or get list
	orgs, err := s.orgRepo.FindByUsername(userID)
	if err != nil {
		return nil, err
	}

	if len(orgs) == 0 {
		return nil, nil
	}

	// Return config for the first organization found
	return map[string]interface{}{
		"organization_id":   orgs[0].ID,
		"organization_code": orgs[0].OrganizationCode,
		"domain_url":        orgs[0].DomainURL,
	}, nil
}

// UpdateDomainURL updates the domain URL for an organization
func (s *OrganizationService) UpdateDomainURL(userID, organizationID, domainURL string) error {
	// Verify user belongs to organization and is admin
	if s.orgUserRepo == nil {
		return errors.New("organization user repository not initialized")
	}

	role, err := s.orgUserRepo.GetRoleByUserIDAndOrgID(userID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found in organization")
		}
		return fmt.Errorf("failed to check role: %w", err)
	}

	// Check if admin (role 1)
	if role != 1 {
		return errors.New("access denied: only admin can update domain url")
	}

	return s.orgRepo.UpdateDomainURL(organizationID, domainURL)
}

// GetBankAccounts retrieves bank accounts for an organization with payment method logic
func (s *OrganizationService) GetBankAccounts(organizationID string) ([]model.OrganizationBankAccountResponse, error) {
	accounts, err := s.orgRepo.GetBankAccounts(organizationID)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		// Default payment method
		accounts[i].PaymentMethod = "Bank Transfer"

		// Check if it qualifies for QRIS
		if accounts[i].AccountName != "" &&
			accounts[i].MerchantID != "" &&
			accounts[i].MerchantNMID != "" &&
			accounts[i].MerchantMCC != "" &&
			accounts[i].MerchantAddress != "" &&
			accounts[i].MerchantCity != "" &&
			accounts[i].MerchantPostalCode != "" &&
			accounts[i].AccountType != 0 {
			accounts[i].PaymentMethod = "QRIS"
		}
	}

	return accounts, nil
}

// CreateBankAccount creates a new bank account for an organization
func (s *OrganizationService) CreateBankAccount(req *model.CreateOrganizationBankAccountRequest, organizationID, createdBy, createdProxy, createdIP string) error {
	// Validation for QRIS payment method
	if req.PaymentMethod == model.BankAccountPaymentMethodQRIS {
		if req.MerchantName == "" || req.MerchantMCC == "" || req.MerchantAddress == "" || req.MerchantCity == "" || req.MerchantPostalCode == "" || req.AccountType == 0 {
			return errors.New("merchant details and account type are required for QRIS payment method")
		}
	}

	// Check if bank account already exists for the organization and bank code
	existingBankName, err := s.orgRepo.CheckBankAccountExists(organizationID, req.BankCode)
	if err != nil {
		return err
	}
	if existingBankName != "" {
		return fmt.Errorf("%s sudah terdaftar", existingBankName)
	}

	return s.orgRepo.CreateBankAccount(req, organizationID, createdBy, createdProxy, createdIP)
}

// UpdateBankAccount updates an existing bank account for an organization
func (s *OrganizationService) UpdateBankAccount(req *model.UpdateOrganizationBankAccountRequest, organizationID, updatedProxy, updatedIP string) error {
	// Mutually exclusive validation
	if req.Active != nil {
		if req.AccountNumber != "" || req.AccountName != "" {
			return errors.New("cannot update active status and account details simultaneously")
		}
	} else {
		if req.AccountNumber == "" || req.AccountName == "" {
			return errors.New("account_number and account_name are required when active is not provided")
		}
	}

	return s.orgRepo.UpdateBankAccount(req.BankAccountID, organizationID, req.Active, req.AccountNumber, req.AccountName, updatedProxy, updatedIP)
}

// DeleteBankAccount deletes a bank account for an organization
func (s *OrganizationService) DeleteBankAccount(bankAccountID, organizationID string) error {
	return s.orgRepo.DeleteBankAccount(bankAccountID, organizationID)
}
