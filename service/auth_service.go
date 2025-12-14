package service

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strings"
	"time"
)

type AuthService struct {
	userRepo               *repository.UserRepository
	orgUserRepo            *repository.OrganizationUserRepository
	emailCfg               *configs.EmailConfig
	authTokenExpiryMinutes int
}

func NewAuthService(userRepo *repository.UserRepository, emailCfg *configs.EmailConfig) *AuthService {
	return &AuthService{
		userRepo:               userRepo,
		emailCfg:               emailCfg,
		authTokenExpiryMinutes: 0, // Will use default from helper.GetAuthTokenExpiry()
	}
}

// SetOrganizationUserRepository sets the organization user repository
func (s *AuthService) SetOrganizationUserRepository(orgUserRepo *repository.OrganizationUserRepository) {
	s.orgUserRepo = orgUserRepo
}

func (s *AuthService) Register(username, fullname, email, password, phone string) (*model.User, string, error) {
	// Normalize email to lowercase for consistent checking
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if email already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err == nil && existingUser != nil {
		log.Printf("[INFO] Register attempt with existing email - Email: %s", email)
		return nil, "", NewServiceError(ErrEmailExists, http.StatusBadRequest, "email already exists")
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Error checking email existence - Email: %s, Error: %v", email, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate email")
	}

	// Check if username already exists
	existingUsername, err := s.userRepo.FindByUsername(username)
	if err == nil && existingUsername != nil {
		log.Printf("[INFO] Register attempt with existing username - Username: %s", username)
		return nil, "", NewServiceError(ErrUsernameExists, http.StatusBadRequest, "username already exists")
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Error checking username existence - Username: %s, Error: %v", username, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate username")
	}

	// Normalize phone number: if starts with 0, replace with 62
	normalizedPhone := helper.NormalizePhoneNumber(phone)
	log.Printf("[INFO] Phone normalization - Original: %s, Normalized: %s", phone, normalizedPhone)

	// Check if phone already exists (after normalization)
	existingPhone, err := s.userRepo.FindByPhone(normalizedPhone)
	if err == nil && existingPhone != nil {
		log.Printf("[INFO] Register attempt with existing phone - Phone: %s (normalized: %s)", phone, normalizedPhone)
		return nil, "", NewServiceError(ErrPhoneExists, http.StatusBadRequest, "phone already exists")
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Error checking phone existence - Phone: %s, Error: %v", normalizedPhone, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate phone")
	}

	hashedPassword, err := helper.HashPassword(password)
	if err != nil {
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to hash password")
	}

	userID := helper.GenerateUUID()

	user := &model.User{
		UserID:     userID,
		Username:   username,
		Name:       fullname,
		Email:      email,
		Password:   hashedPassword,
		Phone:      normalizedPhone,
		IsActive:   true,
		IsVerified: false,
	}

	user, err = s.userRepo.Create(user)
	if err != nil {
		log.Printf("[ERROR] Failed to create user - Username: %s, Email: %s, Error: %v", username, email, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create user")
	}

	otp := helper.GenerateOTP(0) // 0 means use default from env or 8

	// Generate token from email and user_id (will be used as Redis key)
	registerToken, err := helper.EncryptData(email, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to generate token - Email: %s, UserID: %s, Error: %v", email, userID, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate token")
	}

	// Store OTP with token as key (token contains email and user_id)
	if err = helper.SetOTP(registerToken, otp); err != nil {
		log.Printf("[ERROR] Failed to store OTP - Token: %s, Error: %v", registerToken, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to store OTP")
	}

	if err = helper.SendOTPEmail(s.emailCfg, email, username, otp); err != nil {
		log.Printf("[ERROR] Failed to send OTP email - Email: %s, Username: %s, Error: %v", email, username, err)
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send OTP email")
	}

	return user, registerToken, nil
}

func (s *AuthService) VerifyOTP(token, otp string) error {
	// Decrypt token to get email and user_id
	email, userID, err := helper.DecryptData(token)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt token - Error: %v", err)
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid token")
	}

	// Get OTP from Redis using token as key (token contains email and user_id)
	storedOTP, err := helper.GetOTP(token)
	if err != nil {
		// Check if it's redis nil error (OTP expired or not found)
		errStr := err.Error()
		if errStr == "redis: nil" || errStr == "redis: nil: key does not exist" {
			return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "OTP expired")
		}
		log.Printf("[ERROR] Failed to get OTP from Redis - UserID: %s, Error: %v", userID, err)
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "OTP expired")
	}

	// Check if OTP matches
	if storedOTP != otp {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "missmatch")
	}

	// Verify user exists
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// Verify email matches
	if user.Email != email {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "missmatch")
	}

	// Update is_verified to true
	if err = s.userRepo.VerifyUser(userID); err != nil {
		log.Printf("[ERROR] Failed to verify user - UserID: %s, Error: %v", userID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to verify user")
	}

	// Delete OTP from Redis using token as key
	helper.DeleteOTP(token)

	// Send success email
	if err = helper.SendRegisterSuccessEmail(s.emailCfg, email, user.Username); err != nil {
		log.Printf("[ERROR] Failed to send success email - Email: %s, Error: %v", email, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send success email")
	}

	return nil
}

func (s *AuthService) ResendOTP(email, token string) (string, error) {
	var userEmail, userID string
	var err error

	// If email is provided, use email directly (no need for token)
	if email != "" {
		// Find user by email
		user, err := s.userRepo.FindByEmail(email)
		if err != nil {
			log.Printf("[DEBUG] ResendOTP user not found by email - Email: %s", email)
			return "", NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
		}
		userEmail = user.Email
		userID = user.UserID
	} else if token != "" {
		// If token is provided, decrypt it to get email and user_id
		userEmail, userID, err = helper.DecryptData(token)
		if err != nil {
			log.Printf("[ERROR] Failed to decrypt token - Error: %v", err)
			return "", NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid token")
		}
		tp := token
		if len(tp) > 16 {
			tp = tp[:8] + "..." + tp[len(tp)-8:]
		}
		log.Printf("[DEBUG] ResendOTP token decrypted - TokenPreview: %s, Email: %s, UserID: %s", tp, userEmail, userID)
	} else {
		return "", NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "either email or token is required")
	}

	// Find user by email and verify user_id matches
	user, err := s.userRepo.FindByEmail(userEmail)
	if err != nil {
		log.Printf("[DEBUG] ResendOTP user not found on confirm - Email: %s", userEmail)
		return "", NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// If using token, verify user_id from decrypt matches user_id in database
	if token != "" && email == "" {
		if user.UserID != userID {
			log.Printf("[DEBUG] ResendOTP user_id mismatch - Decrypted: %s, DB: %s", userID, user.UserID)
			return "", NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
		}
	}

	// Check if user is already verified
	if user.IsVerified {
		return "", NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "user telah diverifikasi")
	}

	// Check if user is inactive
	if !user.IsActive {
		return "", NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "user inactive")
	}

	// Generate token from email and user_id (will be used as Redis key)
	newToken, err := helper.EncryptData(userEmail, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to generate token - Email: %s, UserID: %s, Error: %v", userEmail, userID, err)
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate token")
	}

	// Generate new OTP
	otp := helper.GenerateOTP(0) // 0 means use default from env or 8

	// Store OTP with token as key (token contains email and user_id)
	if err = helper.SetOTP(newToken, otp); err != nil {
		log.Printf("[ERROR] Failed to store OTP - Token: %s, Error: %v", newToken, err)
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to store OTP")
	}

	// Send OTP email
	if err = helper.SendOTPEmail(s.emailCfg, userEmail, user.Username, otp); err != nil {
		log.Printf("[ERROR] Failed to send OTP email - Email: %s, Username: %s, Error: %v", userEmail, user.Username, err)
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send OTP email")
	}

	return newToken, nil
}

// LoginResponse represents login response data
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Avatar   string `json:"avatar"`
}

// Login authenticates a user with email/phone and password
// Returns LoginResponse containing token, username, fullname, and avatar
func (s *AuthService) Login(email, phone, password string) (*LoginResponse, error) {
	// Validate: either email or phone must be provided
	if email == "" && phone == "" {
		return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "email or phone is required")
	}

	var user *model.User
	var err error

	// Find user by email or phone
	if email != "" {
		// Normalize email to lowercase
		email = strings.ToLower(strings.TrimSpace(email))
		user, err = s.userRepo.FindByEmail(email)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "invalid credentials")
			}
			log.Printf("[ERROR] Error finding user by email - Email: %s, Error: %v", email, err)
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to authenticate")
		}
	} else if phone != "" {
		// Normalize phone number
		phone = helper.NormalizePhoneNumber(phone)
		user, err = s.userRepo.FindByPhone(phone)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "invalid credentials")
			}
			log.Printf("[ERROR] Error finding user by phone - Phone: %s, Error: %v", phone, err)
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to authenticate")
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "user is inactive")
	}

	// Check if user is verified
	if !user.IsVerified {
		return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "user is not verified")
	}

	// Verify password
	if !helper.CheckPasswordHash(password, user.Password) {
		log.Printf("[INFO] Invalid password attempt - UserID: %s", user.UserID)
		return nil, NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "invalid credentials")
	}

	// Get organization_id and role from organization_users table
	organizationID := ""
	organizationRole := 0
	if s.orgUserRepo != nil {
		orgID, role, err := s.orgUserRepo.GetOrganizationAndRoleByUserID(user.UserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("[ERROR] Error getting organization and role - UserID: %s, Error: %v", user.UserID, err)
			// Continue without organization data if error (user might not have organization)
		} else if err == nil {
			organizationID = orgID
			organizationRole = role
		}
	}

	// Generate JWT token
	token, err := helper.GenerateAuthToken(
		user.Name,
		organizationID,
		user.Username,
		user.UserID,
		organizationRole,
		user.Gender,
		user.IsAdmin,
		s.authTokenExpiryMinutes,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to generate auth token - UserID: %s, Error: %v", user.UserID, err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate token")
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

	return &LoginResponse{
		Token:    token,
		Username: user.Username,
		Fullname: user.Name,
		Avatar:   avatar,
	}, nil
}

// RequestResetPassword handles password reset request
// Validates user is active, verified, and not deleted, then sends reset password email
func (s *AuthService) RequestResetPassword(email string, resetPasswordURL string, expiryMinutes int) error {
	// Normalize email to lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if user exists or not for security
			return nil // Return success even if user not found
		}
		log.Printf("[ERROR] Error finding user by email for reset password - Email: %s, Error: %v", email, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to process reset password request")
	}

	// Check if user is active
	if !user.IsActive {
		// Don't reveal user status for security
		return nil // Return success even if user is inactive
	}

	// Check if user is verified
	if !user.IsVerified {
		// Don't reveal user status for security
		return nil // Return success even if user is not verified
	}

	// Check if user is deleted (deleted_at is NULL)
	if user.DeletedAt != nil {
		// Don't reveal user status for security
		return nil // Return success even if user is deleted
	}

	// Generate reset password token
	token, err := helper.EncryptResetPasswordToken(email, user.UserID, expiryMinutes)
	if err != nil {
		log.Printf("[ERROR] Failed to generate reset password token - Email: %s, UserID: %s, Error: %v", email, user.UserID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate reset password token")
	}

	// Create reset password link
	resetLink := fmt.Sprintf("%s?token=%s", resetPasswordURL, token)

	// Send reset password email
	if err = helper.SendResetPasswordEmail(s.emailCfg, email, user.Username, resetLink, expiryMinutes); err != nil {
		log.Printf("[ERROR] Failed to send reset password email - Email: %s, Username: %s, Error: %v", email, user.Username, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send reset password email")
	}

	return nil
}

// UpdatePassword updates user password using reset password token
// Validates token, expiry, and password match before updating
func (s *AuthService) UpdatePassword(token, newPassword, confirmPassword string) error {
	// Decrypt token to get email, user_id, and expiry
	email, userID, expiryTimestamp, err := helper.DecryptResetPasswordToken(token)
	if err != nil {
		if err.Error() == "token expired" {
			return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "reset password link has expired")
		}
		log.Printf("[ERROR] Failed to decrypt reset password token - Error: %v", err)
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid reset password link")
	}

	// Check if expiry is still valid (should already be checked in DecryptResetPasswordToken, but double-check)
	currentTime := time.Now().Unix()
	if expiryTimestamp < currentTime {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "reset password link has expired")
	}

	// Validate password match
	if newPassword != confirmPassword {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "new password and confirm password do not match")
	}

	// Validate password length
	if len(newPassword) < 6 {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "password must be at least 6 characters")
	}

	// Find user by email and user_id
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
		}
		log.Printf("[ERROR] Error finding user for password update - Email: %s, UserID: %s, Error: %v", email, userID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update password")
	}

	// Verify user_id matches
	if user.UserID != userID {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid reset password link")
	}

	// Check if user is active
	if !user.IsActive {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "user is inactive")
	}

	// Check if user is verified
	if !user.IsVerified {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "user is not verified")
	}

	// Check if user is deleted
	if user.DeletedAt != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// Check if new password is different from existing password
	if helper.CheckPasswordHash(newPassword, user.Password) {
		return NewServiceError(ErrInvalidCredentials, http.StatusBadRequest, "password must be different from existing password")
	}

	// Hash new password
	hashedPassword, err := helper.HashPassword(newPassword)
	if err != nil {
		log.Printf("[ERROR] Failed to hash password - UserID: %s, Error: %v", userID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update password")
	}

	// Update password in database
	if err = s.userRepo.UpdatePassword(userID, hashedPassword); err != nil {
		log.Printf("[ERROR] Failed to update password in database - UserID: %s, Error: %v", userID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update password")
	}

	return nil
}
