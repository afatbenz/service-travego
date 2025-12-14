package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
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

// UploadCommon uploads a file to assets/<type> and compresses images larger than 2MB
func (s *UploadService) UploadCommon(sourceFilePath, uploadType string) (string, error) {
	uploadTypeEnum := configs.UploadType(uploadType)
	if !uploadTypeEnum.IsValid() {
		return "", NewServiceError(errors.New("invalid upload type"), http.StatusBadRequest, "invalid upload type")
	}

	storagePath := uploadTypeEnum.GetStoragePath()
	if storagePath == "" {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "invalid storage path")
	}

	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		return "", NewServiceError(errors.New("source file does not exist"), http.StatusBadRequest, "source file does not exist")
	}

	storageDir := strings.TrimPrefix(storagePath, "/")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to create storage directory: %v", err))
	}

	ext := strings.ToLower(filepath.Ext(sourceFilePath))
	timestamp := time.Now().Unix()
	baseName := fmt.Sprintf("%s-%d", uploadType, timestamp)
	destExt := ext

	// Determine file size
	info, err := os.Stat(sourceFilePath)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to stat source file: %v", err))
	}

	maxBytes := int64(2 * 1024 * 1024)
	destinationPath := filepath.Join(storageDir, baseName+destExt)

	if info.Size() <= maxBytes {
		// Copy directly
		if err := copyFile(sourceFilePath, destinationPath); err != nil {
			return "", err
		}
	} else {
		// Compress image
		// Decode image
		src, err := os.Open(sourceFilePath)
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to open source file: %v", err))
		}
		defer src.Close()

		img, format, err := image.Decode(src)
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to decode image: %v", err))
		}

		// Prefer JPEG for compression
		qualityLevels := []int{92, 88, 84, 80, 76, 72}
		var out []byte
		for i, q := range qualityLevels {
			buf := &bytes.Buffer{}
			if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: q}); err != nil {
				return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to encode jpeg: %v", err))
			}
			if int64(buf.Len()) <= maxBytes || i == len(qualityLevels)-1 {
				out = buf.Bytes()
				break
			}
		}

		// If original was PNG, switch extension to .jpg for compressed file
		if format == "png" || ext == ".png" {
			destExt = ".jpg"
			destinationPath = filepath.Join(storageDir, baseName+destExt)
		}

		if err := os.WriteFile(destinationPath, out, 0644); err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to write compressed file: %v", err))
		}
	}

	fullPath := storagePath + "/" + baseName + destExt
	return helper.GetAssetURL(fullPath), nil
}

func (s *UploadService) DeleteFiles(paths []string) ([]string, []string, error) {
	if len(paths) == 0 {
		return nil, nil, NewServiceError(errors.New("empty paths"), http.StatusBadRequest, "paths is required")
	}
	appHost := os.Getenv("APP_HOST")
	deleted := make([]string, 0, len(paths))
	failed := make([]string, 0)
	for _, p := range paths {
		if p == "" {
			failed = append(failed, p)
			continue
		}
		stripped := p
		if appHost != "" && strings.HasPrefix(stripped, appHost) {
			stripped = strings.TrimPrefix(stripped, appHost)
		}
		if strings.HasPrefix(stripped, "http://") || strings.HasPrefix(stripped, "https://") {
			failed = append(failed, p)
			continue
		}
		if !strings.HasPrefix(stripped, "/assets/") && !strings.HasPrefix(stripped, "assets/") {
			failed = append(failed, p)
			continue
		}
		local := strings.TrimPrefix(stripped, "/")
		local = filepath.FromSlash(local)
		if _, err := os.Stat(local); err != nil {
			if os.IsNotExist(err) {
				failed = append(failed, p)
				continue
			}
			return deleted, failed, NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to stat file: %v", err))
		}
		if err := os.Remove(local); err != nil {
			failed = append(failed, p)
			continue
		}
		deleted = append(deleted, p)
	}
	return deleted, failed, nil
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to open source file: %v", err))
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to create destination file: %v", err))
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath)
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to copy file: %v", err))
	}
	return nil
}
