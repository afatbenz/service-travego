package handler

import (
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type LeaveManagementHandler struct {
	service *service.LeaveManagementService
}

func NewLeaveManagementHandler(s *service.LeaveManagementService) *LeaveManagementHandler {
	return &LeaveManagementHandler{service: s}
}

func (h *LeaveManagementHandler) GetLeaveTypes(c *fiber.Ctx) error {
	data, err := h.service.GetLeaveTypes()
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Leave types loaded", data)
}

func (h *LeaveManagementHandler) GetLeaveList(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	month := c.Query("month")
	year := c.Query("year")

	data, err := h.service.ListLeaveManagement(orgID, month, year)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Leave management loaded", data)
}

func (h *LeaveManagementHandler) UploadAttachment(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		file, err = c.FormFile("attachment")
		if err != nil || file == nil {
			file, err = c.FormFile("files")
			if err != nil || file == nil {
				return helper.BadRequestResponse(c, "file is required")
			}
		}
	}

	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, file.Filename)
	if err := c.SaveFile(file, tmpPath); err != nil {
		return helper.BadRequestResponse(c, "failed to save uploaded file")
	}
	defer os.Remove(tmpPath)

	path, filename, err := h.service.UploadAttachment(tmpPath, file.Filename)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Attachment uploaded", fiber.Map{
		"attachment_path": path,
		"filename":        filename,
	})
}

func (h *LeaveManagementHandler) CreateLeave(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return helper.BadRequestResponse(c, "missing user context")
	}

	var req model.LeaveManagementCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}
	if validationErrors := helper.ValidateStruct(&req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	req.EmployeeID = strings.TrimSpace(req.EmployeeID)
	req.SubstituteID = strings.TrimSpace(req.SubstituteID)
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.EndDate = strings.TrimSpace(req.EndDate)
	req.Reason = strings.TrimSpace(req.Reason)
	req.AttachmentPath = strings.TrimSpace(req.AttachmentPath)

	leaveID, err := h.service.CreateLeave(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Leave created", fiber.Map{
		"leave_id": leaveID,
	})
}
