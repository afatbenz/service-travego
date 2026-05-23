package service

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"service-travego/config"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/veritrans/go-midtrans"
)

// PaymentService adalah interface untuk logika bisnis payment
type PaymentService interface {
	CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error)
	PaymentNotifications(req *model.MidtransWebhookRequest) error
	UpdatePaymentStatus(orderID string, orderType int64, status int, paymentStatus int) error
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

func (s *paymentService) ProcessPaymentNotification(req *model.MidtransWebhookRequest) error {
	if req.StatusCode != "200" {
		return nil
	}

	orgID, totalAmount, orderTypeFromOrder, err := s.repo.GetOrderDetails(req.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order details: %w", err)
	}

	grossAmount, err := strconv.ParseFloat(req.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid gross amount: %w", err)
	}

	if err := s.repo.UpdatePaymentOrderNotification(req.OrderID, orgID, totalAmount, grossAmount, req.TransactionID, req.PaymentType); err != nil {
		return fmt.Errorf("failed to update payment order: %w", err)
	}

	remaining, err := s.repo.GetLatestPaymentOrderRemainingAmount(req.OrderID, orgID, orderTypeFromOrder)
	if err != nil {
		return fmt.Errorf("failed to get remaining amount: %w", err)
	}

	paymentStatus := 4
	if !remaining.Valid || remaining.Float64 <= 0 {
		paymentStatus = 1
	}

	if err := s.UpdatePaymentStatus(req.OrderID, orderTypeFromOrder, 1, paymentStatus); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	createdAt := time.Now().Format("2006-01-02 15:04:05")
	if err := s.repo.InsertPaymentMidtrans(req, createdAt); err != nil {
		return fmt.Errorf("failed to insert payment midtrans: %w", err)
	}

	invoiceNumber, orderTypeFromPaymentOrder, createdBy, err := s.repo.GetPaymentOrderMeta(req.OrderID, orgID)
	if err != nil {
		return fmt.Errorf("failed to get payment order meta: %w", err)
	}

	orderType := orderTypeFromOrderID(req.OrderID)
	if orderType == 0 {
		orderType = orderTypeFromPaymentOrder
	}
	if orderType == 0 {
		orderType = orderTypeFromOrder
	}

	if invoiceNumber != "" {
		exists, err := s.repo.TransactionExistsByInvoice(orgID, invoiceNumber)
		if err != nil {
			return fmt.Errorf("failed to check existing transaction: %w", err)
		}
		if exists {
			return nil
		}
	}

	transactionDate := parseMidtransTransactionTime(req.TransactionTime)

	transactionType := int(model.TransactionTypeIncomeOtherIncome)
	if orderType == 1 {
		transactionType = int(model.TransactionTypeIncomeRental)
	} else if orderType == 2 {
		transactionType = int(model.TransactionTypeIncomeTourPackage)
	}

	transactionID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate transaction_id: %w", err)
	}

	if err := s.repo.InsertTransactionMidtrans(
		transactionID.String(),
		orderType,
		invoiceNumber,
		"Midtrans - Order ID "+req.OrderID,
		transactionDate,
		1004,
		grossAmount,
		orgID,
		transactionType,
		int(model.TransactionMarkIncome),
		time.Now(),
		createdBy,
	); err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	customerName, customerEmail, fleetName, pickupLocation, startDate, endDate, destination, err := s.repo.GetFleetOrderEmailData(req.OrderID, orgID)
	if err == nil && strings.TrimSpace(customerEmail) != "" {
		tokenPayload := model.OrderTokenPayload{
			OrderID: req.OrderID,
			PriceID: "",
		}
		tokenBytes, _ := json.Marshal(tokenPayload)
		token, terr := helper.EncryptString(string(tokenBytes))
		if terr == nil && strings.TrimSpace(token) != "" {
			emailCfg := &configs.EmailConfig{
				From:     os.Getenv("EMAIL_FROM"),
				Password: os.Getenv("EMAIL_PASSWORD"),
				SMTPHost: os.Getenv("EMAIL_SMTP_HOST"),
				SMTPPort: os.Getenv("EMAIL_SMTP_PORT"),
			}
			if configs.ValidateEmailConfig(emailCfg) == nil {
				baseURL := "http://localhost:5174"
				baseURL = strings.TrimSuffix(baseURL, "/")

				duration := ""
				if !startDate.IsZero() && !endDate.IsZero() {
					days := int(endDate.Sub(startDate).Hours()/24) + 1
					if days < 1 {
						days = 1
					}
					duration = fmt.Sprintf("%d hari", days)
				}

				emailData := helper.PaymentSuccessEmailData{
					CustomerName:   customerName,
					TransactionID:  req.TransactionID,
					OrderID:        req.OrderID,
					PaymentMethod:  req.PaymentType,
					PaymentDate:    req.TransactionTime,
					TotalPrice:     helper.FormatRupiah(grossAmount),
					FleetName:      fleetName,
					Duration:       duration,
					PickupLocation: pickupLocation,
					Destination:    destination,
					OrderDetailUrl: fmt.Sprintf("%s/order/detail/armada/%s", baseURL, token),
					ReviewUrl:      fmt.Sprintf("%s/order/review", baseURL),
				}

				go func() {
					if err := helper.SendPaymentSuccessEmail(emailCfg, customerEmail, emailData); err != nil {
						fmt.Println("failed to send payment success email:", err)
					}
				}()
			}
		}
	}

	return nil
}

func orderTypeFromOrderID(orderID string) int64 {
	id := strings.TrimSpace(orderID)
	if id == "" {
		return 0
	}
	id = strings.ToUpper(id)

	prefix := id
	if i := strings.Index(prefix, "-"); i > 0 {
		prefix = prefix[:i]
	}

	if strings.HasPrefix(prefix, "FO") {
		return 1
	}
	if strings.HasPrefix(prefix, "TO") {
		return 2
	}
	return 0
}

func parseMidtransTransactionTime(s string) time.Time {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return time.Now()
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	return time.Now()
}

func (s *paymentService) UpdatePaymentStatus(orderID string, orderType int64, status int, paymentStatus int) error {
	return s.repo.UpdateOrderStatus(orderID, orderType, status, paymentStatus)
}

func (s *paymentService) CreatePayment(req *model.PaymentRequest) (*model.PaymentResponse, error) {

	var paymentAmount int64

	if req.PaymentType == 1004 {
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
