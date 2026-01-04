package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"time"

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
	encryptedOrderID := c.Params("encryptOrderId")
	if encryptedOrderID == "" {
		return helper.BadRequestResponse(c, "encryptOrderId is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.GetOrderDetail(encryptedOrderID, orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order detail retrieved", res)
}

func (h *OrderHandler) FindOrder(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if orderID == "" {
		return helper.BadRequestResponse(c, "order_id is required")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.FindOrderDetail(orderID, orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order detail retrieved", res)
}

func (h *OrderHandler) CreateOrderPayment(c *fiber.Ctx) error {
	var req model.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if req.Token == "" || req.PaymentMethod == "" || req.PaymentType == 0 {
		return helper.BadRequestResponse(c, "Required fields missing")
	}

	if orgID, ok := c.Locals("organization_id").(string); ok {
		req.OrganizationID = orgID
	} else {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.CreateOrderPayment(&req)
	if err != nil {
		fmt.Println("Error creating payment:", err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Payment created", res)
}

func (h *OrderHandler) GetPaymentMethods(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	res, err := h.service.GetPaymentMethods(orgID)
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Payment methods retrieved", res)
}

func (h *OrderHandler) ConfirmPayment(c *fiber.Ctx) error {
	var req model.PaymentConfirmationRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid payload")
	}

	if req.Token == "" || req.OrderType == "" {
		return helper.BadRequestResponse(c, "Required fields missing")
	}

	if orgID, ok := c.Locals("organization_id").(string); ok {
		req.OrganizationID = orgID
	} else {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	err := h.service.ConfirmPayment(&req)
	if err != nil {
		fmt.Println("Error confirming payment:", err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Payment confirmed", nil)
}

func (h *OrderHandler) UploadPaymentEvidence(c *fiber.Ctx) error {
	token := c.FormValue("token")
	if token == "" {
		return helper.BadRequestResponse(c, "Token is required")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return helper.BadRequestResponse(c, "Image file is required")
	}

	// Decrypt Token
	decrypted, err := helper.DecryptString(token)
	if err != nil {
		return helper.BadRequestResponse(c, "Invalid token")
	}

	var orderID string
	var payload model.OrderTokenPayload
	if err := json.Unmarshal([]byte(decrypted), &payload); err == nil && payload.OrderID != "" {
		orderID = payload.OrderID
	} else {
		orderID = decrypted
	}

	organizationID, ok := c.Locals("organization_id").(string)
	if !ok || organizationID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	// Ensure directory exists
	uploadDir := "config/payment-attachment"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create upload directory")
	}

	// Generate Filename: {order_id}-{YYMMDDHHmm}.ext
	timestamp := time.Now().Format("0601021504")
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s-%s%s", orderID, timestamp, ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save File
	if err := c.SaveFile(file, filePath); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to save file")
	}

	// Update DB
	if err := h.service.UploadPaymentEvidence(orderID, organizationID, filePath); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Payment evidence uploaded successfully", nil)
}
