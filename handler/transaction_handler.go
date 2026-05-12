package handler

import (
	"encoding/json"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

var paymentStatusOnce sync.Once
var paymentStatusMap map[int]string

var transactionTypeMap = map[int]string{
	int(model.TransactionTypeIncomeRental):      "Income Rental",
	int(model.TransactionTypeIncomeTourPackage): "Income Tour Package",
	int(model.TransactionTypeIncomeComission):   "Income Commission",
	int(model.TransactionTypeIncomeOtherIncome): "Income Other Income",
	int(model.TransactionTypeIncomeAds):         "Income Ads",

	int(model.TransactionTypeExpenseFuel):               "Expense Fuel",
	int(model.TransactionTypeExpenseTol):                "Expense Toll",
	int(model.TransactionTypeExpenseDriverAllowance):    "Expense Driver Allowance",
	int(model.TransactionTypeExpenseGuideFee):           "Expense Guide Fee",
	int(model.TransactionTypeExpenseCrewMeal):           "Expense Crew Meal",
	int(model.TransactionTypeExpenseVehicleMaintenance): "Expense Vehicle Maintenance",
	int(model.TransactionTypeExpenseVehicleTax):         "Expense Vehicle Tax",
	int(model.TransactionTypeExpenseVehicleInsurance):   "Expense Vehicle Insurance",
	int(model.TransactionTypeExpenseHotel):              "Expense Hotel",
	int(model.TransactionTypeExpenseRestaurant):         "Expense Restaurant",
	int(model.TransactionTypeExpenseAttractionTicket):   "Expense Attraction Ticket",
	int(model.TransactionTypeExpenseSalary):             "Expense Salary",
	int(model.TransactionTypeExpenseOfficeRent):         "Expense Office Rent",
	int(model.TransactionTypeExpenseUtility):            "Expense Utility",
	int(model.TransactionTypeExpenseMarketing):          "Expense Marketing",
	int(model.TransactionTypeExpenseBankCharge):         "Expense Bank Charge",
	int(model.TransactionTypeExpenseOtherExpenses):      "Expense Other Expenses",
	int(model.TransactionTypeExpenseCommission):         "Expense Commission",
}

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func ensurePaymentStatusLoaded() {
	paymentStatusOnce.Do(func() {
		paymentStatusMap = map[int]string{}
		f, err := os.Open("config/common.json")
		if err != nil {
			return
		}
		defer f.Close()

		var cfg model.CommonConfig
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return
		}
		for _, it := range cfg.PaymentStatus {
			paymentStatusMap[it.ID] = it.Label
		}
	})
}

func (h *TransactionHandler) ListAllRevenue(c *fiber.Ctx) error {
	return h.listTransactions(c, "revenue")
}

func (h *TransactionHandler) ListAllExpenses(c *fiber.Ctx) error {
	return h.listTransactions(c, "expenses")
}

func (h *TransactionHandler) listTransactions(c *fiber.Ctx, mode string) error {
	var req model.TransactionListRequest
	if err := c.QueryParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid query parameters")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	var rows []model.TransactionListItem
	var err error

	if mode == "revenue" {
		rows, err = h.service.ListAllRevenue(orgID, &req)
	} else {
		rows, err = h.service.ListAllExpenses(orgID, &req)
	}

	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	ensurePaymentStatusLoaded()

	transformedRes := make([]model.TransactionListItem, len(rows))
	for i, row := range rows {
		transactionDateStr := ""
		if row.TransactionDate != "" {
			transactionDateStr = row.TransactionDate
		}

		statusLabel := ""
		if label, ok := paymentStatusMap[row.Status]; ok && label != "" {
			statusLabel = label
		} else if row.Status != 0 {
			statusLabel = strconv.Itoa(row.Status)
		}

		transactionTypeLabel := ""
		if label, ok := transactionTypeMap[row.TransactionType]; ok {
			transactionTypeLabel = label
		}

		transactionMarkLabel := ""
		if row.TransactionMark == 1 {
			transactionMarkLabel = "income"
		} else if row.TransactionMark == 2 {
			transactionMarkLabel = "expenses"
		}

		createdAtStr := ""
		if row.CreatedAt != "" {
			createdAtStr = row.CreatedAt
		}

		transformedRes[i] = model.TransactionListItem{
			TransactionID:        row.TransactionID,
			OrderType:            row.OrderType,
			InvoiceNumber:        row.InvoiceNumber,
			Description:          row.Description,
			TransactionType:      row.TransactionType,
			TransactionTypeLabel: transactionTypeLabel,
			TransactionMark:      row.TransactionMark,
			TransactionMarkLabel: transactionMarkLabel,
			TransactionDate:      transactionDateStr,
			Status:               int(row.Status),
			StatusLabel:          statusLabel,
			CreatedAt:            createdAtStr,
			CreatedBy:            row.CreatedBy,
			Amount:               row.Amount,
		}
	}

	msg := "Transactions retrieved"
	if mode == "revenue" {
		msg = "Revenue transactions retrieved"
	} else {
		msg = "Expense transactions retrieved"
	}

	return helper.SuccessResponse(c, fiber.StatusOK, msg, transformedRes)
}

func (h *TransactionHandler) CreateManualRevenue(c *fiber.Ctx) error {
	var req struct {
		Description     string  `json:"description"`
		TransactionDate string  `json:"transaction_date"`
		Status          int     `json:"status"`
		TransactionType int     `json:"transaction_type"`
		Amount          float64 `json:"amount"`
		PaymentMethod   int     `json:"payment_method"`
		BankAccount     string  `json:"bank_account,omitempty"`
		BankCode        string  `json:"bank_code,omitempty"`
		OrderType       int     `json:"order_type"`
		OrderID         string  `json:"order_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	if req.Description == "" || req.TransactionDate == "" || req.Status == 0 || req.TransactionType == 0 || req.Amount <= 0 {
		return helper.BadRequestResponse(c, "Missing required fields: description, transaction_date, status, transaction_type, amount must be greater than 0")
	}

	if req.PaymentMethod == 1002 {
		if req.BankAccount == "" || req.BankCode == "" {
			return helper.BadRequestResponse(c, "bank_account and bank_code are required for payment method 1002")
		}
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	err := h.service.CreateManualRevenue(orgID, userID, &model.CreateManualRevenueRequest{
		Description:     req.Description,
		TransactionDate: req.TransactionDate,
		Status:          req.Status,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		PaymentMethod:   req.PaymentMethod,
		BankAccount:     req.BankAccount,
		BankCode:        req.BankCode,
		OrderType:       req.OrderType,
		OrderID:         req.OrderID,
	})

	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Manual revenue created successfully", nil)
}

func (h *TransactionHandler) ListTransactionLabels(c *fiber.Ctx) error {
	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	filteredBy := strings.ToLower(strings.TrimSpace(c.Query("filteredby")))
	if filteredBy == "expnse" {
		filteredBy = "expense"
	}

	keys := make([]int, 0, len(transactionTypeMap))
	for id := range transactionTypeMap {
		if filteredBy == "income" && id > 100 {
			continue
		}
		if filteredBy == "expense" && id <= 100 {
			continue
		}
		keys = append(keys, id)
	}
	sort.Ints(keys)

	types := make([]map[string]interface{}, 0, len(keys))
	for _, id := range keys {
		types = append(types, map[string]interface{}{
			"id":    id,
			"label": transactionTypeMap[id],
		})
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Transaction types retrieved", types)
}

// GetOrderTypes returns all order types with their labels
func (h *TransactionHandler) GetOrderTypes(c *fiber.Ctx) error {
	// Get all order types from the service map
	orderTypes := make([]map[string]interface{}, 0, len(service.OrderTypeLabel))
	for id, label := range service.OrderTypeLabel {
		orderTypes = append(orderTypes, map[string]interface{}{
			"id":    id,
			"label": label,
		})
	}

	// Sort by order type id
	sort.Slice(orderTypes, func(i, j int) bool {
		return orderTypes[i]["id"].(int) < orderTypes[j]["id"].(int)
	})

	return helper.SuccessResponse(c, fiber.StatusOK, "Order types retrieved successfully", orderTypes)
}
