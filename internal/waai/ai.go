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
	"service-travego/model"
	"service-travego/repository"
	"service-travego/service"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
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
}

// NewAIClient creates a new AI client
func NewAIClient(apiKey string, db *sql.DB, dbDriver string, rdb *redis.Client) *AIClient {
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
	if err != nil {
		return "", fmt.Errorf("tenant lookup failed: %w", err)
	}
	ctx = withAuthorizedTenantContext(ctx, tenant)

	// Get business snapshot
	snapshot, err := ac.tenantRepo.GetOrganizationSnapshot(ctx, tenant.OrganizationID)
	if err != nil {
		snapshot = map[string]interface{}{} // Use empty snapshot if error
	}

	// Load conversation history
	history, err := ac.sessionMgr.LoadSession(ctx, phone)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
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

	prompt := fmt.Sprintf(`You are a helpful WhatsApp AI Assistant for %s, an ERP rental bus management system.

User Information:
- Name: %s
- Role: %s
- Organization: %s

Current Business Status:
- Fleet Count: %v
- Available Units: %v
- Today's Bookings: %v

You have access to the following functions to help users:
1. get_business_snapshot - Get current business metrics
2. get_fleet_availability - Check vehicle availability
3. get_fleet_list - View owned fleets
4. get_fleet_detail - View fleet detail
5. get_fleet_units - View owned fleet units
6. get_city_list - View city list
7. get_preference_cities - View served cities
8. get_customer_list - View customer list
9. get_customer_detail - View customer detail
10. get_booking_list - View bookings
11. get_revenue_summary - Get revenue data

Please respond in Indonesian (Bahasa Indonesia) unless the user asks otherwise.
Help the user with their inquiries related to the bus rental business.
If the user asks who you are, what your name is, or what assistant they are talking to, identify yourself as "Trave AI Assistant Travego".
If the user asks who developed, created, or made you, answer that you were created by Afatbenz Tech and that they can contact 6281335884729 or visit mafatichulfuadi.com for further discussion.
If the user asks how to register for or enjoy the AI Assistant service, answer that they should register on https://www.travego.id and add their WhatsApp number in the Pengaturan > AI Assistant menu.
Do not say you are Kiro, Claude, Anthropic, or mention the provider/model name unless explicitly asked about technical backend details.
Be professional and concise in your responses.`,
		tenant.OrganizationName,
		displayName,
		tenant.Role,
		tenant.OrganizationName,
		snapshot["fleet_count"],
		snapshot["unit_count"],
		snapshot["today_bookings"],
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
