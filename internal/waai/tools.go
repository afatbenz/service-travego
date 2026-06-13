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
				Description: "Search customers by name. Returns customer_id, name, phone, email, and address. Use this first when the user asks about a customer by name before calling get_customer_detail.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"customer_name": map[string]interface{}{
							"type":        "string",
							"description": "Customer name to search (partial match supported)",
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
				Description: "Get full customer contact details by customer_id. When user asks by name, call get_customer_list first to obtain the customer_id.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"customer_id": map[string]interface{}{
							"type":        "string",
							"description": "Customer ID obtained from get_customer_list",
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
		{
			Type: "function",
			Name: "get_organization_info",
			Function: FunctionDefinition{
				Name:        "get_organization_info",
				Description: "Get full business / organization information including name, address, phone, WhatsApp, email, NPWP, and domain",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_order_list",
			Function: FunctionDefinition{
				Name:        "get_order_list",
				Description: "Get fleet order (pesanan) list with summary counts. Each order includes payment_status_label (Lunas/Belum bayar/Belum dikonfirmasi/Belum lunas). Use latest_payment_type only for payment type (DP/Cicilan), not as payment status.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"period": map[string]interface{}{
							"type":        "string",
							"description": "Order month filter in YYYY-MM format (e.g. 2026-06 for June 2026). Use for 'bulan ini' or 'bulan lalu'.",
						},
						"order_date_from": map[string]interface{}{
							"type":        "string",
							"description": "Optional order date start (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)",
						},
						"order_date_to": map[string]interface{}{
							"type":        "string",
							"description": "Optional order date end (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)",
						},
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional trip start date from filter",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional trip start date to filter",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Optional search by order_id, customer name, or fleet name",
						},
						"payment_status": map[string]interface{}{
							"type":        "integer",
							"description": "Optional payment status filter",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_order_detail",
			Function: FunctionDefinition{
				Name:        "get_order_detail",
				Description: "Get detailed fleet order information by order_id, including itinerary (rencana perjalanan) and payment_summary with payment_remaining (sisa pembayaran)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_order_payment_history",
			Function: FunctionDefinition{
				Name:        "get_order_payment_history",
				Description: "Get riwayat pembayaran for a specific order_id",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "approve_order",
			Function: FunctionDefinition{
				Name:        "approve_order",
				Description: "Setujui (approve) an order by order_id",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID to approve",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "reject_order",
			Function: FunctionDefinition{
				Name:        "reject_order",
				Description: "Tolak (reject) an order by order_id",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID to reject",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_employee_shift_schedule",
			Function: FunctionDefinition{
				Name:        "get_employee_shift_schedule",
				Description: "Get jadwal tim (employee shift schedule) including total off days",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional start date in YYYY-MM-DD format",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional end date in YYYY-MM-DD format",
						},
						"role_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional role ID filter",
						},
						"division_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional division ID filter",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "add_employee_off_day",
			Function: FunctionDefinition{
				Name:        "add_employee_off_day",
				Description: "Tambah hari off (add off day) for an employee",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"employee_id": map[string]interface{}{
							"type":        "string",
							"description": "Employee ID",
						},
						"shift_date": map[string]interface{}{
							"type":        "string",
							"description": "Off day date in YYYY-MM-DD format",
						},
						"shift_type": map[string]interface{}{
							"type":        "integer",
							"description": "Shift type (default off type)",
						},
					},
					"required": []string{"employee_id", "shift_date"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_monthly_revenue",
			Function: FunctionDefinition{
				Name:        "get_monthly_revenue",
				Description: "Get pendapatan bulan ini including total revenue, total expenses, and estimated profit (total_revenue - total_expenses)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"month": map[string]interface{}{
							"type":        "string",
							"description": "Month in YYYY-MM format (default to current month)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_top_fleets",
			Function: FunctionDefinition{
				Name:        "get_top_fleets",
				Description: "Get unit armada paling banyak orderan (top fleets by number of orders)",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_top_destinations",
			Function: FunctionDefinition{
				Name:        "get_top_destinations",
				Description: "Get kota tujuan paling populer (top destinations)",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_top_customers",
			Function: FunctionDefinition{
				Name:        "get_top_customers",
				Description: "Get customer paling loyal (top customers by number of orders)",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_schedule_list",
			Function: FunctionDefinition{
				Name:        "get_schedule_list",
				Description: "Get schedule list filtered by period, order, fleet, or search keywords",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"period": map[string]interface{}{
							"type":        "string",
							"description": "Month filter in YYYY-MM format",
						},
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional order ID filter",
						},
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional fleet ID filter",
						},
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Optional search keyword",
						},
						"fleet_name": map[string]interface{}{
							"type":        "string",
							"description": "Optional fleet name filter",
						},
						"plate": map[string]interface{}{
							"type":        "string",
							"description": "Optional plate number filter",
						},
						"production_year": map[string]interface{}{
							"type":        "string",
							"description": "Optional production year filter",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_schedule_detail",
			Function: FunctionDefinition{
				Name:        "get_schedule_detail",
				Description: "Get schedule trip detail by schedule_number",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Schedule number (Surat Jalan / SPJ)",
						},
					},
					"required": []string{"schedule_number"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_spj_total_biaya",
			Function: FunctionDefinition{
				Name:        "get_spj_total_biaya",
				Description: "Get total biaya operasional (total amount) for a specific Surat Jalan / SPJ (schedule_number)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Surat Jalan / SPJ (schedule number)",
						},
					},
					"required": []string{"schedule_number"},
				},
			},
		},
		{
			Type: "function",
			Name: "tambah_pengeluaran_spj",
			Function: FunctionDefinition{
				Name:        "tambah_pengeluaran_spj",
				Description: "Tambah pengeluaran untuk Surat Jalan / SPJ (schedule_number). Untuk biaya operasional, gunakan transaction_item = TRX-I00 dan payment_method = 1. Jenis pengeluaran (transaction_item) bisa diambil dari daftar transaction-items di config/common.json.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Surat Jalan / SPJ (schedule number)",
						},
						"transaction_item": map[string]interface{}{
							"type":        "string",
							"description": "Jenis pengeluaran (transaction item ID, contoh: TRX-I00 = Biaya Operasional Perjalanan, TRX-I01 = Biaya Bahan Bakar, dll)",
						},
						"payment_method": map[string]interface{}{
							"type":        "integer",
							"description": "Jenis pembayaran (1 = Biaya Operasional / Kas, 2 = Reimburse)",
							"default":     1,
						},
						"amount": map[string]interface{}{
							"type":        "number",
							"description": "Jumlah pengeluaran (dalam rupiah, tanpa titik koma atau simbol)",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Deskripsi pengeluaran (opsional)",
						},
					},
					"required": []string{"schedule_number", "transaction_item", "amount"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_spj_pengeluaran",
			Function: FunctionDefinition{
				Name:        "get_spj_pengeluaran",
				Description: "Dapatkan daftar pengeluaran untuk Surat Jalan / SPJ tertentu",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Surat Jalan / SPJ (schedule number)",
						},
					},
					"required": []string{"schedule_number"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_spj_ringkasan_pembayaran",
			Function: FunctionDefinition{
				Name:        "get_spj_ringkasan_pembayaran",
				Description: "Dapatkan ringkasan total pengeluaran SPJ berdasarkan jenis pembayaran (biaya operasional dan reimburse)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Surat Jalan / SPJ (schedule number)",
						},
					},
					"required": []string{"schedule_number"},
				},
			},
		},
		{
			Type: "function",
			Name: "print_surat_jalan",
			Function: FunctionDefinition{
				Name:        "print_surat_jalan",
				Description: "Mencetak dan mengirimkan surat jalan / SPJ (Surat Pertanggungjawaban) dalam format PDF ke WhatsApp. Gunakan tool ini ketika pengguna meminta untuk mencetak, print, atau kirim surat jalan.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"schedule_number": map[string]interface{}{
							"type":        "string",
							"description": "Nomor surat jalan / SPJ (schedule number)",
						},
					},
					"required": []string{"schedule_number"},
				},
			},
		},
	}
}
