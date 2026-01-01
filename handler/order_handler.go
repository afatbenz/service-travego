package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(service *service.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) GetFleetOrderSummary(c *fiber.Ctx) error {
	var req model.OrderFleetSummaryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if req.FleetID == "" || req.PriceID == "" {
		return helper.BadRequestResponse(c, "fleet_id and price_id are required")
	}

	res, err := h.service.GetFleetOrderSummary(&req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order summary retrieved", res)
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req model.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	// Basic Validation
	if req.FleetID == "" || req.PriceID == "" || req.Qty < 1 {
		return helper.BadRequestResponse(c, "Required fields missing or invalid")
	}

	if orgID, ok := c.Locals("organization_id").(string); ok {
		req.OrganizationID = orgID
	}
	if orgCode, ok := c.Locals("organization_code").(string); ok {
		req.OrganizationCode = orgCode
	}

	res, err := h.service.CreateOrder(&req)
	if err != nil {
		fmt.Println("Error creating order:", err)
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order created successfully", res)
}

func (h *OrderHandler) GetOrderList(c *fiber.Ctx) error {
	var req model.GetOrderListRequest
	if err := c.QueryParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid query parameters")
	}

	if orgID, ok := c.Locals("organization_id").(string); ok {
		req.OrganizationID = orgID
	} else {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.GetOrderList(&req)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order list retrieved", res)
}

func (h *OrderHandler) GetOrderDetail(c *fiber.Ctx) error {
	orderID := c.Params("orderid")
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}

	res, err := h.service.GetOrderDetail(orderID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order detail retrieved", res)
}
