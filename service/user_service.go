package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"time"
)

type UserService struct {
	userRepo      *repository.UserRepository
	orgUserRepo   *repository.OrganizationUserRepository
	orgRepo       *repository.OrganizationRepository
	citiesName    map[string]string
	provincesName map[string]string
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// SetOrganizationUserRepository sets the organization user repository
func (s *UserService) SetOrganizationUserRepository(orgUserRepo *repository.OrganizationUserRepository) {
	s.orgUserRepo = orgUserRepo
}

// SetOrganizationRepository sets the organization repository
func (s *UserService) SetOrganizationRepository(orgRepo *repository.OrganizationRepository) {
	s.orgRepo = orgRepo
}

func (s *UserService) GetAllUsers() ([]model.User, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch users")
	}
	return users, nil
}

func (s *UserService) GetUserByID(id string) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}
	return user, nil
}

func (s *UserService) CreateUser(user *model.User) (*model.User, error) {
	if user.Email == "" {
		return nil, NewServiceError(errors.New("validation error"), http.StatusBadRequest, "email is required")
	}

	existingUser, _ := s.userRepo.FindByEmail(user.Email)
	if existingUser != nil {
		return nil, NewServiceError(ErrEmailExists, http.StatusConflict, "email already exists")
	}

	if user.Password != "" {
		hashedPassword, err := helper.HashPassword(user.Password)
		if err != nil {
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to hash password")
		}
		user.Password = hashedPassword
	}

	createdUser, err := s.userRepo.Create(user)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create user")
	}

	return createdUser, nil
}

func (s *UserService) UpdateUser(id string, user *model.User) (*model.User, error) {
	existingUser, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	if user.Name != "" {
		existingUser.Name = user.Name
	}
	if user.Phone != "" {
		existingUser.Phone = user.Phone
	}
	if user.Address != "" {
		existingUser.Address = user.Address
	}
	if user.City != "" {
		existingUser.City = user.City
	}
	if user.Province != "" {
		existingUser.Province = user.Province
	}

	updatedUser, err := s.userRepo.Update(existingUser)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update user")
	}

	return updatedUser, nil
}

func (s *UserService) DeleteUser(id string) error {
	_, err := s.userRepo.FindByID(id)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	if err = s.userRepo.Delete(id); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to delete user")
	}

	return nil
}

func (s *UserService) UpdateProfile(user *model.User) (*model.User, error) {
	existingUser, err := s.userRepo.FindByID(user.UserID)
	if err != nil {
		return nil, NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	if user.Name != "" {
		existingUser.Name = user.Name
	}
	if user.Phone != "" {
		existingUser.Phone = user.Phone
	}
	if user.NPWP != "" {
		existingUser.NPWP = user.NPWP
	}
	if user.Gender != "" {
		existingUser.Gender = user.Gender
	}
	if user.DateOfBirth != nil {
		existingUser.DateOfBirth = user.DateOfBirth
	}
	if user.Address != "" {
		existingUser.Address = user.Address
	}
	if user.City != "" {
		existingUser.City = user.City
	}
	if user.Province != "" {
		existingUser.Province = user.Province
	}
	if user.PostalCode != "" {
		existingUser.PostalCode = user.PostalCode
	}
	if user.Avatar != "" {
		existingUser.Avatar = user.Avatar
	}

	updatedUser, err := s.userRepo.UpdateProfile(existingUser)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update profile")
	}

	return updatedUser, nil
}

// UpdatePassword updates user password after verifying current password
func (s *UserService) UpdatePassword(userID, currentPassword, newPassword string) error {
	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// Verify current password
	if !helper.CheckPasswordHash(currentPassword, user.Password) {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "current password is incorrect")
	}

	// Check if new password is the same as current password
	if helper.CheckPasswordHash(newPassword, user.Password) {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "new password must be different from current password")
	}

	// Hash new password
	hashedPassword, err := helper.HashPassword(newPassword)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to hash password")
	}

	// Update password
	if err = s.userRepo.UpdatePassword(userID, hashedPassword); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update password")
	}

	return nil
}

// ProfileResponse represents the profile response with organization data
type ProfileResponse struct {
	UserID        string               `json:"user_id"`
	Username      string               `json:"username"`
	Name          string               `json:"name"`
	Email         string               `json:"email"`
	Phone         string               `json:"phone"`
	Address       string               `json:"address"`
	City          string               `json:"city"`
	CityLabel     string               `json:"city_label"`
	Province      string               `json:"province"`
	ProvinceLabel string               `json:"province_label"`
	PostalCode    string               `json:"postal_code"`
	NPWP          string               `json:"npwp"`
	Gender        string               `json:"gender"` // M = Male, F = Female
	DateOfBirth   *time.Time           `json:"date_of_birth"`
	Avatar        string               `json:"avatar"`
	IsActive      bool                 `json:"is_active"`
	IsVerified    bool                 `json:"is_verified"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Organization  *OrganizationProfile `json:"organization"`
}

// OrganizationProfile represents organization data in profile response
type OrganizationProfile struct {
	OrganizationCode string    `json:"organization_code"`
	OrganizationName string    `json:"organization_name"`
	CompanyName      string    `json:"company_name"`
	JoinDate         time.Time `json:"join_date"`
	OrganizationRole int       `json:"organization_role"`
}

// GetProfile retrieves user profile with organization data
func (s *UserService) GetProfile(userID string) (*ProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// Set default avatar based on gender if avatar is empty
	avatar := user.Avatar
	if avatar == "" {
		if user.Gender == "F" {
			avatar = "/assets/avatar/default-avatar-female.png"
		} else {
			avatar = "/assets/avatar/default-avatar.png"
		}
	}

	// Add APP_HOST prefix to avatar URL if it starts with /assets
	avatar = helper.GetAssetURL(avatar)

	profile := &ProfileResponse{
		UserID:       user.UserID,
		Username:     user.Username,
		Name:         user.Name,
		Email:        user.Email,
		Phone:        user.Phone,
		Address:      user.Address,
		City:         user.City,
		Province:     user.Province,
		PostalCode:   user.PostalCode,
		NPWP:         user.NPWP,
		Gender:       user.Gender,
		DateOfBirth:  user.DateOfBirth,
		Avatar:       avatar,
		IsActive:     user.IsActive,
		IsVerified:   user.IsVerified,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Organization: nil, // Default to nil, will be set if organization exists
	}

	s.ensureLocationsLoaded()
	if profile.City != "" {
		if name, ok := s.citiesName[profile.City]; ok && name != "" {
			profile.CityLabel = name
		} else {
			profile.CityLabel = profile.City
		}
	}
	if profile.Province != "" {
		if name, ok := s.provincesName[profile.Province]; ok && name != "" {
			profile.ProvinceLabel = name
		} else {
			profile.ProvinceLabel = profile.Province
		}
	}

	// Get organization data if available
	if s.orgUserRepo != nil {
		orgCode, orgName, companyName, joinDate, orgRole, err := s.orgUserRepo.GetOrganizationWithJoinDateByUserID(userID)
		if err == nil {
			profile.Organization = &OrganizationProfile{
				OrganizationCode: orgCode,
				OrganizationName: orgName,
				CompanyName:      companyName,
				JoinDate:         joinDate,
				OrganizationRole: orgRole,
			}
		} else if err != sql.ErrNoRows {
			// Log error but don't fail the request
			// If error is sql.ErrNoRows, user doesn't have organization (optional)
		}
	}

	return profile, nil
}

func (s *UserService) CheckPassword(userID, password string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}
	if !helper.CheckPasswordHash(password, user.Password) {
		return NewServiceError(ErrInvalidCredentials, http.StatusOK, "Password Tidak Sesuai")
	}
	return nil
}

func (s *UserService) SendUpdatePasswordOTP(orgID, userID string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}
	if user.Email == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "email is required")
	}
	if s.orgUserRepo != nil {
		gotOrgID, _, err := s.orgUserRepo.GetOrganizationAndRoleByUserID(userID)
		if err != nil {
			return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "unauthorized")
		}
		if gotOrgID != orgID {
			return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "unauthorized")
		}
	}

	emailCfg := &configs.EmailConfig{
		From:     os.Getenv("EMAIL_FROM"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		SMTPHost: os.Getenv("EMAIL_SMTP_HOST"),
		SMTPPort: os.Getenv("EMAIL_SMTP_PORT"),
	}
	if err := configs.ValidateEmailConfig(emailCfg); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "email config not set")
	}

	otp := helper.GenerateOTP(6)
	key := fmt.Sprintf("Password_%s", userID)
	if err := helper.SetOTPWithTTL(key, otp, 5*time.Minute); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to store OTP")
	}
	if err := helper.SendResetPasswordOTPEmail(emailCfg, user.Email, user.Username, otp); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send OTP email")
	}

	return nil
}

func (s *UserService) UpdatePasswordWithOTP(userID, otp, existingPassword, newPassword, confirmPassword string) error {
	if newPassword != confirmPassword {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "confirm_password must match new_password")
	}

	key := fmt.Sprintf("Password_%s", userID)
	storedOTP, err := helper.GetOTP(key)
	if err != nil || storedOTP != otp {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "INVALID_OTP")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}
	if !helper.CheckPasswordHash(existingPassword, user.Password) {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "INVALID_PASSWORD")
	}

	hashedPassword, err := helper.HashPassword(newPassword)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to hash password")
	}
	if err = s.userRepo.UpdatePassword(userID, hashedPassword); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update password")
	}
	helper.DeleteOTP(key)
	return nil
}

func (s *UserService) ensureLocationsLoaded() {
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
	cm := make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		cm[c.ID] = c.Name
	}
	pm := make(map[string]string, len(loc.Provinces))
	for _, p := range loc.Provinces {
		pm[p.ID] = p.Name
	}
	s.citiesName = cm
	s.provincesName = pm
}
