import sys

with open('tools.go', 'r') as f:
    content = f.read()

# Find the unique marker near the end
marker = '\t\t\t\tName:        "print_invoice",'
idx = content.find(marker)
if idx < 0:
    print("ERROR: marker not found")
    sys.exit(1)

# Find the closing of GetCompanyToolDefinitions (after print_invoice block)
# The pattern is: after the closing }, of print_invoice, then \t\t},\n\t}\n}
closing_marker = '\t\t},\n\t}\n}'
last_closing = content.rfind(closing_marker)

new_tools = """\t\t\t{
\t\t\t\tType: "function",
\t\t\t\tName: "get_fleet_prices",
\t\t\t\tFunction: FunctionDefinition{
\t\t\t\t\tName:        "get_fleet_prices",
\t\t\t\t\tDescription: "Get rental prices for a specific fleet by fleet_id and service type. Service type (type_id): 1 = CityTour (in-city), 2 = Overland (inter-city), 3 = Drop Only (one way). Ask the customer for the fleet and desired service type first.",
\t\t\t\t\tParameters: map[string]interface{}{
\t\t\t\t\t\t"type": "object",
\t\t\t\t\t\t"properties": map[string]interface{}{
\t\t\t\t\t\t\t"fleet_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Fleet ID to get prices for",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"type_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Service type: 1 = CityTour, 2 = Overland, 3 = Drop Only",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t},
\t\t\t\t\t\t"required": []string{"fleet_id", "type_id"},
\t\t\t\t\t},
\t\t\t\t},
\t\t\t},
\t\t\t{
\t\t\t\tType: "function",
\t\t\t\tName: "get_fleet_addons",
\t\t\t\tFunction: FunctionDefinition{
\t\t\t\t\tName:        "get_fleet_addons",
\t\t\t\t\tDescription: "Get list of available add-ons/extra services for a specific fleet. Ask the customer if they want to see available add-ons during booking.",
\t\t\t\t\tParameters: map[string]interface{}{
\t\t\t\t\t\t"type": "object",
\t\t\t\t\t\t"properties": map[string]interface{}{
\t\t\t\t\t\t\t"fleet_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Fleet ID to get add-ons for",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t},
\t\t\t\t\t\t"required": []string{"fleet_id"},
\t\t\t\t\t},
\t\t\t\t},
\t\t\t},
\t\t\t{
\t\t\t\tType: "function",
\t\t\t\tName: "create_order",
\t\t\t\tFunction: FunctionDefinition{
\t\t\t\t\tName:        "create_order",
\t\t\t\t\tDescription: "Create a new booking/order for fleet rental. ALL required parameters must be collected from the customer before calling this. Required: fleet_id, price_id, fullname, email, address, start_date, end_date, pickup_city_id, pickup_location, qty. Customer phone is taken automatically.",
\t\t\t\t\tParameters: map[string]interface{}{
\t\t\t\t\t\t"type": "object",
\t\t\t\t\t\t"properties": map[string]interface{}{
\t\t\t\t\t\t\t"fleet_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Fleet/armada ID to book",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"price_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Price ID from get_fleet_prices",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"fullname": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Customer full name",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"email": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Customer email address",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"address": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Customer home address",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"start_date": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Departure date in YYYY-MM-DD HH:MM format",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"end_date": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Return date in YYYY-MM-DD HH:MM format",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"pickup_city_id": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Pickup city ID (use get_city_list to find city_id)",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"pickup_location": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Detailed pickup location address",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"destinations": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": 'JSON array of daily trip destinations: [{"location": "City name", "city_id": "1"}]',
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"qty": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "integer",
\t\t\t\t\t\t\t\t"description": "Number of fleet units to book (default: 1)",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"addons": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": 'JSON array of addon IDs from get_fleet_addons: ["addon_id_1", "addon_id_2"]',
\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t"additional_request": map[string]interface{}{
\t\t\t\t\t\t\t\t"type":        "string",
\t\t\t\t\t\t\t\t"description": "Optional additional notes or requests",
\t\t\t\t\t\t\t},
\t\t\t\t\t\t},
\t\t\t\t\t\t"required": []string{"fleet_id", "price_id", "fullname", "email", "address", "start_date", "end_date", "pickup_city_id", "pickup_location"},
\t\t\t\t\t},
\t\t\t\t},
\t\t\t},
"""

prev_end = content[:last_closing] + '\t\t},' + new_tools + '\t}\n}\n'
with open('tools.go', 'w') as f:
    f.write(prev_end)

print("SUCCESS")
