package service

import (
	"log"
	"net/http"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
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

func (s *AuthService) Register(username, email, password string) (*model.User, error) {
	existingUser, _ := s.userRepo.FindByEmail(email)
	if existingUser != nil {
		return nil, NewServiceError(ErrEmailExists, http.StatusConflict, "email already exists")
	}

	existingUsername, _ := s.userRepo.FindByUsername(username)
	if existingUsername != nil {
		return nil, NewServiceError(ErrUsernameExists, http.StatusConflict, "username already exists")
	}

	hashedPassword, err := helper.HashPassword(password)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to hash password")
	}

	userID := helper.GenerateUUID()

	user := &model.User{
		UserID:   userID,
		Username: username,
		Email:    email,
		Password: hashedPassword,
		Status:   2,
	}

	user, err = s.userRepo.Create(user)
	if err != nil {
		log.Printf("[ERROR] Failed to create user - Username: %s, Email: %s, Error: %v", username, email, err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create user")
	}

	otp := helper.GenerateOTP()

	if err = helper.SetOTP(email, otp); err != nil {
		log.Printf("[ERROR] Failed to store OTP - Email: %s, Error: %v", email, err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to store OTP")
	}

	if err = helper.SendOTPEmail(s.emailCfg, email, username, otp); err != nil {
		log.Printf("[ERROR] Failed to send OTP email - Email: %s, Username: %s, Error: %v", email, username, err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send OTP email")
	}

	return user, nil
}

func (s *AuthService) VerifyOTP(email, otp string) error {
	storedOTP, err := helper.GetOTP(email)
	if err != nil {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid or expired OTP")
	}

	if storedOTP != otp {
		return NewServiceError(ErrInvalidOTP, http.StatusBadRequest, "invalid OTP")
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return NewServiceError(ErrUserNotFound, http.StatusNotFound, "user not found")
	}

	if err = s.userRepo.UpdateStatus(user.UserID, 1); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update user status")
	}

	helper.DeleteOTP(email)

	if err = helper.SendRegisterSuccessEmail(s.emailCfg, email, user.Username); err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to send success email")
	}

	return nil
}
