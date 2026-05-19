package service

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"service-travego/config"
	"service-travego/model"
	"service-travego/repository"
	"strconv"

	"github.com/veritrans/go-midtrans"
)

// PaymentService adalah interface untuk logika bisnis payment
type PaymentService interface {
	CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error)
	HandleWebhook(req *model.MidtransWebhookRequest) error
}

type paymentService struct {
	repo           repository.PaymentRepository
	midtransConfig *config.MidtransConfig
}

// NewPaymentService membuat instance baru dari PaymentService
func NewPaymentService(repo repository.PaymentRepository, midtransConfig *config.MidtransConfig) PaymentService {
	return &paymentService{
		repo:           repo,
		midtransConfig: midtransConfig,
	}
}

// CreatePayment menangani pembuatan token Snap Midtrans
func (s *paymentService) CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error) {
	// 1. Validasi payment_type
	if req.PaymentType != 1 && req.PaymentType != 2 {
		return nil, fmt.Errorf("invalid payment type: %d", req.PaymentType)
	}

	var paymentAmount int64

	// 2. Jika payment_type == 1, ambil total_amount dari repository
	if req.PaymentType == 1 {
		amount, err := s.repo.GetOrderTotalAmount(req.OrderID, req.OrderType)
		if err != nil {
			return nil, fmt.Errorf("failed to get order amount: %w", err)
		}
		paymentAmount = amount
	} else {
		// 3. Jika payment_type == 2, gunakan payment_amount dari request
		paymentAmount = req.PaymentAmount
	}

	// 4. Panggil Midtrans Snap API untuk generate snap_token
	snapReq := &midtrans.SnapReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: paymentAmount,
		},
	}

	snapResp, err := s.midtransConfig.Snap.GetToken(snapReq)
	if err != nil {
		return nil, fmt.Errorf("midtrans error: %w", err)
	}

	return &model.PaymentResponse{
		SnapToken: snapResp.Token,
		OrderID:   req.OrderID,
	}, nil
}

// HandleWebhook menangani notifikasi dari Midtrans
func (s *paymentService) HandleWebhook(req *model.MidtransWebhookRequest) error {
	// 1. Verifikasi signature key (SHA512: order_id + status_code + gross_amount + server_key)
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	payload := req.OrderID + req.StatusCode + req.GrossAmount + serverKey

	h := sha512.New()
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))

	if signature != req.SignatureKey {
		return fmt.Errorf("invalid signature key")
	}

	// Ambil gross amount dari webhook
	amountFloat, err := strconv.ParseFloat(req.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid gross amount format: %w", err)
	}
	amount := int64(amountFloat)

	// Kita perlu tahu orderType. Karena webhook Midtrans tidak mengirim orderType secara default,
	// kita bisa asumsikan dari format OrderID atau menyimpannya di metadata saat create.
	// Namun berdasarkan instruksi, kita cek fleet_orders.

	// Coba cek fleet_orders dulu
	totalAmount, err := s.repo.GetOrderTotalAmount(req.OrderID, 1) // orderType 1 = fleet
	orderType := int64(1)
	if err != nil {
		// Jika tidak ketemu di fleet_orders, coba di tour_package_orders
		totalAmount, err = s.repo.GetOrderTotalAmount(req.OrderID, 2) // orderType 2 = tour
		orderType = 2
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}
	}

	// 3. Update payment_status berdasarkan amount
	// 1: paid (full), 4: partial/settlement
	status := 0
	if amount >= totalAmount {
		status = 1 // Paid
	} else {
		status = 4 // Partial
	}

	err = s.repo.UpdatePaymentStatus(req.OrderID, orderType, status)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}
