package service

import (
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
	orgRepo  *repository.OrganizationRepository
	userRepo *repository.UserRepository
}

func NewOrganizationService(orgRepo *repository.OrganizationRepository, userRepo *repository.UserRepository) *OrganizationService {
	return &OrganizationService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

// generateOrganizationCode generates organization code from organization name
// Format: 4 vowels from org name + 4 random digits
func (s *OrganizationService) generateOrganizationCode(orgName string) (string, error) {
	// Extract vowels from organization name
	vowels := "aeiouAEIOU"
	var extractedVowels []string

	for _, char := range orgName {
		if strings.ContainsRune(vowels, char) {
			extractedVowels = append(extractedVowels, strings.ToUpper(string(char)))
		}
	}

	// Take first 4 vowels (or pad with available vowels)
	var code string
	for i := 0; i < 4 && i < len(extractedVowels); i++ {
		code += extractedVowels[i]
	}

	// If less than 4 vowels, pad with random vowels
	for len(code) < 4 {
		randomVowel := string("AEIOU"[rand.Intn(5)])
		code += randomVowel
	}

	// Generate 4 random digits
	rand.Seed(time.Now().UnixNano())
	digits := fmt.Sprintf("%04d", rand.Intn(10000))
	code += digits

	// Check if code already exists
	existingOrg, err := s.orgRepo.FindByCode(code)
	if err == nil && existingOrg != nil {
		// If code exists, regenerate
		return s.generateOrganizationCode(orgName)
	}

	return code, nil
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(userID string, org *model.Organization) (*model.Organization, error) {
	// Check if user exists
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user has completed required fields (phone, name, email)
	if user.Phone == "" || user.Name == "" || user.Email == "" {
		return nil, errors.New("user must complete phone, name, and email before creating organization")
	}

	// Generate organization ID with UUID
	org.ID = helper.GenerateUUID()

	// Generate organization code
	orgCode, err := s.generateOrganizationCode(org.OrganizationName)
	if err != nil {
		return nil, errors.New("failed to generate organization code")
	}
	org.OrganizationCode = orgCode

	// Set user ID
	org.UserID = userID

	// Create organization
	createdOrg, err := s.orgRepo.Create(org)
	if err != nil {
		return nil, errors.New("failed to create organization")
	}

	return createdOrg, nil
}
