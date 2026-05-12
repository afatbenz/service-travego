package service

import (
	"net/http"
	"service-travego/model"
	"service-travego/repository"
	"strings"
)

// OrderType constants as per requirements
const (
	OrderTypeFleet      = 1
	OrderTypeTour       = 2
	OrderTypeOperations = 3
	OrderTypeOther      = 4
)

// OrderTypeLabel maps order_type int to human-readable label
var OrderTypeLabel = map[int]string{
	OrderTypeFleet:      "Fleet Order",
	OrderTypeTour:       "Tour Order",
	OrderTypeOperations: "Operations",
	OrderTypeOther:      "Other",
}

type FinanceRevenueExpenseService struct {
	txnRepo *repository.TransactionRepository
}

type GroupedRevenueEntry struct {
	OrderType   int     `json:"order_type"`
	Label       string  `json:"label"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}

type GroupedExpenseEntry struct {
	OrderType   int     `json:"order_type"`
	Label       string  `json:"label"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}

func NewFinanceRevenueExpenseService(repo *repository.TransactionRepository) *FinanceRevenueExpenseService {
	return &FinanceRevenueExpenseService{
		txnRepo: repo,
	}
}

// GetOrderTypeLabel returns the human-readable label for a given order_type
func GetOrderTypeLabel(orderType int) string {
	if label, ok := OrderTypeLabel[orderType]; ok {
		return label
	}
	return "Unknown Order Type"
}

// GetGroupedRevenueByOrderType returns revenue grouped by each order_type
func (s *FinanceRevenueExpenseService) GetGroupedRevenueByOrderType(orgID string, req *model.TransactionListRequest) ([]GroupedRevenueEntry, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if req.Month < 0 || req.Month > 12 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "month tidak valid")
	}
	if req.Year < 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "year tidak valid")
	}

	// Get all revenue transactions
	rows, err := s.txnRepo.ListAllRevenue(orgID, req)
	if err != nil {
		return nil, err
	}

	// Group by order_type
	grouped := make(map[int]struct {
		total float64
		count int
	})

	for _, row := range rows {
		grouped[row.OrderType] = struct {
			total float64
			count int
		}{
			total: grouped[row.OrderType].total + row.Amount,
			count: grouped[row.OrderType].count + 1,
		}
	}

	// Convert to response slice
	var out []GroupedRevenueEntry
	for orderType, data := range grouped {
		out = append(out, GroupedRevenueEntry{
			OrderType:   orderType,
			Label:       GetOrderTypeLabel(orderType),
			TotalAmount: data.total,
			Count:       data.count,
		})
	}

	return out, nil
}

// GetGroupedExpensesByOrderType returns expenses grouped by each order_type
func (s *FinanceRevenueExpenseService) GetGroupedExpensesByOrderType(orgID string, req *model.TransactionListRequest) ([]GroupedExpenseEntry, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "Organization not found")
	}
	if req.Month < 0 || req.Month > 12 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "month tidak valid")
	}
	if req.Year < 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "year tidak valid")
	}

	// Get all expense transactions
	rows, err := s.txnRepo.ListAllExpenses(orgID, req)
	if err != nil {
		return nil, err
	}

	// Group by order_type
	grouped := make(map[int]struct {
		total float64
		count int
	})

	for _, row := range rows {
		grouped[row.OrderType] = struct {
			total float64
			count int
		}{
			total: grouped[row.OrderType].total + row.Amount,
			count: grouped[row.OrderType].count + 1,
		}
	}

	// Convert to response slice
	var out []GroupedExpenseEntry
	for orderType, data := range grouped {
		out = append(out, GroupedExpenseEntry{
			OrderType:   orderType,
			Label:       GetOrderTypeLabel(orderType),
			TotalAmount: data.total,
			Count:       data.count,
		})
	}

	return out, nil
}
