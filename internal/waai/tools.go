package waai

// GetToolDefinitions returns all available tool definitions for the AI assistant
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Name: "get_business_snapshot",
			Function: FunctionDefinition{
				Name:        "get_business_snapshot",
				Description: "Get a summary of the business today including fleet count, available units, bookings, and revenue",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_fleet_availability",
			Function: FunctionDefinition{
				Name:        "get_fleet_availability",
				Description: "Check fleet availability for a specific date range",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Start date in YYYY-MM-DD HH:MM format",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "End date in YYYY-MM-DD HH:MM format",
						},
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional fleet ID to filter a specific fleet",
						},
					},
					"required": []string{"start_date", "end_date"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_fleet_list",
			Function: FunctionDefinition{
				Name:        "get_fleet_list",
				Description: "Get list of owned fleets with optional filters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fleet_type": map[string]interface{}{
							"type":        "string",
							"description": "Filter by fleet type",
						},
						"fleet_name": map[string]interface{}{
							"type":        "string",
							"description": "Filter by fleet name",
						},
						"fleet_body": map[string]interface{}{
							"type":        "string",
							"description": "Filter by fleet body",
						},
						"fleet_engine": map[string]interface{}{
							"type":        "string",
							"description": "Filter by fleet engine",
						},
						"pickup_location": map[string]interface{}{
							"type":        "integer",
							"description": "Filter by pickup city ID",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_fleet_detail",
			Function: FunctionDefinition{
				Name:        "get_fleet_detail",
				Description: "Get detailed information for a fleet by fleet ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Fleet ID",
						},
					},
					"required": []string{"fleet_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_fleet_units",
			Function: FunctionDefinition{
				Name:        "get_fleet_units",
				Description: "Get list of fleet units owned by the organization",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional fleet ID filter",
						},
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional order ID filter",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Optional search keyword",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_city_list",
			Function: FunctionDefinition{
				Name:        "get_city_list",
				Description: "Get city list with optional filters by province or search",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"province_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional province ID filter",
						},
						"province": map[string]interface{}{
							"type":        "string",
							"description": "Optional province name filter",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Optional city name search",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_preference_cities",
			Function: FunctionDefinition{
				Name:        "get_preference_cities",
				Description: "Get the list of cities served by the organization",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city_id": map[string]interface{}{
							"type":        "integer",
							"description": "Optional city ID filter",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_customer_list",
			Function: FunctionDefinition{
				Name:        "get_customer_list",
				Description: "Get customer list with optional name filter",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"customer_name": map[string]interface{}{
							"type":        "string",
							"description": "Optional customer name search",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_customer_detail",
			Function: FunctionDefinition{
				Name:        "get_customer_detail",
				Description: "Get detailed customer information by customer ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"customer_id": map[string]interface{}{
							"type":        "string",
							"description": "Customer ID",
						},
					},
					"required": []string{"customer_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_booking_list",
			Function: FunctionDefinition{
				Name:        "get_booking_list",
				Description: "Get list of bookings with optional filtering by status",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Filter by booking status (e.g., 'pending', 'confirmed', 'completed')",
							"enum":        []string{"pending", "confirmed", "completed", "cancelled"},
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of bookings to return",
							"default":     10,
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_revenue_summary",
			Function: FunctionDefinition{
				Name:        "get_revenue_summary",
				Description: "Get revenue summary for a specific period",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"period": map[string]interface{}{
							"type":        "string",
							"description": "Time period for revenue summary",
							"enum":        []string{"daily", "weekly", "monthly"},
						},
					},
					"required": []string{"period"},
				},
			},
		},
	}
}
