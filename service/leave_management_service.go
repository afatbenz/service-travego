package service

import (
	"net/http"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"time"
)

type LeaveManagementService struct {
	repo *repository.LeaveManagementRepository
}

func NewLeaveManagementService(repo *repository.LeaveManagementRepository) *LeaveManagementService {
	return &LeaveManagementService{repo: repo}
}

func (s *LeaveManagementService) GetLeaveTypes() ([]model.LeaveManagementTypeItem, error) {
	return s.repo.ListLeaveTypes()
}

func (s *LeaveManagementService) ListLeaveManagement(orgID, month, year string) ([]model.LeaveManagementListItem, error) {
	var start *time.Time
	var end *time.Time

	if month != "" || year != "" {
		if month == "" || year == "" {
			return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "month and year is required")
		}

		mm, err := strconv.Atoi(month)
		if err != nil || mm < 1 || mm > 12 {
			return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid month")
		}

		yy, err := strconv.Atoi(year)
		if err != nil || yy < 1900 || yy > 2100 {
			return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid year")
		}

		startTime := time.Date(yy, time.Month(mm), 1, 0, 0, 0, 0, time.UTC)
		endTime := startTime.AddDate(0, 1, 0).Add(-time.Second)
		start = &startTime
		end = &endTime
	}

	return s.repo.ListEmployeeLeaves(orgID, start, end)
}
