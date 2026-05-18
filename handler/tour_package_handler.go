package handler

import (
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type TourPackageHandler struct {
	service *service.TourPackageService
}

func NewTourPackageHandler(s *service.TourPackageService) *TourPackageHandler {
	return &TourPackageHandler{service: s}
}

func (h *TourPackageHandler) GetTourPackages(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.GetTourPackages(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "OK", items)
}

func (h *TourPackageHandler) GetTourPackageOrderList(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.GetTourPackageOrderList(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "OK", items)
}

func (h *TourPackageHandler) CreateTourPackage(c *fiber.Ctx) error {
	var req model.CreateTourPackageRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.CreateTourPackage(c.Context(), &req, orgID, userID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package created", nil)
}

func (h *TourPackageHandler) UpdateTourPackage(c *fiber.Ctx) error {
	var req model.UpdateTourPackageRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.UpdateTourPackage(c.Context(), &req, orgID, userID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package updated", nil)
}

func (h *TourPackageHandler) TourPackageDetail(c *fiber.Ctx) error {
	var req model.TourPackageDetailRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	res, err := h.service.GetTourPackageDetail(c.Context(), orgID, req.PackageID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "OK", res)
}

func (h *TourPackageHandler) SetTourPackageActiveStatus(c *fiber.Ctx) error {
	var req model.TourPackageActiveStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.SetTourPackageActiveStatus(c.Context(), orgID, userID, req.Action, req.PackageID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Status updated", nil)
}

func (h *TourPackageHandler) DeleteTourPackage(c *fiber.Ctx) error {
	packageID := c.Params("packageid")
	if packageID == "" {
		return helper.BadRequestResponse(c, "package_id is required")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.DeleteTourPackage(c.Context(), orgID, userID, packageID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Tour package deleted", nil)
}

func (h *TourPackageHandler) CreateTourPackageOrder(c *fiber.Ctx) error {
	var req model.TourPackageOrderCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orderID, err := h.service.CreateTourPackageOrder(c.Context(), orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order created", fiber.Map{
		"order_id": orderID,
	})
}

func (h *TourPackageHandler) UpdateTourPackageOrder(c *fiber.Ctx) error {
	var req model.TourPackageOrderUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	orgID, _ := c.Locals("organization_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.UpdateTourPackageOrder(c.Context(), orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order updated", nil)
}

func (h *TourPackageHandler) GetTourPackageOrderDetail(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	res, err := h.service.GetTourPackageOrderDetail(c.Context(), orgID, orderID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "OK", res)
}
