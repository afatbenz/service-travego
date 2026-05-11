package handler

import (
	"encoding/json"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
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

	int(model.TransactionTypeExpenseFuel):             "Expense Fuel",
	int(model.TransactionTypeExpenseTol):              "Expense Toll",
	int(model.TransactionTypeExpenseDriverAllowance):  "Expense Driver Allowance",
	int(model.TransactionTypeExpenseGuideFee):         "Expense Guide Fee",
	int(model.TransactionTypeExpenseCrewMeal):         "Expense Crew Meal",
	int(model.TransactionTypeExpenseVehicleService):   "Expense Vehicle Service",
	int(model.TransactionTypeExpenseVehicleTax):       "Expense Vehicle Tax",
	int(model.TransactionTypeExpenseVehicleInsurance): "Expense Vehicle Insurance",
	int(model.TransactionTypeExpenseHotel):            "Expense Hotel",
	int(model.TransactionTypeExpenseRestaurant):       "Expense Restaurant",
	int(model.TransactionTypeExpenseAttractionTicket): "Expense Attraction Ticket",
	int(model.TransactionTypeExpenseSalary):           "Expense Salary",
	int(model.TransactionTypeExpenseOfficeRent):       "Expense Office Rent",
	int(model.TransactionTypeExpenseUtility):          "Expense Utility",
	int(model.TransactionTypeExpenseMarketing):        "Expense Marketing",
	int(model.TransactionTypeExpenseBankCharge):       "Expense Bank Charge",
	int(model.TransactionTypeExpenseOtherExpenses):    "Expense Other Expenses",
	int(model.TransactionTypeExpenseCommission):       "Expense Commission",
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

func (h *TransactionHandler) ListAllIncome(c *fiber.Ctx) error {
	var req model.TransactionListRequest
	if err := c.QueryParser(&req); err != nil {
		return helper.BadRequestResponse(c, "Invalid query parameters")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || orgID == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	rows, err := h.service.ListAllIncome(orgID, &req)
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

	return helper.SuccessResponse(c, fiber.StatusOK, "Transactions retrieved", transformedRes)
}
