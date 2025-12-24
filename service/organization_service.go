package service

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"service-travego/helper"
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
		if existingOrg != nil {
			// Code exists, regenerate digits
			rand.Seed(time.Now().UnixNano() + int64(attempts))
			digits = fmt.Sprintf("%04d", rand.Intn(10000))
			// Keep the consonant part, only change digits
			code = consonantPart + digits
			attempts++
		} else {
			// Code doesn't exist
			return code, nil
		}
	}

	// If we exhausted attempts, return error
	return "", fmt.Errorf("failed to generate unique organization code after %d attempts", maxAttempts)
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(userID string, org *model.Organization) (*model.Organization, error) {
	// Check if user exists
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user profile is complete: address, city, province, date_of_birth, gender
	if user.Address == "" {
		return nil, errors.New("profile must be completed before creating organization")
	}
	if user.City == "" {
		return nil, errors.New("profile city must be completed before creating organization")
	}
	if user.Province == "" {
		return nil, errors.New("profile province must be completed before creating organization")
	}
	if user.DateOfBirth == nil {
		return nil, errors.New("profile date_of_birth must be completed before creating organization")
	}
	if user.Gender == "" {
		return nil, errors.New("profile gender must be completed before creating organization")
	}

	// Generate organization ID with UUID
	org.ID = helper.GenerateUUID()

	// Generate organization code from organization name (if not provided in payload)
	if org.OrganizationCode == "" {
		orgCode, err := s.generateOrganizationCode(org.OrganizationName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate organization code: %w", err)
		}
		org.OrganizationCode = orgCode
	} else {
		// If organization_code is provided, check if it already exists
		existingOrg, err := s.orgRepo.FindByCode(org.OrganizationCode)
		if err == nil && existingOrg != nil {
			return nil, errors.New("organization code already exists")
		}
		// If error is not "not found", return error
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check organization code: %w", err)
		}
	}

	// Set created_by using canonical DB key
	org.CreatedBy = user.UserID

	// Set username
	org.Username = user.Username

	// Validate organization type if repository available
	if s.orgTypeRepo != nil {
		if _, err := s.orgTypeRepo.FindByID(org.OrganizationType); err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.New("organization type invalid")
			}
			return nil, fmt.Errorf("failed to validate organization type: %w", err)
		}
	}

	// Create organization
	createdOrg, err := s.orgRepo.Create(org)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Insert into organization_users with role 1, is_active true
	if s.orgUserRepo != nil {
		orgUser := &model.OrganizationUser{
			UUID:             helper.GenerateUUID(),
			UserID:           userID,
			OrganizationID:   createdOrg.ID,
			OrganizationRole: 1,
			IsActive:         true,
			CreatedAt:        time.Now(),
			CreatedBy:        userID,
			UpdatedAt:        time.Now(),
			UpdatedBy:        userID,
		}

		if err = s.orgUserRepo.CreateOrganizationUser(orgUser); err != nil {
			// Log error but don't fail the organization creation
			// In production, you might want to rollback the organization creation
			return nil, fmt.Errorf("failed to create organization user: %w", err)
		}
	}

	return createdOrg, nil
}

// GetAPIConfig generates an encrypted API token for admin users
func (s *OrganizationService) GetAPIConfig(userID, organizationID string) (string, error) {
	// Verify user role in organization
	if s.orgUserRepo == nil {
		return "", errors.New("organization user repository not initialized")
	}

	role, err := s.orgUserRepo.GetRoleByUserIDAndOrgID(userID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("user not found in organization")
		}
		return "", fmt.Errorf("failed to check role: %w", err)
	}

	// Check if admin (role 1)
	if role != 1 {
		return "", errors.New("access denied: only admin can generate api config")
	}

	// Encrypt organization ID
	token, err := helper.EncryptString(organizationID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
