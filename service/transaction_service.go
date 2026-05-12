package service

import (
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
			TransactionID:   r.TransactionID,
			OrderType:       r.OrderType,
			InvoiceNumber:   r.InvoiceNumber,
			Description:     r.Description,
			TransactionType: r.TransactionType,
			TransactionMark: r.TransactionMark,
			TransactionDate: r.TransactionDate.Format("2006-01-02"),
			Status:          int(r.Status),
			Amount:          r.Amount,
			CreatedAt:       r.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:       r.CreatedBy,
		})
	}
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
