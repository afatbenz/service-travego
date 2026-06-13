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
	"regexp"
	"service-travego/configs"
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

// AIClient handles communication with Anthropic API
type AIClient struct {
	apiKey                string
	model                 string
	baseURL               string
	authMgr               *AuthManager
	tenantRepo            *TenantRepository
	sessionMgr            *SessionManager
	toolExec              *ToolExecutor
	fleetService          *service.FleetService
	fleetUnitService      *service.FleetUnitService
	generalService        *service.GeneralService
	preferenceCityService *service.PreferenceCityService
	customersService      *service.CustomersService
	organizationService   *service.OrganizationService
	scheduleService       *service.ScheduleService
	orderService          *service.OrderService
	dashboardService      *service.DashboardService
	transactionService    *service.TransactionService
	printService          *service.PrintManagementService
	wagyClient            *WagyClient
}

// NewAIClient creates a new AI client
func NewAIClient(apiKey string, db *sql.DB, dbDriver string, rdb *redis.Client, wagyClient *WagyClient) *AIClient {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	baseURL := os.Getenv("ANTHROPIC_API_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

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

	// Load minimal email config for OrderService
	emailCfg := &configs.EmailConfig{
		From:     os.Getenv("EMAIL_FROM"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		SMTPHost: os.Getenv("EMAIL_SMTP_HOST"),
		SMTPPort: os.Getenv("EMAIL_SMTP_PORT"),
	}

	return &AIClient{
		apiKey:                apiKey,
		model:                 model,
		baseURL:               baseURL,
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
		transactionService:    service.NewTransactionService(transactionRepo),
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
func (ac *AIClient) ProcessMessage(ctx context.Context, phone, incomingMessage string) (string, error) {
	// Get tenant info
	tenant, err := ac.tenantRepo.GetTenantByPhone(ctx, phone)
	var snapshot map[string]interface{}

	if err == nil {
		// Tenant found - proceed with tenant context
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
		return "", fmt.Errorf("failed to load session: %w", err)
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
	finalResponse, err := ac.callAnthropicWithTools(ctx, systemPrompt, history)
	if err != nil {
		return "", fmt.Errorf("anthropic call failed: %w", err)
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

func (ac *AIClient) callAnthropicFinal(ctx context.Context, systemPrompt string, messages []ConversationMessage) (*AnthropicResponse, error) {
	return ac.callAnthropicRequest(ctx, systemPrompt, messages, nil)
}

func (ac *AIClient) callAnthropicRequest(ctx context.Context, systemPrompt string, messages []ConversationMessage, tools []ToolDefinition) (*AnthropicResponse, error) {
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

Tool usage rules:
- [CRITICAL] Data dalam database dapat BERUBAH sewaktu-waktu. JANGAN PERCAYA jawaban Anda dari riwayat percakapan sebelumnya. Selalu PANGGIL TOOL setiap kali user menanyakan data (pesanan, pelanggan, jadwal, armada, dll.) untuk mendapatkan data TERBARU dari database.
- GUESTS CANNOT USE TOOLS THAT REQUIRE ORGANIZATION CONTEXT. If a guest asks for data, explain how to register.
- When the user asks about their business or organization name, answer using Business Name from context above. For full organization details (address, phone, NPWP, etc.), call get_organization_info.
- When the user asks for customer contact or details by name (not customer_id), you MUST:
  1. Call get_customer_list with customer_name set to the name provided
  2. If one match is found, call get_customer_detail with that customer_id and share the contact info
  3. If multiple matches are found, list them and ask the user to clarify
  4. If no match is found, tell the user the customer was not found
- When the user asks about pesanan/order (e.g. "ada pesanan bulan ini?", "berapa order bulan Juni?"), you MUST call get_order_list with period set to the relevant YYYY-MM (use %s for "bulan ini"). Answer ONLY from the tool result summary (total_orders, paid, unpaid, revenue). Never guess from Today's Bookings — that number is for today only.
- For order detail by order_id, call get_order_detail — JANGAN PERCAYA jawaban sebelumnya, selalu panggil tool untuk data terbaru.
- For itinerary of order detail by order_id, call get_order_detail, get orders.itinerary[].
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
  4. Jika user ingin mencetak / mengirim surat jalan dalam format PDF, panggil print_surat_jalan dengan schedule_number.
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
- Do not use # headings, **bold**, __underline__, or [link](url) Markdown syntax.`,
		userContext,
		currentDate,
		currentMonth,
		currentMonth,
	)

	return prompt
}

// executeTool executes a tool and returns the result
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
		res, err := ac.organizationService.GetOrganizationDetail(orgID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return res

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
		totalExpenses, totalReimburse, err := ac.transactionService.GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"schedule_number":            scheduleNumber,
			"total_jatah_uang":           totalAmount,
			"biaya_operasional":          totalAmount,
			"biaya_operasional_terpakai": totalExpenses,
			"total_reimburse":            totalReimburse,
			"total_pengeluaran":          totalExpenses + totalReimburse,
			"saldo_sisa":                 totalAmount - (totalExpenses + totalReimburse),
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
		totalExpenses, totalReimburse, err := ac.transactionService.GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		return map[string]interface{}{
			"schedule_number":            scheduleNumber,
			"total_jatah_uang":           totalAmount,
			"biaya_operasional":          totalAmount,
			"biaya_operasional_terpakai": totalExpenses,
			"reimburse":                  totalReimburse,
			"total_pengeluaran":          totalExpenses + totalReimburse,
			"saldo_sisa":                 totalAmount - (totalExpenses + totalReimburse),
		}

	case "print_surat_jalan":
		scheduleNumber := getStringParam(params, "schedule_number")
		if scheduleNumber == "" {
			return map[string]interface{}{"error": "schedule_number is required"}
		}

		// Generate PDF
		pdfData, err := ac.printService.GenerateFleetTripsPDF(orgID, scheduleNumber)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		// Get phone number from context
		phone, _ := ctx.Value(phoneKey).(string)
		if phone == "" {
			return map[string]interface{}{"error": "phone number missing in context"}
		}

		// Send PDF via WhatsApp
		filename := fmt.Sprintf("surat-jalan-%s.pdf", scheduleNumber)
		caption := fmt.Sprintf("Berikut surat jalan untuk *%s*", scheduleNumber)
		_, err = ac.wagyClient.SendDocument(phone, filename, pdfData, caption)
		if err != nil {
			log.Printf("[WAAI][AI] Failed to send PDF: %v", err)
			return map[string]interface{}{"error": "Gagal mengirim surat jalan: " + err.Error()}
		}

		return map[string]interface{}{
			"status":  "success",
			"message": "Surat jalan berhasil dikirim",
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

type waaiContextKey string

const (
	contextOrganizationID waaiContextKey = "organization_id"
	contextUserID         waaiContextKey = "user_id"
	contextRoleName       waaiContextKey = "role_name"
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

	if orgID == "" || userID == "" {
		return "", fmt.Errorf("missing organization_id or user_id in context")
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
)

// formatWhatsAppReply normalizes model output to WhatsApp-friendly formatting.
func formatWhatsAppReply(text string) string {
	if text == "" {
		return text
	}

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

	return strings.TrimSpace(text)
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
