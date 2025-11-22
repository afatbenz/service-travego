package service

import (
	"errors"
	"net/http"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
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

	updatedUser, err := s.userRepo.Update(existingUser)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update profile")
	}

	return updatedUser, nil
}
