package waai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// Handler handles WhatsApp AI webhook requests
type Handler struct {
	config      *Config
	wagyClient  *WagyClient
	aiClient    *AIClient
	tenantRepo  *TenantRepository
	sessionMgr  *SessionManager
}

// NewHandler creates a new webhook handler
func NewHandler(cfg *Config, db *sql.DB, dbDriver string, rdb *redis.Client) *Handler {
	return &Handler{
		config:     cfg,
		wagyClient: NewWagyClient(cfg.WagyDeviceID, cfg.WagyToken),
		aiClient:   NewAIClient(cfg.AnthropicAPIKey, db, dbDriver, rdb),
		tenantRepo: NewTenantRepository(db, dbDriver),
		sessionMgr: NewSessionManager(rdb),
	}
}

// HandleWebhookGET handles Wagy webhook URL verification
// Wagy sends a GET request with a challenge parameter during registration
func (h *Handler) HandleWebhookGET(c *fiber.Ctx) error {
	challenge := c.Query("challenge")
	if challenge == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing challenge parameter")
	}

	// Return the challenge as plain text
	c.Set("Content-Type", "text/plain")
	return c.SendString(challenge)
}

// HandleWebhookPOST handles incoming WhatsApp messages from Wagy
func (h *Handler) HandleWebhookPOST(c *fiber.Ctx) error {
	// Verify signature
	signature := c.Get("X-Wagy-Signature")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing X-Wagy-Signature header",
		})
	}

	// Read request body
	rawBody := c.Body()

	// Verify signature
	if !VerifySignature(rawBody, signature, h.config.WagyWebhookSecret) {
		log.Printf("Signature verification failed for phone webhook")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	// Parse webhook payload
	var payload WebhookPayload
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		log.Printf("Failed to parse webhook payload: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid payload",
		})
	}

	// Filter: only process message.received events from WhatsApp
	if payload.Event != "message.received" || payload.Source != "whatsapp" {
		// Ignore other events
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ignored",
		})
	}

	// Extract phone number
	phone := ExtractPhoneNumber(payload.Data.Content.PhoneJID)
	messageText := payload.Data.Content.Message

	log.Printf("[WAAI] Incoming message from %s: %s", phone, messageText)

	// Check if tenant exists (quick check)
	_, err = h.tenantRepo.GetTenantByPhone(phone)
	if err != nil {
		// Tenant not found - send error message and return
		log.Printf("[WAAI] Tenant not found for phone: %s", phone)
		replyText := "Maaf, nomor Anda belum terdaftar dalam sistem. Hubungi administrator untuk pendaftaran."
		_ = h.sendMessage(phone, replyText)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "tenant_not_found",
		})
	}

	// Process message in background goroutine (must return quickly)
	go h.processMessageAsync(phone, messageText)

	// Return 200 immediately to Wagy
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "received",
	})
}

// processMessageAsync processes the message asynchronously
func (h *Handler) processMessageAsync(phone, messageText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Process message with AI
	response, err := h.aiClient.ProcessMessage(ctx, phone, messageText)
	if err != nil {
		log.Printf("[WAAI] Error processing message from %s: %v", phone, err)
		response = "Maaf, terjadi kesalahan saat memproses permintaan Anda. Silakan coba lagi."
	}

	// Send response
	if err := h.sendMessage(phone, response); err != nil {
		log.Printf("[WAAI] Error sending message to %s: %v", phone, err)
	}
}

// sendMessage sends a message via Wagy API
func (h *Handler) sendMessage(phone, message string) error {
	_, err := h.wagyClient.SendMessage(phone, message)
	if err != nil {
		return err
	}
	log.Printf("[WAAI] Message sent to %s", phone)
	return nil
}

// HealthCheck returns the health status of the WAAI module
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
		"module": "waai",
	})
}

// ClearSessionHandler clears the conversation history for a phone number
// This is for admin use only
func (h *Handler) ClearSessionHandler(c *fiber.Ctx) error {
	phone := c.Params("phone")
	if phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Phone parameter is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.sessionMgr.ClearSession(ctx, phone)
	if err != nil {
		log.Printf("[WAAI] Error clearing session for %s: %v", phone, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to clear session",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "session cleared",
	})
}

// RegisterRoutes registers all WAAI routes with the Fiber app
func RegisterRoutes(app *fiber.App, cfg *Config, db *sql.DB, dbDriver string, rdb *redis.Client) error {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create handler
	handler := NewHandler(cfg, db, dbDriver, rdb)

	// Webhook routes (public, no auth required)
	waaiGroup := app.Group("/waai")
	waaiGroup.Get("/webhook", handler.HandleWebhookGET)
	waaiGroup.Post("/webhook", handler.HandleWebhookPOST)

	// Admin routes (should be protected in production)
	adminGroup := waaiGroup.Group("/admin")
	adminGroup.Delete("/session/:phone", handler.ClearSessionHandler)
	adminGroup.Get("/health", handler.HealthCheck)

	log.Println("[WAAI] Routes registered successfully")
	return nil
}
