package handler

import (
	"database/sql"
	"log"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type TourPackageHandler struct {
	service *service.TourPackageService
}

func NewTourPackageHandler(service *service.TourPackageService) *TourPackageHandler {
	return &TourPackageHandler{
		service: service,
	}
}

func (h *TourPackageHandler) GetTourPackages(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	packages, err := h.service.GetTourPackages(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour packages retrieved successfully", packages)
}

func (h *TourPackageHandler) CreateTourPackage(c *fiber.Ctx) error {
	var req model.CreateTourPackageRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if validationErrors := helper.ValidateStruct(req); len(validationErrors) > 0 {
		return helper.SendValidationErrorResponse(c, validationErrors)
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		log.Printf("[ERROR] Organization ID not found in context - Path: %s", c.Path())
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		log.Printf("[ERROR] User ID not found in context - Path: %s", c.Path())
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := h.service.CreateTourPackage(c.Context(), &req, orgID, userID); err != nil {
		log.Printf("[ERROR] CreateTourPackage failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Tour package created successfully", nil)
}

func (h *TourPackageHandler) TourPackageDetail(c *fiber.Ctx) error {
	var req model.TourPackageDetailRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] BodyParser failed - Path: %s, Error: %v", c.Path(), err)
		return helper.BadRequestResponse(c, "Invalid request body")
	}
	if req.PackageID == "" {
		return helper.BadRequestResponse(c, "package_id is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.GetTourPackageDetail(c.Context(), orgID, req.PackageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return helper.SendErrorResponse(c, fiber.StatusNotFound, "Tour package not found")
		}
		log.Printf("[ERROR] TourPackageDetail failed - Path: %s, Error: %v", c.Path(), err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package detail loaded", res)
}
