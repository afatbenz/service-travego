package handler

import (
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (h *TransactionHandler) GetFleetTripSummary(c *fiber.Ctx) error {
	scheduleNumber := strings.TrimSpace(c.Query("schedule_number"))
	if scheduleNumber == "" {
		return helper.BadRequestResponse(c, "Missing query parameter: schedule_number")
	}

	orgID, ok := c.Locals("organization_id").(string)
	if !ok || strings.TrimSpace(orgID) == "" {
		return helper.SendErrorResponse(c, fiber.StatusUnauthorized, "Organization not found")
	}

	totalAmount, err := h.service.GetFleetTripTotalAmount(scheduleNumber)
	if err != nil {
		fmt.Println("failed to get fleet trip total amount: ", err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	summary, err := h.service.GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber, orgID)
	if err != nil {
		fmt.Println("failed to get fleet trip amount summary by payment method: ", err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	rows, err := h.service.ListFleetTripExpenses(scheduleNumber, orgID)
	if err != nil {
		fmt.Println("failed to list fleet trip expenses: ", err)
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	ensurePaymentStatusLoaded()

	expenses := make([]model.FleetTripExpenseItem, 0, len(rows))
	for _, row := range rows {
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
		transactionItemLabel := ""
		if transactionItemKey != "" {
			if label, ok := transactionItemMap[transactionItemKey]; ok && label != "" {
				transactionItemLabel = label
			} else {
				transactionItemLabel = transactionItemKey
			}
		}

		paymentMethodLabel := ""
		switch row.PaymentMethod {
		case 1:
			paymentMethodLabel = "Operasional"
		case 2:
			paymentMethodLabel = "Reimburse"
		default:
			if row.PaymentMethod != 0 {
				paymentMethodLabel = strconv.Itoa(row.PaymentMethod)
			}
		}

		createdAt := ""
		if !row.CreatedAt.IsZero() {
			createdAt = row.CreatedAt.Format("2006-01-02 15:04:05")
		}

		expenses = append(expenses, model.FleetTripExpenseItem{
			TransactionTripID:        row.TransactionTripID,
			TransactionCategory:      transactionCategoryKey,
			TransactionCategoryLabel: transactionCategoryLabel,
			TransactionItem:          transactionItemKey,
			TransactionItemLabel:     transactionItemLabel,
			Amount:                   row.Amount,
			Status:                   row.Status,
			RemainingClaim:           summary.RemainingClaim,
			PaymentMethod:            row.PaymentMethod,
			PaymentMethodLabel:       paymentMethodLabel,
			Description:              row.Description,
			CreatedAt:                createdAt,
			CreatedBy:                row.CreatedBy,
		})
	}

	balance := totalAmount - summary.TotalExpenses

	return helper.SuccessResponse(c, fiber.StatusOK, "Fleet trip summary retrieved", map[string]interface{}{
		"total_amount":         totalAmount,
		"total_expenses":       summary.TotalExpenses,
		"total_claimed":        summary.TotalClaimed,
		"total_reimburse":      summary.TotalReimburse,
		"total_item_reimburse": summary.TotalItemReimburse,
		"remaining_claim":      summary.RemainingClaim,
		"balance":              balance,
		"expenses":             expenses,
	})
}
