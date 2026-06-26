package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strings"
	"time"
)

type TransactionService struct {
	repo                *repository.TransactionRepository
	notificationService *NotificationService
}

func NewTransactionService(repo *repository.TransactionRepository, notificationService *NotificationService) *TransactionService {
	return &TransactionService{
		repo:                repo,
		notificationService: notificationService,
	}
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
			PaymentMethod:            r.PaymentType,
			PaymentType:              r.PaymentType,
			Status:                   r.Status,
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
		if s.notificationService != nil {
			go func() {
				baseURL := os.Getenv("BASE_URL")
				_, _ = s.notificationService.CreateNotification(orgID, NotificationPayload{
					Title:   "Pengeluaran Reimbursement Baru",
					Message: fmt.Sprintf("Ada pengeluaran reimbursement sebesar %.2f untuk SJP %s", amount, scheduleNumber),
					URL:     baseURL + "/dashboard/schedules/fleet-schedules/detail/" + scheduleNumber,
				})
			}()
		}
		return s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 2, 0, amount, "reimbursement - "+description)
	}
	if totalExpenses+amount <= totalAmount {
		return s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 1, 1, amount, description)
	}

	firstAmount := remaining
	secondAmount := amount - remaining

	if firstAmount > 0 {
		if err := s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 1, 1, firstAmount, description); err != nil {
			return err
		}
	}
	if secondAmount > 0 {
		if err := s.repo.CreateFleetTripExpenseTransaction(orgID, userID, orderID, scheduleNumber, transactionItem, 2, 0, secondAmount, "reimbursement - "+description); err != nil {
			return err
		}
		if s.notificationService != nil {
			go func() {
				baseURL := os.Getenv("BASE_URL")
				_, _ = s.notificationService.CreateNotification(orgID, NotificationPayload{
					Title:   "Pengeluaran Reimbursement Baru",
					Message: fmt.Sprintf("Ada pengeluaran reimbursement sebesar %.2f untuk SJP %s", secondAmount, scheduleNumber),
					URL:     baseURL + "/dashboard/schedules/fleet-schedules/detail/" + scheduleNumber,
				})
			}()
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

func (s *TransactionService) GetFleetTripAmountSummaryByPaymentMethod(scheduleNumber, orgID string) (model.FleetTripAmountSummary, error) {
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	orgID = strings.TrimSpace(orgID)
	result := model.FleetTripAmountSummary{}
	if orgID == "" {
		return result, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if scheduleNumber == "" {
		return result, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}

	return s.repo.GetFleetTripAmountSummary(scheduleNumber, orgID)
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

func (s *TransactionService) DeleteFleetTripExpense(orgID, userID, scheduleNumber, transactionTripID string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	transactionTripID = strings.TrimSpace(transactionTripID)

	if orgID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if userID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}
	if scheduleNumber == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "schedule_number is required")
	}
	if transactionTripID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_trip_id is required")
	}

	err := s.repo.DeleteFleetTripExpense(orgID, scheduleNumber, transactionTripID)
	if errors.Is(err, sql.ErrNoRows) {
		return NewServiceError(ErrNotFound, http.StatusNotFound, "fleet trip expense not found")
	}
	return err
}

func (s *TransactionService) SubmitExpenseTransaction(orgID, userID string, req *model.SubmitExpenseTransactionRequest) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	if orgID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if userID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}
	if req == nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "Invalid request body")
	}

	description := strings.TrimSpace(req.Description)
	unitID := strings.TrimSpace(req.UnitID)
	transactionCategory := strings.ToUpper(strings.TrimSpace(req.TransactionCategory))
	transactionItem := strings.ToUpper(strings.TrimSpace(req.TransactionItem))
	transactionDateStr := strings.TrimSpace(req.TransactionDate)

	if req.Amount <= 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "amount must be greater than 0")
	}
	if description == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "description is required")
	}
	if req.PaymentMethod == 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "payment_method is required")
	}
	if req.PaymentType == 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "payment_type is required")
	}
	if transactionDateStr == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_date is required")
	}
	transactionDate, err := time.Parse("2006-01-02", transactionDateStr)
	if err != nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_date must be YYYY-MM-DD")
	}
	if transactionCategory == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_category is required")
	}
	if transactionItem == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_item is required")
	}

	return s.repo.CreateExpenseTransaction(orgID, userID, &repository.CreateExpenseTransactionRequest{
		Amount:              req.Amount,
		Description:         description,
		UnitID:              unitID,
		PaymentMethod:       req.PaymentMethod,
		PaymentType:         req.PaymentType,
		TransactionDate:     transactionDate,
		TransactionCategory: transactionCategory,
		TransactionItem:     transactionItem,
	})
}

func (s *TransactionService) DeleteExpenseTransaction(orgID, transactionID string) error {
	orgID = strings.TrimSpace(orgID)
	transactionID = strings.TrimSpace(transactionID)
	if orgID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if transactionID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_id is required")
	}

	err := s.repo.SoftDeleteExpenseTransaction(orgID, transactionID)
	if errors.Is(err, sql.ErrNoRows) {
		return NewServiceError(ErrNotFound, http.StatusNotFound, "transaction not found")
	}
	return err
}

func (s *TransactionService) UpdateExpenseTransaction(orgID, userID string, req *model.UpdateExpenseTransactionRequest) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	if orgID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if userID == "" {
		return NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "User not found")
	}
	if req == nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "Invalid request body")
	}

	transactionID := strings.TrimSpace(req.TransactionID)
	unitID := strings.TrimSpace(req.UnitID)
	transactionCategory := strings.ToUpper(strings.TrimSpace(req.TransactionCategory))
	transactionItem := strings.ToUpper(strings.TrimSpace(req.TransactionItem))
	transactionDateStr := strings.TrimSpace(req.TransactionDate)

	if transactionID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_id is required")
	}
	if req.Amount <= 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "amount must be greater than 0")
	}
	if req.PaymentMethod == 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "payment_method is required")
	}
	if transactionDateStr == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_date is required")
	}
	transactionDate, err := time.Parse("2006-01-02", transactionDateStr)
	if err != nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_date must be YYYY-MM-DD")
	}
	if transactionCategory == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_category is required")
	}
	if transactionItem == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "transaction_item is required")
	}

	err = s.repo.UpdateExpenseTransaction(orgID, userID, &repository.UpdateExpenseTransactionRequest{
		TransactionID:       transactionID,
		Amount:              req.Amount,
		UnitID:              unitID,
		PaymentMethod:       req.PaymentMethod,
		TransactionDate:     transactionDate,
		TransactionCategory: transactionCategory,
		TransactionItem:     transactionItem,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return NewServiceError(ErrNotFound, http.StatusNotFound, "transaction not found")
	}
	return err
}

func (s *TransactionService) SubmitFleetTripReimbursement(orgID, userID, scheduleNumber string, recipientID string, paymentMethodID string, transactionDateStr string) error {
	orgID = strings.TrimSpace(orgID)
	userID = strings.TrimSpace(userID)
	scheduleNumber = strings.TrimSpace(scheduleNumber)
	recipientID = strings.TrimSpace(recipientID)
	paymentMethodID = strings.TrimSpace(paymentMethodID)
	if orgID == "" || userID == "" || scheduleNumber == "" || recipientID == "" || paymentMethodID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization_id, user_id, schedule_number, recipient_id, and payment_method_id are required")
	}

	amount, err := s.repo.GetReimbursementAmount(scheduleNumber)
	if err != nil {
		return err
	}

	recordReimbursement := &model.FleetTripReimbursement{
		ScheduleNumber:  scheduleNumber,
		Amount:          amount,
		RecipientID:     recipientID,
		PaymentMethodID: paymentMethodID,
		TransactionDate: transactionDateStr,
	}
	err = s.repo.CreateFleetTripReimbursement(orgID, userID, recordReimbursement)
	if err != nil {
		return err
	}

	updateErr := s.repo.MarkReimbursementPaid(scheduleNumber)
	if updateErr != nil {
		return updateErr
	}
	return nil
}
