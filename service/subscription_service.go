package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/config"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/veritrans/go-midtrans"
)

type SubscriptionService struct {
	subscriptionRepo *repository.SubscriptionRepository
	orgUserRepo      *repository.OrganizationUserRepository
	orgRepo          *repository.OrganizationRepository
	paymentRepo      *repository.PaymentRepository
	midtransConfig   *config.MidtransConfig
	packages         []model.Package
	once             sync.Once
	loadErr          error
}

func NewSubscriptionService(subscriptionRepo *repository.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *SubscriptionService) SetMidtransConfig(cfg *config.MidtransConfig) {
	s.midtransConfig = cfg
}

// SetOrganizationUserRepository sets the organization user repository
func (s *SubscriptionService) SetOrganizationUserRepository(orgUserRepo *repository.OrganizationUserRepository) {
	s.orgUserRepo = orgUserRepo
}

// SetOrganizationRepository sets the organization repository
func (s *SubscriptionService) SetOrganizationRepository(orgRepo *repository.OrganizationRepository) {
	s.orgRepo = orgRepo
}

// SetPaymentRepository sets the payment repository
func (s *SubscriptionService) SetPaymentRepository(paymentRepo *repository.PaymentRepository) {
	s.paymentRepo = paymentRepo
}

func (s *SubscriptionService) loadPackages() error {
	s.once.Do(func() {
		f, err := os.Open("config/packages.json")
		if err != nil {
			s.loadErr = err
			return
		}
		defer f.Close()

		var data struct {
			Packages []model.Package `json:"packages"`
		}
		if err := json.NewDecoder(f).Decode(&data); err != nil {
			s.loadErr = err
			return
		}
		s.packages = data.Packages
	})
	return s.loadErr
}

func (s *SubscriptionService) GetSubscription(orgID string) (model.SubscriptionDetail, error) {
	if err := s.loadPackages(); err != nil {
		return model.SubscriptionDetail{}, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to load packages")
	}

	subscriptions, err := s.subscriptionRepo.GetSubscriptionDetails(orgID)
	if err != nil {
		fmt.Println(err)
		return model.SubscriptionDetail{}, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch subscriptions")
	}
	sub := subscriptions[0]

	for _, pkg := range s.packages {
		fmt.Println("pkg packageID ", pkg.PackageID)
		encryptedID, _ := helper.EncryptString(pkg.PackageID)
		if pkg.PackageID == sub.PackageID {
			sub.PackageID = encryptedID
			sub.PackageName = pkg.PackageName
			break
		}
	}

	today := time.Now()
	if sub.ExpireDate.Before(today) {
		sub.Status = "Berakhir"
	} else {
		sub.Status = "Aktif"
	}

	return sub, nil
}

func (s *SubscriptionService) GetSubscriptionHistory(userID, orgID string) ([]model.SubscriptionHistory, error) {
	if err := s.loadPackages(); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to load packages")
	}

	subscriptions, err := s.subscriptionRepo.GetSubscriptionHistory(userID, orgID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch subscription history")
	}

	// Process each subscription
	for i := range subscriptions {
		sub := &subscriptions[i]

		// Find package name
		for _, pkg := range s.packages {
			if pkg.PackageID == sub.PackageID {
				sub.PackageName = pkg.PackageName
				break
			}
		}

		// Format dates as YYYY-MM-DD
		sub.StartDateFormatted = sub.StartDate.Format("2006-01-02")
		sub.ExpiryDateFormatted = sub.ExpiryDate.Format("2006-01-02")
	}

	return subscriptions, nil
}

func (s *SubscriptionService) SubmitSubscriptionPayment(packageID, userID, orgID string) (*model.PaymentResponse, error) {
	if err := s.loadPackages(); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to load packages")
	}

	// Find the package from packages.json
	var selectedPackage *model.Package
	for _, pkg := range s.packages {
		if pkg.PackageID == packageID {
			selectedPackage = &pkg
			break
		}
	}
	if selectedPackage == nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "package not found")
	}

	// Get current subscription to compare
	currentSubs, err := s.subscriptionRepo.GetSubscriptionDetails(orgID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get current subscription")
	}

	// Determine package amount
	var packageAmount int
	if len(currentSubs) > 0 && currentSubs[0].PackageID == packageID {
		// Same package, use package price
		packageAmount = selectedPackage.PackagePrice
	} else if len(currentSubs) > 0 {
		// Different package, check if new price is higher
		if selectedPackage.PackagePrice > int(currentSubs[0].PackagePrice) {
			packageAmount = selectedPackage.PackagePrice - int(currentSubs[0].PackagePrice)
		} else {
			// New package is cheaper or same, use new package price
			packageAmount = selectedPackage.PackagePrice
		}
	} else {
		// No current subscription, use package price
		packageAmount = selectedPackage.PackagePrice
	}

	// Generate invoice number
	invoiceNumber, err := s.subscriptionRepo.GenerateSubsInvoiceID()
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate invoice number")
	}

	// Generate transaction ID
	transactionID := uuid.New().String()

	// Prepare dates
	now := time.Now()
	transactionDate := now.Format("2006-01-02 15:04:05")
	startDate := transactionDate
	expiryTime := now.AddDate(0, 0, selectedPackage.PackageDuration)
	// Set time to 23:59:59
	expiryTime = time.Date(expiryTime.Year(), expiryTime.Month(), expiryTime.Day(), 23, 59, 59, 0, expiryTime.Location())
	expiryDate := expiryTime.Format("2006-01-02 15:04:05")
	createdAt := transactionDate
	createdBy := userID
	status := 2

	// Insert into travego_transactions
	err = s.subscriptionRepo.InsertTravegoTransaction(
		transactionID,
		transactionDate,
		invoiceNumber,
		packageID,
		startDate,
		expiryDate,
		status,
		userID,
		orgID,
		createdAt,
		createdBy,
	)
	if err != nil {
		fmt.Println(err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to insert transaction")
	}

	// Get Midtrans snap token
	baseURL := os.Getenv("APP_BASE_URL")
	finishURL := fmt.Sprintf("%s/dashboard/partner/subscription/payment/success/%s", baseURL, invoiceNumber)
	errorURL := fmt.Sprintf("%s/dashboard/partner/subscription/payment/error/%s", baseURL, invoiceNumber)
	snapReq := &midtrans.SnapReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  invoiceNumber,
			GrossAmt: int64(packageAmount),
		},
		Callbacks: &midtrans.Callbacks{
			Finish: finishURL,
			Error:  errorURL,
		},
	}

	snapResp, err := s.midtransConfig.Snap.GetToken(snapReq)
	if err != nil {
		fmt.Println("Midtrans error:", err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "midtrans error")
	}

	return &model.PaymentResponse{
		SnapToken: snapResp.Token,
		OrderID:   invoiceNumber,
	}, nil
}

func (s *SubscriptionService) GetSubscriptionSummary(packageID, orgID string) (*model.SubmitSubscriptionResponse, error) {
	if err := s.loadPackages(); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to load packages")
	}

	// Find the package from packages.json
	var selectedPackage *model.Package
	for _, pkg := range s.packages {
		if pkg.PackageID == packageID {
			selectedPackage = &pkg
			break
		}
	}
	if selectedPackage == nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "package not found")
	}

	// Get current subscription to compare
	currentSubs, err := s.subscriptionRepo.GetSubscriptionDetails(orgID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get current subscription")
	}

	// Determine package amount
	var packageAmount int
	var currentPackagePrice float64 = 0
	if len(currentSubs) > 0 {
		// Current subscription exists, calculate difference
		currentPackagePrice = currentSubs[0].PackagePrice
		if selectedPackage.PackagePrice > int(currentSubs[0].PackagePrice) {
			packageAmount = selectedPackage.PackagePrice - int(currentSubs[0].PackagePrice)
		} else {
			// New package is cheaper or same, use full price
			packageAmount = selectedPackage.PackagePrice
		}
	} else {
		// No current subscription, use full package price
		packageAmount = selectedPackage.PackagePrice
	}

	// Encrypt package ID
	encryptedPackageID, err := helper.EncryptString(packageID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to encrypt package ID")
	}

	// Calculate discount price
	discountPrice := selectedPackage.OriginalPrice - selectedPackage.PackagePrice - packageAmount

	// Create response
	response := &model.SubmitSubscriptionResponse{
		PackageID:           encryptedPackageID,
		PackageName:         selectedPackage.PackageName,
		PackageDuration:     selectedPackage.PackageDuration,
		PackageDescription:  selectedPackage.PackageDescription,
		Features:            selectedPackage.Features,
		PaymentAmount:       packageAmount,
		PackagePrice:        selectedPackage.PackagePrice,
		OriginalPrice:       selectedPackage.OriginalPrice,
		CurrentPackagePrice: currentPackagePrice,
		DiscountPrice:       discountPrice,
	}

	return response, nil
}

func (s *SubscriptionService) GetSubscriptionDetailByInvoice(invoiceNumber string) (*model.SubscriptionDetailByInvoiceResponse, error) {
	if err := s.loadPackages(); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to load packages")
	}

	if s.paymentRepo == nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "payment repository not initialized")
	}

	_, packageID, startDate, expiryDate, _, _, paymentMethod, createdAt, paymentAmount, err := (*s.paymentRepo).GetSubscriptionDetail(invoiceNumber)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "subscription detail not found")
	}

	var selectedPackage *model.Package
	for _, pkg := range s.packages {
		if pkg.PackageID == packageID {
			selectedPackage = &pkg
			break
		}
	}
	if selectedPackage == nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "package not found")
	}

	var pm string
	if paymentMethod.Valid {
		pm = paymentMethod.String
	}

	var pa float64
	if paymentAmount.Valid {
		pa = paymentAmount.Float64
	}

	return &model.SubscriptionDetailByInvoiceResponse{
		PackageID:       packageID,
		PackageName:     selectedPackage.PackageName,
		PackageDuration: selectedPackage.PackageDuration,
		StartDate:       startDate,
		ExpiryDate:      expiryDate,
		CreatedAt:       createdAt,
		PaymentMethod:   pm,
		PaymentAmount:   pa,
	}, nil
}
