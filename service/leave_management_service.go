package service

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
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

func (s *LeaveManagementService) CreateLeave(organizationID, userID string, req *model.LeaveManagementCreateRequest) (string, error) {
	employeeID := strings.TrimSpace(req.EmployeeID)
	substituteID := strings.TrimSpace(req.SubstituteID)
	if employeeID == "" || substituteID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "employee_id and substitute_id is required")
	}

	employeeExists, err := s.repo.EmployeeUUIDExists(organizationID, employeeID)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate employee_id")
	}
	if !employeeExists {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "employee_id not found")
	}

	subExists, err := s.repo.EmployeeUUIDExists(organizationID, substituteID)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate substitute_id")
	}
	if !subExists {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "substitute_id not found")
	}

	startDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.StartDate))
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "start_date must be YYYY-MM-DD")
	}
	endDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.EndDate))
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "end_date must be YYYY-MM-DD")
	}
	if endDate.Before(startDate) {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "end_date must be greater than or equal start_date")
	}

	leaveID := uuid.New().String()
	if err := s.repo.CreateEmployeeLeave(leaveID, organizationID, employeeID, substituteID, startDate, endDate, req.LeaveType, time.Now(), userID); err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create leave")
	}
	return leaveID, nil
}

func (s *LeaveManagementService) UploadAttachment(sourceFilePath, originalFilename string) (string, string, error) {
	storageDir := filepath.FromSlash("assets/common/leave")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create storage directory")
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(sourceFilePath))
	}
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".pdf" {
		return "", "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "attachment must be image or pdf")
	}

	info, err := os.Stat(sourceFilePath)
	if err != nil {
		return "", "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to stat file")
	}

	maxBytes := int64(2 * 1024 * 1024)
	filename := "leave-" + uuid.New().String() + ext
	destPath := filepath.Join(storageDir, filename)

	if info.Size() <= maxBytes {
		if err := copyLocalFile(sourceFilePath, destPath); err != nil {
			return "", "", err
		}
		return "/assets/common/leave/" + filename, filename, nil
	}

	if ext == ".pdf" {
		if err := api.OptimizeFile(sourceFilePath, destPath, nil); err != nil {
			_ = os.Remove(destPath)
			return "", "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to compress pdf")
		}
		if st, err := os.Stat(destPath); err != nil || st.Size() > maxBytes {
			_ = os.Remove(destPath)
			return "", "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "file size exceeds 2MB")
		}
		return "/assets/common/leave/" + filename, filename, nil
	}

	outBytes, outExt, err := compressImageUnderLimit(sourceFilePath, ext, maxBytes)
	if err != nil {
		return "", "", err
	}

	if outExt != ext {
		filename = "leave-" + uuid.New().String() + outExt
		destPath = filepath.Join(storageDir, filename)
	}
	if err := os.WriteFile(destPath, outBytes, 0644); err != nil {
		return "", "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to save attachment")
	}
	return "/assets/common/leave/" + filename, filename, nil
}

func copyLocalFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to open source file")
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create destination file")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(dstPath)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to copy file")
	}
	return nil
}

func compressImageUnderLimit(sourcePath, ext string, maxBytes int64) ([]byte, string, error) {
	f, err := os.Open(sourcePath)
	if err != nil {
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to open image")
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid image")
	}

	targetExt := ext
	if format == "png" || ext == ".png" {
		targetExt = ".jpg"
	}

	qualityLevels := []int{92, 88, 84, 80, 76, 72, 68, 64, 60, 56, 52, 48, 44, 40, 36, 32, 28, 24, 20}
	for _, q := range qualityLevels {
		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to encode image")
		}
		if int64(buf.Len()) <= maxBytes {
			return buf.Bytes(), targetExt, nil
		}
	}

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 20}); err != nil {
		return nil, "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to encode image")
	}
	if int64(buf.Len()) > maxBytes {
		return nil, "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "file size exceeds 2MB")
	}
	return buf.Bytes(), targetExt, nil
}
