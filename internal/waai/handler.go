package waai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"service-travego/internal/wagy"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// Handler handles all WhatsApp AI webhook requests
type Handler struct {
	config         *Config
	wagyClient     *wagy.WagyClient
	aiClient       *AIClient
	tenantRepo     *TenantRepository
	sessionMgr     *SessionManager
	asstCustRepo   *AssistantCustomerRepository
	clientRegistry *WagyClientRegistry
}

// NewHandler creates a new webhook handler
func NewHandler(cfg *Config, db *sql.DB, dbDriver string, rdb *redis.Client) *Handler {
	authMgr := NewAuthManager(rdb)
	wagyClient := wagy.NewWagyClient(cfg.WagyDeviceID, cfg.WagyToken)

	return &Handler{
		config:         cfg,
		wagyClient:     wagyClient,
		asstCustRepo:   NewAssistantCustomerRepository(db, dbDriver),
		aiClient:       NewAIClient(cfg.AnthropicAPIKey, db, dbDriver, rdb, wagyClient),
		tenantRepo:     NewTenantRepository(db, dbDriver, authMgr),
		sessionMgr:     NewSessionManager(rdb),
		clientRegistry: NewWagyClientRegistry(),
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
// Ini adalah single entry point untuk SEMUA device (Skenario 1: TraveGO ERP, Skenario 2: Company Assistants)
func (h *Handler) HandleWebhookPOST(c *fiber.Ctx) error {
	signature := c.Get("X-Wagy-Signature")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing X-Wagy-Signature header",
		})
	}

	rawBody := c.Body()

	if !VerifySignature(rawBody, signature, h.config.WagyWebhookSecret) {
		log.Printf("[WAAI] Signature verification failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	var payload WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		log.Printf("[WAAI] Failed to parse payload: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid payload",
		})
	}

	if payload.Event != "message.received" || payload.Source != "whatsapp" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ignored"})
	}

	customerPhone := ExtractPhoneNumber(payload.Data.Content.PhoneJID)
	ownerPhone := ExtractPhoneNumber(payload.Data.OwnerJID)
	messageText := payload.Data.Content.Message
	wagyDeviceID := payload.Data.DeviceID

	log.Printf("[WAAI] Event=message.received | wagy_device=%s | owner=%s | from=%s | msg=%s",
		wagyDeviceID, ownerPhone, customerPhone, messageText)

	switch {
	case ownerPhone == h.config.ServiceAccount:
		h.processERPAssistant(customerPhone, messageText)
	default:
		h.processCompanyAssistant(customerPhone, ownerPhone, messageText)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "received"})
}

// processERPAssistant — Skenario 1: User mengirim ke bot perusahaan TraveGO
func (h *Handler) processERPAssistant(customerPhone, messageText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.tenantRepo.GetTenantByPhone(ctx, customerPhone)
	if err != nil {
		if isCapabilitiesQuestion(messageText) || isIdentityOrDeveloperQuestion(messageText) || isRegistrationQuestion(messageText) {
			go h.processMessageAsync(customerPhone, messageText)
			return
		}
		replyText := buildUnregisteredReply(messageText)
		_ = h.sendMessage(customerPhone, replyText)
		return
	}

	go h.processMessageAsync(customerPhone, messageText)
}

// processCompanyAssistant — Skenario 2: Customer mengirim ke nomor perusahaan customer
func (h *Handler) processCompanyAssistant(customerPhone, ownerPhone, messageText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	asstCust, found, err := h.asstCustRepo.FindByDeviceID(ctx, ownerPhone)
	if err != nil {
		log.Printf("[WAAI][Company] DB error for %s: %v", ownerPhone, err)
		return
	}
	if !found {
		log.Printf("[WAAI][Company] Ignored: %s not registered in assistant_customers", ownerPhone)
		return
	}

	log.Printf("[WAAI][Company] org=%s | assistant_device=%s | from=%s",
		asstCust.OrganizationID, asstCust.AssistantDeviceID, customerPhone)

	go h.processCompanyMessageAsync(customerPhone, messageText, asstCust)
}

// processCompanyMessageAsync memproses pesan untuk company assistant (Skenario 2)
func (h *Handler) processCompanyMessageAsync(customerPhone, messageText string, asstCust *AssistantCustomer) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sendClient := h.clientRegistry.GetClient(asstCust.DeviceID, asstCust.DeviceToken)
	if sendClient == nil {
		fmt.Println("sendClient is nil -- deviceToken ", asstCust.DeviceToken)
		fmt.Println("sendClient is nil -- deviceID ", asstCust.DeviceID)
		log.Printf("[WAAI][Company] Cannot get WagyClient for device %s", asstCust.DeviceID)
		finalResponse := "Maaf, layanan assistant sedang tidak tersedia. Silakan hubungi kantor langsung."
		_ = h.sendMessage(customerPhone, finalResponse)
		return
	}

	tenant := &TenantInfo{
		OrganizationID:   asstCust.OrganizationID,
		OrganizationName: "",
		Role:             "CustomerAssistant",
	}

	snapshot := make(map[string]interface{})
	if tenant.OrganizationID != "" {
		ctxAuth := withAuthorizedTenantContext(ctx, tenant)
		var err error
		snapshot, err = h.tenantRepo.GetOrganizationSnapshot(ctxAuth, tenant.OrganizationID)
		if err != nil {
			log.Printf("[WAAI][Company] Failed snapshot org %s: %v", tenant.OrganizationID, err)
			snapshot = map[string]interface{}{}
		}
		if tenant.OrganizationName == "" {
			if name, ok := snapshot["organization_name"].(string); ok && name != "" {
				tenant.OrganizationName = name
			}
		}
	}

	history, err := h.sessionMgr.LoadSessionFor(ctx, asstCust.OrganizationID, customerPhone)
	if err != nil {
		log.Printf("[WAAI][Company] Failed load session: %v", err)
		history = []ConversationMessage{}
	}

	if len(history) > 20 {
		history = history[len(history)-20:]
	}

	history = append(history, ConversationMessage{
		Role:    "user",
		Content: messageText,
	})

	// Set customer phone in context for phone validation in tools (get_order_list, get_order_detail, print_invoice)
	ctx = context.WithValue(ctx, phoneKey, customerPhone)

	systemPrompt := h.aiClient.BuildCompanySystemPrompt(tenant, snapshot, messageText, asstCust.DeviceName)

	// Use Company-specific AI method with restricted tool definitions
	finalResponse, err := h.aiClient.callAnthropicWithCompanyTools(ctx, systemPrompt, history)
	if err != nil {
		log.Printf("[WAAI][Company] AI error: %v", err)
		finalResponse = "Maaf, layanan sedang sibuk. Silakan coba lagi."
	}
	finalResponse = formatWhatsAppReply(finalResponse)

	history = append(history, ConversationMessage{
		Role:    "assistant",
		Content: finalResponse,
	})
	_ = h.sessionMgr.SaveSessionFor(ctx, asstCust.OrganizationID, customerPhone, history)

	if err := h.sendMessageWithClient(customerPhone, finalResponse, sendClient); err != nil {
		log.Printf("[WAAI][Company] Failed send via device %s: %v", asstCust.AssistantDeviceID, err)
	}

	log.Printf("[WAAI][Company] Reply sent | device=%s | to=%s", asstCust.AssistantDeviceID, customerPhone)
}

func (h *Handler) sendMessageWithClient(phone, message string, client *wagy.WagyClient) error {
	if client == nil {
		return fmt.Errorf("WagyClient is nil")
	}
	_, err := client.SendMessage(phone, message)
	if err != nil {
		return err
	}
	log.Printf("[WAAI] Message sent to %s", phone)
	return nil
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
		fmt.Printf("Error sending message to %s: %v", phone, err)
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

	// Register both legacy and API-prefixed routes to avoid breaking existing integrations.
	for _, basePath := range []string{"/waai", "/api/waai"} {
		waaiGroup := app.Group(basePath)
		waaiGroup.Get("/webhook", handler.HandleWebhookGET)
		waaiGroup.Post("/webhook", handler.HandleWebhookPOST)

		// Admin routes (should be protected in production)
		adminGroup := waaiGroup.Group("/admin")
		adminGroup.Delete("/session/:phone", handler.ClearSessionHandler)
		adminGroup.Get("/health", handler.HealthCheck)
	}

	log.Println("[WAAI] Routes registered successfully")
	return nil
}

func buildUnregisteredReply(messageText string) string {
	if isIdentityOrDeveloperQuestion(messageText) {
		return "Halo! Saya Trave AI Assistant Travego.\n\n" +
			"Trave AI Assistant Travego diciptakan oleh Afatbenz Tech.\n" +
			"Untuk diskusi lebih lanjut, Anda bisa hubungi 6281335884729 atau kunjungi mafatichulfuadi.com.\n\n" +
			"Jika Anda ingin mengetahui lebih lanjut tentang layanan Travego, saya siap membantu."
	}
	if isRegistrationQuestion(messageText) {
		return "Untuk mendaftar dan menikmati layanan AI Assistant, silakan register di platform https://www.travego.id lalu tambahkan nomor WhatsApp Anda di menu Pengaturan > AI Assistant."
	}

	return "Halo! \nMaaf, sepertinya nomor Anda belum terdaftar di sistem kami.\n\n" +
		"Ingin mengoptimalkan operasional bisnis transportasi Anda dengan bantuan AI Assistant dan sistem ERP Travego? " +
		"Segera daftar dan nikmati kemudahannya.\n\n" +
		"Informasi lebih lanjut, silakan kunjungi:\n" +
		"Website: http://www.travego.id\n" +
		"Whatsapp: 6281335884729\n\n" +
		"Terimakasih"
}

func isIdentityOrDeveloperQuestion(messageText string) bool {
	text := strings.ToLower(strings.TrimSpace(messageText))
	if text == "" {
		return false
	}

	keywords := []string{
		"kamu siapa",
		"siapa kamu",
		"siapa anda",
		"nama kamu",
		"nama anda",
		"asisten apa",
		"assistant apa",
		"siapa developer",
		"siapa pencipta",
		"siapa pembuat",
		"dibuat oleh siapa",
		"diciptakan oleh siapa",
		"developer kamu",
		"pencipta kamu",
		"pembuat kamu",
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func isRegistrationQuestion(messageText string) bool {
	text := strings.ToLower(strings.TrimSpace(messageText))
	if text == "" {
		return false
	}

	keywords := []string{
		"cara daftar",
		"bagaimana daftar",
		"cara register",
		"bagaimana register",
		"cara menikmati layanan",
		"menikmati layanan ai assistant",
		"cara pakai ai assistant",
		"cara menggunakan ai assistant",
		"daftar ai assistant",
		"register ai assistant",
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func isCapabilitiesQuestion(messageText string) bool {
	text := strings.ToLower(strings.TrimSpace(messageText))
	if text == "" {
		return false
	}

	keywords := []string{
		"bisa apa",
		"apa yang bisa",
		"apa saja yang bisa",
		"kemampuan kamu",
		"fitur apa",
		"fitur asisten",
		"bantuan",
		"help",
		"layanan apa",
		"fungsi kamu",
		"apa kegunaan",
		"manfaat kamu",
		"tugas kamu",
		"ngapain aja",
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}
