package waai

import (
	"encoding/json"
)

// ToolDefinition represents an Anthropic tool definition
type ToolDefinition struct {
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Function FunctionDefinition     `json:"function"`
}

// FunctionDefinition defines a tool function
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// GetToolDefinitions returns all available tool definitions for the AI assistant
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Name: "get_business_snapshot",
			Function: FunctionDefinition{
				Name:        "get_business_snapshot",
				Description: "Get a summary of the business today including fleet count, available units, bookings, and revenue",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {},
					"required": []
				}`),
			},
		},
		{
			Type: "function",
			Name: "get_fleet_availability",
			Function: FunctionDefinition{
				Name:        "get_fleet_availability",
				Description: "Check available fleet units for a specific date range",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date_start": {
							"type": "string",
							"description": "Start date in YYYY-MM-DD format"
						},
						"date_end": {
							"type": "string",
							"description": "End date in YYYY-MM-DD format"
						}
					},
					"required": ["date_start", "date_end"]
				}`),
			},
		},
		{
			Type: "function",
			Name: "get_booking_list",
			Function: FunctionDefinition{
				Name:        "get_booking_list",
				Description: "Get list of bookings with optional filtering by status",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"status": {
							"type": "string",
							"description": "Filter by booking status (e.g., 'pending', 'confirmed', 'completed')",
							"enum": ["pending", "confirmed", "completed", "cancelled"]
						},
						"limit": {
							"type": "integer",
							"description": "Maximum number of bookings to return",
							"default": 10
						}
					},
					"required": []
				}`),
			},
		},
		{
			Type: "function",
			Name: "get_revenue_summary",
			Function: FunctionDefinition{
				Name:        "get_revenue_summary",
				Description: "Get revenue summary for a specific period",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"period": {
							"type": "string",
							"description": "Time period for revenue summary",
							"enum": ["daily", "weekly", "monthly"]
						}
					},
					"required": ["period"]
				}`),
			},
		},
	}
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
