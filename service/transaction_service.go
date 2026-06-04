package service

import (
	"fmt"
	"net/http"
	"service-travego/model"
	"service-travego/repository"
	"strings"
)

type TransactionService struct {
	repo *repository.TransactionRepository
}

func NewTransactionService(repo *repository.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) ListAllRevenue(orgID string, req *model.TransactionListRequest) ([]model.TransactionListItem, error) {
	return s.listTransactions(orgID, req, "revenue")
}

func (s *TransactionService) ListAllExpenses(orgID string, req *model.TransactionListRequest) ([]model.TransactionListItem, error) {
	return s.listTransactions(orgID, req, "expenses")
}

func (s *TransactionService) listTransactions(orgID string, req *model.TransactionListRequest, mode string) ([]model.TransactionListItem, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if req.Month < 0 || req.Month > 12 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "month tidak valid")
	}
	if req.Year < 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "year tidak valid")
	}

	var rows []model.TransactionListRow
	var err error

	if mode == "revenue" {
		rows, err = s.repo.ListAllRevenue(orgID, req)
	} else {
		rows, err = s.repo.ListAllExpenses(orgID, req)
	}

	if err != nil {
		return nil, err
	}

	out := make([]model.TransactionListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.TransactionListItem{
			TransactionID:            r.TransactionID,
			OrderType:                r.OrderType,
			InvoiceNumber:            r.InvoiceNumber,
			Description:              r.Description,
			TransactionType:          r.TransactionType,
			TransactionItem:          r.TransactionItem,
			TransactionCategory:      r.TransactionCategory,
			TransactionCategoryLabel: r.TransactionCategoryLabel,
			TransactionDate:          r.TransactionDate.Format("2006-01-02"),
			Status:                   int(r.Status),
			Amount:                   r.Amount,
			CreatedAt:                r.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:                r.CreatedBy,
		})
	}
	fmt.Println(out)
	return out, nil
}

func (s *TransactionService) CreateManualRevenue(orgID, userID string, req *model.CreateManualRevenueRequest) error {
	if strings.TrimSpace(orgID) == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if strings.TrimSpace(userID) == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}

	err := s.repo.CreateManualTransaction(orgID, userID, &repository.CreateManualTransactionRequest{
		OrderType:       req.OrderType,
		OrderID:         req.OrderID,
		Description:     req.Description,
		TransactionDate: req.TransactionDate,
		Status:          req.Status,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		PaymentMethod:   req.PaymentMethod,
		BankAccount:     req.BankAccount,
		BankCode:        req.BankCode,
	})
	return err
}

// CreateManualExpense creates a manual expense transaction with the specified order_type
func (s *TransactionService) CreateManualExpense(orgID, userID string, req *model.CreateManualRevenueRequest) error {
	if strings.TrimSpace(orgID) == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if strings.TrimSpace(userID) == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}

	err := s.repo.CreateManualTransaction(orgID, userID, &repository.CreateManualTransactionRequest{
		OrderType:       req.OrderType,
		OrderID:         req.OrderID,
		Description:     req.Description,
		TransactionDate: req.TransactionDate,
		Status:          req.Status,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		PaymentMethod:   req.PaymentMethod,
		BankAccount:     req.BankAccount,
		BankCode:        req.BankCode,
	})
	return err
}

func (s *TransactionService) SubmitFleetTripExpense(orgID, userID, transactionItem, scheduleNumber string, paymentMethod int, amount float64, description string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	transactionItem = strings.ToUpper(strings.TrimSpace(transactionItem))
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	description = strings.TrimSpace(description)

	if orgID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if userID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}
	if transactionItem == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_item is required")
	}
	if scheduleNumber == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}
	if amount <= 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "amount must be greater than 0")
	}

	if transactionItem == "TRX-I00" {
		paymentMethod = 1
	} else if paymentMethod == 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "payment_method is required")
	}

	orderID, ok, err := s.repo.GetFleetOrderIDByScheduleNumber(scheduleNumber, orgID)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get schedule order")
	}
	if !ok || strings.TrimSpace(orderID) == "" {
		return NewServiceError(ErrNotFound, http.StatusNotFound, "SCHEDULE_NOT_FOUND")
	}

	if transactionItem == "TRX-I00" {
		desc := fmt.Sprintf("Biaya Operasional %s", scheduleNumber)
		return s.repo.CreateFleetTripOperationalExpenseTransaction(orgID, userID, orderID, scheduleNumber, amount, desc)
	}

	totalAmount, err := s.repo.SumTransactionsAmountByReferenceID(scheduleNumber)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get total amount")
	}
	totalExpenses, err := s.repo.SumFleetTripAmountByScheduleNumber(scheduleNumber)
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get total expenses")
	}

	remaining := totalAmount - totalExpenses
	if remaining <= 0 {
		return s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 2, amount, description)
	}
	if totalExpenses+amount <= totalAmount {
		return s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 1, amount, description)
	}

	firstAmount := remaining
	secondAmount := amount - remaining

	if firstAmount > 0 {
		if err := s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 1, firstAmount, description); err != nil {
			return err
		}
	}
	if secondAmount > 0 {
		if err := s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 2, secondAmount, description); err != nil {
			return err
		}
	}
	return nil
}

func (s *TransactionService) GetFleetTripTotalAmount(scheduleNumber string) (float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return 0, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}

	total, err := s.repo.SumTransactionsAmountByReferenceID(scheduleNumber)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *TransactionService) GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber string) (float64, float64, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	if scheduleNumber == "" {
		return 0, 0, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}

	m, err := s.repo.SumFleetTripAmountByScheduleNumberAndPaymentMethod(scheduleNumber)
	if err != nil {
		return 0, 0, err
	}

	totalExpenses := 0.0
	if v, ok := m[1]; ok {
		totalExpenses = v
	}
	totalReimburse := 0.0
	if v, ok := m[2]; ok {
		totalReimburse = v
	}
	return totalExpenses, totalReimburse, nil
}

func (s *TransactionService) ListFleetTripExpenses(scheduleNumber, orgID string) ([]model.FleetTripExpenseRow, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if scheduleNumber == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}
	return s.repo.ListFleetTripExpensesByScheduleNumber(scheduleNumber, orgID)
}
