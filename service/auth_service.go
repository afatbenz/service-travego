package service

import (
	"database/sql"
	"log"
	"net/http"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strings"
)

type AuthService struct {
	userRepo *repository.UserRepository
	emailCfg *configs.EmailConfig
}

func NewAuthService(userRepo *repository.UserRepository, emailCfg *configs.EmailConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		emailCfg: emailCfg,
	}
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
	} else {
		return "", NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "either email or token is required")
	}

	// Find user by email and verify user_id matches
	user, err := s.userRepo.FindByEmail(userEmail)
	if err != nil {
		return "", NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	// If using token, verify user_id from decrypt matches user_id in database
	if token != "" && email == "" {
		if user.UserID != userID {
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
