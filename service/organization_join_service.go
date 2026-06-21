package service

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"time"

	"github.com/google/uuid"
)

type OrganizationJoinService struct {
	orgRepo         *repository.OrganizationRepository
	orgUserRepo     *repository.OrganizationUserRepository
	userRepo        *repository.UserRepository
	notificationSvc *NotificationService
	emailCfg        *configs.EmailConfig
}

func NewOrganizationJoinService(orgRepo *repository.OrganizationRepository, orgUserRepo *repository.OrganizationUserRepository, userRepo *repository.UserRepository, notificationSvc *NotificationService, emailCfg *configs.EmailConfig) *OrganizationJoinService {
	return &OrganizationJoinService{
		orgRepo:         orgRepo,
		orgUserRepo:     orgUserRepo,
		userRepo:        userRepo,
		notificationSvc: notificationSvc,
		emailCfg:        emailCfg,
	}
}

// JoinOrganization handles user joining an organization
func (s *OrganizationJoinService) JoinOrganization(userID, organizationCode string) error {
	org, err := s.orgRepo.FindByCode(organizationCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrUserNotFound, http.StatusBadRequest, "organization not found")
		}
		log.Printf("[ERROR] Error finding organization by code - Code: %s, Error: %v", organizationCode, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to find organization")
	}

	// Check if user already exists in organization_users for this organization
	exists, err := s.orgUserRepo.CheckUserInOrganization(userID, org.OrganizationId)
	if err != nil {
		log.Printf("[ERROR] Error checking user in organization - UserID: %s, OrgID: %s, Error: %v", userID, org.OrganizationId, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to check organization membership")
	}

	// Get current user for created_by
	currentUser, err := s.userRepo.FindByID(userID)
	if err != nil {
		log.Printf("[ERROR] Error finding current user - UserID: %s, Error: %v", userID, err)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to find user")
	}

	now := time.Now()

	if !exists {
		// User doesn't exist in organization, insert with role 1
		orgUser := &model.OrganizationUser{
			UUID:             uuid.New().String(),
			UserID:           userID,
			OrganizationID:   org.OrganizationId,
			OrganizationRole: 1,
			IsActive:         false,
			CreatedAt:        now,
			CreatedBy:        userID,
			UpdatedAt:        now,
			UpdatedBy:        userID,
		}

		if err = s.orgUserRepo.CreateOrganizationUser(orgUser); err != nil {
			log.Printf("[ERROR] Failed to create organization user - UserID: %s, OrgID: %s, Error: %v", userID, org.OrganizationId, err)
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to join organization")
		}
	} else {
		// User already exists, update role to 2
		if err = s.orgUserRepo.UpdateOrganizationUserRole(userID, org.OrganizationId, 2); err != nil {
			log.Printf("[ERROR] Failed to update organization user role - UserID: %s, OrgID: %s, Error: %v", userID, org.OrganizationId, err)
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update organization role")
		}
	}

	// Get base URL for approval link
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("APP_HOST")
	}
	approveURL := ""
	if baseURL != "" {
		approveURL = baseURL + "/dashboard/partner/organization/request-user"
	}

	// Track if we need to create notification
	notificationCreated := false

	// Get all users in the organization (excluding the current user)
	orgUsers, err := s.orgUserRepo.GetUsersByOrganizationID(org.OrganizationId)
	if err != nil {
		log.Printf("[ERROR] Failed to get organization users - OrgID: %s, Error: %v", org.OrganizationId, err)
		// Continue even if this fails, as the join was successful
	} else {
		// Send email to existing users (excluding current user) for approval
		for _, orgUser := range orgUsers {
			if orgUser.UserID != userID {
				// Get user details for email
				user, err := s.userRepo.FindByID(orgUser.UserID)
				if err != nil {
					log.Printf("[ERROR] Failed to get user for email - UserID: %s, Error: %v", orgUser.UserID, err)
					continue
				}

				// Send approval email
				if err = helper.SendJoinOrganizationApprovalEmail(s.emailCfg, user.Email, user.Username, currentUser.Username, org.OrganizationName, approveURL); err != nil {
					log.Printf("[ERROR] Failed to send approval email - Email: %s, Error: %v", user.Email, err)
					// Continue even if email fails
				} else if !notificationCreated && s.notificationSvc != nil {
					// Create notification only once after first successful email
					_, _ = s.notificationSvc.CreateNotification(org.OrganizationId, NotificationPayload{
						Title:   "Permintaan User Baru",
						Message: "Tinjau user sebelum memberikan persetujuan akses",
						URL:     approveURL,
					})
					notificationCreated = true
				}
			}
		}
	}

	return nil
}
