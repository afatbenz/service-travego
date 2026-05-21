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
	"time"

	"github.com/google/uuid"
	"github.com/veritrans/go-midtrans"
)

// PaymentService adalah interface untuk logika bisnis payment
type PaymentService interface {
	CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error)
	PaymentNotifications(req *model.MidtransWebhookRequest) error
	UpdatePaymentStatus(orderID string, orderType int64, status int) error
	ProcessPaymentNotification(req *model.MidtransWebhookRequest) error
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

// ProcessPaymentNotification handles the logic for Midtrans notification
func (s *paymentService) ProcessPaymentNotification(req *model.MidtransWebhookRequest) error {
	if req.StatusCode != "200" {
		return nil
	}

	orgID, totalAmount, orderType, err := s.repo.GetOrderDetails(req.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order details: %w", err)
	}

	grossAmount, err := strconv.ParseFloat(req.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid gross amount: %w", err)
	}

	if err := s.repo.UpdatePaymentOrderNotification(req.OrderID, orgID, totalAmount, grossAmount, req.TransactionID); err != nil {
		return fmt.Errorf("failed to update payment order: %w", err)
	}

	if err := s.UpdatePaymentStatus(req.OrderID, orderType, 1); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	createdAt := time.Now().Format("2006-01-02 15:04:05")
	if err := s.repo.InsertPaymentMidtrans(req, createdAt); err != nil {
		return fmt.Errorf("failed to insert payment midtrans: %w", err)
	}

	return nil
}

func (s *paymentService) UpdatePaymentStatus(orderID string, orderType int64, status int) error {
	return s.repo.UpdateOrderStatus(orderID, orderType, status)
}

func (s *paymentService) CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error) {
	if req.PaymentType != 1 && req.PaymentType != 2 {
		return nil, fmt.Errorf("invalid payment type: %d", req.PaymentType)
	}

	var paymentAmount int64

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

	// 4. Update status_payment menjadi 3 di tabel order yang sesuai
	err := s.repo.UpdatePaymentStatus(req.OrderID, req.OrderType, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to update order payment status: %w", err)
	}

	// 5. Generate invoice number
	invoiceNumber, err := s.repo.GetNextInvoiceNumber(req.OrganizationID, int(req.OrderType))
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// 6. Insert ke payment_orders
	paymentID := uuid.New().String()
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[DEBUG] Service CreatePayment - calling InsertPaymentOrder with orgID: %s, userID: %s\n", req.OrganizationID, req.UserID)
	err = s.repo.InsertPaymentOrder(paymentID, req.OrderType, req.OrderID, req.OrganizationID, req.PaymentType, 1004, invoiceNumber, now, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert payment order: %w", err)
	}

	// 7. Panggil Midtrans Snap API untuk generate snap_token
	snapReq := &midtrans.SnapReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: paymentAmount,
		},
	}

	snapResp, err := s.midtransConfig.Snap.GetToken(snapReq)
	if err != nil {
		fmt.Println("Midtrans error:", err)
		return nil, fmt.Errorf("midtrans error: %w", err)
	}

	return &model.PaymentResponse{
		SnapToken: snapResp.Token,
		OrderID:   req.OrderID,
	}, nil
}

// PaymentNotifications menangani notifikasi dari Midtrans
func (s *paymentService) PaymentNotifications(req *model.MidtransWebhookRequest) error {
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
	_, err := strconv.ParseFloat(req.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid gross amount format: %w", err)
	}

	// Ambil orderType dengan mengecek tabel
	orderType := int64(1)
	_, err = s.repo.GetOrderTotalAmount(req.OrderID, 1) // orderType 1 = fleet
	if err != nil {
		// Jika tidak ketemu di fleet_orders, coba di tour_package_orders
		_, err = s.repo.GetOrderTotalAmount(req.OrderID, 2) // orderType 2 = tour
		orderType = 2
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}
	}

	// Update status_payment menjadi 3 sesuai logic yang diminta
	status := 3
	err = s.repo.UpdatePaymentStatus(req.OrderID, orderType, status)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}
