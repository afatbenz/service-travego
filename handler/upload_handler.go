package handler

import (
	"mime/multipart"
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

// UploadCommon handles POST /api/common/upload
func (h *UploadHandler) UploadCommon(c *fiber.Ctx) error {
	uploadType := c.FormValue("type")
	if uploadType == "" {
		return helper.BadRequestResponse(c, "type is required")
	}

	// Validate type
	validTypes := []string{"armada", "package", "order", "content"}
	isValid := false
	for _, vt := range validTypes {
		if uploadType == vt {
			isValid = true
			break
		}
	}
	if !isValid {
		return helper.BadRequestResponse(c, "type must be one of: armada, package, order, content")
	}

	// Support multiple files
	var files []*multipart.FileHeader
	if form, err := c.MultipartForm(); err == nil && form != nil {
		if f := form.File["files"]; len(f) > 0 {
			files = f
		}
	}
	if len(files) == 0 {
		if f, err := c.FormFile("files"); err == nil && f != nil {
			files = []*multipart.FileHeader{f}
		}
	}
	if len(files) == 0 {
		return helper.BadRequestResponse(c, "files is required")
	}

	tempDir := os.TempDir()
	uploaded := make([]string, 0, len(files))
	for _, file := range files {
		tempFilePath := filepath.Join(tempDir, file.Filename)
		if err := c.SaveFile(file, tempFilePath); err != nil {
			return helper.BadRequestResponse(c, "failed to save uploaded file")
		}
		// Upload with compression if needed
		filePath, err := h.uploadService.UploadCommon(tempFilePath, uploadType)
		// Cleanup temp
		os.Remove(tempFilePath)
		if err != nil {
			statusCode := service.GetStatusCode(err)
			return helper.SendErrorResponse(c, statusCode, err.Error())
		}
		uploaded = append(uploaded, filePath)
	}

	responseData := map[string]interface{}{
		"files":     uploaded,
		"count":     len(uploaded),
		"first_url": uploaded[0],
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Files uploaded successfully", responseData)
}

type deleteFilesPayload struct {
	Pathfile string   `json:"pathfile"`
	Files    []string `json:"files"`
}

func (h *UploadHandler) DeleteFilesCommon(c *fiber.Ctx) error {
	var payload deleteFilesPayload
	if err := c.BodyParser(&payload); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	var paths []string
	if len(payload.Files) > 0 {
		paths = payload.Files
	}
	if payload.Pathfile != "" {
		paths = append(paths, payload.Pathfile)
	}
	if len(paths) == 0 {
		return helper.BadRequestResponse(c, "pathfile or files is required")
	}
	deleted, failed, err := h.uploadService.DeleteFiles(paths)
	if err != nil {
		statusCode := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, statusCode, err.Error())
	}
	resp := map[string]interface{}{
		"deleted":       deleted,
		"failed":        failed,
		"count_deleted": len(deleted),
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Files deleted successfully", resp)
}
