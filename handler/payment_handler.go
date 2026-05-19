package handler

import (
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

	resp, err := h.paymentService.CreatePayment(&req)
	if err != nil {
		// Jika error karena payment_type invalid, return 400
		if err.Error() == "invalid payment type: 0" || err.Error() == "invalid payment type" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		// Untuk error lainnya return 500 atau sesuaikan
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// HandleWebhook menangani POST /api/services/order/payment/webhook
func (h *PaymentHandler) HandleWebhook(c *fiber.Ctx) error {
	var req model.MidtransWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid webhook payload"})
	}

	err := h.paymentService.HandleWebhook(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(model.WebhookResponse{Message: "OK"})
}
