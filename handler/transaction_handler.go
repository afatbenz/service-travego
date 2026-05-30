package handler

import (
	"encoding/json"
	"os"
	"service-travego/configs"
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
var paymentMethodMap map[int]string
var transactionCategoryMap map[string]string

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func ensurePaymentStatusLoaded() {
	paymentStatusOnce.Do(func() {
		paymentStatusMap = map[int]string{}
		paymentMethodMap = map[int]string{}
		transactionCategoryMap = map[string]string{}
		f, err := os.Open("config/common.json")
		if err != nil {
			return
		}
		defer f.Close()

		var cfg struct {
			PaymentStatus         []model.CommonItem `json:"payment-status"`
			PaymentMethod         []model.CommonItem `json:"payment-method"`
			TransactionCategories []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"transaction-categories"`
		}
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return
		}
		for _, it := range cfg.PaymentStatus {
			paymentStatusMap[it.ID] = it.Label
		}
		for _, it := range cfg.PaymentMethod {
			paymentMethodMap[it.ID] = it.Label
		}
		for _, it := range cfg.TransactionCategories {
			transactionCategoryMap[it.ID] = it.Label
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
		if label, ok := configs.TransactionTypeLabel[row.TransactionType]; ok {
			transactionTypeLabel = label
		}

		paymentMethodLabel := ""
		if label, ok := paymentMethodMap[row.PaymentMethod]; ok && label != "" {
			paymentMethodLabel = label
		} else if row.PaymentMethod != 0 {
			paymentMethodLabel = strconv.Itoa(row.PaymentMethod)
		}

		transactionCategoryKey := strings.ToUpper(strings.TrimSpace(row.TransactionCategory))
		transactionCategoryLabel := ""
		if transactionCategoryKey != "" {
			if label, ok := transactionCategoryMap[transactionCategoryKey]; ok && label != "" {
				transactionCategoryLabel = label
			} else {
				transactionCategoryLabel = transactionCategoryKey
			}
		}

		TransactionItemLabel := ""
		switch row.TransactionType {
		case 1:
			TransactionItemLabel = "income"
		case 2:
			TransactionItemLabel = "expenses"
		}

		createdAtStr := ""
		if row.CreatedAt != "" {
			createdAtStr = row.CreatedAt
		}

		transformedRes[i] = model.TransactionListItem{
			TransactionID:            row.TransactionID,
			OrderType:                row.OrderType,
			InvoiceNumber:            row.InvoiceNumber,
			Description:              row.Description,
			TransactionDate:          transactionDateStr,
			TransactionType:          row.TransactionType,
			TransactionTypeLabel:     transactionTypeLabel,
			TransactionItem:          row.TransactionItem,
			TransactionItemLabel:     TransactionItemLabel,
			PaymentMethod:            row.PaymentMethod,
			PaymentMethodLabel:       paymentMethodLabel,
			TransactionCategory:      transactionCategoryKey,
			TransactionCategoryLabel: transactionCategoryLabel,
			Status:                   int(row.Status),
			StatusLabel:              statusLabel,
			CreatedAt:                createdAtStr,
			CreatedBy:                row.CreatedBy,
			Amount:                   row.Amount,
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

	keys := make([]int, 0, len(configs.TransactionTypeLabel))
	for id := range configs.TransactionTypeLabel {
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
			"label": configs.TransactionTypeLabel[id],
		})
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Transaction types retrieved", types)
}

// GetOrderTypes returns all order types with their labels
func (h *TransactionHandler) GetTransactionTypes(c *fiber.Ctx) error {
	filteredBy := strings.ToLower(strings.TrimSpace(c.Query("filteredby")))
	if filteredBy == "" {
		return helper.BadRequestResponse(c, "Missing query parameter: filteredby")
	}
	if filteredBy != "categories" && filteredBy != "items" {
		return helper.BadRequestResponse(c, "Invalid query parameter: filteredby must be 'categories' or 'items'")
	}

	reqType := strings.ToLower(strings.TrimSpace(c.Query("type")))
	if reqType == "" {
		return helper.BadRequestResponse(c, "Missing query parameter: type")
	}
	if reqType == "expenses" {
		reqType = "expense"
	}
	if reqType != "income" && reqType != "expense" {
		return helper.BadRequestResponse(c, "Invalid query parameter: type must be 'income' or 'expense'")
	}

	f, err := os.Open("config/common.json")
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load common config")
	}
	defer f.Close()

	var cfg struct {
		TransactionCategories []struct {
			ID    string   `json:"id"`
			Label string   `json:"label"`
			Type  []string `json:"type"`
		} `json:"transaction-categories"`
		TransactionItems []struct {
			ID    string   `json:"id"`
			Label string   `json:"label"`
			Type  []string `json:"type"`
		} `json:"transaction-items"`
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to parse common config")
	}

	matchesType := func(types []string, t string) bool {
		if len(types) == 0 {
			return true
		}
		for _, v := range types {
			v = strings.ToLower(strings.TrimSpace(v))
			if v == "expenses" {
				v = "expense"
			}
			if v == t {
				return true
			}
		}
		return false
	}

	res := make([]map[string]interface{}, 0)

	if filteredBy == "categories" {
		cats := make([]struct {
			ID    string
			Label string
		}, 0, len(cfg.TransactionCategories))
		for _, it := range cfg.TransactionCategories {
			if !matchesType(it.Type, reqType) {
				continue
			}
			cats = append(cats, struct {
				ID    string
				Label string
			}{ID: it.ID, Label: it.Label})
		}
		sort.Slice(cats, func(i, j int) bool {
			return cats[i].ID < cats[j].ID
		})
		res = make([]map[string]interface{}, 0, len(cats))
		for _, it := range cats {
			res = append(res, map[string]interface{}{
				"id":    it.ID,
				"label": it.Label,
			})
		}
	} else {
		items := make([]struct {
			ID    string
			Label string
		}, 0, len(cfg.TransactionItems))
		for _, it := range cfg.TransactionItems {
			if !matchesType(it.Type, reqType) {
				continue
			}
			items = append(items, struct {
				ID    string
				Label string
			}{ID: it.ID, Label: it.Label})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].ID < items[j].ID
		})
		res = make([]map[string]interface{}, 0, len(items))
		for _, it := range items {
			res = append(res, map[string]interface{}{
				"id":    it.ID,
				"label": it.Label,
			})
		}
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order types retrieved successfully", res)
}
