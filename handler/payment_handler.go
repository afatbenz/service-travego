package handler

import (
	"fmt"
	"service-travego/model"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// PaymentHandler menangani request HTTP untuk payment
type PaymentHandler struct {
	paymentService service.PaymentService
}

// NewPaymentHandler membuat instance baru dari PaymentHandler
func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePayment menangani POST /api/services/order/payment
func (h *PaymentHandler) CreatePayment(c *fiber.Ctx) error {
	var req model.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Extract from locals (populated by DualAuthMiddleware or JWT middleware)
	if orgID := c.Locals("organization_id"); orgID != nil {
		req.OrganizationID = fmt.Sprintf("%v", orgID)
	}
	if userID := c.Locals("user_id"); userID != nil {
		req.UserID = fmt.Sprintf("%v", userID)
	}

	fmt.Printf("[DEBUG] CreatePayment - req: %+v\n", req)

	// Validation
	if req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order_id is required"})
	}
	if req.OrderType != 1 && req.OrderType != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_type. Must be 1 (fleet) or 2 (tour package)"})
	}

	resp, err := h.paymentService.CreatePayment(&req)
	if err != nil {
		fmt.Println("Error creating payment:", err)
		if err.Error() == "invalid payment type: 0" || err.Error() == "invalid payment type" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// PaymentNotifications menangani POST /api/services/order/payment/webhook
func (h *PaymentHandler) PaymentNotifications(c *fiber.Ctx) error {
	var req model.MidtransWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook payload"})
	}

	err := h.paymentService.PaymentNotifications(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(model.WebhookResponse{Message: "OK"})
}

func (h *PaymentHandler) HandlePaymentNotification(c *fiber.Ctx) error {
	var req model.MidtransWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	body := string(c.Body())
	fmt.Printf("[WEBHOOK LOG] Received Request Body: %s\n", body)

	err := h.paymentService.ProcessPaymentNotification(&req)
	if err != nil {
		fmt.Println("Error processing payment notification:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to process payment notification: %v", err),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Payment notification processed successfully",
	})
}
