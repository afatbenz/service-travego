package handler

import (
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
	}
}

// UploadPhoto handles POST /api/upload/photo and /api/upload/avatar
func (h *UploadHandler) UploadPhoto(c *fiber.Ctx) error {
	// Get upload type from form
	uploadType := c.FormValue("upload-type")
	if uploadType == "" {
		return helper.BadRequestResponse(c, "upload-type is required")
	}

	// Validate upload type
	validTypes := []string{"profile-user", "icon-company", "content-thumbnail"}
	isValid := false
	for _, validType := range validTypes {
		if uploadType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		return helper.BadRequestResponse(c, "upload-type must be one of: profile-user, icon-company, content-thumbnail")
	}

	// Get file from form
	file, err := c.FormFile("filepath")
	if err != nil {
		// If file is not in form, try to get from "file" field
		file, err = c.FormFile("file")
		if err != nil {
			return helper.BadRequestResponse(c, "file is required")
		}
	}

	// Create temporary file to save uploaded file
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, file.Filename)

	// Save uploaded file to temporary location
	if err := c.SaveFile(file, tempFilePath); err != nil {
		return helper.BadRequestResponse(c, "failed to save uploaded file")
	}

	// Defer cleanup of temporary file
	defer os.Remove(tempFilePath)

	// Upload photo using service
	filePath, err := h.uploadService.UploadPhoto(tempFilePath, uploadType)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}

	responseData := map[string]interface{}{
		"filepath":  filePath,
		"full_path": filePath, // Full path with APP_HOST prefix
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Photo uploaded successfully", responseData)
}
