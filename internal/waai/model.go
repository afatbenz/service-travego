package waai

// TenantInfo holds tenant/organization information for a WhatsApp contact.
type TenantInfo struct {
	Phone            string
	Name             string
	FullName         string
	Role             string
	OrganizationID   string
	OrganizationName string
	IsActive         bool
	AssistantID      string
	UserID           string
	Avatar           string
	RoleName         string
	DivisionName     string
	AccountNumber    string
	UserType         int
}

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// WebhookPayload represents the structure of a Wagy webhook event
type WebhookPayload struct {
	Event  string `json:"event"`
	Source string `json:"source"`
	Data   struct {
		ID       int64  `json:"id"`
		DeviceID string `json:"device_id"`
		OwnerJID string `json:"owner_jid"`
		Content  struct {
			PhoneJID  string `json:"pn_jid"`
			Message   string `json:"content"`
			MessageID string `json:"message_id"`
			Timestamp string `json:"timestamp"`
		} `json:"content"`
	} `json:"data"`
}



// ToolDefinition represents an Anthropic tool definition
type ToolDefinition struct {
	Type     string             `json:"type"`
	Name     string             `json:"name"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a tool function
type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
}

// NewToolResult creates a new tool result
func NewToolResult(toolUseID string, content interface{}) ToolResult {
	return ToolResult{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
	}
}
