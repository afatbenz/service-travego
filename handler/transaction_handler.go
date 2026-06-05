package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var paymentStatusOnce sync.Once
var paymentStatusMap map[int]string
var paymentMethodMap map[int]string
var transactionCategoryMap map[string]string
var transactionItemMap map[string]string

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
		transactionItemMap = map[string]string{}
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
			TransactionItems []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"transaction-items"`
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
		for _, it := range cfg.TransactionItems {
			transactionItemMap[it.ID] = it.Label
		}
	})
}

func loadTransactionCategoryItemSets() (map[string]struct{}, map[string]struct{}, error) {
	f, err := os.Open("config/common.json")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var cfg struct {
		TransactionCategories []struct {
			ID string `json:"id"`
		} `json:"transaction-categories"`
		TransactionItems []struct {
			ID string `json:"id"`
		} `json:"transaction-items"`
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, nil, err
	}

	categories := make(map[string]struct{}, len(cfg.TransactionCategories))
	for _, it := range cfg.TransactionCategories {
		k := strings.ToUpper(strings.TrimSpace(it.ID))
		if k == "" {
			continue
		}
		categories[k] = struct{}{}
	}
	items := make(map[string]struct{}, len(cfg.TransactionItems))
	for _, it := range cfg.TransactionItems {
		k := strings.ToUpper(strings.TrimSpace(it.ID))
		if k == "" {
			continue
		}
		items[k] = struct{}{}
	}
	return categories, items, nil
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

		transactionItemKey := strings.ToUpper(strings.TrimSpace(row.TransactionItem))
		fmt.Println("get transaction item key:", row.TransactionItem, transactionItemKey)
		transactionItemLabel := ""
		if transactionItemKey != "" {
			if label, ok := transactionItemMap[transactionItemKey]; ok && label != "" {
				transactionItemLabel = label
			} else {
				transactionItemLabel = transactionItemKey
			}
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
			TransactionItemLabel:     transactionItemLabel,
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

func (h *TransactionHandler) SubmitExpenseTransaction(c *fiber.Ctx) error {
	var req model.SubmitExpenseTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	req.Description = strings.TrimSpace(req.Description)
	req.UnitID = strings.TrimSpace(req.UnitID)
	req.TransactionDate = strings.TrimSpace(req.TransactionDate)
	req.TransactionCategory = strings.ToUpper(strings.TrimSpace(req.TransactionCategory))
	req.TransactionItem = strings.ToUpper(strings.TrimSpace(req.TransactionItem))

	if req.Amount <= 0 {
		return helper.BadRequestResponse(c, "amount must be greater than 0")
	}
	if req.Description == "" {
		return helper.BadRequestResponse(c, "description is required")
	}
	if req.PaymentMethod == 0 {
		return helper.BadRequestResponse(c, "payment_method is required")
	}
	if req.PaymentType == 0 {
		return helper.BadRequestResponse(c, "payment_type is required")
	}
	if req.TransactionDate == "" {
		return helper.BadRequestResponse(c, "transaction_date is required")
	}
	if _, err := time.Parse("2006-01-02", req.TransactionDate); err != nil {
		return helper.BadRequestResponse(c, "transaction_date must be YYYY-MM-DD")
	}
	if req.TransactionCategory == "" {
		return helper.BadRequestResponse(c, "transaction_category is required")
	}
	if req.TransactionItem == "" {
		return helper.BadRequestResponse(c, "transaction_item is required")
	}
	if req.UnitID != "" {
		if _, err := uuid.Parse(req.UnitID); err != nil {
			return helper.BadRequestResponse(c, "unit_id must be a valid uuid")
		}
	}

	catSet, itemSet, err := loadTransactionCategoryItemSets()
	if err != nil {
		return helper.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to load common config")
	}
	if _, ok := catSet[req.TransactionCategory]; !ok {
		return helper.BadRequestResponse(c, "transaction_category not found")
	}
	if _, ok := itemSet[req.TransactionItem]; !ok {
		return helper.BadRequestResponse(c, "transaction_item not found")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || strings.TrimSpace(orgID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := h.service.SubmitExpenseTransaction(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusCreated, "Expense transaction submitted successfully", nil)
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

	orderType := ""
	if filteredBy == "items" {
		orderType = strings.ToLower(strings.TrimSpace(c.Query("order_type")))
		if orderType == "fleets" {
			orderType = "fleet"
		}
		if orderType != "" && orderType != "fleet" && orderType != "tour" {
			return helper.BadRequestResponse(c, "Invalid query parameter: order_type must be 'fleet' or 'tour'")
		}
	}

	tagsRaw := strings.TrimSpace(c.Query("tags"))
	reqTags := make([]string, 0)
	if tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			reqTags = append(reqTags, t)
		}
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
			Tags  []string `json:"tags"`
		} `json:"transaction-categories"`
		TransactionItems []struct {
			ID    string   `json:"id"`
			Label string   `json:"label"`
			Type  []string `json:"type"`
			Tags  []string `json:"tags"`
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

	matchesOrderType := func(types []string, t string) bool {
		if t == "" {
			return true
		}
		for _, v := range types {
			v = strings.ToLower(strings.TrimSpace(v))
			if v == "fleets" {
				v = "fleet"
			}
			if v == t {
				return true
			}
		}
		return false
	}

	matchesTags := func(itemTags []string, tags []string) bool {
		if len(tags) == 0 {
			return true
		}
		if len(itemTags) == 0 {
			return false
		}
		itemSet := make(map[string]struct{}, len(itemTags))
		for _, t := range itemTags {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			itemSet[t] = struct{}{}
		}
		for _, t := range tags {
			if _, ok := itemSet[t]; ok {
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
			if !matchesTags(it.Tags, reqTags) {
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
			if !matchesOrderType(it.Type, orderType) {
				continue
			}
			if !matchesTags(it.Tags, reqTags) {
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

func (h *TransactionHandler) SubmitFleetTripExpenseForm(c *fiber.Ctx) error {
	var req struct {
		TransactionItem string  `json:"transaction_item"`
		TransactionDate string  `json:"transaction_date,omitempty"`
		ScheduleNumber  string  `json:"schedule_number"`
		PaymentMethod   int     `json:"payment_method"`
		Amount          float64 `json:"amount"`
		Description     string  `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid request body")
	}

	req.TransactionItem = strings.ToUpper(strings.TrimSpace(req.TransactionItem))
	req.TransactionDate = strings.TrimSpace(req.TransactionDate)
	req.ScheduleNumber = strings.TrimSpace(req.ScheduleNumber)
	req.Description = strings.TrimSpace(req.Description)

	if req.TransactionDate == "" {
		req.TransactionDate = time.Now().Format("2006-01-02")
	}

	if req.TransactionItem == "" || req.ScheduleNumber == "" || req.Amount <= 0 {
		return helper.BadRequestResponse(c, "Missing required fields: transaction_item, schedule_number, amount must be greater than 0")
	}

	if req.TransactionItem == "TRX-I00" {
		req.PaymentMethod = 1
	} else if req.PaymentMethod == 0 {
		return helper.BadRequestResponse(c, "Missing required fields: payment_method")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || strings.TrimSpace(orgID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "User not found")
	}

	err := h.service.SubmitFleetTripExpense(orgID, userID, req.TransactionItem, req.ScheduleNumber, req.PaymentMethod, req.Amount, req.Description)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	totalAmount, err := h.service.GetFleetTripTotalAmount(req.ScheduleNumber)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	totalExpenses, totalReimburse, err := h.service.GetFleetTripAmountSummaryByPaymentMethod(req.ScheduleNumber)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet trip expense submitted successfully", map[string]interface{}{
		"total_amount":    totalAmount,
		"total_expenses":  totalExpenses,
		"total_reimburse": totalReimburse,
	})
}
