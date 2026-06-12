package waai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// AIClient handles communication with Anthropic API
type AIClient struct {
	apiKey    string
	model     string
	baseURL   string
	tenantRepo *TenantRepository
	sessionMgr *SessionManager
}

// NewAIClient creates a new AI client
func NewAIClient(apiKey string, db *sql.DB, dbDriver string, rdb *redis.Client) *AIClient {
	return &AIClient{
		apiKey:     apiKey,
		model:      "claude-sonnet-4-6",
		baseURL:    "https://api.anthropic.com",
		tenantRepo: NewTenantRepository(db, dbDriver),
		sessionMgr: NewSessionManager(rdb),
	}
}

// AnthropicRequest represents the request body for Anthropic API
type AnthropicRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	SystemPrompt string                `json:"system"`
	Messages    []ConversationMessage  `json:"messages"`
	Tools       []ToolDefinition       `json:"tools"`
}

// AnthropicResponse represents the response from Anthropic API
type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type      string `json:"type"`
		Text      string `json:"text,omitempty"`
		ID        string `json:"id,omitempty"`
		Name      string `json:"name,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
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
	tenant, err := ac.tenantRepo.GetTenantByPhone(phone)
	if err != nil {
		return "", fmt.Errorf("tenant lookup failed: %w", err)
	}

	// Get business snapshot
	snapshot, err := ac.tenantRepo.GetOrganizationSnapshot(tenant.OrganizationID)
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
	for i := 0; i < 5; i++ { // Max 5 iterations to prevent infinite loops
		response, err := ac.callAnthropic(ctx, systemPrompt, messages)
		if err != nil {
			return "", err
		}

		// Check if we got a tool use
		hasToolUse := false
		for _, content := range response.Content {
			if content.Type == "tool_use" {
				hasToolUse = true
				// Execute the tool
				toolResult := ac.executeTool(content.Name, content.Input)

				// Add assistant response and tool result to messages
				assistantMsg := ConversationMessage{
					Role: "assistant",
					Content: []map[string]interface{}{
						{
							"type": "tool_use",
							"id":   content.ID,
							"name": content.Name,
							"input": json.RawMessage(content.Input),
						},
					},
				}
				messages = append(messages, assistantMsg)

				// Add tool result
				toolResultMsg := ConversationMessage{
					Role: "user",
					Content: []map[string]interface{}{
						{
							"type":       "tool_result",
							"tool_use_id": content.ID,
							"content":    toolResult,
						},
					},
				}
				messages = append(messages, toolResultMsg)
			}
		}

		// If no tool use, return the text response
		if !hasToolUse {
			for _, content := range response.Content {
				if content.Type == "text" && content.Text != "" {
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("max tool use iterations reached without text response")
}

// callAnthropic makes a single call to Anthropic API
func (ac *AIClient) callAnthropic(ctx context.Context, systemPrompt string, messages []ConversationMessage) (*AnthropicResponse, error) {
	req := AnthropicRequest{
		Model:        ac.model,
		MaxTokens:    1024,
		SystemPrompt: systemPrompt,
		Messages:     messages,
		Tools:        GetToolDefinitions(),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ac.baseURL+"/v1/messages", bytes.NewBuffer(body))
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

	var response AnthropicResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s - %s", response.Error.Type, response.Error.Message)
	}

	return &response, nil
}

// buildSystemPrompt builds the system prompt with tenant context
func (ac *AIClient) buildSystemPrompt(tenant *TenantInfo, snapshot map[string]interface{}) string {
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
3. get_booking_list - View bookings
4. get_revenue_summary - Get revenue data

Please respond in Indonesian (Bahasa Indonesia) unless the user asks otherwise.
Help the user with their inquiries related to the bus rental business.
Be professional and concise in your responses.`,
		tenant.OrganizationName,
		tenant.Name,
		tenant.Role,
		tenant.OrganizationName,
		snapshot["fleet_count"],
		snapshot["unit_count"],
		snapshot["today_bookings"],
	)

	return prompt
}

// executeTool executes a tool and returns the result
func (ac *AIClient) executeTool(toolName string, input json.RawMessage) interface{} {
	switch toolName {
	case "get_business_snapshot":
		return map[string]interface{}{
			"fleet_count":    5,
			"available_units": 12,
			"today_bookings": 3,
			"today_revenue":  2500000,
		}
	case "get_fleet_availability":
		return map[string]interface{}{
			"available_units": 8,
			"date_range": "2026-06-12 to 2026-06-19",
		}
	case "get_booking_list":
		return []map[string]interface{}{
			{
				"id":     1001,
				"status": "confirmed",
				"date":   "2026-06-15",
				"route":  "Jakarta - Bandung",
			},
			{
				"id":     1002,
				"status": "pending",
				"date":   "2026-06-16",
				"route":  "Jakarta - Sukabumi",
			},
		}
	case "get_revenue_summary":
		return map[string]interface{}{
			"period": "daily",
			"revenue": 2500000,
			"transactions": 5,
		}
	default:
		return map[string]interface{}{
			"error": "Unknown tool: " + toolName,
		}
	}
}
