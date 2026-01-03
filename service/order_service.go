package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderService struct {
	fleetRepo   *repository.FleetRepository
	contentRepo *repository.ContentRepository
	orgRepo     *repository.OrganizationRepository
	emailCfg    *configs.EmailConfig
	citiesName  map[string]string
}

func NewOrderService(fleetRepo *repository.FleetRepository, contentRepo *repository.ContentRepository, orgRepo *repository.OrganizationRepository, emailCfg *configs.EmailConfig) *OrderService {
	return &OrderService{
		fleetRepo:   fleetRepo,
		contentRepo: contentRepo,
		orgRepo:     orgRepo,
		emailCfg:    emailCfg,
	}
}

func (s *OrderService) CreateOrder(req *model.CreateOrderRequest) (*model.CreateOrderResponse, error) {
	// 1. Calculate Total Amount
	// Get Fleet Price
	price, rentType, err := s.fleetRepo.GetPriceByID(req.PriceID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "price not found")
	}

	// Logic for destinations based on rent_type (1=Citytour, 3=Citytour Drop / Pickup Only)
	// If rent_type is 1 or 3, force destination city_id to match pickup_city_id
	if rentType == 1 || rentType == 3 {
		for i := range req.Destinations {
			req.Destinations[i].CityID = req.PickupCityID
		}
	}

	// Get Addon Price Sum
	addonTotal, err := s.fleetRepo.GetAddonPriceSum(req.Addons)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to calc addons")
	}

	// Formula: unit_qty * (price + total_addon_price)
	totalAmount := float64(req.Qty) * (price + addonTotal)

	// 2. Generate Order ID
	// {orgcode}{YYDDMMhh:mm}{count}-FRT

	if req.OrganizationID == "" {
		return nil, NewServiceError(ErrInternalServer, http.StatusBadRequest, "organization_id is missing")
	}

	// Get Order Count
	count, err := s.fleetRepo.GetOrderCountByOrgID(req.OrganizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, fmt.Sprintf("failed to get order count: %v", err))
	}

	// Prepare Org Code Part (3 first chars + 2 last digits)
	orgCode := req.OrganizationCode
	var truncatedCode string
	if len(orgCode) >= 5 {
		truncatedCode = orgCode[:3] + orgCode[len(orgCode)-2:]
	} else {
		truncatedCode = orgCode
	}

	now := time.Now()
	// Format: YYDDMMhh -> 06020115
	timePart := now.Format("06020115")

	orderID := fmt.Sprintf("%s%s%d-FRT", truncatedCode, timePart, count+1)

	// 3. Save to DB
	err = s.fleetRepo.CreateFleetOrder(orderID, totalAmount, req)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create order")
	}

	// Generate Token
	tokenPayload := model.OrderTokenPayload{
		OrderID: orderID,
		PriceID: req.PriceID,
	}
	tokenBytes, _ := json.Marshal(tokenPayload)
	token, err := helper.EncryptString(string(tokenBytes))
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to encrypt token")
	}

	// 4. Send Email Notification
	// Fetch fleet details for email
	fleetSummary, err := s.fleetRepo.GetFleetOrderSummary(req.FleetID, req.PriceID)
	if err != nil {
		// Log error but don't fail the order
		log.Printf("[WARN] Failed to get fleet summary for email: %v", err)
	} else {
		// Construct email data
		facilities := strings.Join(fleetSummary.Facilities, ", ")

		destinations := make([]string, len(req.Destinations))
		for i, d := range req.Destinations {
			destinations[i] = d.Location
		}
		destStr := strings.Join(destinations, ", ")

		// Fetch Organization Specific Content
		var orgLogo, brandName, companyName string

		// Logo
		if content, err := s.contentRepo.FindByTagAndOrgID("brand-logo", req.OrganizationID); err == nil && content != nil {
			orgLogo = content.Content
		}

		// Brand Name
		if content, err := s.contentRepo.FindByTagAndOrgID("brand-name", req.OrganizationID); err == nil && content != nil {
			brandName = content.Content
		}

		// Company Name
		if content, err := s.contentRepo.FindByTagAndOrgID("company-name", req.OrganizationID); err == nil && content != nil {
			companyName = content.Content
		}

		// Contact List
		contactList, _ := s.contentRepo.GetContentListByTag("contact", req.OrganizationID)

		// Fetch Domain URL
		domainURL, err := s.orgRepo.GetDomainURL(req.OrganizationID)
		if err != nil {
			log.Printf("[WARN] Failed to get domain url for org %s: %v", req.OrganizationID, err)
		}

		baseURL := "http://localhost:5174" // Default fallback
		if domainURL != "" {
			baseURL = domainURL
		}
		baseURL = strings.TrimSuffix(baseURL, "/")

		// Generate Order Detail URL
		orderDetailUrl := fmt.Sprintf("%s/order/detail/armada/%s", baseURL, token)

		emailData := helper.OrderSuccessEmailData{
			CustomerName:     req.Fullname,
			OrderID:          orderID,
			FleetName:        fleetSummary.FleetName,
			Duration:         fmt.Sprintf("%d %s", fleetSummary.Duration, fleetSummary.Uom),
			Facilities:       facilities,
			PickupLocation:   req.PickupLocation,
			Destination:      destStr,
			TotalPrice:       helper.FormatRupiah(totalAmount),
			OrganizationLogo: orgLogo,
			BrandName:        brandName,
			CompanyName:      companyName,
			ContactList:      contactList,
			OrderDetailUrl:   orderDetailUrl,
		}

		// Send email asynchronously
		go func() {
			if err := helper.SendOrderSuccessEmail(s.emailCfg, req.Email, emailData); err != nil {
				log.Printf("[ERROR] Failed to send order success email to %s: %v", req.Email, err)
			}
		}()
	}

	return &model.CreateOrderResponse{
		Token: token,
	}, nil
}

func (s *OrderService) GetPaymentMethods(organizationID string) (*model.PaymentMethodGroupedResponse, error) {
	return s.orgRepo.GetPaymentMethods(organizationID)
}

func (s *OrderService) GetFleetOrderSummary(req *model.OrderFleetSummaryRequest) (*model.OrderFleetSummaryResponse, error) {
	res, err := s.fleetRepo.GetFleetOrderSummary(req.FleetID, req.PriceID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet or price not found")
	}

	// Map RentType to RentTypeLabel
	switch res.RentType {
	case 1:
		res.RentTypeLabel = "Citytour"
	case 2:
		res.RentTypeLabel = "Overland"
	case 3:
		res.RentTypeLabel = "Citytour Drop / Pickup Only"
	}

	s.ensureCitiesLoaded()
	for i := range res.PickupPoints {
		key := strconv.Itoa(res.PickupPoints[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.PickupPoints[i].CityName = name
		}
	}

	return res, nil
}

func (s *OrderService) ensureCitiesLoaded() {
	if s.citiesName != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		s.citiesName = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		s.citiesName = map[string]string{}
		return
	}
	m := make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		m[c.ID] = c.Name
	}
	s.citiesName = m
}

func (s *OrderService) GetOrderList(req *model.GetOrderListRequest) (*model.GetOrderListResponse, error) {
	data, total, err := s.fleetRepo.GetOrderList(req)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch orders")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	totalPage := (total + limit - 1) / limit
	currentPage := req.Page
	if currentPage <= 0 {
		currentPage = 1
	}

	return &model.GetOrderListResponse{
		Data:        data,
		TotalData:   total,
		TotalPage:   totalPage,
		CurrentPage: currentPage,
	}, nil
}

func (s *OrderService) GetOrderDetail(encryptedOrderID, organizationID string) (*model.OrderDetailResponse, error) {
	// Decrypt Order ID
	decrypted, err := helper.DecryptString(encryptedOrderID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusBadRequest, "invalid order id")
	}

	var orderID, priceID string
	var payload model.OrderTokenPayload

	// Try to parse as JSON object {order_id, price_id}
	if err := json.Unmarshal([]byte(decrypted), &payload); err == nil && payload.OrderID != "" {
		orderID = payload.OrderID
		priceID = payload.PriceID
	} else {
		// Fallback: assume the decrypted string is the orderID itself
		orderID = decrypted
	}

	res, err := s.fleetRepo.GetOrderDetail(orderID, priceID, organizationID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusBadRequest, "order not found")
	}

	// Map RentType
	switch res.RentType {
	case 1:
		res.RentTypeLabel = "Citytour"
	case 2:
		res.RentTypeLabel = "Overland"
	case 3:
		res.RentTypeLabel = "Citytour Drop / Pickup Only"
	}

	// Map Cities
	s.ensureCitiesLoaded()

	// Pickup City
	if name, ok := s.citiesName[res.Pickup.PickupCity]; ok {
		res.Pickup.PickupCity = name
	}

	// Destination Cities
	for i := range res.Destination {
		if name, ok := s.citiesName[res.Destination[i].City]; ok {
			res.Destination[i].City = name
		}
	}

	return res, nil
}

func (s *OrderService) CreateOrderPayment(req *model.CreatePaymentRequest) (*model.FleetOrderPayment, error) {
	// 1. Decrypt Token
	decrypted, err := helper.DecryptString(req.Token)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusBadRequest, "invalid payment token")
	}

	// Try to unmarshal as token payload
	var orderID, priceID string
	var payload model.OrderTokenPayload
	if err := json.Unmarshal([]byte(decrypted), &payload); err == nil && payload.OrderID != "" {
		orderID = payload.OrderID
		priceID = payload.PriceID
	} else {
		// Fallback: assume it's just the orderID string
		orderID = decrypted
	}

	// 2. Get Order Total Amount
	totalAmount, err := s.fleetRepo.GetFleetOrderTotalAmount(orderID, priceID, req.OrganizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Error: order not found or invalid organization:", err)
			return nil, NewServiceError(ErrNotFound, http.StatusBadRequest, "order not found or invalid organization")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order amount")
	}

	// 3. Calculate Amounts
	var paymentAmount, remaining float64

	if req.PaymentType == 1 { // Full Payment
		paymentAmount = totalAmount
		remaining = 0
	} else if req.PaymentType == 2 { // Partial/Down Payment
		paymentAmount = (req.PaymentPercentage / 100) * totalAmount
		remaining = totalAmount - paymentAmount
	} else {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid payment type")
	}

	// Map Payment Method
	var paymentMethod model.PaymentMethod
	switch strings.ToLower(req.PaymentMethod) {
	case "bank":
		paymentMethod = model.PaymentMethodBank
	case "qris":
		paymentMethod = model.PaymentMethodQris
	default:
		// Try parsing as int string "1", "2"
		if val, err := strconv.Atoi(req.PaymentMethod); err == nil {
			paymentMethod = model.PaymentMethod(val)
		} else {
			return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid payment method")
		}
	}

	// 4. Create Payment Object
	payment := &model.FleetOrderPayment{
		OrderPaymentID:    uuid.New().String(),
		OrderID:           orderID,
		OrganizationID:    req.OrganizationID,
		PaymentMethod:     paymentMethod,
		PaymentType:       req.PaymentType,
		PaymentPercentage: req.PaymentPercentage,
		PaymentAmount:     paymentAmount,
		TotalAmount:       totalAmount,
		PaymentRemaining:  remaining,
		Status:            model.PaymentStatusPendingVerification, // Default 2
		CreatedAt:         time.Now(),
	}

	// 5. Save
	if err := s.fleetRepo.CreateOrderPayment(payment); err != nil {
		fmt.Println("Error: failed to create payment record:", err)
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create payment record")
	}

	return payment, nil
}
