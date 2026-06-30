package waai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"service-travego/configs"
	"service-travego/internal/wagy"
	"service-travego/model"
	"service-travego/repository"
	"service-travego/service"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	phoneKey contextKey = "phone"
)

// AIClient handles communication with AI provider (Anthropic / Gemini)
type AIClient struct {
	apiKey                string
	model                 string
	fallbackModels        []string
	baseURL               string
	provider              string // "anthropic" or "gemini"
	db                    *sql.DB
	driver                string
	authMgr               *AuthManager
	tenantRepo            *TenantRepository
	sessionMgr            *SessionManager
	toolExec              *ToolExecutor
	fleetService          *service.FleetService
	fleetUnitService      *service.FleetUnitService
	partnerService        *service.PartnerService
	generalService        *service.GeneralService
	preferenceCityService *service.PreferenceCityService
	customersService      *service.CustomersService
	organizationService   *service.OrganizationService
	scheduleService       *service.ScheduleService
	orderService          *service.OrderService
	dashboardService      *service.DashboardService
	transactionService    *service.TransactionService
	inventoryService      *service.InventoryService
	garageService         *service.GarageService
	printService          *service.PrintManagementService
	wagyClient            *wagy.WagyClient
}

// NewAIClient creates a new AI client (supports Anthropic or Gemini)
func NewAIClient(apiKey string, db *sql.DB, dbDriver string, rdb *redis.Client, wagyClient *wagy.WagyClient) *AIClient {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if provider == "" {
		provider = "anthropic"
	}

	var model, baseURL string
	var fallbackModels []string

	switch provider {
	case "gemini":
		fallbackModels = buildGeminiModelFallbacks()
		model = fallbackModels[0]
		baseURL = os.Getenv("GEMINI_API_URL")
		if baseURL == "" {
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		baseURL = strings.TrimRight(baseURL, "/")
		if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
			apiKey = geminiKey
		}
	default:
		provider = "anthropic"
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-sonnet-4-6"
		}
		baseURL = os.Getenv("ANTHROPIC_API_URL")
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		baseURL = strings.TrimRight(baseURL, "/")
	}

	authMgr := NewAuthManager(rdb)
	fleetRepo := repository.NewFleetRepository(db, dbDriver)
	fleetUnitRepo := repository.NewFleetUnitRepository(db, dbDriver)
	generalRepo := repository.NewGeneralRepository(db, dbDriver)
	preferenceCityRepo := repository.NewPreferenceCityRepository(db, dbDriver)
	customersRepo := repository.NewCustomersRepository(db, dbDriver)
	partnerRepo := repository.NewPartnerRepository(db, dbDriver)
	orgRepo := repository.NewOrganizationRepository(db, dbDriver)
	userRepo := repository.NewUserRepository(db, dbDriver)
	scheduleRepo := repository.NewScheduleRepository(db, dbDriver)
	contentRepo := repository.NewContentRepository(db, dbDriver)
	dashboardRepo := repository.NewDashboardRepository(db, dbDriver)
	transactionRepo := repository.NewTransactionRepository(db, dbDriver)
	printRepo := repository.NewPrintManagementRepository(db, dbDriver)
	inventoryRepo := repository.NewInventoryRepository(db, dbDriver)
	garageRepo := repository.NewGarageRepository(db, dbDriver)

	// Load minimal email config for OrderService
	emailCfg := &configs.EmailConfig{
		From:     os.Getenv("EMAIL_FROM"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		SMTPHost: os.Getenv("EMAIL_SMTP_HOST"),
		SMTPPort: os.Getenv("EMAIL_SMTP_PORT"),
	}

	notificationSvc := service.NewNotificationService(db, dbDriver)

	return &AIClient{
		apiKey: apiKey,
		model:  model,
		fallbackModels: func() []string {
			if len(fallbackModels) > 1 {
				return fallbackModels[1:]
			}
			return nil
		}(),
		baseURL:               baseURL,
		provider:              provider,
		db:                    db,
		driver:                dbDriver,
		authMgr:               authMgr,
		tenantRepo:            NewTenantRepository(db, dbDriver, authMgr),
		sessionMgr:            NewSessionManager(rdb),
		toolExec:              NewToolExecutor(db, dbDriver),
		fleetService:          service.NewFleetService(fleetRepo),
		fleetUnitService:      service.NewFleetUnitService(fleetUnitRepo, partnerRepo, orgRepo),
		generalService:        service.NewGeneralService("config/general-config.json", "config/web-menu.json", "config/location.json", generalRepo),
		preferenceCityService: service.NewPreferenceCityService(preferenceCityRepo, "config/location.json"),
		customersService:      service.NewCustomersService(customersRepo),
		organizationService:   service.NewOrganizationService(orgRepo, userRepo),
		scheduleService:       service.NewScheduleService(scheduleRepo),
		orderService:          service.NewOrderService(fleetRepo, contentRepo, orgRepo, emailCfg),
		dashboardService:      service.NewDashboardService(dashboardRepo),
		transactionService:    service.NewTransactionService(transactionRepo, notificationSvc),
		inventoryService:      service.NewInventoryService(inventoryRepo, notificationSvc),
		garageService:         service.NewGarageService(garageRepo),
		printService:          service.NewPrintManagementService(printRepo),
		wagyClient:            wagyClient,
	}
}

// AnthropicRequest represents the request body for Anthropic API
type AnthropicRequest struct {
	Model        string                `json:"model"`
	MaxTokens    int                   `json:"max_tokens"`
	SystemPrompt string                `json:"system"`
	Messages     []ConversationMessage `json:"messages"`
	Tools        []ToolDefinition      `json:"tools"`
	Stream       bool                  `json:"stream"`
}

// AnthropicResponse represents the response from Anthropic API
type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ProcessMessage processes an incoming WhatsApp message
func (ac *AIClient) ProcessMessage(ctx context.Context, phone, incomingMessage string) (finalResponse string, retErr error) {
	// Get tenant info
	tenant, err := ac.tenantRepo.GetTenantByPhone(ctx, phone)
	var snapshot map[string]interface{}
	var statOrganizationID string

	defer func() {
		if statOrganizationID == "" {
			return
		}
		status := 1
		if retErr != nil {
			status = 0
		}
		insertAssistantAccountStat(ac.db, ac.driver, statOrganizationID, 1, status)
	}()

	if err == nil {
		// Tenant found - proceed with tenant context
		statOrganizationID = tenant.OrganizationID
		ctx = withAuthorizedTenantContext(ctx, tenant)

		// Get business snapshot
		snapshot, err = ac.tenantRepo.GetOrganizationSnapshot(ctx, tenant.OrganizationID)
		if err != nil {
			snapshot = map[string]interface{}{} // Use empty snapshot if error
		}
		if tenant.OrganizationName == "" {
			if name, ok := snapshot["organization_name"].(string); ok && name != "" {
				tenant.OrganizationName = name
			}
		}
	} else {
		// Tenant not found - handle as guest
		log.Printf("[WAAI][AI] Processing message for unregistered phone: %s", phone)
		tenant = &TenantInfo{
			Phone: phone,
			Role:  "Guest",
		}
		snapshot = map[string]interface{}{}
	}

	// Add phone number to context so executeTool can use it
	ctx = context.WithValue(ctx, phoneKey, phone)

	// Load conversation history
	history, err := ac.sessionMgr.LoadSession(ctx, phone)
	if err != nil {
		retErr = fmt.Errorf("failed to load session: %w", err)
		return "", retErr
	}

	// Limit history to last 20 messages to balance context while avoiding outdated data
	if len(history) > 20 {
		history = history[len(history)-20:]
	}

	// Add user message to history
	userMsg := ConversationMessage{
		Role:    "user",
		Content: incomingMessage,
	}
	history = append(history, userMsg)

	// Build system prompt
	systemPrompt := ac.buildSystemPrompt(tenant, snapshot)

	// Call Anthropic API with tool support
	finalResponse, err = ac.callAnthropicWithTools(ctx, systemPrompt, history)
	if err != nil {
		retErr = fmt.Errorf("anthropic call failed: %w", err)
		return "", retErr
	}
	finalResponse = formatWhatsAppReply(finalResponse)

	// Save updated history
	assistantMsg := ConversationMessage{
		Role:    "assistant",
		Content: finalResponse,
	}
	history = append(history, assistantMsg)

	err = ac.sessionMgr.SaveSession(ctx, phone, history)
	if err != nil {
		// Log but don't fail - message should still be sent
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}

	return finalResponse, nil
}

// callAnthropicWithTools calls Anthropic API and handles tool use
func (ac *AIClient) callAnthropicWithTools(ctx context.Context, systemPrompt string, messages []ConversationMessage) (string, error) {
	lastTextResponse := ""

	for i := 0; i < 5; i++ { // Max 5 iterations to prevent infinite loops
		response, err := ac.callAnthropic(ctx, systemPrompt, messages)
		if err != nil {
			return "", err
		}
		log.Printf("[WAAI][AI] Iteration %d stop_reason=%s content=%s", i+1, response.StopReason, summarizeAnthropicContent(response.Content))

		hasToolUse := false
		textResponse := ""
		assistantBlocks := make([]map[string]interface{}, 0, len(response.Content))

		for _, content := range response.Content {
			switch content.Type {
			case "text":
				if content.Text != "" {
					if textResponse != "" {
						textResponse += "\n"
					}
					textResponse += content.Text
					assistantBlocks = append(assistantBlocks, map[string]interface{}{
						"type": "text",
						"text": content.Text,
					})
				}
			case "tool_use":
				hasToolUse = true
				assistantBlocks = append(assistantBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    content.ID,
					"name":  content.Name,
					"input": json.RawMessage(content.Input),
				})
			}
		}

		if textResponse != "" {
			lastTextResponse = textResponse
		}

		if len(assistantBlocks) > 0 {
			messages = append(messages, ConversationMessage{
				Role:    "assistant",
				Content: assistantBlocks,
			})
		}

		for _, content := range response.Content {
			if content.Type != "tool_use" {
				continue
			}

			toolResult := ac.executeTool(ctx, content.Name, content.Input)
			toolResultMsg := ConversationMessage{
				Role: "user",
				Content: []map[string]interface{}{
					{
						"type":        "tool_result",
						"tool_use_id": content.ID,
						"content":     formatToolResult(toolResult),
					},
				},
			}
			messages = append(messages, toolResultMsg)
		}

		if !hasToolUse {
			if textResponse != "" {
				return textResponse, nil
			}
		}
	}

	if lastTextResponse != "" {
		return lastTextResponse, nil
	}

	// Force one final pass without tools so the model must answer with text.
	finalResponse, err := ac.callAnthropicFinal(ctx, systemPrompt, messages)
	if err == nil {
		log.Printf("[WAAI][AI] Final no-tools pass stop_reason=%s content=%s", finalResponse.StopReason, summarizeAnthropicContent(finalResponse.Content))
		for _, content := range finalResponse.Content {
			if content.Type == "text" && content.Text != "" {
				return content.Text, nil
			}
		}
	}

	return "", fmt.Errorf("max tool use iterations reached without text response")
}

// callAnthropic makes a single call to Anthropic API
func (ac *AIClient) callAnthropic(ctx context.Context, systemPrompt string, messages []ConversationMessage) (*AnthropicResponse, error) {
	return ac.callAnthropicRequest(ctx, systemPrompt, messages, GetToolDefinitions())
}

// callAnthropicCompany makes a single call to Anthropic API using Company Assistant tool definitions
func (ac *AIClient) callAnthropicCompany(ctx context.Context, systemPrompt string, messages []ConversationMessage) (*AnthropicResponse, error) {
	return ac.callAnthropicRequest(ctx, systemPrompt, messages, getCompanyAssistantToolDefinitions())
}

// callAnthropicWithCompanyTools calls Anthropic API with tool support, using Company Assistant tool definitions.
// Mirrors callAnthropicWithTools but uses getCompanyToolDefinitions for restricted tool access.
func (ac *AIClient) callAnthropicWithCompanyTools(ctx context.Context, systemPrompt string, messages []ConversationMessage) (finalResponse string, updatedMessages []ConversationMessage, retErr error) {
	lastTextResponse := ""
	latestUserMessage := latestUserText(messages)
	toolRequired := companyMessageNeedsTool(latestUserMessage)
	createOrderToolRequired := false
	if isAffirmationReply(latestUserMessage) && conversationSuggestsCreateOrderFlow(messages) {
		toolRequired = true
		createOrderToolRequired = true
	}
	correctionInjected := false
	createOrderForceAttempted := false
	toolUsed := false
	toolContextNotes := make([]string, 0, 4)
	createOrderSucceeded := false
	createOrderFailed := false
	createOrderMissing := []string{}
	createOrderError := ""
	createOrderID := ""
	bankAccountsBlocked := false
	bankAccountsLoaded := false
	createdOrderStatus := 0
	createdOrderStatusKnown := false
	tripMinDays := 0
	tripDistanceKm := 0.0
	tripMinDaysOverland := 0
	fleetInfoLoaded := false
	fleetPriceLoaded := false
	preferenceCitiesChecked := false
	cityListUnfiltered := false
	knownOrderIDs := map[string]struct{}{}
	knownOrderIDsLoaded := false
	orgID, _ := getAuthorizedContextValues(ctx)

	defer func() {
		if orgID == "" {
			return
		}
		status := 1
		if retErr != nil {
			status = 0
		}
		insertAssistantCustomerStat(ac.db, ac.driver, orgID, 1, status)
	}()

	for i := 0; i < 5; i++ {
		response, err := ac.callAnthropicCompany(ctx, systemPrompt, messages)
		if err != nil {
			retErr = err
			return "", messages, retErr
		}
		log.Printf("[WAAI][Company] Iteration %d stop_reason=%s content=%s", i+1, response.StopReason, summarizeAnthropicContent(response.Content))

		hasToolUse := false
		textResponse := ""
		assistantBlocks := make([]map[string]interface{}, 0, len(response.Content))

		for _, content := range response.Content {
			switch content.Type {
			case "text":
				if content.Text != "" {
					if textResponse != "" {
						textResponse += "\n"
					}
					textResponse += content.Text
					assistantBlocks = append(assistantBlocks, map[string]interface{}{
						"type": "text",
						"text": content.Text,
					})
				}
			case "tool_use":
				hasToolUse = true
				assistantBlocks = append(assistantBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    content.ID,
					"name":  content.Name,
					"input": json.RawMessage(content.Input),
				})
			}
		}

		if textResponse != "" {
			lastTextResponse = textResponse
		}
		if hasToolUse {
			toolUsed = true
		}

		if len(assistantBlocks) > 0 {
			messages = append(messages, ConversationMessage{
				Role:    "assistant",
				Content: assistantBlocks,
			})
		}

		for _, content := range response.Content {
			if content.Type != "tool_use" {
				continue
			}

			log.Printf("[WAAI][Company] Executing tool name=%s input=%s", content.Name, truncateResponseBody(content.Input))
			toolResult := ac.executeTool(ctx, content.Name, content.Input)
			formattedToolResult := formatToolResult(toolResult)
			log.Printf("[WAAI][Company] Tool result name=%s output=%s", content.Name, truncateResponseBody([]byte(formattedToolResult)))
			toolContextNotes = append(toolContextNotes, fmt.Sprintf("Tool %s result: %s", content.Name, formattedToolResult))
			if content.Name == "create_order" {
				createOrderSucceeded, createOrderFailed, createOrderMissing, createOrderError, createOrderID = analyzeCreateOrderToolResult(toolResult)
				if createOrderSucceeded && !createdOrderStatusKnown && strings.TrimSpace(createOrderID) != "" && ac != nil && ac.fleetService != nil {
					if detail, err := ac.fleetService.GetPartnerOrderDetail(strings.TrimSpace(createOrderID), orgID); err == nil && detail != nil {
						createdOrderStatus = detail.Status
						createdOrderStatusKnown = true
					}
				}
			}
			if content.Name == "get_order_list" {
				knownOrderIDs = extractOrderIDsFromOrderListToolResult(toolResult)
				knownOrderIDsLoaded = true
			}
			if content.Name == "get_bank_accounts" {
				bankAccountsBlocked = bankAccountsBlocked || bankAccountsToolResultBlocked(toolResult)
				bankAccountsLoaded = true
			}
			if content.Name == "get_fleet_list" || content.Name == "get_fleet_detail" || content.Name == "get_fleet_units" {
				fleetInfoLoaded = true
			}
			if content.Name == "get_fleet_prices" {
				fleetPriceLoaded = true
			}
			if content.Name == "get_trip_distance" {
				if v, ok := toolResult.(map[string]interface{}); ok {
					if n := getIntParam(v, "min_days"); n > 0 {
						tripMinDays = n
					}
					if n := getIntParam(v, "min_days_overland"); n > 0 {
						tripMinDaysOverland = n
					}
					if f := getFloatParam(v, "distance_km"); f > 0 {
						tripDistanceKm = f
					}
				}
			}
			if content.Name == "get_preference_cities" {
				preferenceCitiesChecked = true
			}
			if content.Name == "get_city_list" {
				var inputParams map[string]interface{}
				_ = json.Unmarshal(content.Input, &inputParams)
				searchText := strings.TrimSpace(getStringValue(inputParams["search"]))
				if searchText == "" {
					if n := sliceLen(toolResult); n > 50 {
						cityListUnfiltered = true
					}
				}
			}

			toolResultMsg := ConversationMessage{
				Role: "user",
				Content: []map[string]interface{}{
					{
						"type":        "tool_result",
						"tool_use_id": content.ID,
						"content":     formattedToolResult,
					},
				},
			}
			messages = append(messages, toolResultMsg)
		}

		if !hasToolUse {
			if textResponse != "" {
				if !createOrderSucceeded && !createOrderForceAttempted && (containsOrderIDCandidate(textResponse) || companyResponseClaimsOrderSuccess(textResponse) || companyResponseClaimsOrderAccepted(textResponse)) {
					createOrderForceAttempted = true
					forcedCtx := context.WithValue(ctx, contextSuppressAdminNotify, true)
					forcedResult := ac.executeTool(forcedCtx, "create_order", json.RawMessage(`{}`))
					formattedForced := formatToolResult(forcedResult)
					toolContextNotes = append(toolContextNotes, fmt.Sprintf("Tool create_order result (forced): %s", formattedForced))
					toolUsed = true
					createOrderSucceeded, createOrderFailed, createOrderMissing, createOrderError, createOrderID = analyzeCreateOrderToolResult(forcedResult)
					if createOrderSucceeded && !createdOrderStatusKnown && strings.TrimSpace(createOrderID) != "" && ac != nil && ac.fleetService != nil {
						if detail, err := ac.fleetService.GetPartnerOrderDetail(strings.TrimSpace(createOrderID), orgID); err == nil && detail != nil {
							createdOrderStatus = detail.Status
							createdOrderStatusKnown = true
						}
					}
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "SISTEM: Anda menyebut status/nomor pesanan tanpa hasil tool create_order. Sistem memanggil create_order dengan input {}. Hasil: " + formattedForced + "\n\nInstruksi: Jika missing_required muncul, ajukan tepat satu pertanyaan klarifikasi yang paling penting agar create_order bisa berhasil. Jika status success, sebut order_id dari hasil tool. Jangan menyebut rekening bank kecuali status pesanan = 1 (Pesanan Dikonfirmasi) dan nomor rekening berasal dari tool get_bank_accounts.",
					})
					continue
				}
				if createOrderToolRequired && !createOrderSucceeded && !createOrderForceAttempted {
					createOrderForceAttempted = true
					forcedCtx := context.WithValue(ctx, contextSuppressAdminNotify, true)
					forcedResult := ac.executeTool(forcedCtx, "create_order", json.RawMessage(`{}`))
					formattedForced := formatToolResult(forcedResult)
					toolContextNotes = append(toolContextNotes, fmt.Sprintf("Tool create_order result (forced): %s", formattedForced))
					toolUsed = true
					createOrderSucceeded, createOrderFailed, createOrderMissing, createOrderError, createOrderID = analyzeCreateOrderToolResult(forcedResult)
					if createOrderSucceeded && !createdOrderStatusKnown && strings.TrimSpace(createOrderID) != "" && ac != nil && ac.fleetService != nil {
						if detail, err := ac.fleetService.GetPartnerOrderDetail(strings.TrimSpace(createOrderID), orgID); err == nil && detail != nil {
							createdOrderStatus = detail.Status
							createdOrderStatusKnown = true
						}
					}
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "SISTEM: Customer sudah konfirmasi tetapi Anda belum memanggil tool create_order. Sistem memanggil create_order dengan input {}. Hasil: " + formattedForced + "\n\nInstruksi: Jika missing_required muncul, ajukan tepat satu pertanyaan klarifikasi yang paling penting agar create_order bisa berhasil. Jika status success, sebut order_id dari hasil tool. Jangan menyebut rekening bank kecuali status pesanan = 1 (Pesanan Dikonfirmasi) dan nomor rekening berasal dari tool get_bank_accounts.",
					})
					continue
				}
				if responseContainsBankAccountDetails(textResponse) && !bankAccountsLoaded {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan menyebut nomor rekening atau detail bank tanpa memanggil tool get_bank_accounts. Jika customer menanyakan pembayaran, jelaskan singkat bahwa info pembayaran akan diberikan setelah pesanan dibuat (create_order) dan dikonfirmasi (status=1).",
					})
					continue
				}
				if responseContainsBankAccountDetails(textResponse) && (createOrderToolRequired && !createOrderSucceeded) {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Customer sudah konfirmasi, tetapi create_order belum berhasil. Jangan kirim rekening bank sebelum create_order status success. Panggil create_order sekarang (atau minta 1 data wajib yang paling penting jika masih kurang).",
					})
					continue
				}
				if responseContainsBankAccountDetails(textResponse) && !createOrderSucceeded && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan kirim rekening bank sebelum tool create_order berhasil (status success) dan pesanan berstatus 1 (Pesanan Dikonfirmasi). Panggil create_order sekarang atau minta 1 data wajib yang paling penting jika masih kurang.",
					})
					continue
				}
				if responseContainsBankAccountDetails(textResponse) && createdOrderStatusKnown && createdOrderStatus != 1 {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan kirim rekening bank jika status pesanan bukan 1 (Pesanan Dikonfirmasi). Sampaikan bahwa pembayaran dapat dilakukan setelah pesanan dikonfirmasi oleh tim.",
					})
					continue
				}
				if createOrderFailed && !createOrderSucceeded && (companyResponseClaimsOrderSuccess(textResponse) || companyResponseClaimsOrderAccepted(textResponse) || containsOrderIDCandidate(textResponse)) {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: buildCreateOrderFailureCorrection(createOrderMissing, createOrderError),
					})
					continue
				}
				if !createOrderSucceeded && !createOrderFailed && createOrderToolRequired && companyResponseClaimsOrderSuccess(textResponse) && containsOrderIDCandidate(textResponse) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Anda mengklaim pesanan berhasil dibuat dan menyebut nomor pesanan, tetapi tool create_order belum berhasil. Panggil tool create_order sekarang (atau minta 1 data wajib yang paling penting jika masih kurang). Jangan mengarang nomor pesanan.",
					})
					continue
				}
				if !createOrderSucceeded && !createOrderFailed && createOrderToolRequired && companyResponseClaimsOrderAccepted(textResponse) && containsOrderIDCandidate(textResponse) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Anda menyatakan pesanan sudah dicatat dan menyebut nomor pesanan, tetapi tool create_order belum dipanggil/berhasil. Panggil tool create_order sekarang (atau minta 1 data wajib yang paling penting jika masih kurang). Jangan mengarang nomor pesanan.",
					})
					continue
				}
				if createOrderSucceeded && strings.TrimSpace(createOrderID) != "" {
					unknown := extractUnknownOrderIDsFromText(textResponse, map[string]struct{}{createOrderID: {}})
					if len(unknown) > 0 {
						correctionInjected = true
						messages = append(messages, ConversationMessage{
							Role:    "user",
							Content: "PERBAIKAN WAJIB: Anda menyebut order_id yang tidak sama dengan hasil tool create_order. Jangan mengarang nomor pesanan. Gunakan hanya order_id dari hasil tool create_order.",
						})
						continue
					}
				}
				if !createOrderSucceeded && (createOrderToolRequired || conversationSuggestsCreateOrderFlow(messages)) && textContainsFleetOrderID(textResponse) {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan memberikan atau memilih nomor pesanan dari histori (get_order_list) ketika customer sedang meminta dibuatkan pesanan baru. Nomor pesanan hanya boleh disebut jika berasal dari hasil tool create_order (status success). Jika pesanan baru belum berhasil dibuat, minta data yang kurang atau jelaskan kendalanya.",
					})
					continue
				}
				if bankAccountsBlocked && responseContainsBankAccountDetails(textResponse) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan kirim nomor rekening ke customer jika pesanan berstatus 2 (belum dikonfirmasi). Sampaikan bahwa pembayaran dapat dilakukan setelah pesanan selesai ditinjau dan dikonfirmasi oleh tim.",
					})
					continue
				}
				if tripMinDays > 1 && responseClaimsOneDay(textResponse) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: fmt.Sprintf("PERBAIKAN WAJIB: Berdasarkan perhitungan jarak, minimal durasi sewa adalah %d hari. Jangan mengatakan bisa 1 hari / sehari. Jelaskan singkat ke customer bahwa minimal durasi %d hari.", tripMinDays, tripMinDays),
					})
					continue
				}
				if durationQuestionNeedsValidation(latestUserMessage) && !preferenceCitiesChecked && tripDistanceKm == 0 && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Pertanyaan customer tentang durasi/minimal hari harus divalidasi. Gunakan get_city_list dengan parameter search untuk mencari city_id tujuan, lalu panggil get_preference_cities (city_id). Jika kota tujuan tidak ada di preference, panggil get_trip_distance lalu terapkan aturan minimal hari (overland vs drop only).",
					})
					continue
				}
				if cityListUnfiltered && durationQuestionNeedsValidation(latestUserMessage) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan panggil get_city_list tanpa search karena hasilnya terlalu besar dan tidak spesifik. Panggil lagi get_city_list dengan search=nama kota tujuan.",
					})
					continue
				}
				if tripDistanceKm > 250 && tripMinDaysOverland >= 2 && durationQuestionOneDay(latestUserMessage) && responseClaimsOneDayPossible(textResponse) && !mentionsDropOnlyContext(latestUserMessage) && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Untuk jarak > 250 km, pulang-pergi (Overland) minimal 2 hari. Jangan bilang '1 hari memungkinkan' untuk Overland. Tanyakan apakah kebutuhannya *Drop Only* (antar/jemput saja) atau *Overland* (pulang-pergi). Jika Drop Only dan jarak 250-400 km, 1 hari masih memungkinkan.",
					})
					continue
				}
				if responseContainsSeatCapacity(textResponse) && !fleetInfoLoaded && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Customer menanyakan kapasitas/seat. Jangan mengarang kapasitas (angka seat/kursi). Panggil get_fleet_list untuk menemukan armada yang dimaksud, lalu get_fleet_detail untuk membaca kapasitas resmi. Jika armada belum jelas, ajukan 1 pertanyaan klarifikasi (Big Bus yang mana).",
					})
					continue
				}
				if responseContainsUnitRecommendation(textResponse) && !fleetInfoLoaded && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan merekomendasikan jumlah unit (mis. 2 unit/3 unit) tanpa data kapasitas armada dari tool. Panggil get_fleet_list/get_fleet_detail dulu atau minta klarifikasi armada yang dipilih.",
					})
					continue
				}
				if responseContainsPriceAmount(textResponse) && !fleetPriceLoaded && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "PERBAIKAN WAJIB: Jangan menyebut harga (Rp ...) tanpa hasil tool get_fleet_prices. Panggil get_fleet_prices atau minta data yang diperlukan (armada & jenis layanan).",
					})
					continue
				}
				if createOrderToolRequired && !createOrderSucceeded && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "Customer sudah menyetujui untuk dibuatkan pesanan. Panggil tool create_order sekarang menggunakan data yang sudah ada di percakapan. Jika ada data wajib yang belum lengkap, ajukan tepat satu pertanyaan klarifikasi yang paling penting. Jangan mengatakan pesanan sedang diproses / berhasil sebelum create_order status success.",
					})
					continue
				}
				if knownOrderIDsLoaded && len(knownOrderIDs) > 0 {
					unknown := extractUnknownOrderIDsFromText(textResponse, knownOrderIDs)
					if len(unknown) > 0 {
						correctionInjected = true
						messages = append(messages, ConversationMessage{
							Role:    "user",
							Content: "PERBAIKAN WAJIB: Anda menyebut order_id yang TIDAK ADA di hasil tool get_order_list. Jangan mengarang order_id. Gunakan hanya order_id yang ada di hasil tool atau minta customer menyebut order_id yang dimaksud.",
						})
						continue
					}
					if companyResponseClaimsOrderNumber(textResponse) && len(knownOrderIDs) > 1 && !textContainsFleetOrderID(latestUserMessage) {
						correctionInjected = true
						messages = append(messages, ConversationMessage{
							Role:    "user",
							Content: "PERBAIKAN WAJIB: Hasil get_order_list berisi lebih dari satu pesanan. Jangan memilih salah satu secara acak. Minta customer menyebutkan order_id yang dimaksud (atau tanggal sewa) dengan tepat satu pertanyaan klarifikasi.",
						})
						continue
					}
				}
				if toolRequired && !toolUsed && !correctionInjected {
					correctionInjected = true
					messages = append(messages, ConversationMessage{
						Role:    "user",
						Content: "Jawaban terakhir belum valid untuk customer. Jangan tampilkan instruksi internal, daftar kemampuan, atau pembahasan soal tool. Jika pertanyaan customer membutuhkan data perusahaan, order, lokasi, harga, rekening, armada, invoice, atau validasi durasi (mis. '1 hari'), panggil tool yang sesuai sekarang. Untuk validasi durasi: cek get_preference_cities dulu; jika kota tujuan tidak ada, panggil get_trip_distance lalu ikuti aturan minimal hari. Jika parameter tool belum cukup, ajukan tepat satu pertanyaan klarifikasi yang paling penting dalam Bahasa Indonesia.",
					})
					continue
				}
				return textResponse, messages, nil
			}
		}
	}

	if lastTextResponse != "" {
		return lastTextResponse, messages, nil
	}

	finalMessages := messages
	if len(toolContextNotes) > 0 {
		finalMessages = append(finalMessages, ConversationMessage{
			Role:    "user",
			Content: "Gunakan hasil tool berikut sebagai sumber data final. Jangan panggil tool lagi. Jawab langsung ke customer dalam Bahasa Indonesia dengan singkat dan natural.\n\n" + strings.Join(toolContextNotes, "\n\n"),
		})
	}

	finalAnthropicResponse, err := ac.callAnthropicFinal(ctx, systemPrompt, finalMessages)
	if err == nil {
		log.Printf("[WAAI][Company] Final no-tools pass stop_reason=%s content=%s", finalAnthropicResponse.StopReason, summarizeAnthropicContent(finalAnthropicResponse.Content))
		for _, content := range finalAnthropicResponse.Content {
			if content.Type == "text" && content.Text != "" {
				return content.Text, finalMessages, nil
			}
		}
	}

	retErr = fmt.Errorf("max tool use iterations reached without text response")
	return "", finalMessages, retErr
}

func (ac *AIClient) callAnthropicFinal(ctx context.Context, systemPrompt string, messages []ConversationMessage) (*AnthropicResponse, error) {
	return ac.callAnthropicRequest(ctx, systemPrompt, messages, nil)
}

// ——— Gemini provider ———

// callGeminiRequest calls Gemini API and converts to AnthropicResponse for unified tool-loop handling.
func (ac *AIClient) callGeminiRequest(ctx context.Context, systemPrompt string, messages []ConversationMessage, tools []ToolDefinition, noTools bool) (*AnthropicResponse, error) {
	// Konversi ConversationMessage ke Gemini contents[]
	gContents := make([]map[string]interface{}, 0, len(messages)+1)

	// System prompt Gemini dikirim sebagai user message pertama (atau via system_instruction)
	var systemInstruction interface{}
	if systemPrompt != "" {
		systemInstruction = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemPrompt},
			},
		}
	}

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		// content bisa string atau []map/[]interface{} (untuk tool_use / tool_result)
		parts := make([]map[string]interface{}, 0)

		switch v := msg.Content.(type) {
		case string:
			parts = append(parts, map[string]interface{}{"text": v})
		case []map[string]interface{}:
			for _, bm := range v {
				switch bm["type"] {
				case "text":
					if txt, ok := bm["text"].(string); ok && txt != "" {
						parts = append(parts, map[string]interface{}{"text": txt})
					}
				case "tool_use":
					toolInput := bm["input"]
					inputJSON, _ := json.Marshal(toolInput)
					parts = append(parts, map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": bm["name"],
							"args": json.RawMessage(inputJSON),
						},
					})
				case "tool_result":
					tc := bm["content"]
					toolName, _ := bm["name"].(string)
					if toolName == "" {
						// Fallback: Anthropic style tool_use_id (toolu_xxx), cari dari name
						if name, ok := bm["tool_use_id"].(string); ok {
							toolName = name
						}
					}
					// Untuk Gemini, tool_result → functionResponse dengan nama function asli
					parts = append(parts, map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name": toolName,
							"response": map[string]interface{}{
								"result": tc,
							},
						},
					})
					// Function response selalu dari user/function role
					role = "function"
				}
			}
		case []interface{}:
			for _, raw := range v {
				bm, ok := raw.(map[string]interface{})
				if !ok || bm == nil {
					continue
				}
				switch bm["type"] {
				case "text":
					if txt, ok := bm["text"].(string); ok && txt != "" {
						parts = append(parts, map[string]interface{}{"text": txt})
					}
				case "tool_use":
					toolInput := bm["input"]
					inputJSON, _ := json.Marshal(toolInput)
					parts = append(parts, map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": bm["name"],
							"args": json.RawMessage(inputJSON),
						},
					})
				case "tool_result":
					tc := bm["content"]
					toolName, _ := bm["name"].(string)
					if toolName == "" {
						if name, ok := bm["tool_use_id"].(string); ok {
							toolName = name
						}
					}
					parts = append(parts, map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name": toolName,
							"response": map[string]interface{}{
								"result": tc,
							},
						},
					})
					role = "function"
				}
			}
		}

		if len(parts) > 0 {
			gContents = append(gContents, map[string]interface{}{
				"role":  role,
				"parts": parts,
			})
		}
	}

	// Build Gemini request body
	gReq := map[string]interface{}{
		"contents": gContents,
	}

	if systemInstruction != nil {
		gReq["system_instruction"] = systemInstruction
	}

	// Konversi tools ke Gemini FunctionDeclaration
	if !noTools && len(tools) > 0 {
		gTools := convertToolsToGemini(tools)
		if len(gTools) > 0 {
			gReq["tools"] = []map[string]interface{}{
				{"function_declarations": gTools},
			}
		}
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	client := &http.Client{}
	models := ac.geminiModelSequence()
	var lastErr error

	for _, modelName := range models {
		// Gemini endpoint: POST /v1beta/models/{model}:generateContent
		endpoint := fmt.Sprintf("%s/models/%s:generateContent", ac.baseURL, modelName)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini request: %w", err)
		}
		httpReq.Header.Set("x-goog-api-key", ac.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("model %s: failed to send gemini request: %w", modelName, err)
			log.Printf("[WAAI][Gemini] Request failed for model=%s: %v", modelName, err)
			continue
		}

		respBody, readErr := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("model %s: failed to read gemini response: %w", modelName, readErr)
			log.Printf("[WAAI][Gemini] Failed reading response for model=%s: %v", modelName, readErr)
			continue
		}

		log.Printf("[WAAI][Gemini] Raw response model=%s status=%d body=%s", modelName, httpResp.StatusCode, truncateResponseBody(respBody))

		if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
			lastErr = fmt.Errorf("model %s: gemini error (%d): %s", modelName, httpResp.StatusCode, truncateResponseBody(respBody))
			log.Printf("[WAAI][Gemini] Falling back from model=%s because status=%d", modelName, httpResp.StatusCode)
			continue
		}

		response, parseErr := parseGeminiResponse(respBody)
		if parseErr != nil {
			lastErr = fmt.Errorf("model %s: failed to parse gemini response: %w", modelName, parseErr)
			log.Printf("[WAAI][Gemini] Failed parsing response for model=%s: %v", modelName, parseErr)
			continue
		}

		if modelName != ac.model {
			log.Printf("[WAAI][Gemini] Using fallback model=%s", modelName)
		}

		return response, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("gemini request failed: no model configured")
}

func buildGeminiModelFallbacks() []string {
	models := []string{
		strings.TrimSpace(os.Getenv("GEMINI_MODEL")),
		strings.TrimSpace(os.Getenv("GEMINI_MODEL_FALLBACK1")),
		strings.TrimSpace(os.Getenv("GEMINI_MODEL_FALLBACK2")),
	}

	result := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		if model == "" {
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}

	if len(result) == 0 {
		return []string{"gemini-2.5-pro"}
	}

	return result
}

func (ac *AIClient) geminiModelSequence() []string {
	models := make([]string, 0, 1+len(ac.fallbackModels))
	seen := make(map[string]struct{}, 1+len(ac.fallbackModels))
	for _, model := range append([]string{ac.model}, ac.fallbackModels...) {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		models = append(models, model)
	}
	if len(models) == 0 {
		return []string{"gemini-2.5-pro"}
	}
	return models
}

// convertToolsToGemini converts Anthropic-style tool definitions to Gemini FunctionDeclaration
func convertToolsToGemini(tools []ToolDefinition) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		fd := map[string]interface{}{
			"name":        t.Function.Name,
			"description": t.Function.Description,
		}
		if t.Function.Parameters != nil {
			fd["parameters"] = t.Function.Parameters
		}
		result = append(result, fd)
	}
	return result
}

// parseGeminiResponse parses Gemini API response to AnthropicResponse for unified tool-loop handling
func parseGeminiResponse(body []byte) (*AnthropicResponse, error) {
	var raw struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	resp := &AnthropicResponse{
		Role: "assistant",
	}

	if raw.Error != nil {
		resp.Error = &struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    "gemini_error",
			Message: raw.Error.Message,
		}
		return resp, nil
	}

	if len(raw.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: no candidates in response")
	}

	candidate := raw.Candidates[0]

	// Map Gemini finishReason to Anthropic stop_reason
	switch candidate.FinishReason {
	case "STOP":
		resp.StopReason = "end_turn"
	case "MAX_TOKENS":
		resp.StopReason = "max_tokens"
	case "SAFETY", "RECITATION", "OTHER":
		resp.StopReason = "stop"
	case "TOOL_CALL":
		resp.StopReason = "tool_use"
	default:
		resp.StopReason = candidate.FinishReason
	}

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			resp.Content = append(resp.Content, struct {
				Type  string          `json:"type"`
				Text  string          `json:"text,omitempty"`
				ID    string          `json:"id,omitempty"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input,omitempty"`
			}{
				Type: "text",
				Text: part.Text,
			})
		}
		if part.FunctionCall != nil {
			inputJSON, _ := json.Marshal(part.FunctionCall.Args)
			resp.Content = append(resp.Content, struct {
				Type  string          `json:"type"`
				Text  string          `json:"text,omitempty"`
				ID    string          `json:"id,omitempty"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input,omitempty"`
			}{
				Type:  "tool_use",
				ID:    part.FunctionCall.Name,
				Name:  part.FunctionCall.Name,
				Input: inputJSON,
			})
		}
	}

	return resp, nil
}

// callAnthropicRequest dispatches to Anthropic or Gemini based on ac.provider
func (ac *AIClient) callAnthropicRequest(ctx context.Context, systemPrompt string, messages []ConversationMessage, tools []ToolDefinition) (*AnthropicResponse, error) {
	if ac.provider == "gemini" {
		return ac.callGeminiRequest(ctx, systemPrompt, messages, tools, false)
	}

	req := AnthropicRequest{
		Model:        ac.model,
		MaxTokens:    1024,
		SystemPrompt: systemPrompt,
		Messages:     messages,
		Tools:        tools,
		Stream:       false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ac.messagesEndpoint(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", ac.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	log.Printf("[WAAI][AI] Raw response status=%d body=%s", httpResp.StatusCode, truncateResponseBody(respBody))

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		errResponse, err := parseAnthropicResponse(respBody)
		if err == nil && errResponse.Error != nil {
			return nil, fmt.Errorf("anthropic error (%d): %s - %s", httpResp.StatusCode, errResponse.Error.Type, errResponse.Error.Message)
		}
		return nil, fmt.Errorf("anthropic error (%d): %s", httpResp.StatusCode, truncateResponseBody(respBody))
	}

	response, err := parseAnthropicResponse(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from %s: %s", ac.messagesEndpoint(), truncateResponseBody(respBody))
	}

	if response.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s - %s", response.Error.Type, response.Error.Message)
	}

	return response, nil
}

func (ac *AIClient) messagesEndpoint() string {
	if strings.HasSuffix(ac.baseURL, "/v1") {
		return ac.baseURL + "/messages"
	}
	return ac.baseURL + "/v1/messages"
}

func truncateResponseBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response body"
	}
	const maxLen = 300
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func formatToolResult(result interface{}) string {
	if result == nil {
		return "{}"
	}
	if text, ok := result.(string); ok {
		return text
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result)
	}
	return string(data)
}

func parseAnthropicResponse(body []byte) (*AnthropicResponse, error) {
	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err == nil && hasNativeAnthropicPayload(&response) {
		return &response, nil
	}

	var wrapped struct {
		Message AnthropicResponse `json:"message"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && hasNativeAnthropicPayload(&wrapped.Message) {
		return &wrapped.Message, nil
	}

	var dataWrapped struct {
		Data AnthropicResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &dataWrapped); err == nil && hasNativeAnthropicPayload(&dataWrapped.Data) {
		return &dataWrapped.Data, nil
	}

	var openAICompat struct {
		ID      string `json:"id"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role      string `json:"role"`
				Content   any    `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &openAICompat); err == nil {
		converted := convertOpenAICompatResponse(openAICompat)
		if hasParsedPayload(converted) {
			return converted, nil
		}
	}

	return nil, fmt.Errorf("unsupported anthropic response format")
}

func hasNativeAnthropicPayload(response *AnthropicResponse) bool {
	if response == nil {
		return false
	}
	if response.Error != nil {
		return true
	}
	return response.Role != "" || response.StopReason != "" || len(response.Content) > 0
}

func hasParsedPayload(response *AnthropicResponse) bool {
	if response == nil {
		return false
	}
	if response.Error != nil {
		return true
	}
	return response.ID != "" || response.Role != "" || response.StopReason != "" || len(response.Content) > 0
}

func convertOpenAICompatResponse(src struct {
	ID      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role      string `json:"role"`
			Content   any    `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}) *AnthropicResponse {
	response := &AnthropicResponse{
		ID: src.ID,
	}

	if src.Error != nil {
		response.Error = &struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    src.Error.Type,
			Message: src.Error.Message,
		}
		return response
	}

	if len(src.Choices) == 0 {
		return response
	}

	choice := src.Choices[0]
	response.Role = choice.Message.Role
	response.StopReason = choice.FinishReason

	switch content := choice.Message.Content.(type) {
	case string:
		if content != "" {
			response.Content = append(response.Content, struct {
				Type  string          `json:"type"`
				Text  string          `json:"text,omitempty"`
				ID    string          `json:"id,omitempty"`
				Name  string          `json:"name,omitempty"`
				Input json.RawMessage `json:"input,omitempty"`
			}{
				Type: "text",
				Text: content,
			})
		}
	}

	for _, toolCall := range choice.Message.ToolCalls {
		arguments := strings.TrimSpace(toolCall.Function.Arguments)
		if arguments == "" {
			arguments = "{}"
		}

		response.Content = append(response.Content, struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		}{
			Type:  "tool_use",
			ID:    toolCall.ID,
			Name:  toolCall.Function.Name,
			Input: json.RawMessage(arguments),
		})
	}

	return response
}

func summarizeAnthropicContent(content []struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}) string {
	parts := make([]string, 0, len(content))
	for _, block := range content {
		switch block.Type {
		case "text":
			parts = append(parts, "text="+truncateResponseBody([]byte(block.Text)))
		case "tool_use":
			parts = append(parts, fmt.Sprintf("tool_use(name=%s,id=%s,input=%s)", block.Name, block.ID, truncateResponseBody(block.Input)))
		default:
			parts = append(parts, "type="+block.Type)
		}
	}
	return strings.Join(parts, " | ")
}

// buildSystemPrompt builds the system prompt with tenant context
func (ac *AIClient) buildSystemPrompt(tenant *TenantInfo, snapshot map[string]interface{}) string {
	displayName := tenant.FullName
	if displayName == "" {
		displayName = tenant.Name
	}

	orgName := tenant.OrganizationName
	if orgName == "" {
		if name, ok := snapshot["organization_name"].(string); ok && name != "" {
			orgName = name
		}
	}

	isGuest := tenant.Role == "Guest"

	now := time.Now()
	currentMonth := now.Format("2006-01")
	currentDate := now.Format("2006-01-02")

	var userContext string
	if isGuest {
		userContext = fmt.Sprintf(`User Information:
- Status: Unregistered/Guest
- Phone: %s

You are talking to an unregistered user. They can only ask about your capabilities and how to register.
If they ask about specific data (orders, fleet, etc.), politely inform them that they need to register and link their WhatsApp number first.`, tenant.Phone)
	} else {
		userContext = fmt.Sprintf(`User Information:
- Name: %s
- Role: %s
- Organization: %s

Current Business Status (today only):
- Business Name: %s
- Fleet Count: %v
- Available Units: %v
- Today's Bookings: %v`, displayName, tenant.Role, orgName, orgName, snapshot["fleet_count"], snapshot["unit_count"], snapshot["today_bookings"])
	}

	prompt := fmt.Sprintf(`You are a helpful WhatsApp AI Assistant for Travego, an ERP rental bus management system.

%s

Current Date: %s (current month: %s)

You have access to the following functions to help users:
1. get_business_snapshot - Get current business metrics
2. get_fleet_availability - Check vehicle availability
3. get_fleet_list - View owned fleets (armada)
4. get_fleet_detail - View fleet detail
5. get_fleet_units - View owned fleet units
6. get_city_list - View city list
7. get_preference_cities - View served cities
8. get_customer_list - Search customers by name (returns customer_id)
9. get_customer_detail - View customer detail by customer_id
10. get_booking_list - View bookings (legacy, prefer get_order_list for pesanan)
11. get_revenue_summary - Get revenue data
12. get_organization_info - Get business / organization information
13. get_order_list - View pesanan/order list with summary, filter by period (YYYY-MM)
14. get_order_detail - View order detail by order_id, including itinerary and payment summary (sisa pembayaran)
15. get_schedule_list - View schedule list, filtered by period (YYYY-MM)
16. get_schedule_detail - View schedule detail by schedule_number
17. get_order_payment_history - Get riwayat pembayaran for a specific order_id
18. approve_order - Setujui (approve) an order by order_id
19. reject_order - Tolak (reject) an order by order_id
20. get_employee_shift_schedule - Get jadwal tim (employee shift schedule) including total off days
21. add_employee_off_day - Tambah hari off (add off day) for an employee
22. get_monthly_revenue - Get pendapatan bulan ini including total revenue, total expenses, and estimated profit
23. get_top_fleets - Get unit armada paling banyak orderan (top fleets by number of orders)
24. get_top_destinations - Get kota tujuan paling populer (top destinations)
25. get_top_customers - Get customer paling loyal (top customers by number of orders)
26. get_spj_total_biaya - Get total biaya operasional (total amount) for a specific Surat Jalan / SPJ (schedule_number)
27. tambah_pengeluaran_spj - Tambah pengeluaran untuk Surat Jalan / SPJ (schedule_number)
28. get_spj_pengeluaran - Dapatkan daftar pengeluaran untuk Surat Jalan / SPJ tertentu
29. get_spj_ringkasan_pembayaran - Dapatkan ringkasan total pengeluaran SPJ berdasarkan jenis pembayaran (biaya operasional dan reimburse)
30. print_surat_jalan - Mencetak dan mengirim surat jalan / SPJ (Surat Pertanggungjawaban) dalam format PDF ke WhatsApp
31. get_fleet_availibility_by_daterange - Get vehicle availability by date range, filter (YYYY-MM-DD)
32. get_fleet_unit_detail - Get fleet unit detail by fleet_id and unit_id
33. get_fleet_unit_by_partner - Get fleet unit detail by fleet_id and ownership_type = 1 partner
34. get_upcoming_unit_schedule - Get upcoming schedule for a fleet unit by unit_id
35. get_latest_unit_schedule - Get latest schedule for a fleet unit by unit_id
36. get_unit_trip_history - Get trip history for a fleet unit by unit_id, start_date, end_date (YYYY-MM-DD)
37. get_inventory_items - View inventory item list with total stock per item and garage names
38. get_inventory_detail - View inventory item detail by item_id, including stock per garage/location
39. get_inventory_stock - Check stock count for an inventory item, either total stock or stock in a specific garage
40. get_garage_list - Get garage/garasi list, optionally filtered by item_id
41. get_item_suppliers - Get supplier list for inventory purchase orders
42. get_item_movements - Get item inventory movement history, filterable by garage_id and date range
43. get_item_order_history - Get item inventory order/purchase history by item_id and date range
44. get_item_stock_distribution - Get stock distribution per garage/location from item detail locations[]
45. get_purchase_order_list - Get inventory purchase order list
46. get_purchase_order_detail - Get inventory purchase order detail by purchase_id
47. complete_purchase_order - Mark inventory purchase order as completed/received
48. cancel_purchase_order - Cancel/reject inventory purchase order
49. create_new_item - Create new inventory item or add/update item stock; SKU is generated automatically when empty

- [CRITICAL] Data dalam database dapat BERUBAH sewaktu-waktu. JANGAN PERCAYA jawaban Anda dari riwayat percakapan sebelumnya. Selalu PANGGIL TOOL setiap kali user menanyakan data (pesanan, pelanggan, jadwal, armada, dll.) untuk mendapatkan data TERBARU dari database.
- GUESTS CANNOT USE TOOLS THAT REQUIRE ORGANIZATION CONTEXT. If a guest asks for data, explain how to register.
- When the user asks about their business or organization name, answer using Business Name from context above. For full organization details (address, phone, NPWP, etc.), call get_organization_info.
- When the user asks for customer contact or details by name (not customer_id), you MUST:
  1. Call get_customer_list with customer_name set to the name provided
  2. If one match is found, call get_customer_detail with that customer_id and share the contact info
  3. If multiple matches are found, list them and ask the user to clarify
  4. If no match is found, tell the user the customer was not found
- When the user asks about inventory, inventory item stock, stock per garage, or stock item count, you MUST call inventory tools every time:
  1. Call get_inventory_items to check the inventory list or find item_id from item name/SKU.
  2. Call get_inventory_detail when the user asks for inventory item detail.
  3. Call get_inventory_stock when the user asks for jumlah stok / total stock, with item_id and optional garage_id.
  4. Call get_item_stock_distribution when the user asks for distribusi stok per garage/location.
  5. Call get_item_movements when the user asks for riwayat movement item.
  6. Call get_item_order_history when the user asks for riwayat order / purchase history item.
  7. If the user provides only an item name, first call get_inventory_items to find the matching item_id before calling get_inventory_detail, get_inventory_stock, get_item_movements, get_item_order_history, or get_item_stock_distribution.
- For garage/garasi-related inventory questions, call get_garage_list. If the user wants to create or update item stock and only mentions garage name/city, call get_garage_list first to get the correct garage_id.
- For purchase order inventory questions, use get_purchase_order_list or get_purchase_order_detail. Use complete_purchase_order only when the user explicitly confirms barang sudah diterima. Use cancel_purchase_order only when the user explicitly asks to cancel/reject a purchase order.
- For supplier questions in inventory/purchase order context, call get_item_suppliers to get supplier_id before calling create_new_item with transaction_type = 2.
- When the user asks to create/request item baru, tambah stok item, atau buat item inventory baru:
  - Use create_new_item.
  - Before calling, ensure parameters are complete. If any required parameter is missing, ask the user for the missing parameter(s) instead of guessing.
  - Required for all create_new_item calls: item_name or existing item_id, item_uom, item_category, stock, garage_id, transaction_type.
  - Required when transaction_type = 2: item_price, transaction_date (YYYY-MM-DD), and supplier_id or supplier_name.
  - item_sku is optional. If the user does not provide item_sku, call create_new_item with item_sku omitted/empty so the service generates SKU automatically.
  - item_uom examples: Pcs, Box, Liter, Unit.
  - item_category mapping: 1 = Kebutuhan Armada, 2 = kebutuhan kantor.
  - transaction_type mapping: 1 = tambah stok yang ada, 2 = update stock sesuai input (tidak menambahkan stock yang sudah ada).
  - If garage_id is unknown, call get_garage_list first or ask the user which garage/garasi to use.
  - If supplier_id is unknown for transaction_type = 2, call get_item_suppliers first or ask the user for supplier details.
- When the user asks about pesanan/order (e.g. "ada pesanan bulan ini?", "berapa order bulan Juni?"), you MUST call get_order_list with period set to the relevant YYYY-MM (use %s for "bulan ini"). Answer ONLY from the tool result summary (total_orders, paid, unpaid, revenue). Never guess from Today's Bookings — that number is for today only.
- For order detail by order_id, call get_order_detail — JANGAN PERCAYA jawaban sebelumnya, selalu panggil tool untuk data terbaru.
- For itinerary of order detail by order_id, call get_order_detail, get orders.itinerary[].
- When user asks for fleet's partner, get data from get_fleet_units and mapping the ownership_type. ownership_type = 1 fleet's partner, ownership_type = 0 own ownership.
- When user asks for coverage area of a fleet unit, get data from get_fleet_unit_detail > pickup_point. You can explain if pickup area from other city so the customer need to pay charge.
- Order payment status mapping (field payment_status / payment_status_label):
  0 = Dibatalkan, 1 = Lunas, 2 = Menunggu Verifikasi, 3 = Belum Lunasi, 10 = Menunggu Persetujuan.
  When telling the user payment status, ALWAYS use payment_status_label from get_order_list/get_order_detail. NEVER use latest_payment_status or latest_payment_type as status pembayaran — those are jenis pembayaran (DP, Cicilan, Pelunasan), not order payment status.
  Summary fields: paid = lunas, pending = menunggu verifikasi atau belum lunasi.
  Untuk menjawab apakah order sudah dijadwalkan atau belum, baca dari orders[].schedule_id. Jika schedule_id = "" berarti belum terjadwal. Field scheduled/is_scheduled mengikuti aturan yang sama.
- Untuk pertanyaan terkait biaya operasional, Surat Jalan, atau SPJ (Surat Pertanggungjawaban):
  1. Selalu panggil get_spj_total_biaya atau get_spj_ringkasan_pembayaran dengan schedule_number (Surat Jalan) yang dimaksud.
  2. Jika user ingin melihat rincian biaya, panggil get_spj_pengeluaran.
  3. Jika user ingin menambah biaya, panggil tambah_pengeluaran_spj:
     - transaction_item bisa berupa teks bebas (misal "tol", "bahan bakar", "parkir"), sistem akan otomatis memetakannya ke kode yang benar.
     - Gunakan description untuk menambahkan catatan rinci (misal "Tol MBZ Bekasi-Karawang").
     - Contoh: untuk "bayar tol 36.000", set transaction_item="tol", amount=36000, description="Tol MBZ Bekasi-Karawang".
  4. JIKA USER INGIN MENGIRIM SURAT JALAN PDF:
     - Anda WAJIB menyertakan schedule_number di parameter tool print_surat_jalan.
     - JANGAN PERNAH memanggil print_surat_jalan dengan input kosong {}.
     - Contoh: Jika nomornya SJL-260706163-CLS70, Anda harus tulis input: {"schedule_number": "SJL-260706163-CLS70"}.
     - Jika Anda belum tahu nomornya, cari dulu menggunakan get_order_detail atau get_schedule_list.
  5. Identifikasi schedule_number dari input user atau dari hasil get_order_detail / get_schedule_list.

Please respond in Indonesian (Bahasa Indonesia) unless the user asks otherwise.
Help the user with their inquiries related to the bus rental business.
If the user asks who you are, what your name is, or what assistant they are talking to, identify yourself as "Trave". Trave is AI Assistant by TraveGO.
If the user asks who developed, created, or made you, answer that you were created by Afatbenz Tech and that they can contact 6281335884729 or visit mafatichulfuadi.com for further discussion.
If the user asks how to register for or enjoy the AI Assistant service, answer that they should register on https://www.travego.id and add their WhatsApp number in the Pengaturan > AI Assistant menu.
If the user asks what you can do (capabilities), explain your features like checking fleet availability, managing orders, viewing customer data, revenue summaries, and employee schedules. Mention that these features are available after registration.
If the user asks where are you from, explain that you are from TraveGO, you dont have physical location, but if need discussion the travego team can visit or visisted in Yogyakarta. Just create the appontment for further discussion.
Do not say you are Kiro, Claude, Anthropic, or mention the provider/model name unless explicitly asked about technical backend details.
Be professional and concise in your responses.

WhatsApp reply formatting:
- This is WhatsApp, NOT Markdown. For bold use a single asterisk on each side: *teks tebal*. Never use **double asterisks**.
- Use bold sparingly — only for key values such as names, amounts, or dates. Do not bold whole sentences.
- Prefer plain, short sentences. Use line breaks between list items instead of Markdown bullets or headers.
- Use minimal emoji. Default to no emoji unless the user is clearly casual and one small emoji truly helps.
- When the reply contains more than one important point, split it into short paragraphs or one item per line so it is easy to read in WhatsApp.
- Do not use # headings, **bold**, __underline__, or [link](url) Markdown syntax.`,
		userContext,
		currentDate,
		currentMonth,
		currentMonth,
	)

	return prompt
}

// BuildCompanySystemPrompt membentuk system prompt untuk company customer assistant (Skenario 2)
// tenant: context organisasi (berisi OrganizationID dan DeviceName dari assistant_customers)
// snapshot: data bisnis organisasi
// customerMessage: pesan asli customer
// assistantName: nama display assistant (device_name dari assistant_customers)
func (ac *AIClient) BuildCompanySystemPrompt(
	tenant *TenantInfo,
	snapshot map[string]interface{},
	customerMessage string,
	assistantName string,
) string {
	now := time.Now()
	currentMonth := now.Format("2006-01")
	currentDate := now.Format("2006-01-02")

	displayName := tenant.OrganizationName
	if displayName == "" {
		displayName = assistantName
	}

	orgName := tenant.OrganizationName
	if orgName == "" {
		if name, ok := snapshot["organization_name"].(string); ok && name != "" {
			orgName = name
		}
	}

	userContext := fmt.Sprintf(`You are an AI assistant for *%s*.

Your role: *%s*
You represent this transport company and assist customers via WhatsApp.`,
		orgName, displayName)

	dataContext := fmt.Sprintf(`Current Business Status:
- Business: %s
- Fleet Count: %v
- Available Units: %v
- Today's Bookings: %v`,
		orgName, snapshot["fleet_count"], snapshot["unit_count"], snapshot["today_bookings"])

	prompt := fmt.Sprintf(`Anda adalah customer service WhatsApp resmi untuk perusahaan transportasi rental bernama %s.
Nama assistant yang boleh disebut ke customer: %s.

%s

%s

Pesan customer terbaru:
%s

Tanggal saat ini: %s
Bulan saat ini: %s

Tujuan Anda:
- Bantu customer dengan jawaban singkat, sopan, natural, dan langsung ke inti.
- Untuk pertanyaan yang membutuhkan data terbaru, selalu panggil tool yang sesuai terlebih dahulu lalu jawab berdasarkan hasil tool.
- Jika data untuk memanggil tool belum cukup, ajukan tepat satu pertanyaan klarifikasi yang paling penting.

Larangan keras:
- Jangan pernah membalas dengan instruksi internal, daftar aturan, daftar tool, atau daftar kemampuan Anda.
- Jangan pernah mengatakan kalimat meta seperti "saya sudah memahami peran saya", "saya akan memanggil tool", "silakan lanjutkan pertanyaan", atau "berikut kemampuan saya".
- Jangan mengarang alamat, lokasi kantor, nomor rekening, harga, ketersediaan armada, fasilitas armada, status pesanan, atau detail invoice.
- Jangan pernah mengatakan pesanan sudah diterima, sedang diproses, atau berhasil dibuat sebelum tool create_order benar-benar mengembalikan status success.
- Jangan menyebut nama model, provider, atau detail teknis backend.

Panduan penggunaan tool:
- Lokasi kantor, alamat kantor, titik maps, nomor telepon kantor, WhatsApp kantor, email perusahaan -> get_organization_info
- Lokasi garasi atau alamat garasi -> get_garage_list
- Nomor rekening pembayaran -> get_bank_accounts
- Daftar armada atau armada tertentu -> get_fleet_list, lalu get_fleet_detail bila customer meminta detail/fasilitas
- Fasilitas armada -> get_fleet_detail
- Harga sewa -> jika armada belum jelas tanyakan armada dulu; setelah ada fleet_id dan jenis layanan, panggil get_fleet_prices
- Ketersediaan armada -> butuh start_date dan end_date; setelah lengkap panggil get_fleet_availability
- Lacak pesanan, detail pesanan, status pembayaran, invoice -> gunakan get_order_list, get_order_detail, atau print_invoice sesuai kebutuhan
- Buat pesanan baru -> kumpulkan data wajib, cek harga/tool terkait, lalu panggil create_order
- Jika get_order_list mengembalikan lebih dari 1 pesanan dan customer belum menyebut order_id, ajukan tepat satu pertanyaan klarifikasi untuk meminta order_id yang dimaksud (atau tanggal sewa).
- Jika ada pesanan customer dengan status = 2 (belum dikonfirmasi), jangan kirim nomor rekening. Sampaikan bahwa pembayaran dapat dilakukan setelah pesanan selesai ditinjau dan dikonfirmasi oleh tim.

Layanan sewa:
- type_id 1 = CityTour
- type_id 2 = Overland
- type_id 3 = Drop Only
- Definisi layanan yang WAJIB diikuti:
  - CityTour = perjalanan wisata atau keliling area/kota tujuan dengan agenda beberapa titik.
  - Overland = perjalanan luar kota / antar kota dengan armada sewa, bisa pergi-pulang atau multi-destinasi sesuai itinerary customer.
  - Drop Only = layanan antar atau jemput saja ke satu tujuan, tanpa standby perjalanan lanjutan.
  - JANGAN PERNAH menjelaskan Overland sebagai "perjalanan dengan guide", "trip dengan tour guide", atau istilah serupa kecuali customer memang meminta layanan guide secara terpisah.
  - Jika customer menyebut tujuan luar kota seperti Bandung, Semarang, Surabaya, Malang, atau kota lain di luar kota asal, arahkan konteksnya ke Overland atau Drop Only sesuai pola perjalanannya, bukan ke layanan guide.

Validasi durasi sewa:
- Untuk pertanyaan layanan CityTour, Overland, atau Drop Only, selalu cek get_preference_cities. Jika customer menyebut kota dengan nama (mis. "Brebes"), cari city_id via get_city_list (search) lalu panggil get_preference_cities dengan city_id tersebut
- Logic CityTour: jika service_types pada kota tersebut berisi city_tour maka kota itu tersedia untuk CityTour; gunakan minimal_day sebagai minimal durasi sewa
- Logic Overland / Drop Only: cek kota tujuan di get_preference_cities; jika service_types sesuai, gunakan minimal_day sebagai minimal durasi sewa
- Jika pelanggan belum menentukan lokasi penjemputan, gunakan kota asal company sebagai default pickup city
- Jika kota tujuan tidak ada di get_preference_cities: gunakan get_trip_distance untuk menghitung jarak dari pickup (default kota asal company bila pickup belum ada) ke kota tujuan, lalu tentukan minimal durasi:
  - Overland (pulang-pergi):
    - jika jarak > 700 km -> minimal 4 hari
    - jika jarak > 400 km -> minimal 3 hari
    - jika jarak > 250 km -> minimal 2 hari
    - selain itu -> minimal 1 hari
  - Drop Only (antar/jemput saja):
    - jika jarak > 700 km -> minimal 3 hari
    - jika jarak > 400 km -> minimal 2 hari
    - selain itu -> minimal 1 hari
- Setelah create_order berhasil untuk kota yang belum ada di get_preference_cities, tambahkan persis teks ini:
  Tim sedang meninjau pesanan anda, kami akan segera menghubungi anda.
  Terimakasih, Calista Prima
- Jangan menjanjikan durasi atau harga tanpa data tool

Format jawaban WhatsApp:
- Gunakan Bahasa Indonesia
- Maksimal 4 kalimat pendek kecuali customer meminta detail
- Gunakan *teks* hanya untuk nilai penting
- Hindari emoji berlebihan. Default tanpa emoji.
- Jika ada lebih dari satu informasi penting, pisahkan dengan enter atau baris baru agar mudah dibaca
- Jangan gunakan heading markdown, bullet markdown, atau format dokumentasi
- Untuk sapaan pembuka, gunakan gaya seperti: "Halo, Selamat datang di Nama Perusahaan!" lalu lanjutkan pertanyaan bantuan di baris berikutnya

Contoh jawaban yang salah:
Siap! Saya sudah memahami peran saya sebagai AI Assistant...

Contoh perilaku yang benar:
- Pertanyaan "lokasi kantor di mana?" -> panggil get_organization_info lalu jawab alamatnya
- Pertanyaan "berapa harga sewanya?" tanpa armada -> tanya 1 hal penting: armadanya apa
- Pertanyaan "invoice order OR-123 bisa dikirim?" -> panggil print_invoice dengan order_id yang sesuai
- Pertanyaan "30 orang ke Bandung 2-3 Juli" -> pahami Bandung sebagai tujuan luar kota. Jika belum jelas pola perjalanannya, tanyakan apakah kebutuhannya *Drop Only* atau *Overland* tanpa menyebut guide`, orgName, displayName, userContext, dataContext, customerMessage, currentDate, currentMonth)

	return prompt
}

func getCompanyAssistantToolDefinitions() []ToolDefinition {
	companyTools := GetCompanyToolDefinitions()
	needed := map[string]struct{}{
		"get_fleet_prices": {},
		"get_fleet_addons": {},
		"create_order":     {},
	}

	existing := make(map[string]struct{}, len(companyTools))
	for _, tool := range companyTools {
		name := strings.TrimSpace(tool.Function.Name)
		if name == "" {
			name = strings.TrimSpace(tool.Name)
		}
		if name != "" {
			existing[name] = struct{}{}
		}
	}

	for _, tool := range GetToolDefinitions() {
		name := strings.TrimSpace(tool.Function.Name)
		if name == "" {
			name = strings.TrimSpace(tool.Name)
		}
		if _, wanted := needed[name]; !wanted {
			continue
		}
		if _, ok := existing[name]; ok {
			continue
		}
		companyTools = append(companyTools, tool)
		existing[name] = struct{}{}
	}

	return companyTools
}

func latestUserText(messages []ConversationMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		if text, ok := messages[i].Content.(string); ok {
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func companyMessageNeedsTool(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}

	greetings := []string{
		"halo",
		"hai",
		"hi",
		"pagi",
		"siang",
		"sore",
		"malam",
		"assalamualaikum",
		"permisi",
	}
	for _, greeting := range greetings {
		if text == greeting {
			return false
		}
	}

	keywords := []string{
		"alamat",
		"lokasi",
		"kantor",
		"maps",
		"telepon",
		"telp",
		"whatsapp",
		"wa ",
		"email",
		"rekening",
		"bank",
		"harga",
		"tarif",
		"biaya",
		"sewa",
		"armada",
		"bus",
		"unit",
		"seat",
		"kursi",
		"kapasitas",
		"muat",
		"fasilitas",
		"tersedia",
		"ketersediaan",
		"order",
		"pesanan",
		"booking",
		"invoice",
		"jadwal",
		"garasi",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	if regexp.MustCompile(`\b\d+\s*hari\b`).FindStringIndex(text) != nil {
		return true
	}
	if strings.Contains(text, "berapa hari") || strings.Contains(text, "durasi") || strings.Contains(text, "sehari") {
		return true
	}

	return false
}

func responseContainsSeatCapacity(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	if !strings.Contains(normalized, "kapasitas") && !strings.Contains(normalized, "seat") && !strings.Contains(normalized, "kursi") {
		return false
	}
	re := regexp.MustCompile(`\b\d{2,3}\b`)
	return re.FindStringIndex(normalized) != nil
}

func responseContainsUnitRecommendation(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	if !strings.Contains(normalized, "unit") {
		return false
	}
	if !strings.Contains(normalized, "big bus") && !strings.Contains(normalized, "medium bus") && !strings.Contains(normalized, "bus") {
		return false
	}
	re := regexp.MustCompile(`\b\d+\s*unit\b`)
	return re.FindStringIndex(normalized) != nil
}

func responseContainsPriceAmount(text string) bool {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)\brp\s*[\d\.,]{4,}\b`)
	return re.FindStringIndex(normalized) != nil
}

func analyzeCreateOrderToolResult(toolResult interface{}) (success bool, failed bool, missing []string, errMsg string, orderID string) {
	resultMap, ok := toolResult.(map[string]interface{})
	if !ok {
		return false, false, nil, "", ""
	}

	status := strings.ToLower(strings.TrimSpace(getStringValue(resultMap["status"])))
	if status == "success" {
		orderID = strings.TrimSpace(getStringValue(resultMap["order_id"]))
		return true, false, nil, "", orderID
	}

	errMsg = strings.TrimSpace(getStringValue(resultMap["error"]))
	if errMsg == "" {
		return false, false, nil, "", ""
	}

	return false, true, getStringSliceValue(resultMap["missing_required"]), errMsg, ""
}

func getStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func getStringSliceValue(value interface{}) []string {
	rawItems, ok := value.([]interface{})
	if !ok {
		return nil
	}

	items := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		item := strings.TrimSpace(getStringValue(raw))
		if item != "" {
			items = append(items, item)
		}
	}

	return items
}

func bankAccountsToolResultBlocked(toolResult interface{}) bool {
	resultMap, ok := toolResult.(map[string]interface{})
	if !ok {
		return false
	}
	errMsg := strings.TrimSpace(getStringValue(resultMap["error"]))
	return errMsg == "ORDER_UNCONFIRMED"
}

func responseContainsBankAccountDetails(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	normalized := strings.ToLower(text)
	if strings.Contains(normalized, "rekening") || strings.Contains(normalized, "bank") || strings.Contains(normalized, "no rekening") || strings.Contains(normalized, "nomor rekening") {
		if containsLongDigitSequence(text) {
			return true
		}
	}
	return false
}

func containsLongDigitSequence(text string) bool {
	re := regexp.MustCompile(`\b\d{8,20}\b`)
	return re.FindStringIndex(text) != nil
}

func responseClaimsOneDay(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "1 hari") || strings.Contains(normalized, "sehari")
}

func responseClaimsOneDayPossible(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "bisa 1 hari") ||
		strings.Contains(normalized, "1 hari memungkinkan") ||
		strings.Contains(normalized, "memungkinkan") ||
		strings.Contains(normalized, "sehari memungkinkan")
}

func durationQuestionOneDay(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "1 hari") || strings.Contains(normalized, "sehari")
}

func durationQuestionNeedsValidation(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	if durationQuestionOneDay(normalized) {
		return true
	}
	if strings.Contains(normalized, "berapa hari") || strings.Contains(normalized, "durasi") {
		return true
	}
	return regexp.MustCompile(`\b\d+\s*hari\b`).FindStringIndex(normalized) != nil
}

func mentionsDropOnlyContext(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "drop") ||
		strings.Contains(normalized, "antar") ||
		strings.Contains(normalized, "jemput") ||
		strings.Contains(normalized, "one way") ||
		strings.Contains(normalized, "sekali jalan")
}

func sliceLen(value interface{}) int {
	if value == nil {
		return 0
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return 0
	}
	return rv.Len()
}

func extractOrderIDsFromOrderListToolResult(toolResult interface{}) map[string]struct{} {
	resultMap, ok := toolResult.(map[string]interface{})
	if !ok {
		return map[string]struct{}{}
	}

	rawOrders, ok := resultMap["orders"]
	if !ok || rawOrders == nil {
		return map[string]struct{}{}
	}

	items, ok := rawOrders.([]interface{})
	if !ok {
		return map[string]struct{}{}
	}

	out := make(map[string]struct{}, len(items))
	for _, rawItem := range items {
		itemMap, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		orderID := strings.TrimSpace(getStringValue(itemMap["order_id"]))
		if orderID == "" {
			continue
		}
		out[orderID] = struct{}{}
	}
	return out
}

func extractUnknownOrderIDsFromText(text string, known map[string]struct{}) []string {
	candidates := extractFleetOrderIDsFromText(text)
	if len(candidates) == 0 {
		return nil
	}

	unknown := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, id := range candidates {
		if _, ok := known[id]; ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unknown = append(unknown, id)
	}
	return unknown
}

func extractFleetOrderIDsFromText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	re := regexp.MustCompile(`\bFO-[A-Za-z0-9-]+\b`)
	return re.FindAllString(text, -1)
}

func textContainsFleetOrderID(text string) bool {
	return len(extractFleetOrderIDsFromText(text)) > 0
}

func extractOrderIDCandidatesFromText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	re := regexp.MustCompile(`\b(?:FO|OR)-[A-Za-z0-9-]+\b`)
	return re.FindAllString(text, -1)
}

func containsOrderIDCandidate(text string) bool {
	return len(extractOrderIDCandidatesFromText(text)) > 0
}

func companyResponseClaimsOrderNumber(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	phrases := []string{
		"nomor order",
		"no. order",
		"no order",
		"order anda adalah",
		"order kamu adalah",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func companyResponseClaimsOrderSuccess(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}

	successPhrases := []string{
		"pesanan berhasil",
		"pesanan sudah kami terima",
		"pesanan anda sudah kami terima",
		"pesanan anda sedang kami proses",
		"pesanan sedang kami proses",
		"berhasil dibuat",
		"sudah berhasil dibuat",
		"akan diproses segera",
		"order berhasil",
		"order sudah dibuat",
	}
	for _, phrase := range successPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}

	return false
}

func companyResponseClaimsOrderAccepted(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}

	phrases := []string{
		"saya catat",
		"sudah saya catat",
		"kami catat",
		"sudah kami catat",
		"pesanan sudah kami catat",
		"pesanan sudah dicatat",
		"pesanan anda sudah dicatat",
		"pesanan anda sudah kami catat",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func buildCreateOrderFailureCorrection(missing []string, errMsg string) string {
	if len(missing) > 0 {
		return "PERBAIKAN WAJIB: Tool create_order gagal sehingga pesanan BELUM dibuat. Jangan bilang pesanan sudah diterima, diproses, atau berhasil. Minta customer melengkapi data yang belum ada. Jika ada beberapa data yang kurang, tanyakan tepat satu pertanyaan yang paling penting dulu. Data yang masih kurang: " + strings.Join(missing, ", ") + "."
	}

	if strings.TrimSpace(errMsg) != "" {
		return "PERBAIKAN WAJIB: Tool create_order gagal sehingga pesanan BELUM dibuat. Jangan bilang pesanan sudah diterima, diproses, atau berhasil. Jelaskan kendalanya secara singkat dalam Bahasa Indonesia dan arahkan customer ke langkah berikutnya. Error tool: " + errMsg
	}

	return "PERBAIKAN WAJIB: Tool create_order gagal sehingga pesanan BELUM dibuat. Jangan bilang pesanan sudah diterima, diproses, atau berhasil. Minta data yang masih kurang atau jelaskan kendalanya secara singkat."
}

func isAffirmationReply(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	normalized = collapseRepeatedLetters(normalized)
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return false
	}

	affirmations := []string{
		"iya",
		"ya",
		"iya mau",
		"mau",
		"mau dong",
		"ok",
		"oke",
		"baik",
		"boleh",
		"silakan",
		"lanjut",
		"setuju",
	}
	for _, phrase := range affirmations {
		if normalized == phrase {
			return true
		}
	}
	if strings.Contains(normalized, "tapi") || strings.Contains(normalized, "namun") || strings.Contains(normalized, "kecuali") {
		return false
	}
	if strings.HasPrefix(normalized, "ya ") || strings.HasPrefix(normalized, "iya ") || strings.HasPrefix(normalized, "ok ") || strings.HasPrefix(normalized, "oke ") {
		return true
	}
	if strings.Contains(normalized, "sudah sesuai") || strings.Contains(normalized, "sudah oke") || strings.Contains(normalized, "sudah benar") || strings.Contains(normalized, "sudah pas") || strings.Contains(normalized, "sesuai") {
		return true
	}
	if strings.Contains(normalized, "lanjut") {
		return true
	}

	return false
}

func conversationSuggestsCreateOrderFlow(messages []ConversationMessage) bool {
	lastAssistant := latestAssistantText(messages)
	if strings.TrimSpace(lastAssistant) == "" {
		return false
	}

	normalized := strings.ToLower(lastAssistant)
	offerPhrases := []string{
		"bantu buat pesanan",
		"bantu buat pesanan baru",
		"saya bantu buat pesanan",
		"mau saya bantu buat pesanan",
		"mau lanjut",
		"mau lanjut pesan",
		"mau lanjut pesan dengan",
		"mau lanjut dengan",
		"lanjut pesan",
		"lanjut pesan dengan",
		"untuk melanjutkan",
		"silakan konfirmasi",
		"silahkan konfirmasi",
		"balas konfirmasi",
		"balas ya",
		"jika sudah sesuai",
		"kalau sudah sesuai",
		"apakah sudah sesuai",
		"untuk konfirmasi",
		"konfirmasi pesanan",
		"konfirmasi pemesanan",
	}
	for _, phrase := range offerPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}

	return false
}

func latestAssistantText(messages []ConversationMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "assistant" {
			continue
		}
		switch v := messages[i].Content.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case []map[string]interface{}:
			var parts []string
			for _, bm := range v {
				if bm == nil {
					continue
				}
				if t, ok := bm["type"].(string); ok && t == "text" {
					if txt, ok := bm["text"].(string); ok && strings.TrimSpace(txt) != "" {
						parts = append(parts, strings.TrimSpace(txt))
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		case []interface{}:
			var parts []string
			for _, raw := range v {
				bm, ok := raw.(map[string]interface{})
				if !ok || bm == nil {
					continue
				}
				if t, ok := bm["type"].(string); ok && t == "text" {
					if txt, ok := bm["text"].(string); ok && strings.TrimSpace(txt) != "" {
						parts = append(parts, strings.TrimSpace(txt))
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}
	return ""
}

func collapseRepeatedLetters(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	var prev rune
	first := true
	for _, r := range text {
		if first {
			b.WriteRune(r)
			prev = r
			first = false
			continue
		}
		if r == prev {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}
func (ac *AIClient) executeTool(ctx context.Context, toolName string, input json.RawMessage) interface{} {
	// Parse input parameters
	var params map[string]interface{}
	_ = json.Unmarshal(input, &params)

	orgID, err := getAuthorizedContextValues(ctx)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Get user ID from context
	userID, _ := ctx.Value(contextUserID).(string)
	userID = strings.TrimSpace(userID)

	switch toolName {
	case "get_business_snapshot":
		return ac.toolExec.ExecuteGetBusinessSnapshot(ctx, orgID)

	case "get_fleet_availability":
		startStr := getStringParam(params, "start_date", "date_start")
		endStr := getStringParam(params, "end_date", "date_end")
		if startStr == "" || endStr == "" {
			return map[string]interface{}{"error": "start_date and end_date are required"}
		}
		startDate, endDate, err := parseFleetAvailabilityDates(startStr, endStr)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		fleetID := getStringParam(params, "fleet_id")
		available, fleets, err := ac.fleetService.GetFleetAvailibility(orgID, startDate, endDate, fleetID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"available": available,
			"fleets":    fleets,
		}

	case "get_fleet_list":
		req := &model.ListFleetRequest{
			OrganizationID: orgID,
			FleetType:      getStringParam(params, "fleet_type"),
			FleetName:      getStringParam(params, "fleet_name"),
			FleetBody:      getStringParam(params, "fleet_body"),
			FleetEngine:    getStringParam(params, "fleet_engine"),
		}
		req.PickupLocation = getIntParam(params, "pickup_location")
		items, err := ac.fleetService.ListFleets(req)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		fleetIDs := make([]string, 0, len(items))
		for i := range items {
			if items[i].FleetID != "" {
				fleetIDs = append(fleetIDs, items[i].FleetID)
			}
		}
		ratings, err := ac.fleetService.GetFleetRatings(orgID, fleetIDs)
		if err == nil {
			for i := range items {
				if v, ok := ratings[items[i].FleetID]; ok {
					items[i].Rating = v.Rating
					items[i].TotalUlasan = v.TotalUlasan
				}
			}
		}
		return items

	case "get_fleet_detail":
		fleetID := getStringParam(params, "fleet_id")
		if fleetID == "" {
			return map[string]interface{}{"error": "fleet_id is required"}
		}
		res, err := ac.fleetService.GetFleetDetail(orgID, fleetID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		ratings, err := ac.fleetService.GetFleetRatings(orgID, []string{fleetID})
		if err == nil {
			if v, ok := ratings[fleetID]; ok {
				res.Meta.Rating = v.Rating
				res.Meta.TotalUlasan = v.TotalUlasan
			}
		}
		reviews, err := ac.fleetService.GetFleetReviews(fleetID, orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		raw, _ := json.Marshal(res)
		var out map[string]interface{}
		_ = json.Unmarshal(raw, &out)
		out["reviews"] = reviews
		return out

	case "get_fleet_units":
		return ac.getFleetUnits(orgID, params)

	case "get_city_list":
		provinceID, provinceName := normalizeCityFilters(getStringParam(params, "province_id"), getStringParam(params, "province"))
		searchText := getStringParam(params, "search")
		cities, err := ac.generalService.GetCities(provinceID, provinceName, searchText)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return cities

	case "get_preference_cities":
		cityID := getOptionalIntParam(params, "city_id")
		items, err := ac.preferenceCityService.GetAll(orgID, cityID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return items

	case "get_customer_list":
		customerName := getStringParam(params, "customer_name")
		items, err := ac.customersService.ListCustomers(orgID, customerName)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return items

	case "get_customer_detail":
		customerID := getStringParam(params, "customer_id")
		if customerID == "" {
			return map[string]interface{}{"error": "customer_id is required"}
		}
		data, err := ac.customersService.GetCustomerDetail(orgID, customerID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return data

	case "get_inventory_items":
		items, err := ac.inventoryService.GetItems(orgID, 0)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		search := strings.ToLower(strings.TrimSpace(getStringParam(params, "search")))
		garageID := strings.ToLower(strings.TrimSpace(getStringParam(params, "garage_id")))
		if search != "" || garageID != "" {
			filtered := make([]model.InventoryItemWithLabel, 0, len(items))
			for _, item := range items {
				itemSearch := strings.ToLower(item.ItemName + " " + item.ItemSKU)
				garageNames := strings.ToLower(item.GarageNames)
				if search != "" && !strings.Contains(itemSearch, search) {
					continue
				}
				if garageID != "" && !strings.Contains(garageNames, garageID) {
					continue
				}
				filtered = append(filtered, item)
			}
			items = filtered
		}

		return map[string]interface{}{
			"items": items,
			"count": len(items),
		}

	case "get_inventory_detail":
		itemID := getStringParam(params, "item_id")
		if itemID == "" {
			return map[string]interface{}{"error": "item_id is required"}
		}
		if _, err := ac.inventoryService.GetItem(orgID, itemID); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		detail, err := ac.inventoryService.GetItemDetail(itemID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		totalStock := 0
		for _, location := range detail.Locations {
			totalStock += location.Stock
		}

		return map[string]interface{}{
			"item_id":       detail.ItemID,
			"item_sku":      detail.ItemSKU,
			"item_name":     detail.ItemName,
			"item_uom":      detail.ItemUOM,
			"item_category": detail.ItemCategory,
			"total_stock":   totalStock,
			"locations":     detail.Locations,
		}

	case "get_inventory_stock":
		itemID := getStringParam(params, "item_id")
		if itemID == "" {
			return map[string]interface{}{"error": "item_id is required"}
		}
		if _, err := ac.inventoryService.GetItem(orgID, itemID); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		detail, err := ac.inventoryService.GetItemDetail(itemID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		garageID := strings.TrimSpace(getStringParam(params, "garage_id"))
		garageIDLower := strings.ToLower(garageID)
		totalStock := 0
		stock := 0
		garageName := ""
		foundGarageID := garageID
		garageFound := garageID == ""
		for _, location := range detail.Locations {
			totalStock += location.Stock
			if garageID != "" && (strings.EqualFold(location.GarageID, garageID) || strings.Contains(strings.ToLower(location.GarageName), garageIDLower)) {
				stock = location.Stock
				garageName = location.GarageName
				foundGarageID = location.GarageID
				garageFound = true
			}
		}
		if garageID != "" && !garageFound {
			return map[string]interface{}{"error": "garage_id not found for this item"}
		}
		if garageID == "" {
			stock = totalStock
		}

		return map[string]interface{}{
			"item_id":       detail.ItemID,
			"item_name":     detail.ItemName,
			"item_uom":      detail.ItemUOM,
			"item_category": detail.ItemCategory,
			"garage_id":     foundGarageID,
			"garage_name":   garageName,
			"stock":         stock,
			"total_stock":   totalStock,
			"locations":     detail.Locations,
		}

	case "get_garage_list":
		itemID := getStringParam(params, "item_id")
		garages, err := ac.garageService.GetGarages(orgID, itemID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return garages

	case "get_item_suppliers":
		suppliers, err := ac.inventoryService.GetSuppliers(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return suppliers

	case "get_item_movements":
		itemID := getStringParam(params, "item_id")
		if itemID == "" {
			return map[string]interface{}{"error": "item_id is required"}
		}
		movements, err := ac.inventoryService.GetItemMovements(orgID, itemID, getStringParam(params, "start_date"), getStringParam(params, "end_date"), getStringParam(params, "garage_id"))
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"movements": movements,
			"count":     len(movements),
		}

	case "get_item_order_history":
		itemID := getStringParam(params, "item_id")
		if itemID == "" {
			return map[string]interface{}{"error": "item_id is required"}
		}
		history, err := ac.inventoryService.GetItemOrderHistory(orgID, itemID, getStringParam(params, "start_date"), getStringParam(params, "end_date"))
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"history": history,
			"count":   len(history),
		}

	case "get_item_stock_distribution":
		itemID := getStringParam(params, "item_id")
		if itemID == "" {
			return map[string]interface{}{"error": "item_id is required"}
		}
		detail, err := ac.inventoryService.GetItemDetail(itemID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		totalStock := 0
		for _, location := range detail.Locations {
			totalStock += location.Stock
		}

		return map[string]interface{}{
			"item_id":       detail.ItemID,
			"item_sku":      detail.ItemSKU,
			"item_name":     detail.ItemName,
			"item_uom":      detail.ItemUOM,
			"item_category": detail.ItemCategory,
			"total_stock":   totalStock,
			"locations":     detail.Locations,
		}

	case "get_purchase_order_list":
		orders, err := ac.inventoryService.GetOrders(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"orders": orders,
			"count":  len(orders),
		}

	case "get_purchase_order_detail":
		purchaseID := getStringParam(params, "purchase_id")
		if purchaseID == "" {
			return map[string]interface{}{"error": "purchase_id is required"}
		}
		order, err := ac.inventoryService.GetOrder(purchaseID, orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return order

	case "complete_purchase_order":
		purchaseID := getStringParam(params, "purchase_id")
		if purchaseID == "" {
			return map[string]interface{}{"error": "purchase_id is required"}
		}
		if err := ac.inventoryService.ReceiveRequest(orgID, userID, &model.ReceiveInventoryOrderRequest{PurchaseID: purchaseID}); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":      "success",
			"message":     "Purchase order completed successfully",
			"purchase_id": purchaseID,
		}

	case "cancel_purchase_order":
		purchaseID := getStringParam(params, "purchase_id")
		if purchaseID == "" {
			return map[string]interface{}{"error": "purchase_id is required"}
		}
		if err := ac.inventoryService.CancelOrder(orgID, userID, purchaseID); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":      "success",
			"message":     "Purchase order cancelled successfully",
			"purchase_id": purchaseID,
		}

	case "create_new_item":
		return ac.createNewItem(ctx, orgID, userID, params)

	case "create_inventory_request":
		return ac.createInventoryRequest(ctx, orgID, userID, params)

	case "approve_inventory_request":
		return ac.approveInventoryRequest(ctx, orgID, userID, params)

	case "reject_inventory_request":
		return ac.rejectInventoryRequest(ctx, orgID, userID, params)

	case "get_booking_list":
		status := getStringParam(params, "status")
		limitVal := 10
		if limit := getIntParam(params, "limit"); limit > 0 {
			limitVal = limit
		}
		return ac.toolExec.ExecuteGetBookingList(ctx, orgID, status, limitVal)

	case "get_revenue_summary":
		period := getStringParam(params, "period")
		if period == "" {
			return map[string]interface{}{"error": "period is required"}
		}
		return ac.toolExec.ExecuteGetRevenueSummary(ctx, orgID, period)

	case "get_organization_info":
		fmt.Println("------ get organization info")
		res, err := ac.organizationService.GetOrganizationDetail(orgID)
		fmt.Println("res:", res)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return res

	case "get_bank_accounts":
		roleName, _ := ctx.Value(contextRoleName).(string)
		phone, _ := ctx.Value(phoneKey).(string)
		if strings.TrimSpace(roleName) == "CustomerAssistant" && strings.TrimSpace(phone) != "" {
			if blocked := ac.customerHasUnconfirmedOrderStatus2(ctx, orgID, strings.TrimSpace(phone)); blocked {
				return map[string]interface{}{
					"error":   "ORDER_UNCONFIRMED",
					"message": "Pembayaran dapat dilakukan setelah pesanan selesai ditinjau dan dikonfirmasi oleh tim.",
				}
			}
			if confirmed := ac.customerHasConfirmedOrderStatus1(ctx, orgID, strings.TrimSpace(phone)); !confirmed {
				return map[string]interface{}{
					"error":   "ORDER_NOT_CONFIRMED",
					"message": "Pembayaran dapat dilakukan setelah pesanan dibuat (create_order) dan dikonfirmasi oleh tim.",
				}
			}
		}
		accounts, err := ac.organizationService.GetBankAccounts(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return accounts

	case "get_trip_distance":
		from := getStringParam(params, "from", "origin", "pickup")
		to := getStringParam(params, "to", "destination", "dest")
		if strings.TrimSpace(to) == "" {
			return map[string]interface{}{"error": "to is required"}
		}
		if strings.TrimSpace(from) == "" {
			orgDetail, err := ac.organizationService.GetOrganizationDetail(orgID)
			if err == nil && orgDetail != nil {
				if v, ok := orgDetail["city_label"].(string); ok && strings.TrimSpace(v) != "" {
					from = strings.TrimSpace(v)
				} else if v, ok := orgDetail["city"].(string); ok && strings.TrimSpace(v) != "" {
					from = strings.TrimSpace(v)
				}
			}
		}

		distanceKm, method, err := estimateTripDistanceKm(ctx, from, to)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		minOverland := 1
		switch {
		case distanceKm > 700:
			minOverland = 4
		case distanceKm > 400:
			minOverland = 3
		case distanceKm > 250:
			minOverland = 2
		}

		minDrop := 1
		switch {
		case distanceKm > 700:
			minDrop = 3
		case distanceKm > 400:
			minDrop = 2
		default:
			minDrop = 1
		}

		return map[string]interface{}{
			"from":               from,
			"to":                 to,
			"distance_km":        distanceKm,
			"min_days":           minOverland,
			"min_days_overland":  minOverland,
			"min_days_drop_only": minDrop,
			"method":             method,
		}

	case "get_order_list":
		fmt.Println("------ get order list")
		req := &model.PartnerOrderListFilter{
			StartDateFrom: getStringParam(params, "start_date"),
			StartDateTo:   getStringParam(params, "end_date"),
			Search:        getStringParam(params, "search"),
		}
		if ps := getStringParam(params, "payment_status"); ps != "" {
			if n, err := strconv.Atoi(ps); err == nil {
				req.PaymentStatus = n
				req.HasPaymentStatus = true
			}
		} else if n := getIntParam(params, "payment_status"); n > 0 {
			req.PaymentStatus = n
			req.HasPaymentStatus = true
		}
		req.OrderDateFrom, req.OrderDateTo = resolveOrderDateRange(params)
		res, err := ac.fleetService.GetPartnerOrdersWithSummary(orgID, req)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		// Filter by phone number for Company Assistant: only show orders belonging to this customer
		phone, _ := ctx.Value(phoneKey).(string)
		roleName, _ := ctx.Value(contextRoleName).(string)
		if roleName == "CustomerAssistant" && phone != "" {
			filtered := make([]model.PartnerOrderListItem, 0, len(res.Orders))
			for _, o := range res.Orders {
				if strings.TrimSpace(o.CustomerPhone) == "" || strings.TrimSpace(o.CustomerPhone) == phone {
					filtered = append(filtered, o)
				}
			}
			res.Orders = filtered
		}

		return enrichOrderListForAI(res)

	case "get_order_detail":
		fmt.Println("------ get order detail")
		orderID := getStringParam(params, "order_id")
		fmt.Println("params:", params)
		fmt.Println("orderID:", orderID)
		if orderID == "" {
			return map[string]interface{}{"error": "order_id is required"}
		}
		res, err := ac.fleetService.GetPartnerOrderDetail(orderID, orgID)
		if err != nil {
			fmt.Println("error get order detail:", err)
			return map[string]interface{}{"error": err.Error()}
		}

		// For Company Assistant: validate this order belongs to the customer
		phone, _ := ctx.Value(phoneKey).(string)
		roleName, _ := ctx.Value(contextRoleName).(string)
		if roleName == "CustomerAssistant" && phone != "" {
			if res.Customer.CustomerPhone != "" && res.Customer.CustomerPhone != phone {
				return map[string]interface{}{"error": "Pesanan ini bukan milik Anda"}
			}
		}

		fmt.Println("payment status", res.PaymentStatus)
		fmt.Println("payment status label", res.PaymentStatusLabel)
		return enrichOrderDetailForAI(res)
	case "get_schedule_list":
		items, err := ac.scheduleService.GetScheduleFleetList(model.ScheduleFleetListServiceInput{
			OrganizationID: orgID,
			Query: model.ScheduleFleetListQuery{
				Period:         getStringParam(params, "period"),
				OrderID:        getStringParam(params, "order_id"),
				FleetID:        getStringParam(params, "fleet_id"),
				Search:         getStringParam(params, "search"),
				FleetName:      getStringParam(params, "fleet_name"),
				PlateNumber:    getStringParam(params, "plate"),
				ProductionYear: getStringParam(params, "production_year"),
			},
		})
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return items

	case "get_schedule_detail":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}
		res, err := ac.scheduleService.GetFleetTripDetail(model.ScheduleFleetTripDetailServiceInput{
			OrganizationID: orgID,
			ScheduleNumber: scheduleNumber,
		})
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return res

	case "get_order_payment_history":
		orderID := getStringParam(params, "order_id")
		if orderID == "" {
			return map[string]interface{}{"error": "order_id is required"}
		}
		history, err := ac.orderService.GetServiceOrderPaymentHistory(orgID, &model.ServiceOrderPaymentHistoryRequest{
			OrderID:   orderID,
			OrderType: 1, // Fleet order type
		})
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return history

	case "approve_order":
		orderID := getStringParam(params, "order_id")
		if orderID == "" {
			return map[string]interface{}{"error": "order_id is required"}
		}
		// Get user ID from context
		userID, _ := ctx.Value(contextUserID).(string)
		err := ac.fleetService.ProcessFleetOrder(orgID, userID, orderID, 1) // 1 = approve
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":   "success",
			"message":  "Order approved successfully",
			"order_id": orderID,
		}

	case "reject_order":
		orderID := getStringParam(params, "order_id")
		if orderID == "" {
			return map[string]interface{}{"error": "order_id is required"}
		}
		// Get user ID from context
		userID, _ := ctx.Value(contextUserID).(string)
		err := ac.fleetService.ProcessFleetOrder(orgID, userID, orderID, 0) // 0 = reject
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":   "success",
			"message":  "Order rejected successfully",
			"order_id": orderID,
		}

	case "get_employee_shift_schedule":
		req := &model.EmployeeShiftScheduleRequest{
			StartDate:  getStringParam(params, "start_date"),
			EndDate:    getStringParam(params, "end_date"),
			RoleID:     getStringParam(params, "role_id"),
			DivisionID: getStringParam(params, "division_id"),
		}
		schedule, err := ac.organizationService.EmployeeShiftSchedule(orgID, req)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return schedule

	case "add_employee_off_day":
		employeeID := getStringParam(params, "employee_id")
		shiftDate := getStringParam(params, "shift_date")
		if employeeID == "" || shiftDate == "" {
			return map[string]interface{}{"error": "employee_id and shift_date are required"}
		}
		// Get user ID from context
		userID, _ := ctx.Value(contextUserID).(string)
		// Get shift type from params, default to a reasonable value
		shiftType := 1
		if st := getIntParam(params, "shift_type"); st > 0 {
			shiftType = st
		}
		req := &model.EmployeeShiftSetScheduleRequest{
			Type:       "submit",
			EmployeeID: employeeID,
			ShiftDate:  shiftDate,
			ShiftType:  shiftType,
		}
		result, err := ac.organizationService.EmployeeShiftSetSchedule(orgID, userID, req)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":  "success",
			"message": "Off day added successfully",
			"result":  result,
		}

	case "get_monthly_revenue":
		monthStr := getStringParam(params, "month")
		// If no month provided, use current month
		now := time.Now()
		if monthStr == "" {
			monthStr = now.Format("2006-01")
		}
		// Parse month to get start and end dates
		monthTime, err := time.Parse("2006-01", monthStr)
		if err != nil {
			monthTime = now
		}
		startDate := time.Date(monthTime.Year(), monthTime.Month(), 1, 0, 0, 0, 0, time.Local)
		endDate := startDate.AddDate(0, 1, -1)
		// Get finance data
		finance, err := ac.dashboardService.GetFinance(orgID, startDate, endDate)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		// Calculate profit
		profit := finance.Summary.TotalRevenue - finance.Summary.TotalExpenses
		return map[string]interface{}{
			"month":          monthStr,
			"total_revenue":  finance.Summary.TotalRevenue,
			"total_expenses": finance.Summary.TotalExpenses,
			"profit":         profit,
			"finance_data":   finance,
		}

	case "get_top_fleets":
		topFleets, err := ac.dashboardService.GetTopFleets(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return topFleets

	case "get_top_destinations":
		topDestinations, err := ac.dashboardService.GetTopDestinations(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return topDestinations

	case "get_top_customers":
		topCustomers, err := ac.dashboardService.GetTopCustomers(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return topCustomers

	case "get_spj_total_biaya":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}
		totalAmount, err := ac.transactionService.GetFleetTripTotalAmount(scheduleNumber)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		summary, err := ac.transactionService.GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber, orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"schedule_number":            scheduleNumber,
			"total_jatah_uang":           totalAmount,
			"biaya_operasional":          totalAmount,
			"biaya_operasional_terpakai": summary.TotalExpenses,
			"total_claimed":              summary.TotalClaimed,
			"total_item_reimburse":       summary.TotalItemReimburse,
			"total_reimburse":            summary.TotalReimburse,
			"remaining_claim":            summary.RemainingClaim,
			"total_pengeluaran":          summary.TotalExpenses + summary.TotalClaimed + summary.TotalItemReimburse,
			"saldo_sisa":                 totalAmount - (summary.TotalExpenses + summary.TotalClaimed + summary.TotalReimburse),
		}

	case "tambah_pengeluaran_spj":
		scheduleNumber := getStringParam(params, "schedule_number")
		transactionItemInput := getStringParam(params, "transaction_item")
		amount := getFloatParam(params, "amount")
		paymentMethod := getIntParam(params, "payment_method")
		description := getStringParam(params, "description")

		if scheduleNumber == "" || transactionItemInput == "" || amount <= 0 {
			return map[string]interface{}{"error": "schedule_number, transaction_item, and amount are required"}
		}

		// Load transaction items from common.json to get valid codes
		type TransactionItem struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		}
		var validTransactionItems []TransactionItem
		var transactionItemCode string = "TRX-I00" // Default

		f, err := os.Open("config/common.json")
		if err == nil {
			defer f.Close()
			var cfg struct {
				TransactionItems []TransactionItem `json:"transaction-items"`
			}
			if err := json.NewDecoder(f).Decode(&cfg); err == nil {
				validTransactionItems = cfg.TransactionItems

				// Check if input is already a valid code
				inputUpper := strings.ToUpper(strings.TrimSpace(transactionItemInput))
				found := false
				for _, item := range validTransactionItems {
					if strings.ToUpper(item.ID) == inputUpper {
						transactionItemCode = item.ID
						found = true
						break
					}
				}

				// If not a valid code, try to match by label
				if !found {
					inputLower := strings.ToLower(transactionItemInput)
					for _, item := range validTransactionItems {
						if strings.Contains(strings.ToLower(item.Label), inputLower) {
							transactionItemCode = item.ID
							// If description is empty, use the original input as description
							if description == "" {
								description = transactionItemInput
							}
							break
						}
					}
				}
			}
		}

		if paymentMethod == 0 {
			paymentMethod = 1 // Default to Biaya Operasional
		}

		err = ac.transactionService.SubmitFleetTripExpense(orgID, userID, transactionItemCode, scheduleNumber, paymentMethod, amount, description)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"status":  "success",
			"message": "Pengeluaran SPJ berhasil ditambahkan",
		}

	case "get_spj_pengeluaran":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}
		expenses, err := ac.transactionService.ListFleetTripExpenses(scheduleNumber, orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return expenses

	case "get_spj_ringkasan_pembayaran":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}
		totalAmount, err := ac.transactionService.GetFleetTripTotalAmount(scheduleNumber)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		summary, err := ac.transactionService.GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber, orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"schedule_number":            scheduleNumber,
			"total_jatah_uang":           totalAmount,
			"biaya_operasional":          totalAmount,
			"biaya_operasional_terpakai": summary.TotalExpenses,
			"total_claimed":              summary.TotalClaimed,
			"total_item_reimburse":       summary.TotalItemReimburse,
			"remaining_claim":            summary.RemainingClaim,
			"reimburse":                  summary.TotalReimburse,
			"total_pengeluaran":          summary.TotalExpenses + summary.TotalClaimed + summary.TotalReimburse,
			"saldo_sisa":                 totalAmount - (summary.TotalExpenses + summary.TotalClaimed + summary.TotalReimburse),
		}

	case "print_surat_jalan":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}
		roleName, _ := ctx.Value(contextRoleName).(string)
		sendResultHook := buildAssistantSendResultHook(ac.db, ac.driver, orgID, roleName)

		log.Printf("[WAAI][AI] print_surat_jalan called with schedule_number: '%s'", scheduleNumber)

		// Generate PDF
		pdfData, err := ac.printService.GenerateFleetTripsPDF(orgID, scheduleNumber)
		if err != nil {
			log.Printf("[WAAI][AI] Failed to generate PDF for %s: %v", scheduleNumber, err)
			return map[string]interface{}{"error": "Gagal membuat file PDF: " + err.Error()}
		}

		// Get phone number from context
		phone, _ := ctx.Value(phoneKey).(string)
		if phone == "" {
			return map[string]interface{}{"error": "phone number missing in context"}
		}

		// Sanitize filename
		filename := fmt.Sprintf("surat-jalan-%s.pdf", strings.ReplaceAll(scheduleNumber, "/", "-"))
		caption := fmt.Sprintf("Berikut surat jalan untuk *%s*", scheduleNumber)

		// Simpan PDF ke folder assets/temp/surat-jalan/ sebagai file sementara
		tempDir := filepath.Join("assets", "temp", "surat-jalan")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			log.Printf("[WAAI][AI] Failed to create temp dir: %v", err)
			return map[string]interface{}{"error": "Gagal menyimpan file sementara: " + err.Error()}
		}
		tempPath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(tempPath, pdfData, 0644); err != nil {
			log.Printf("[WAAI][AI] Failed to write temp file: %v", err)
			return map[string]interface{}{"error": "Gagal menyimpan file sementara: " + err.Error()}
		}

		// Bangun URL publik (APP_HOST_URL > APP_HOST > fallback)
		baseURL := strings.TrimSuffix(os.Getenv("APP_HOST_URL"), "/")
		if baseURL == "" {
			baseURL = strings.TrimSuffix(os.Getenv("APP_HOST"), "/")
		}
		relativePath := strings.ReplaceAll(tempPath, "\\", "/")
		mediaURL := fmt.Sprintf("%s/%s", baseURL, relativePath)

		log.Printf("[WAAI][AI] Attempting to send PDF %s to %s via URL: %s", filename, phone, mediaURL)

		// Kirim via URL — Wagy akan download dari URL ini
		_, err = ac.wagyClient.SendDocumentWithURLAndHook(phone, filename, mediaURL, caption, sendResultHook)
		if err != nil {
			log.Printf("[WAAI][AI] Failed to send PDF: %v", err)
			_ = os.Remove(tempPath) // Bersihkan file meskipun gagal
			return map[string]interface{}{"error": "Gagal mengirim surat jalan ke WhatsApp: " + err.Error()}
		}

		// Hapus file setelah berhasil dikirim
		if err := os.Remove(tempPath); err != nil {
			log.Printf("[WAAI][AI] Warning: failed to remove temp file %s: %v", tempPath, err)
		}

		log.Printf("[WAAI][AI] PDF successfully sent via URL and temp file cleaned up")
		return map[string]interface{}{
			"status":  "success",
			"message": "Surat jalan " + scheduleNumber + " berhasil dikirim ke WhatsApp Anda",
		}

	case "get_fleet_availibility_by_daterange":
		layout := "2006-01-02 15:04"
		startDate, err := time.ParseInLocation(layout, getStringParam(params, "start_date"), time.Local)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		endDate, err := time.ParseInLocation(layout, getStringParam(params, "end_date"), time.Local)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		fleetID := getStringParam(params, "fleet_id")
		availibility, _, err := ac.fleetService.GetFleetAvailibility(orgID, startDate, endDate, fleetID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return availibility

	case "get_fleet_unit_detail":
		unitID := getStringParam(params, "unit_id")
		unit, err := ac.fleetUnitService.Detail(orgID, unitID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return unit

	case "get_fleet_unit_by_partner":
		fleetID := getStringParam(params, "fleet_id")
		items, err := ac.partnerService.Detail(
			&model.OperationPartnerDetailRequest{
				PartnerID: fleetID,
			},
			orgID,
		)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return items

	case "get_upcoming_unit_schedule":
		unitID := getStringParam(params, "unit_id")
		_, _, upcoming, err := ac.fleetUnitService.UnitScheduleStats(orgID, unitID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return upcoming

	case "get_latest_unit_schedule":
		unitID := getStringParam(params, "unit_id")
		_, latest, _, err := ac.fleetUnitService.UnitScheduleStats(orgID, unitID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return latest

	case "get_unit_trip_history":
		unitID := getStringParam(params, "unit_id")
		startDateStr := getStringParam(params, "start_date")
		endDateStr := getStringParam(params, "end_date")
		layout := "2006-01-02"
		startDate, err := time.ParseInLocation(layout, startDateStr, time.Local)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		endDate, err := time.ParseInLocation(layout, endDateStr, time.Local)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		trips, err := ac.fleetUnitService.UnitOrderHistory(orgID, unitID, startDate.Format(layout), endDate.Format(layout))
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return trips

	case "get_fleet_prices":
		fleetID := getStringParam(params, "fleet_id")
		typeID := getStringParam(params, "type_id")
		if fleetID == "" || typeID == "" {
			return map[string]interface{}{"error": "fleet_id and type_id are required"}
		}
		prices, err := ac.fleetService.GetFleetPricesByFleetID(orgID, fleetID, typeID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return prices

	case "get_fleet_addons":
		fleetID := getStringParam(params, "fleet_id")
		if fleetID == "" {
			return map[string]interface{}{"error": "fleet_id is required"}
		}
		addons, err := ac.fleetService.GetFleetAddonList(orgID, fleetID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return addons

	case "create_order":
		return ac.executeCreateOrder(ctx, orgID, userID, params)

	case "print_invoice":
		orderID := getStringParam(params, "order_id")
		if orderID == "" {
			return map[string]interface{}{"error": "order_id is required"}
		}
		roleName, _ := ctx.Value(contextRoleName).(string)
		sendResultHook := buildAssistantSendResultHook(ac.db, ac.driver, orgID, roleName)

		// Validasi nomor customer: hanya bisa akses invoice miliknya sendiri
		phone, _ := ctx.Value(phoneKey).(string)
		if phone != "" {
			detail, err := ac.fleetService.GetPartnerOrderDetail(orderID, orgID)
			if err != nil {
				return map[string]interface{}{"error": "Pesanan tidak ditemukan"}
			}
			if detail.Customer.CustomerPhone != "" && detail.Customer.CustomerPhone != phone {
				return map[string]interface{}{"error": "Invoice ini bukan milik Anda"}
			}
		}

		invoiceNum := getStringParam(params, "invoice_number")
		var invPtr *string
		if invoiceNum != "" {
			invPtr = &invoiceNum
		}

		pdfData, err := ac.printService.GenerateFleetInvoicePDF(orgID, orderID, invPtr)
		if err != nil {
			return map[string]interface{}{"error": "Gagal membuat invoice: " + err.Error()}
		}

		phone, _ = ctx.Value(phoneKey).(string)
		if phone == "" {
			return map[string]interface{}{"error": "Nomor telepon tidak ditemukan di konteks"}
		}

		filename := fmt.Sprintf("invoice-%s.pdf", orderID)
		caption := fmt.Sprintf("Berikut invoice untuk pesanan *%s*", orderID)
		_, err = ac.wagyClient.SendDocumentWithHook(phone, filename, pdfData, caption, sendResultHook)
		if err != nil {
			return map[string]interface{}{"error": "Gagal kirim invoice: " + err.Error()}
		}

		log.Printf("[WAAI][AI] Invoice %s sent to %s", orderID, phone)
		return map[string]interface{}{
			"status":   "success",
			"message":  "Invoice " + orderID + " berhasil dikirim ke WhatsApp Anda",
			"order_id": orderID,
		}

	default:
		return map[string]interface{}{
			"error": "Unknown tool: " + toolName,
		}
	}
}

func (ac *AIClient) getFleetUnits(orgID string, params map[string]interface{}) interface{} {
	items, err := ac.fleetUnitService.List(
		orgID,
		getStringParam(params, "fleet_id"),
		getStringParam(params, "order_id"),
		getStringParam(params, "search"),
	)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return items
}

func (ac *AIClient) createNewItem(ctx context.Context, orgID, userID string, params map[string]interface{}) interface{} {
	itemID := getStringParam(params, "item_id")
	itemName := getStringParam(params, "item_name")
	itemSKU := getStringParam(params, "item_sku")
	itemUOM := getStringParam(params, "item_uom")
	itemCategory := getIntParam(params, "item_category")
	stock := getIntParam(params, "stock")
	garageID := getStringParam(params, "garage_id")
	transactionType := getStringParam(params, "transaction_type")
	transactionDate := getStringParam(params, "transaction_date")
	itemPrice := getFloatParam(params, "item_price")
	supplierID := getStringParam(params, "supplier_id")
	supplierName := getStringParam(params, "supplier_name")
	supplierPhone := getStringParam(params, "supplier_phone")
	supplierURL := getStringParam(params, "supplier_url")
	supplierPrice := getFloatParam(params, "supplier_price")
	notes := getStringParam(params, "notes")

	missing := make([]string, 0)
	if itemID == "" && itemName == "" {
		missing = append(missing, "item_id or item_name")
	}
	if itemUOM == "" {
		missing = append(missing, "item_uom")
	}
	if itemCategory == 0 {
		missing = append(missing, "item_category")
	} else if itemCategory != 1 && itemCategory != 2 {
		missing = append(missing, "item_category must be 1 or 2")
	}
	if stock <= 0 {
		missing = append(missing, "stock")
	}
	if garageID == "" {
		missing = append(missing, "garage_id")
	}
	if transactionType == "" {
		missing = append(missing, "transaction_type")
	} else if transactionType != "1" && transactionType != "2" {
		missing = append(missing, "transaction_type must be 1 or 2")
	}
	if transactionType == "2" {
		if itemPrice <= 0 {
			missing = append(missing, "item_price")
		}
		if transactionDate == "" {
			missing = append(missing, "transaction_date")
		}
		if supplierID == "" && supplierName == "" {
			missing = append(missing, "supplier_id or supplier_name")
		}
	}

	if len(missing) > 0 {
		return map[string]interface{}{
			"error":             "missing required parameters",
			"missing_required":  missing,
			"item_uom_examples": []string{"Pcs", "Box", "Liter", "Unit"},
			"item_category": map[string]string{
				"1": "Kebutuhan Armada",
				"2": "kebutuhan kantor",
			},
			"transaction_type": map[string]string{
				"1": "tambah stok yang ada",
				"2": "update stock sesuai input (tidak menambahkan stock yang sudah ada)",
			},
		}
	}

	if itemSKU == "" {
		generatedSKU, err := ac.inventoryService.GenerateItemSKU(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		itemSKU = generatedSKU
	}

	req := &model.CreateInventoryItemRequest{
		ItemID:          itemID,
		ItemSKU:         itemSKU,
		ItemName:        itemName,
		ItemUOM:         itemUOM,
		ItemCategory:    itemCategory,
		Stock:           stock,
		GarageID:        garageID,
		TransactionType: transactionType,
		TransactionDate: transactionDate,
		SupplierID:      supplierID,
		SupplierName:    supplierName,
		SupplierPhone:   supplierPhone,
		SupplierURL:     supplierURL,
		SupplierPrice:   supplierPrice,
		Notes:           notes,
	}

	item, err := ac.inventoryService.CreateItem(orgID, userID, req)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"status":           "success",
		"message":          "Item created successfully",
		"item_id":          item.ItemID,
		"item_sku":         itemSKU,
		"item_name":        itemName,
		"item_uom":         itemUOM,
		"item_category":    itemCategory,
		"stock":            stock,
		"garage_id":        garageID,
		"transaction_type": transactionType,
	}
}

type waaiContextKey string

const (
	contextOrganizationID      waaiContextKey = "organization_id"
	contextUserID              waaiContextKey = "user_id"
	contextRoleName            waaiContextKey = "role_name"
	contextSuppressAdminNotify waaiContextKey = "suppress_admin_notify"
)

func withAuthorizedTenantContext(ctx context.Context, tenant *TenantInfo) context.Context {
	if tenant == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, contextOrganizationID, tenant.OrganizationID)
	ctx = context.WithValue(ctx, contextUserID, tenant.UserID)
	ctx = context.WithValue(ctx, contextRoleName, tenant.RoleName)
	return ctx
}

func getAuthorizedContextValues(ctx context.Context) (string, error) {
	orgID, _ := ctx.Value(contextOrganizationID).(string)
	userID, _ := ctx.Value(contextUserID).(string)
	roleName, _ := ctx.Value(contextRoleName).(string)
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	roleName = strings.TrimSpace(roleName)

	if orgID == "" {
		return "", fmt.Errorf("missing organization_id in context")
	}
	if userID == "" && roleName != "CustomerAssistant" {
		return "", fmt.Errorf("missing user_id in context")
	}
	if roleName == "" {
		log.Printf("[WAAI][AI] role_name missing in context for org=%s user=%s", orgID, userID)
	}
	return orgID, nil
}

func getStringParam(params map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		raw, ok := params[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case float64:
			if v == float64(int64(v)) {
				return strconv.FormatInt(int64(v), 10)
			}
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		}
	}
	return ""
}

func getIntParam(params map[string]interface{}, key string) int {
	raw, ok := params[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func getFloatParam(params map[string]interface{}, key string) float64 {
	raw, ok := params[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func getOptionalIntParam(params map[string]interface{}, key string) *int {
	n := getIntParam(params, key)
	if n == 0 {
		return nil
	}
	return &n
}

func normalizeCityFilters(provinceID, provinceName string) (string, string) {
	provinceID = strings.TrimSpace(provinceID)
	provinceName = strings.TrimSpace(provinceName)
	if provinceID == "" && provinceName != "" {
		onlyDigits := true
		for i := 0; i < len(provinceName); i++ {
			ch := provinceName[i]
			if ch < '0' || ch > '9' {
				onlyDigits = false
				break
			}
		}
		if onlyDigits {
			return provinceName, ""
		}
	}
	return provinceID, provinceName
}

func parseFleetAvailabilityDates(startStr, endStr string) (time.Time, time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02",
	}

	parseOne := func(value string) (time.Time, error) {
		value = strings.TrimSpace(value)
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid datetime format")
	}

	startDate, err := parseOne(startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date")
	}
	endDate, err := parseOne(endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date")
	}
	return startDate, endDate, nil
}

var (
	markdownBoldTriple = regexp.MustCompile(`\*\*\*([^*\n]+?)\*\*\*`)
	markdownBoldDouble = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	markdownUnderline  = regexp.MustCompile(`__([^_\n]+?)__`)
	markdownHeader     = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	emojiPattern       = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]`)
	multiBlankLines    = regexp.MustCompile(`\n{3,}`)
	openingWelcomeLine = regexp.MustCompile(`(?i)^halo[!,.]?\s*selamat datang di\s+([^\n.!?]+)[.!?]?\s*`)
	helpQuestionLine   = regexp.MustCompile(`(?i)^(ada yang bisa (saya|kami) bantu\??.*)$`)
)

// formatWhatsAppReply normalizes model output to WhatsApp-friendly formatting.
func formatWhatsAppReply(text string) string {
	if text == "" {
		return text
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = markdownBoldTriple.ReplaceAllString(text, "*$1*")
	for strings.Contains(text, "**") {
		next := markdownBoldDouble.ReplaceAllString(text, "*$1*")
		if next == text {
			break
		}
		text = next
	}
	text = markdownUnderline.ReplaceAllString(text, "*$1*")
	text = markdownHeader.ReplaceAllString(text, "")
	text = emojiPattern.ReplaceAllString(text, "")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")
	text = multiBlankLines.ReplaceAllString(text, "\n\n")
	text = normalizeGreetingStyle(text)

	return strings.TrimSpace(text)
}

func normalizeGreetingStyle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}

	if matches := openingWelcomeLine.FindStringSubmatch(text); len(matches) == 2 {
		companyName := strings.TrimSpace(matches[1])
		rest := strings.TrimSpace(openingWelcomeLine.ReplaceAllString(text, ""))
		if rest != "" {
			rest = helpQuestionLine.ReplaceAllString(rest, "$1")
			return "Halo, Selamat datang di " + companyName + "!\n" + rest
		}
		return "Halo, Selamat datang di " + companyName + "!"
	}

	return text
}

func resolveOrderDateRange(params map[string]interface{}) (string, string) {
	if period := strings.TrimSpace(getStringParam(params, "period")); period != "" {
		if from, to, ok := monthPeriodToOrderDateRange(period); ok {
			return from, to
		}
	}

	from := getStringParam(params, "order_date_from", "order_date_start")
	to := getStringParam(params, "order_date_to", "order_date_end")
	return from, to
}

func enrichOrderListForAI(res *model.PartnerOrderListResponse) map[string]interface{} {
	if res == nil {
		return map[string]interface{}{"orders": []map[string]interface{}{}}
	}

	orders := make([]map[string]interface{}, 0, len(res.Orders))
	seenOrderIDs := make(map[string]struct{}, len(res.Orders))
	totalOrders := 0
	paid := 0
	pending := 0
	revenue := 0.0
	for _, o := range res.Orders {
		orderID := strings.TrimSpace(o.OrderID)
		if orderID == "" {
			continue
		}
		if _, exists := seenOrderIDs[orderID]; exists {
			continue
		}
		seenOrderIDs[orderID] = struct{}{}
		totalOrders++

		paymentStatus := int(o.PaymentStatus)
		paymentStatusLabel := paymentStatusLabelForAI(paymentStatus, o.PaymentStatusLabel)
		switch paymentStatus {
		case 1: // Paid
			paid++
			revenue += o.TotalAmount
		case 2, 3, 10: // Pending verification, partial paid, waiting approval
			pending++
			if paymentStatus == 3 { // Partial paid should still count as some revenue?
				revenue += o.TotalAmount
			}
		case 0: // Cancelled
			// Do nothing for cancelled orders
		}

		scheduled := strings.TrimSpace(o.ScheduleID) != ""
		item := map[string]interface{}{
			"order_id":             orderID,
			"customer_name":        o.CustomerName,
			"customer_phone":       o.CustomerPhone,
			"fleet_name":           o.FleetName,
			"start_date":           o.StartDate,
			"end_date":             o.EndDate,
			"total_amount":         o.TotalAmount,
			"payment_status":       paymentStatus,
			"payment_status_label": paymentStatusLabel,
			"scheduled":            scheduled,
			"schedule_id":          o.ScheduleID,
			"is_scheduled":         scheduled,
		}
		if o.LatestPaymentStatus != "" {
			item["latest_payment_type"] = o.LatestPaymentStatus
		}
		orders = append(orders, item)
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_orders":                         totalOrders,
			"paid":                                 paid,
			"pending":                              pending,
			"lunas":                                paid,
			"menunggu_verifikasi_atau_belum_lunas": pending,
			"revenue":                              revenue,
			"ongoing":                              res.Summary.Ongoing,
		},
		"orders": orders,
		"payment_status_legend": map[string]string{
			"0":  "Dibatalkan",
			"1":  "Lunas",
			"2":  "Menunggu Verifikasi",
			"3":  "Belum Lunasi",
			"10": "Menunggu Persetujuan",
		},
	}
}

func paymentStatusLabelForAI(paymentStatus int, currentLabel string) string {
	switch paymentStatus {
	case 0:
		return "Dibatalkan"
	case 1:
		return "Lunas"
	case 2:
		return "Menunggu Verifikasi"
	case 3:
		return "Belum Lunasi"
	case 10:
		return "Menunggu Persetujuan"
	default:
		return strings.TrimSpace(currentLabel)
	}
}

func enrichOrderDetailForAI(res *model.OrderDetailResponse) map[string]interface{} {
	if res == nil {
		return map[string]interface{}{}
	}

	raw, _ := json.Marshal(res)
	out := map[string]interface{}{}
	_ = json.Unmarshal(raw, &out)

	paymentStatus := res.PaymentStatus
	out["payment_status"] = paymentStatus
	out["payment_status_label"] = paymentStatusLabelForAI(paymentStatus, "")
	out["scheduled"] = res.Scheduled
	out["is_scheduled"] = res.Scheduled

	// Calculate payment summary (sisa pembayaran)
	totalAmount := res.TotalAmount
	// For now, we'll need to get payment history to calculate remaining
	// But let's add a placeholder - we'll update this when we have the order service
	out["payment_summary"] = map[string]interface{}{
		"total_amount":      totalAmount,
		"payment_remaining": totalAmount, // Default to total if no payments yet
	}

	return out
}

func monthPeriodToOrderDateRange(period string) (string, string, bool) {
	t, err := time.ParseInLocation("2006-01", period, time.Local)
	if err != nil {
		return "", "", false
	}
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return start.Format("2006-01-02") + " 00:00:00", end.Format("2006-01-02") + " 23:59:59", true
}

func (ac *AIClient) requireAdmin(ctx context.Context) error {
	phone, _ := ctx.Value(phoneKey).(string)
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("unauthorized: admin access required")
	}
	isAdmin, err := ac.inventoryService.IsAdminByAccountNumber(phone)
	if err != nil {
		return fmt.Errorf("failed to verify admin: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("unauthorized: only admin can perform this action")
	}
	return nil
}

func (ac *AIClient) executeCreateOrder(ctx context.Context, orgID, userID string, params map[string]interface{}) interface{} {
	fleetID := getStringParam(params, "fleet_id")
	priceID := getStringParam(params, "price_id")
	fullname := getStringParam(params, "fullname")
	email := getStringParam(params, "email")
	address := getStringParam(params, "address")
	startDate := getStringParam(params, "start_date")
	endDate := getStringParam(params, "end_date")
	pickupCityID := getStringParam(params, "pickup_city_id")
	pickupLocation := getStringParam(params, "pickup_location")
	qty := getIntParam(params, "qty")
	additionalRequest := getStringParam(params, "additional_request")

	// Parse destinations and addons from JSON strings
	var destinations []model.OrderDestination
	if destStr := getStringParam(params, "destinations"); destStr != "" {
		_ = json.Unmarshal([]byte(destStr), &destinations)
	}
	if destinations == nil {
		destinations = []model.OrderDestination{}
	}

	var addons []string
	if addonStr := getStringParam(params, "addons"); addonStr != "" {
		_ = json.Unmarshal([]byte(addonStr), &addons)
	}
	if addons == nil {
		addons = []string{}
	}

	// Get organization code and company origin city
	orgDetail, err := ac.organizationService.GetOrganizationDetail(orgID)
	if err != nil {
		return map[string]interface{}{"error": "Failed to get organization info: " + err.Error()}
	}
	orgCode, _ := orgDetail["organization_code"].(string)
	orgName, _ := orgDetail["organization_name"].(string)
	roleName, _ := ctx.Value(contextRoleName).(string)
	if strings.TrimSpace(pickupCityID) == "" && strings.TrimSpace(roleName) == "CustomerAssistant" {
		if defaultPickupCityID, ok := orgDetail["city"].(string); ok && strings.TrimSpace(defaultPickupCityID) != "" {
			pickupCityID = strings.TrimSpace(defaultPickupCityID)
		}
	}

	missing := make([]string, 0)
	if strings.TrimSpace(roleName) == "CustomerAssistant" {
		if strings.TrimSpace(address) == "" && strings.TrimSpace(pickupLocation) != "" {
			address = pickupLocation
		}
	}
	missingVariable := ""
	if fleetID == "" {
		missingVariable += "fleet_id, "
		missing = append(missing, "fleet_id")
	}
	if priceID == "" {
		missingVariable += "price_id, "
		missing = append(missing, "price_id")
	}
	if fullname == "" {
		missingVariable += "fullname, "
		missing = append(missing, "fullname")
	}
	if strings.TrimSpace(roleName) != "CustomerAssistant" && email == "" {
		missingVariable += "email, "
		missing = append(missing, "email")
	}
	if address == "" {
		missingVariable += "address, "
		missing = append(missing, "address")
	}
	if startDate == "" {
		missingVariable += "start_date, "
		missing = append(missing, "start_date")
	}
	if endDate == "" {
		missingVariable += "end_date, "
		missing = append(missing, "end_date")
	}
	if pickupCityID == "" {
		missingVariable += "pickup_city_id, "
		missing = append(missing, "pickup_city_id")
	}
	if pickupLocation == "" {
		missingVariable += "pickup_location, "
		missing = append(missing, "pickup_location")
	}
	if len(missing) > 0 {
		customerPhone, _ := ctx.Value(phoneKey).(string)
		fmt.Println("==== missingVariable:", missingVariable)
		if suppress, _ := ctx.Value(contextSuppressAdminNotify).(bool); !suppress {
			ac.notifyAdminCreateOrderFailed(ctx, orgID, strings.TrimSpace(orgName), strings.TrimSpace(customerPhone), map[string]interface{}{
				"error":            "Missing required parameters: " + missingVariable,
				"missing_required": missingVariable,
				"fleet_id":         fleetID,
				"price_id":         priceID,
				"fullname":         fullname,
				"email":            email,
				"address":          address,
				"start_date":       startDate,
				"end_date":         endDate,
				"pickup_city_id":   pickupCityID,
				"pickup_location":  pickupLocation,
				"qty":              qty,
			})
		}
		return map[string]interface{}{
			"error":            "Missing required parameters",
			"missing_required": missing,
		}
	}

	if qty <= 0 {
		qty = 1
	}

	phone, _ := ctx.Value(phoneKey).(string)
	reviewRequired := false
	if strings.TrimSpace(roleName) == "CustomerAssistant" {
		var validationErr error
		reviewRequired, validationErr = ac.validateCompanyAssistantOrderRules(orgID, fleetID, priceID, pickupCityID, startDate, endDate, destinations)
		if validationErr != nil {
			return map[string]interface{}{"error": validationErr.Error()}
		}
	}

	req := &model.CreateOrderRequest{
		FleetID:           fleetID,
		PriceID:           priceID,
		Fullname:          fullname,
		Email:             email,
		Phone:             phone,
		Address:           address,
		StartDate:         startDate,
		EndDate:           endDate,
		PickupCityID:      pickupCityID,
		PickupLocation:    pickupLocation,
		Destinations:      destinations,
		Qty:               qty,
		Addons:            addons,
		AdditionalRequest: additionalRequest,
		OrganizationID:    orgID,
		OrganizationCode:  orgCode,
	}
	if cityID, err := strconv.Atoi(strings.TrimSpace(pickupCityID)); err == nil && cityID > 0 {
		req.CityID = cityID
	}

	result, err := ac.orderService.CreateOrder(req)
	if err != nil {
		customerPhone, _ := ctx.Value(phoneKey).(string)
		if suppress, _ := ctx.Value(contextSuppressAdminNotify).(bool); !suppress {
			ac.notifyAdminCreateOrderFailed(ctx, orgID, strings.TrimSpace(orgName), strings.TrimSpace(customerPhone), map[string]interface{}{
				"error":            err.Error(),
				"missing_required": []string{},
				"fleet_id":         fleetID,
				"price_id":         priceID,
				"fullname":         fullname,
				"email":            email,
				"address":          address,
				"start_date":       startDate,
				"end_date":         endDate,
				"pickup_city_id":   pickupCityID,
				"pickup_location":  pickupLocation,
				"qty":              qty,
			})
		}
		return map[string]interface{}{"error": err.Error()}
	}

	message := "Pesanan berhasil dibuat"
	if reviewRequired {
		message += "\n\nTim sedang meninjau pesanan anda, kami akan segera menghubungi anda. \nTerimakasih, Calista Prima"
	}

	return map[string]interface{}{
		"status":   "success",
		"message":  message,
		"order_id": result.OrderID,
		"token":    result.Token,
	}
}

func (ac *AIClient) customerHasUnconfirmedOrderStatus2(ctx context.Context, orgID, phone string) bool {
	if ac == nil || ac.fleetService == nil {
		return false
	}
	orgID = strings.TrimSpace(orgID)
	phone = strings.TrimSpace(phone)
	if orgID == "" || phone == "" {
		return false
	}

	now := time.Now()
	from := now.AddDate(0, 0, -90).Format("2006-01-02") + " 00:00:00"
	to := now.Format("2006-01-02") + " 23:59:59"
	filter := &model.PartnerOrderListFilter{
		OrderDateFrom: from,
		OrderDateTo:   to,
	}

	res, err := ac.fleetService.GetPartnerOrdersWithSummary(orgID, filter)
	if err != nil || res == nil {
		return false
	}

	for _, o := range res.Orders {
		if strings.TrimSpace(o.CustomerPhone) == "" || strings.TrimSpace(o.CustomerPhone) != phone {
			continue
		}
		if o.Status == 2 {
			return true
		}
	}
	return false
}

func (ac *AIClient) customerHasConfirmedOrderStatus1(ctx context.Context, orgID, phone string) bool {
	if ac == nil || ac.fleetService == nil {
		return false
	}
	orgID = strings.TrimSpace(orgID)
	phone = strings.TrimSpace(phone)
	if orgID == "" || phone == "" {
		return false
	}

	now := time.Now()
	from := now.AddDate(0, 0, -90).Format("2006-01-02") + " 00:00:00"
	to := now.Format("2006-01-02") + " 23:59:59"
	filter := &model.PartnerOrderListFilter{
		OrderDateFrom: from,
		OrderDateTo:   to,
	}

	res, err := ac.fleetService.GetPartnerOrdersWithSummary(orgID, filter)
	if err != nil || res == nil {
		return false
	}

	for _, o := range res.Orders {
		if strings.TrimSpace(o.CustomerPhone) == "" || strings.TrimSpace(o.CustomerPhone) != phone {
			continue
		}
		if o.Status == 1 {
			return true
		}
	}
	return false
}

func (ac *AIClient) notifyAdminCreateOrderFailed(ctx context.Context, orgID, orgName, customerPhone string, detail map[string]interface{}) {
	if ac == nil || ac.organizationService == nil || ac.wagyClient == nil {
		return
	}

	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return
	}

	adminPhone, err := ac.organizationService.GetAdminAccountNumber(orgID)
	if err != nil {
		log.Printf("[WAAI][Company] Failed get admin account number org=%s: %v", orgID, err)
		return
	}
	adminPhone = strings.TrimSpace(adminPhone)
	if adminPhone == "" {
		return
	}

	customerPhone = strings.TrimSpace(customerPhone)
	orgName = strings.TrimSpace(orgName)
	if orgName == "" {
		orgName = orgID
	}

	message := "Ada kendala saat membuat pesanan via AI Assistant.\n\n" +
		"Perusahaan: " + orgName + "\n" +
		"Nomor customer: " + customerPhone + "\n"

	if detail != nil {
		if fullname := strings.TrimSpace(getStringValue(detail["fullname"])); fullname != "" {
			message += "Nama: " + fullname + "\n"
		}
		if fleetID := strings.TrimSpace(getStringValue(detail["fleet_id"])); fleetID != "" {
			message += "fleet_id: " + fleetID + "\n"
		}
		if priceID := strings.TrimSpace(getStringValue(detail["price_id"])); priceID != "" {
			message += "price_id: " + priceID + "\n"
		}
		if startDate := strings.TrimSpace(getStringValue(detail["start_date"])); startDate != "" {
			message += "start_date: " + startDate + "\n"
		}
		if endDate := strings.TrimSpace(getStringValue(detail["end_date"])); endDate != "" {
			message += "end_date: " + endDate + "\n"
		}
		if pickupCityID := strings.TrimSpace(getStringValue(detail["pickup_city_id"])); pickupCityID != "" {
			message += "pickup_city_id: " + pickupCityID + "\n"
		}
		if pickupLocation := strings.TrimSpace(getStringValue(detail["pickup_location"])); pickupLocation != "" {
			message += "pickup_location: " + pickupLocation + "\n"
		}
		if qtyRaw := getIntParam(detail, "qty"); qtyRaw > 0 {
			message += "qty: " + strconv.Itoa(qtyRaw) + "\n"
		}
		if errMsg := strings.TrimSpace(getStringValue(detail["error"])); errMsg != "" {
			message += "\nError: " + errMsg + "\n"
		}
		if missing := getStringSliceValue(detail["missing_required"]); len(missing) > 0 {
			message += "Missing: " + strings.Join(missing, ", ") + "\n"
		}
	}

	adminPhone = service.NormalizeAssistantAccountNumber(adminPhone)
	go func() {
		if _, err := ac.wagyClient.SendMessage(adminPhone, strings.TrimSpace(message)); err != nil {
			log.Printf("[WAAI][Company] Failed notify admin %s: %v", adminPhone, err)
		}
	}()
}

func (ac *AIClient) validateCompanyAssistantOrderRules(orgID, fleetID, priceID, pickupCityID, startDate, endDate string, destinations []model.OrderDestination) (bool, error) {
	selectedPrice, err := ac.findFleetPriceByPriceID(orgID, fleetID, priceID)
	if err != nil {
		return false, err
	}

	rentalDays, err := calculateRentalDays(startDate, endDate)
	if err != nil {
		return false, err
	}

	prefs, err := ac.preferenceCityService.GetAll(orgID, nil)
	if err != nil {
		return false, err
	}

	prefByCityID := make(map[string]model.PreferenceCityWithLabels, len(prefs))
	for _, pref := range prefs {
		prefByCityID[strconv.Itoa(pref.CityID)] = pref
	}

	switch selectedPrice.RentType {
	case model.ServiceTypeCityTour:
		pref, ok := prefByCityID[strings.TrimSpace(pickupCityID)]
		if !ok {
			return true, nil
		}
		if !containsString(pref.ServiceTypes, model.ServiceTypeLabels[model.ServiceTypeCityTour]) {
			return false, fmt.Errorf("kota penjemputan belum tersedia untuk layanan city tour")
		}
		if pref.MinimalDay > 0 && rentalDays < pref.MinimalDay {
			return false, fmt.Errorf("minimal durasi sewa untuk city tour di %s adalah %d hari", fallbackCityLabel(pref, "kota tersebut"), pref.MinimalDay)
		}
		return false, nil
	case model.ServiceTypeOverland, model.ServiceTypeDropOnly:
		targetCityID := firstDestinationCityID(destinations)
		if targetCityID == "" {
			return false, fmt.Errorf("city_id tujuan wajib diisi untuk layanan overland atau drop only")
		}
		pref, ok := prefByCityID[targetCityID]
		if !ok {
			return true, nil
		}
		expectedType := model.ServiceTypeLabels[selectedPrice.RentType]
		if !containsString(pref.ServiceTypes, expectedType) {
			return false, fmt.Errorf("kota tujuan %s belum tersedia untuk layanan %s", fallbackCityLabel(pref, "tersebut"), strings.ReplaceAll(expectedType, "_", " "))
		}
		if pref.MinimalDay > 0 && rentalDays < pref.MinimalDay {
			return false, fmt.Errorf("minimal durasi sewa untuk layanan %s ke %s adalah %d hari", strings.ReplaceAll(expectedType, "_", " "), fallbackCityLabel(pref, "kota tersebut"), pref.MinimalDay)
		}
		return false, nil
	default:
		return false, nil
	}
}

func (ac *AIClient) findFleetPriceByPriceID(orgID, fleetID, priceID string) (*model.FleetPriceListItem, error) {
	for _, typeID := range []string{
		strconv.Itoa(model.ServiceTypeCityTour),
		strconv.Itoa(model.ServiceTypeOverland),
		strconv.Itoa(model.ServiceTypeDropOnly),
	} {
		items, err := ac.fleetService.GetFleetPricesByFleetID(orgID, fleetID, typeID)
		if err != nil {
			continue
		}
		for i := range items {
			if strings.TrimSpace(items[i].PriceID) == strings.TrimSpace(priceID) {
				return &items[i], nil
			}
		}
	}
	return nil, fmt.Errorf("price_id tidak ditemukan pada daftar harga armada")
}

func calculateRentalDays(startDate, endDate string) (int, error) {
	layout := "2006-01-02 15:04"
	start, err := time.ParseInLocation(layout, strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return 0, fmt.Errorf("format start_date harus YYYY-MM-DD HH:MM")
	}
	end, err := time.ParseInLocation(layout, strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return 0, fmt.Errorf("format end_date harus YYYY-MM-DD HH:MM")
	}
	if end.Before(start) {
		return 0, fmt.Errorf("end_date tidak boleh lebih awal dari start_date")
	}

	durationHours := end.Sub(start).Hours()
	days := int(durationHours / 24)
	if durationHours == 0 || durationHours-float64(days*24) > 0 {
		days++
	}
	if days <= 0 {
		days = 1
	}
	return days, nil
}

func firstDestinationCityID(destinations []model.OrderDestination) string {
	for _, destination := range destinations {
		if cityID := strings.TrimSpace(destination.CityID); cityID != "" {
			return cityID
		}
	}
	return ""
}

func containsString(items []string, needle string) bool {
	needle = strings.TrimSpace(strings.ToLower(needle))
	for _, item := range items {
		if strings.TrimSpace(strings.ToLower(item)) == needle {
			return true
		}
	}
	return false
}

func fallbackCityLabel(pref model.PreferenceCityWithLabels, fallback string) string {
	if strings.TrimSpace(pref.CityLabel) != "" {
		return pref.CityLabel
	}
	return fallback
}

func insertAssistantAccountStat(db *sql.DB, driver string, organizationID string, messageType int, status int) {
	insertAssistantStat(db, driver, "assistant_account_stats", "AssistantAccountStat", organizationID, messageType, status)
}

func insertAssistantCustomerStat(db *sql.DB, driver string, organizationID string, messageType int, status int) {
	insertAssistantStat(db, driver, "assistant_customer_stats", "AssistantCustomerStat", organizationID, messageType, status)
}

func insertAssistantStat(db *sql.DB, driver string, tableName string, logPrefix string, organizationID string, messageType int, status int) {
	if db == nil || strings.TrimSpace(organizationID) == "" {
		return
	}

	period := time.Now().Format("2006-01-02")
	query := fmt.Sprintf(`
		INSERT INTO %s (period, count, organization_id, type, status)
		VALUES ($1, 1, $2, $3, $4)
		ON CONFLICT (period, type, status, organization_id)
		DO UPDATE SET count = %s.count + 1
	`, tableName, tableName)

	_, err := db.Exec(query, period, organizationID, messageType, status)
	if err != nil {
		log.Printf("[%s] Failed to insert stat for org %s, type %d, status %d, driver %s: %v", logPrefix, organizationID, messageType, status, driver, err)
		return
	}

	log.Printf("[%s] Stat recorded for org %s, type %d, status %d", logPrefix, organizationID, messageType, status)
}

func buildAssistantSendResultHook(db *sql.DB, driver string, organizationID string, roleName string) func(error) {
	roleName = strings.TrimSpace(roleName)
	organizationID = strings.TrimSpace(organizationID)

	return func(err error) {
		if organizationID == "" {
			return
		}

		status := 1
		if err != nil {
			status = 0
		}

		if roleName == "CustomerAssistant" {
			insertAssistantCustomerStat(db, driver, organizationID, 2, status)
			return
		}

		insertAssistantAccountStat(db, driver, organizationID, 2, status)
	}
}

func (ac *AIClient) createInventoryRequest(ctx context.Context, orgID, userID string, params map[string]interface{}) interface{} {
	itemID := getStringParam(params, "item_id")
	itemName := getStringParam(params, "item_name")
	quantity := getIntParam(params, "quantity")
	garageID := getStringParam(params, "garage_id")
	employeeID := getStringParam(params, "employee_id")
	itemUOM := getStringParam(params, "item_uom")
	itemCategory := getIntParam(params, "item_category")
	notes := getStringParam(params, "notes")

	if itemID != "" && itemName != "" {
		return map[string]interface{}{"error": "send item_id or item_name, not both"}
	}
	if itemID == "" && itemName == "" {
		return map[string]interface{}{"error": "item_id or item_name is required"}
	}
	if quantity <= 0 {
		return map[string]interface{}{"error": "quantity must be greater than 0"}
	}
	if garageID == "" {
		return map[string]interface{}{"error": "garage_id is required"}
	}

	req := &model.CreateInventoryRequestRequest{
		ItemID:       itemID,
		ItemName:     itemName,
		Quantity:     quantity,
		GarageID:     garageID,
		EmployeeID:   employeeID,
		ItemUOM:      itemUOM,
		ItemCategory: itemCategory,
		Notes:        notes,
	}

	request, err := ac.inventoryService.CreateRequest(orgID, userID, req)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"status":         "success",
		"message":        "Inventory request created",
		"request_id":     request.RequestID,
		"request_number": request.RequestNumber,
	}
}

func (ac *AIClient) approveInventoryRequest(ctx context.Context, orgID, userID string, params map[string]interface{}) interface{} {
	if err := ac.requireAdmin(ctx); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	requestID := getStringParam(params, "request_id")
	itemID := getStringParam(params, "item_id")

	if requestID == "" {
		return map[string]interface{}{"error": "request_id is required"}
	}

	req := &model.ApproveInventoryRequestRequest{
		RequestID: requestID,
		ItemID:    itemID,
	}

	if err := ac.inventoryService.ApproveRequest(orgID, userID, req); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Inventory request approved successfully",
	}
}

func (ac *AIClient) rejectInventoryRequest(ctx context.Context, orgID, userID string, params map[string]interface{}) interface{} {
	if err := ac.requireAdmin(ctx); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	requestID := getStringParam(params, "request_id")
	if requestID == "" {
		return map[string]interface{}{"error": "request_id is required"}
	}

	req := &model.RejectInventoryRequestRequest{
		RequestID: requestID,
	}

	if err := ac.inventoryService.RejectRequest(orgID, userID, req); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Inventory request rejected successfully",
	}
}
