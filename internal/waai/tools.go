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
			Name: "get_trip_distance",
			Function: FunctionDefinition{
				Name:        "get_trip_distance",
				Description: "Estimate trip distance (km) between pickup/origin and destination using public routing/geocoding. Returns recommended minimal rental days for overland (pulang-pergi) vs drop-only based on distance thresholds.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"from": map[string]interface{}{
							"type":        "string",
							"description": "Origin/pickup city or location name. If empty, will default to company city.",
						},
						"to": map[string]interface{}{
							"type":        "string",
							"description": "Destination city or location name (e.g. Brebes)",
						},
					},
					"required": []string{"to"},
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
			Name: "get_inventory_items",
			Function: FunctionDefinition{
				Name:        "get_inventory_items",
				Description: "Get daftar inventory item aktif beserta total stok dan garage yang memiliki stok. Gunakan untuk cek inventory umum atau mencari item_id dari nama item/SKU.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Optional search by item name or item SKU",
						},
						"garage_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional garage ID filter",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_inventory_detail",
			Function: FunctionDefinition{
				Name:        "get_inventory_detail",
				Description: "Get detail inventory item by item_id, including stock per garage/location.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Inventory item ID",
						},
					},
					"required": []string{"item_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_inventory_stock",
			Function: FunctionDefinition{
				Name:        "get_inventory_stock",
				Description: "Get jumlah stok item, either total stock across garages or stock in a specific garage.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Inventory item ID",
						},
						"garage_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional garage ID to get stock only for that garage",
						},
					},
					"required": []string{"item_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_garage_list",
			Function: FunctionDefinition{
				Name:        "get_garage_list",
				Description: "Get daftar garage/garasi untuk organization. Gunakan untuk mencari garage_id saat membuat item atau cek stok per garage.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional item ID to filter garages that have this item",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_item_suppliers",
			Function: FunctionDefinition{
				Name:        "get_item_suppliers",
				Description: "Get daftar supplier untuk purchase order item inventory.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_item_movements",
			Function: FunctionDefinition{
				Name:        "get_item_movements",
				Description: "Get riwayat movement item inventory. Bisa filter by item_id, start_date, end_date, dan garage_id.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Inventory item ID",
						},
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional start date in YYYY-MM-DD format",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional end date in YYYY-MM-DD format",
						},
						"garage_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional garage ID filter",
						},
					},
					"required": []string{"item_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_item_order_history",
			Function: FunctionDefinition{
				Name:        "get_item_order_history",
				Description: "Get riwayat purchase order / history untuk item inventory.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Inventory item ID",
						},
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional start date in YYYY-MM-DD format",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional end date in YYYY-MM-DD format",
						},
					},
					"required": []string{"item_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_item_stock_distribution",
			Function: FunctionDefinition{
				Name:        "get_item_stock_distribution",
				Description: "Get distribusi stok item per garage/location dari inventory item detail locations[].",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Inventory item ID",
						},
					},
					"required": []string{"item_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_purchase_order_list",
			Function: FunctionDefinition{
				Name:        "get_purchase_order_list",
				Description: "Get daftar purchase order inventory.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_purchase_order_detail",
			Function: FunctionDefinition{
				Name:        "get_purchase_order_detail",
				Description: "Get detail purchase order inventory by purchase_id.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"purchase_id": map[string]interface{}{
							"type":        "string",
							"description": "Purchase order ID / purchase_id",
						},
					},
					"required": []string{"purchase_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "complete_purchase_order",
			Function: FunctionDefinition{
				Name:        "complete_purchase_order",
				Description: "Set purchase order inventory menjadi completed/received. Gunakan hanya setelah barang diterima.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"purchase_id": map[string]interface{}{
							"type":        "string",
							"description": "Purchase order ID / purchase_id",
						},
					},
					"required": []string{"purchase_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "cancel_purchase_order",
			Function: FunctionDefinition{
				Name:        "cancel_purchase_order",
				Description: "Cancel/reject purchase order inventory. Gunakan hanya ketika user meminta membatalkan atau menolak PO.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"purchase_id": map[string]interface{}{
							"type":        "string",
							"description": "Purchase order ID / purchase_id",
						},
					},
					"required": []string{"purchase_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "create_new_item",
			Function: FunctionDefinition{
				Name:        "create_new_item",
				Description: "Create inventory item baru atau tambah/update stok item. Jika item_sku kosong, SKU akan digenerate otomatis. item_category: 1 = Kebutuhan Armada, 2 = kebutuhan kantor. transaction_type: 1 = tambah stok yang ada, 2 = update stock sesuai input (tidak menambahkan stock yang sudah ada).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional existing item ID. Required only when updating existing item instead of creating by name",
						},
						"item_name": map[string]interface{}{
							"type":        "string",
							"description": "Nama item. Required jika item_id tidak diberikan atau membuat item baru",
						},
						"item_sku": map[string]interface{}{
							"type":        "string",
							"description": "Optional SKU item. Jika kosong, sistem akan generate otomatis",
						},
						"item_uom": map[string]interface{}{
							"type":        "string",
							"description": "Satuan item, contoh: Pcs, Box, Liter, Unit",
						},
						"item_category": map[string]interface{}{
							"type":        "integer",
							"description": "Kategori item: 1 = Kebutuhan Armada, 2 = kebutuhan kantor",
						},
						"stock": map[string]interface{}{
							"type":        "integer",
							"description": "Jumlah stok. Untuk transaction_type 1 = stok yang ditambahkan. Untuk transaction_type 2 = stok akhir sesuai input",
						},
						"garage_id": map[string]interface{}{
							"type":        "string",
							"description": "Garage ID lokasi stok",
						},
						"transaction_type": map[string]interface{}{
							"type":        "string",
							"description": "1 = tambah stok yang ada, 2 = update stock sesuai input (tidak menambahkan stock yang sudah ada)",
							"enum":        []string{"1", "2"},
						},
						"transaction_date": map[string]interface{}{
							"type":        "string",
							"description": "Tanggal transaksi dalam format YYYY-MM-DD. Required untuk transaction_type 2",
						},
						"item_price": map[string]interface{}{
							"type":        "number",
							"description": "Harga per unit item. Required untuk transaction_type 2",
						},
						"supplier_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional supplier ID. Gunakan get_item_suppliers untuk mencari supplier_id",
						},
						"supplier_name": map[string]interface{}{
							"type":        "string",
							"description": "Optional supplier name. Required jika supplier_id kosong saat transaction_type 2",
						},
						"supplier_phone": map[string]interface{}{
							"type":        "string",
							"description": "Optional supplier phone untuk membuat supplier baru jika supplier_id tidak ditemukan",
						},
						"supplier_url": map[string]interface{}{
							"type":        "string",
							"description": "Optional supplier URL",
						},
						"supplier_price": map[string]interface{}{
							"type":        "number",
							"description": "Optional harga supplier",
						},
						"notes": map[string]interface{}{
							"type":        "string",
							"description": "Optional catatan untuk transaksi",
						},
					},
					"required": []string{"item_uom", "item_category", "stock", "garage_id", "transaction_type"},
				},
			},
		},
		{
			Type: "function",
			Name: "create_inventory_request",
			Function: FunctionDefinition{
				Name:        "create_inventory_request",
				Description: "Buat permintaan inventory request. Kirim item_id atau item_name (tidak boleh keduanya), quantity, garage_id, dan optional employee_id, item_uom, item_category, notes. request_number dan status (default 2) di-generate otomatis.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional item ID. Kirim item_id atau item_name, tidak boleh keduanya",
						},
						"item_name": map[string]interface{}{
							"type":        "string",
							"description": "Optional item name. Kirim item_id atau item_name, tidak boleh keduanya",
						},
						"quantity": map[string]interface{}{
							"type":        "integer",
							"description": "Jumlah yang diminta",
						},
						"garage_id": map[string]interface{}{
							"type":        "string",
							"description": "Garage ID tujuan permintaan",
						},
						"employee_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional employee ID karyawan yang meminta",
						},
						"item_uom": map[string]interface{}{
							"type":        "string",
							"description": "Optional satuan item",
						},
						"item_category": map[string]interface{}{
							"type":        "integer",
							"description": "Optional kategori item: 1 = Kebutuhan Armada, 2 = kebutuhan kantor",
						},
						"notes": map[string]interface{}{
							"type":        "string",
							"description": "Optional catatan untuk permintaan",
						},
					},
					"required": []string{"quantity", "garage_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "approve_inventory_request",
			Function: FunctionDefinition{
				Name:        "approve_inventory_request",
				Description: "Setujui (approve) inventory request. Hanya admin yang bisa menggunakan tool ini. Akan mengurangi stok di inventory_item_garage jika item_id diberikan.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"request_id": map[string]interface{}{
							"type":        "string",
							"description": "Request ID yang akan disetujui",
						},
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "Optional item ID untuk update stock di garage",
						},
					},
					"required": []string{"request_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "reject_inventory_request",
			Function: FunctionDefinition{
				Name:        "reject_inventory_request",
				Description: "Tolak (reject) inventory request. Hanya admin yang bisa menggunakan tool ini.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"request_id": map[string]interface{}{
							"type":        "string",
							"description": "Request ID yang akan ditolak",
						},
					},
					"required": []string{"request_id"},
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
				Description: "Tolak (reject) an order by order_id. Sertakan reason singkat agar customer menerima alasan pembatalan dari tim.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID to reject",
						},
						"reason": map[string]interface{}{
							"type":        "string",
							"description": "Alasan singkat penolakan/pembatalan pesanan untuk dikirim ke customer",
						},
					},
					"required": []string{"order_id", "reason"},
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
		{
			Type: "function",
			Name: "get_fleet_prices",
			Function: FunctionDefinition{
				Name:        "get_fleet_prices",
				Description: "Get rental prices for a specific fleet. type_id: 1 = CityTour, 2 = Overland, 3 = Drop Only.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Fleet ID",
						},
						"type_id": map[string]interface{}{
							"type":        "string",
							"description": "Service type: 1 = CityTour, 2 = Overland, 3 = Drop Only",
						},
					},
					"required": []string{"fleet_id", "type_id"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_fleet_addons",
			Function: FunctionDefinition{
				Name:        "get_fleet_addons",
				Description: "Get available add-ons/extra services for a specific fleet.",
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
			Name: "create_order",
			Function: FunctionDefinition{
				Name:        "create_order",
				Description: "Create a new booking/order for fleet rental. All required params must be collected first: fleet_id, price_id, fullname, email, address, start_date, end_date, pickup_city_id, pickup_location, qty.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fleet_id": map[string]interface{}{
							"type":        "string",
							"description": "Fleet/armada ID",
						},
						"price_id": map[string]interface{}{
							"type":        "string",
							"description": "Price ID from get_fleet_prices",
						},
						"fullname": map[string]interface{}{
							"type":        "string",
							"description": "Customer full name",
						},
						"email": map[string]interface{}{
							"type":        "string",
							"description": "Customer email",
						},
						"address": map[string]interface{}{
							"type":        "string",
							"description": "Customer address",
						},
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Departure YYYY-MM-DD HH:MM",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "Return YYYY-MM-DD HH:MM",
						},
						"pickup_city_id": map[string]interface{}{
							"type":        "string",
							"description": "Pickup city ID (get from get_city_list)",
						},
						"pickup_location": map[string]interface{}{
							"type":        "string",
							"description": "Pickup location address",
						},
						"destinations": map[string]interface{}{
							"type":        "string",
							"description": "JSON: [{\"location\": \"...\", \"city_id\": \"1\"}]",
						},
						"qty": map[string]interface{}{
							"type":        "integer",
							"description": "Number of units (default 1)",
						},
						"addons": map[string]interface{}{
							"type":        "string",
							"description": "JSON: [\"addon_id_1\"]",
						},
						"additional_request": map[string]interface{}{
							"type":        "string",
							"description": "Optional notes",
						},
					},
					"required": []string{"fleet_id", "price_id", "fullname", "email", "address", "start_date", "end_date", "pickup_city_id", "pickup_location"},
				},
			},
		},
	}
}

// GetCompanyToolDefinitions returns tool definitions for the Company Assistant (Skenario 2)
// Only includes tools relevant for customer-facing interactions, not internal operations.
func GetCompanyToolDefinitions() []ToolDefinition {
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
				Description: "Check vehicle availability by date range",
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
				Description: "Get list of available fleets/armada with optional filters",
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
				Description: "Get detailed information for a fleet by fleet ID, including facilities (fasilitas), reviews, and ratings",
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
				Description: "Get the list of cities served by the organization, including minimal rental days (minimal_sewa) and service types (Overland, CityTour, DropOnly)",
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
			Name: "get_trip_distance",
			Function: FunctionDefinition{
				Name:        "get_trip_distance",
				Description: "Estimate trip distance (km) between pickup/origin and destination using public routing/geocoding. Returns recommended minimal rental days for overland (pulang-pergi) vs drop-only based on distance thresholds.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"from": map[string]interface{}{
							"type":        "string",
							"description": "Origin/pickup city or location name. If empty, will default to company city.",
						},
						"to": map[string]interface{}{
							"type":        "string",
							"description": "Destination city or location name (e.g. Brebes)",
						},
					},
					"required": []string{"to"},
				},
			},
		},
		{
			Type: "function",
			Name: "get_customer_list",
			Function: FunctionDefinition{
				Name:        "get_customer_list",
				Description: "Search customers by name. Returns customer_id, name, phone, email, and address.",
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
			Name: "get_order_list",
			Function: FunctionDefinition{
				Name:        "get_order_list",
				Description: "View bookings/orders (customer-facing). Only shows orders belonging to the customer's phone number. Includes summary counts and payment status.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"period": map[string]interface{}{
							"type":        "string",
							"description": "Order month filter in YYYY-MM format (e.g. 2026-06 for June 2026)",
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
				Description: "View order detail by order_id including itinerary, payment summary, and customer info",
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
					},
					"required": []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_organization_info",
			Function: FunctionDefinition{
				Name:        "get_organization_info",
				Description: "Get company information including address, phone, WhatsApp, email, and location coordinates",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_garage_list",
			Function: FunctionDefinition{
				Name:        "get_garage_list",
				Description: "Get daftar garage/garasi/lokasi perusahaan. Gunakan untuk menjawab pertanyaan tentang lokasi garasi atau alamat garasi",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "get_bank_accounts",
			Function: FunctionDefinition{
				Name:        "get_bank_accounts",
				Description: "Get daftar rekening pembayaran perusahaan (bank accounts). Gunakan untuk menjawab pertanyaan tentang nomor rekening untuk pembayaran",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Name: "print_invoice",
			Function: FunctionDefinition{
				Name:        "print_invoice",
				Description: "Generate and send invoice PDF for an order to WhatsApp. Only works for orders belonging to the customer's phone number",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Order ID",
						},
						"invoice_number": map[string]interface{}{
							"type":        "string",
							"description": "Optional invoice number",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
	}
}
