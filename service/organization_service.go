package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strings"
	"time"
)

type OrganizationService struct {
	orgRepo       *repository.OrganizationRepository
	orgUserRepo   *repository.OrganizationUserRepository
	userRepo      *repository.UserRepository
	orgTypeRepo   *repository.OrganizationTypeRepository
	citiesName    map[string]string
	provincesName map[string]string
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

func (s *OrganizationService) UpdateOrganizationDetail(orgID string, payload map[string]interface{}) error {
	name, _ := payload["organization_name"].(string)
	company, _ := payload["company_name"].(string)
	phone, _ := payload["phone"].(string)
	address, _ := payload["address"].(string)
	email, _ := payload["email"].(string)
	orgCode, _ := payload["organization_code"].(string)

	if name == "" || company == "" || phone == "" || address == "" || email == "" {
		return NewServiceError(ErrInvalidInput, 400, "organization_name, company_name, phone, address, email wajib")
	}

	var provinceStr *string
	if v, ok := payload["province"]; ok {
		switch t := v.(type) {
		case float64:
			s := fmt.Sprintf("%d", int(t))
			provinceStr = &s
		case int:
			s := fmt.Sprintf("%d", t)
			provinceStr = &s
		case string:
			s := t
			provinceStr = &s
		}
	}

	var cityStr *string
	if v, ok := payload["city"]; ok {
		switch t := v.(type) {
		case float64:
			s := fmt.Sprintf("%d", int(t))
			cityStr = &s
		case int:
			s := fmt.Sprintf("%d", t)
			cityStr = &s
		case string:
			s := t
			cityStr = &s
		}
	}

	var npwpPtr *string
	if v, ok := payload["npwp_number"].(string); ok {
		s := v
		npwpPtr = &s
	}
	var postalPtr *string
	if v, ok := payload["postal_code"].(string); ok {
		s := v
		postalPtr = &s
	}
	var orgTypePtr *int
	if v, ok := payload["organization_type"]; ok {
		switch t := v.(type) {
		case float64:
			iv := int(t)
			orgTypePtr = &iv
		case int:
			iv := t
			orgTypePtr = &iv
		}
	}

	if orgCode == "" {
		return NewServiceError(ErrInvalidInput, 400, "organization_code diperlukan untuk verifikasi")
	}

	if err := s.orgRepo.UpdateByIDAndCode(orgID, orgCode, name, company, phone, address, email, provinceStr, cityStr, npwpPtr, postalPtr, orgTypePtr); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, 404, "organization tidak ditemukan atau code tidak cocok")
		}
		return err
	}
	return nil
}

func (s *OrganizationService) UpdateOrganizationLogo(orgID, sourceFilePath string) (string, error) {
	if sourceFilePath == "" {
		return "", NewServiceError(ErrInvalidInput, 400, "file_path wajib")
	}
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		return "", NewServiceError(ErrInvalidInput, 400, "file_path tidak ditemukan")
	}

	org, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", NewServiceError(ErrNotFound, 404, "organization tidak ditemukan")
		}
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(sourceFilePath))
	if ext == "" {
		ext = ".png"
	}
	// Ensure storage directory exists
	storageDir := filepath.FromSlash("assets/logo")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", NewServiceError(ErrInternalServer, 500, fmt.Sprintf("gagal membuat direktori: %v", err))
	}

	filename := org.OrganizationCode + ext
	destLocalPath := filepath.Join(storageDir, filename)

	// Copy file
	if err := copyFile(sourceFilePath, destLocalPath); err != nil {
		return "", err
	}

	// Store path in DB as web path
	webPath := "/assets/logo/" + filename
	if err := s.orgRepo.UpdateLogo(orgID, webPath); err != nil {
		return "", err
	}

	return helper.GetAssetURL(webPath), nil
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
func (s *OrganizationService) ensureLocationsLoaded() {
	if s.citiesName != nil && s.provincesName != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		s.citiesName = map[string]string{}
		s.provincesName = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		s.citiesName = map[string]string{}
		s.provincesName = map[string]string{}
		return
	}
	s.citiesName = make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		s.citiesName[c.ID] = c.Name
	}
	s.provincesName = make(map[string]string, len(loc.Provinces))
	for _, p := range loc.Provinces {
		s.provincesName[p.ID] = p.Name
	}
}

func (s *OrganizationService) GetOrganizationDetail(organizationID string) (map[string]interface{}, error) {
	org, err := s.orgRepo.FindByID(organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, 404, "organization not found")
		}
		return nil, err
	}

	s.ensureLocationsLoaded()
	cityName := org.City
	if name, ok := s.citiesName[org.City]; ok {
		cityName = name
	}
	provinceName := org.Province
	if name, ok := s.provincesName[org.Province]; ok {
		provinceName = name
	}

	var orgTypeLabel string
	switch org.OrganizationType {
	case 1:
		orgTypeLabel = "RENTAL KENDARAAN"
	case 2:
		orgTypeLabel = "BIRO PERJALANAN WISATA"
	case 3:
		orgTypeLabel = "RENTAL DAN JASA PERJALANAN WISATA"
	default:
		orgTypeLabel = ""
	}

	res := map[string]interface{}{
		"organization_code": org.OrganizationCode,
		"organization_name": org.OrganizationName,
		"company_name":      org.CompanyName,
		"address":           org.Address,
		"city":              org.City,
		"province":          org.Province,
		"phone":             org.Phone,
		"npwp_number":       org.NPWPNumber,
		"email":             org.Email,
		"organization_type": orgTypeLabel,
		"postal_code":       org.PostalCode,
		"domain_url":        org.DomainURL,
		"logo":              org.Logo,
		"city_name":         cityName,
		"province_name":     provinceName,
	}
	return res, nil
}
