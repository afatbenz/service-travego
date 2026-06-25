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
	"service-travego/internal/wagy"
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
	orgRepo        *repository.OrganizationRepository
	midtransConfig *config.MidtransConfig
}

// NewPaymentService membuat instance baru dari PaymentService
func NewPaymentService(repo repository.PaymentRepository, orgRepo *repository.OrganizationRepository, midtransConfig *config.MidtransConfig) PaymentService {
	return &paymentService{
		repo:           repo,
		orgRepo:        orgRepo,
		midtransConfig: midtransConfig,
	}
}

func (s *paymentService) ProcessPaymentNotification(req *model.MidtransWebhookRequest) error {
	if req.StatusCode != "200" {
		return nil
	}

	paymentMethod := determinePaymentMethod(req)

	// Check if it's a subscription order (starts with TRV)
	if strings.HasPrefix(req.OrderID, "TRV") {
		// Get subscription detail
		_, packageID, _, expiryDate, _, organizationID, _, _, _, err := s.repo.GetSubscriptionDetail(req.OrderID)
		if err != nil {
			return fmt.Errorf("failed to get subscription detail: %w", err)
		}

		grossAmount, err := strconv.ParseFloat(req.GrossAmount, 64)
		// Update travego_transactions
		if err := s.repo.UpdateTravegoTransactionStatus(req.OrderID, paymentMethod, grossAmount); err != nil {
			return fmt.Errorf("failed to update travego transaction: %w", err)
		}

		// Parse gross amount
		if err != nil {
			return fmt.Errorf("invalid gross amount: %w", err)
		}

		// Check if subscription exists
		subscriptionExists, err := s.repo.GetSubscriptionByOrganization(organizationID)
		if err != nil {
			return fmt.Errorf("failed to check subscription existence: %w", err)
		}

		activateDate := time.Now()
		if subscriptionExists {
			// Update existing subscription
			if err := s.repo.UpdateSubscription(organizationID, packageID, activateDate, expiryDate, grossAmount); err != nil {
				return fmt.Errorf("failed to update subscription: %w", err)
			}
		} else {
			// Insert new subscription
			subscriptionID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("failed to generate subscription ID: %w", err)
			}
			if err := s.repo.InsertSubscription(subscriptionID.String(), organizationID, packageID, activateDate, expiryDate, grossAmount, activateDate); err != nil {
				return fmt.Errorf("failed to insert subscription: %w", err)
			}
		}

		// Get organization name
		_, orgName, _, err := s.orgRepo.GetOrganizationEmailAndName(organizationID)
		if err != nil {
			fmt.Printf("warning: failed to get organization name: %v\n", err)
		}

		// Send WhatsApp notification
		go func() {
			waClient := wagy.NewWagyClient(os.Getenv("WAGY_DEVICE_ID"), os.Getenv("WAGY_TOKEN"))
			phone := os.Getenv("ADMINISTRATOR_PHONE")
			if phone == "" {
				fmt.Printf("warning: ADMINISTRATOR_PHONE environment variable not set\n")
				return
			}
			message := fmt.Sprintf(
				"[PAYMENT SUCCESS]\n"+
					"Organization: %s\n"+
					"Invoice: %s\n"+
					"Amount: Rp %s\n"+
					"Thank you!",
				orgName,
				req.OrderID,
				helper.FormatRupiah(grossAmount),
			)
			if _, err := waClient.SendMessage(phone, message); err != nil {
				fmt.Printf("warning: failed to send WhatsApp notification: %v\n", err)
			}
		}()

		return nil
	}

	orgID, _, orderTypeFromOrder, orderID, err := s.repo.GetOrderDetails(req.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order details: %w", err)
	}

	totalAmount, err := s.repo.GetOrderTotalAmount(orderID, orderTypeFromOrder)
	if err != nil {
		return fmt.Errorf("failed to get order total amount: %w", err)
	}

	grossAmount, err := strconv.ParseFloat(req.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("invalid gross amount: %w", err)
	}

	if err := s.repo.UpdatePaymentOrderNotification(req.OrderID, orgID, totalAmount, grossAmount, req.TransactionID, req.PaymentType); err != nil {
		return fmt.Errorf("failed to update payment order: %w", err)
	}

	totalPaid, err := s.repo.GetOrderTotalPaidAmount(orderID, orderTypeFromOrder, orgID)
	if err != nil {
		return fmt.Errorf("failed to get total paid amount: %w", err)
	}

	remainingAmount := float64(totalAmount) - totalPaid
	paymentStatus := 4
	if remainingAmount <= 0 {
		paymentStatus = 1
	}

	if err := s.repo.UpdateOrderPaymentStatus(orderID, orderTypeFromOrder, paymentStatus); err != nil {
		return fmt.Errorf("failed to update order payment status: %w", err)
	}

	createdAt := time.Now().Format("2006-01-02 15:04:05")
	if err := s.repo.InsertPaymentMidtrans(req, createdAt); err != nil {
		return fmt.Errorf("failed to insert payment midtrans: %w", err)
	}

	invoiceNumber, orderTypeFromPaymentOrder, paymentTypeFromPaymentOrder, paymentMethodFromPaymentOrder, createdBy, err := s.repo.GetPaymentOrderMeta(orderID, orgID)
	if err != nil {
		return fmt.Errorf("failed to get payment order meta: %w", err)
	}

	orderType := orderTypeFromPaymentOrder

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
	formattedPaymentDate := formatPaymentDate(transactionDate)

	transactionCategory := ""
	switch orderType {
	case 1:
		transactionCategory = "TRX01"
	case 2:
		transactionCategory = "TRX02"
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
		paymentTypeFromPaymentOrder,
		paymentMethodFromPaymentOrder,
		grossAmount,
		orgID,
		transactionCategory,
		time.Now(),
		createdBy,
		orderID,
	); err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	emailCfg := &configs.EmailConfig{
		From:     os.Getenv("EMAIL_FROM"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		SMTPHost: os.Getenv("EMAIL_SMTP_HOST"),
		SMTPPort: os.Getenv("EMAIL_SMTP_PORT"),
	}
	if configs.ValidateEmailConfig(emailCfg) == nil {
		baseURL := os.Getenv("APP_BASE_URL")
		baseURL = strings.TrimSuffix(baseURL, "/")

		tokenPayload := model.OrderTokenPayload{
			OrderID: req.OrderID,
			PriceID: "",
		}
		tokenBytes, _ := json.Marshal(tokenPayload)
		token, terr := helper.EncryptString(string(tokenBytes))
		orderDetailUrl := ""
		dashboardOrderDetailUrl := ""

		orgEmail, orgName, domainURL, oerr := s.orgRepo.GetOrganizationEmailAndName(orgID)
		dashboardOrderDetailUrl = fmt.Sprintf("%s/dashboard/partner/orders/fleet/detail/%s", baseURL, req.OrderID)
		if terr == nil && strings.TrimSpace(token) != "" && strings.TrimSpace(domainURL) != "" {
			orderDetailUrl = fmt.Sprintf("%s/order/detail/armada/%s", domainURL, token)
		}
		if oerr == nil && strings.TrimSpace(orgEmail) != "" {
			orgEmailData := helper.PaymentSuccessEmailData{
				OrganizationName:        orgName,
				TransactionID:           req.TransactionID,
				OrderID:                 orderID,
				PaymentMethod:           req.PaymentType,
				PaymentDate:             formattedPaymentDate,
				TotalPrice:              helper.FormatRupiah(grossAmount),
				DashboardOrderDetailUrl: dashboardOrderDetailUrl,
			}

			go func() {
				if err := helper.SendPaymentReceivedEmail(emailCfg, orgEmail, orgEmailData); err != nil {
					fmt.Println("failed to send payment received email to organization:", err)
				}
			}()
		}

		customerName, customerEmail, fleetName, pickupLocation, startDate, endDate, destination, ferr := s.repo.GetFleetOrderEmailData(req.OrderID, orgID)
		if ferr == nil && strings.TrimSpace(customerEmail) != "" {
			duration := ""
			if !startDate.IsZero() && !endDate.IsZero() {
				days := int(endDate.Sub(startDate).Hours()/24) + 1
				if days < 1 {
					days = 1
				}
				duration = fmt.Sprintf("%d hari", days)
			}

			customerEmailData := helper.PaymentSuccessEmailData{
				CustomerName:   customerName,
				TransactionID:  req.TransactionID,
				OrderID:        req.OrderID,
				PaymentMethod:  req.PaymentType,
				PaymentDate:    formattedPaymentDate,
				TotalPrice:     helper.FormatRupiah(grossAmount),
				FleetName:      fleetName,
				Duration:       duration,
				PickupLocation: pickupLocation,
				Destination:    destination,
				OrderDetailUrl: orderDetailUrl,
				ReviewUrl:      fmt.Sprintf("%s/order/review", domainURL),
			}

			go func() {
				if err := helper.SendPaymentSuccessEmail(emailCfg, customerEmail, customerEmailData); err != nil {
					fmt.Println("failed to send payment success email:", err)
				}
			}()
		}
	}

	return nil
}

func determinePaymentMethod(req *model.MidtransWebhookRequest) string {
	if len(req.VaNumbers) > 0 && req.VaNumbers[0].Bank != "" {
		return fmt.Sprintf("Midtrans - Virtual Account - %s", req.VaNumbers[0].Bank)
	}
	return fmt.Sprintf("Midtrans - %s", req.PaymentType)
}

func formatPaymentDate(t time.Time) string {
	months := [...]string{
		"Januari",
		"Februari",
		"Maret",
		"April",
		"Mei",
		"Juni",
		"Juli",
		"Agustus",
		"September",
		"Oktober",
		"November",
		"Desember",
	}

	local := t.Local()
	monthName := ""
	monthIdx := int(local.Month())
	if monthIdx >= 1 && monthIdx <= 12 {
		monthName = months[monthIdx-1]
	}

	return fmt.Sprintf("%02d %s %04d %02d:%02d", local.Day(), monthName, local.Year(), local.Hour(), local.Minute())
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
	fmt.Println("invoiceNumber:", invoiceNumber)

	// 6. Insert ke payment_orders
	paymentID := uuid.New().String()
	now := time.Now().Format("2006-01-02 15:04:05")

	err = s.repo.InsertPaymentOrder(paymentID, req.OrderType, req.OrderID, req.OrganizationID, req.PaymentType, 1004, invoiceNumber, now, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert payment order: %w", err)
	}

	// 7. Panggil Midtrans Snap API untuk generate snap_token
	snapReq := &midtrans.SnapReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  invoiceNumber,
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
		OrderID:   invoiceNumber,
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
