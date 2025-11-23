package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"service-travego/configs"
	"service-travego/helper"
	"strings"
	"time"
)

type UploadService struct{}

func NewUploadService() *UploadService {
	return &UploadService{}
}

// UploadPhoto uploads a photo from filepath to the appropriate storage path based on upload type
func (s *UploadService) UploadPhoto(sourceFilePath, uploadType string) (string, error) {
	// Validate upload type
	uploadTypeEnum := configs.UploadType(uploadType)
	if !uploadTypeEnum.IsValid() {
		return "", NewServiceError(errors.New("invalid upload type"), http.StatusBadRequest, "invalid upload type")
	}

	// Get storage path based on upload type
	storagePath := uploadTypeEnum.GetStoragePath()
	if storagePath == "" {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "invalid storage path")
	}

	// Check if source file exists
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		return "", NewServiceError(errors.New("source file does not exist"), http.StatusBadRequest, "source file does not exist")
	}

	// Ensure storage directory exists
	storageDir := strings.TrimPrefix(storagePath, "/")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to create storage directory: %v", err))
	}

	// Generate unique filename: {upload-type}-timestamp
	ext := filepath.Ext(sourceFilePath)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s-%d%s", uploadType, timestamp, ext)
	destinationPath := filepath.Join(storageDir, filename)

	// Copy file from source to destination
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to open source file: %v", err))
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to create destination file: %v", err))
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		os.Remove(destinationPath) // Clean up on error
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to copy file: %v", err))
	}

	// Return the full path with APP_HOST prefix
	fullPath := storagePath + "/" + filename
	return helper.GetAssetURL(fullPath), nil
}
