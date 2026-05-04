package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"
)

type TourPackageService struct {
	repo       *repository.TourPackageRepository
	citiesName map[string]string
}

func NewTourPackageService(repo *repository.TourPackageRepository) *TourPackageService {
	return &TourPackageService{
		repo: repo,
	}
}

func (s *TourPackageService) GetTourPackages(orgID string) ([]model.TourPackageListItem, error) {
	return s.repo.GetTourPackagesByOrgID(orgID)
}

func (s *TourPackageService) GetTourPackageOrderList(orgID string) ([]model.TourPackageOrderListItem, error) {
	return s.repo.ListTourPackageOrders(orgID)
}

func (s *TourPackageService) CreateTourPackageOrder(ctx context.Context, orgID, userID string, req *model.TourPackageOrderCreateRequest) (string, error) {
	if orgID == "" || userID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}
	if req == nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid payload")
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "customer_id is required")
	}
	if strings.TrimSpace(req.PackageID) == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "package_id is required")
	}
	if strings.TrimSpace(req.PriceID) == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "price_id is required")
	}

	startDate, err := normalizeTourPackageDateTime(req.StartDate)
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid start_date")
	}
	endDate, err := normalizeTourPackageDateTime(req.EndDate)
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid end_date")
	}

	customerExists, err := s.repo.CustomerExistsByOrgID(orgID, strings.TrimSpace(req.CustomerID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate customer")
	}
	if !customerExists {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "CUSTOMER_NOT_FOUND")
	}

	packageExists, err := s.repo.TourPackageExistsByOrgID(orgID, strings.TrimSpace(req.PackageID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate package")
	}
	if !packageExists {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "PACKAGE_NOT_FOUND")
	}

	price, priceOk, err := s.repo.GetTourPackagePriceByID(orgID, strings.TrimSpace(req.PriceID))
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate price")
	}
	if !priceOk {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "INVALID_PRICE_ID")
	}

	addonIDs := make([]string, 0, len(req.Addons))
	seen := map[string]struct{}{}
	for _, id := range req.Addons {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		addonIDs = append(addonIDs, id)
	}

	addonTotal := 0.0
	if len(addonIDs) > 0 {
		total, allExist, err := s.repo.GetTourPackageAddonTotalByIDs(orgID, addonIDs)
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate addons")
		}
		if !allExist {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "INVALID_ADDON_ID")
		}
		addonTotal = total
	}

	memberPax := req.MemberPax
	if memberPax < 0 {
		memberPax = 0
	}
	officialPax := req.OfficialPax
	if officialPax < 0 {
		officialPax = 0
	}
	totalPax := memberPax + officialPax

	discountAmount := req.DiscountAmount
	if discountAmount < 0 {
		discountAmount = 0
	}
	additionalAmount := req.AdditionalAmount
	if additionalAmount < 0 {
		additionalAmount = 0
	}

	totalAmount := price + additionalAmount + addonTotal - discountAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	orgCode, err := s.repo.GetOrganizationCodeByOrgID(orgID)
	if err != nil || strings.TrimSpace(orgCode) == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}

	count, err := s.repo.GetTourPackageOrderCountByOrgID(orgID)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order count")
	}

	truncatedCode := orgCode
	if len(orgCode) >= 5 {
		truncatedCode = orgCode[:3] + orgCode[len(orgCode)-2:]
	}
	timePart := time.Now().Format("06020115")
	orderID := fmt.Sprintf("%s%s%d-PCK", truncatedCode, timePart, count+1)

	if err := s.repo.CreateTourPackageOrder(ctx, repository.CreateTourPackageOrderInput{
		OrderID:          orderID,
		OrganizationID:   orgID,
		UserID:           userID,
		TourPackageID:    strings.TrimSpace(req.PackageID),
		CustomerID:       strings.TrimSpace(req.CustomerID),
		StartDate:        startDate,
		EndDate:          endDate,
		PickupAddress:    strings.TrimSpace(req.PickupAddress),
		PickupCityID:     strings.TrimSpace(req.PickupCityID),
		DiscountAmount:   discountAmount,
		AdditionalAmount: additionalAmount,
		OfficialPax:      officialPax,
		MemberPax:        memberPax,
		TotalPax:         totalPax,
		TotalAmount:      totalAmount,
		AddonIDs:         addonIDs,
	}); err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create order")
	}
	return orderID, nil
}

func (s *TourPackageService) UpdateTourPackageOrder(ctx context.Context, orgID, userID string, req *model.TourPackageOrderUpdateRequest) error {
	if orgID == "" || userID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}
	if req == nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid payload")
	}
	if strings.TrimSpace(req.OrderID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "customer_id is required")
	}
	if strings.TrimSpace(req.PackageID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "package_id is required")
	}
	if strings.TrimSpace(req.PriceID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "price_id is required")
	}

	startDate, err := normalizeTourPackageDateTime(req.StartDate)
	if err != nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid start_date")
	}
	endDate, err := normalizeTourPackageDateTime(req.EndDate)
	if err != nil {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid end_date")
	}

	orderExists, err := s.repo.TourPackageOrderExistsByOrgID(orgID, strings.TrimSpace(req.OrderID))
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate order")
	}
	if !orderExists {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "ORDER_NOT_FOUND")
	}

	customerExists, err := s.repo.CustomerExistsByOrgID(orgID, strings.TrimSpace(req.CustomerID))
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate customer")
	}
	if !customerExists {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "CUSTOMER_NOT_FOUND")
	}

	packageExists, err := s.repo.TourPackageExistsByOrgID(orgID, strings.TrimSpace(req.PackageID))
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate package")
	}
	if !packageExists {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "PACKAGE_NOT_FOUND")
	}

	price, priceOk, err := s.repo.GetTourPackagePriceByID(orgID, strings.TrimSpace(req.PriceID))
	if err != nil {
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate price")
	}
	if !priceOk {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "INVALID_PRICE_ID")
	}

	addonIDs := make([]string, 0, len(req.Addons))
	seen := map[string]struct{}{}
	for _, id := range req.Addons {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		addonIDs = append(addonIDs, id)
	}

	addonTotal := 0.0
	if len(addonIDs) > 0 {
		total, allExist, err := s.repo.GetTourPackageAddonTotalByIDs(orgID, addonIDs)
		if err != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to validate addons")
		}
		if !allExist {
			return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "INVALID_ADDON_ID")
		}
		addonTotal = total
	}

	memberPax := req.MemberPax
	if memberPax < 0 {
		memberPax = 0
	}
	officialPax := req.OfficialPax
	if officialPax < 0 {
		officialPax = 0
	}
	totalPax := memberPax + officialPax

	discountAmount := req.DiscountAmount
	if discountAmount < 0 {
		discountAmount = 0
	}
	additionalAmount := req.AdditionalAmount
	if additionalAmount < 0 {
		additionalAmount = 0
	}

	totalAmount := price + additionalAmount + addonTotal - discountAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	return s.repo.UpdateTourPackageOrder(ctx, repository.UpdateTourPackageOrderInput{
		OrderID:          strings.TrimSpace(req.OrderID),
		OrganizationID:   orgID,
		UserID:           userID,
		TourPackageID:    strings.TrimSpace(req.PackageID),
		CustomerID:       strings.TrimSpace(req.CustomerID),
		StartDate:        startDate,
		EndDate:          endDate,
		PickupAddress:    strings.TrimSpace(req.PickupAddress),
		PickupCityID:     strings.TrimSpace(req.PickupCityID),
		DiscountAmount:   discountAmount,
		AdditionalAmount: additionalAmount,
		OfficialPax:      officialPax,
		MemberPax:        memberPax,
		TotalPax:         totalPax,
		TotalAmount:      totalAmount,
		AddonIDs:         addonIDs,
	})
}

func (s *TourPackageService) GetTourPackageOrderDetail(ctx context.Context, orgID, orderID string) (map[string]interface{}, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}

	order, addons, err := s.repo.GetTourPackageOrderDetail(ctx, orgID, orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "ORDER_NOT_FOUND")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order detail")
	}

	customerID := ""
	if v, ok := order["customer_id"]; ok && v != nil {
		switch vv := v.(type) {
		case string:
			customerID = strings.TrimSpace(vv)
		case []byte:
			customerID = strings.TrimSpace(string(vv))
		default:
			customerID = strings.TrimSpace(fmt.Sprintf("%v", vv))
		}
	}
	customer := map[string]interface{}{}
	if customerID != "" {
		cust, err := s.repo.GetCustomerInfoByOrgID(ctx, orgID, customerID)
		if err == nil && cust != nil {
			customer = cust
		}
	}

	s.ensureCitiesLoaded()
	if len(customer) > 0 {
		custCityID := ""
		if v, ok := customer["customer_city"]; ok && v != nil {
			switch vv := v.(type) {
			case string:
				custCityID = strings.TrimSpace(vv)
			case []byte:
				custCityID = strings.TrimSpace(string(vv))
			default:
				custCityID = strings.TrimSpace(fmt.Sprintf("%v", vv))
			}
		}
		if custCityID != "" {
			if name, ok := s.citiesName[custCityID]; ok {
				customer["customer_city"] = name
			} else {
				customer["customer_city"] = ""
			}
		} else {
			customer["customer_city"] = ""
		}
	}
	cityKey := ""
	if v, ok := order["pickup_city_id"]; ok && v != nil {
		switch vv := v.(type) {
		case string:
			cityKey = strings.TrimSpace(vv)
		case []byte:
			cityKey = strings.TrimSpace(string(vv))
		default:
			cityKey = strings.TrimSpace(fmt.Sprintf("%v", vv))
		}
	}
	if cityKey != "" {
		if name, ok := s.citiesName[cityKey]; ok {
			order["pickup_city_label"] = name
		} else {
			order["pickup_city_label"] = ""
		}
	} else {
		order["pickup_city_label"] = ""
	}

	getString := func(m map[string]interface{}, key string) string {
		v, ok := m[key]
		if !ok || v == nil {
			return ""
		}
		switch vv := v.(type) {
		case string:
			return strings.TrimSpace(vv)
		case []byte:
			return strings.TrimSpace(string(vv))
		default:
			return strings.TrimSpace(fmt.Sprintf("%v", vv))
		}
	}
	getFloat64 := func(m map[string]interface{}, key string) float64 {
		v, ok := m[key]
		if !ok || v == nil {
			return 0
		}
		switch vv := v.(type) {
		case float64:
			return vv
		case float32:
			return float64(vv)
		case int:
			return float64(vv)
		case int64:
			return float64(vv)
		case uint64:
			return float64(vv)
		case string:
			s := strings.TrimSpace(vv)
			if s == "" {
				return 0
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
			return 0
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", vv))
			if s == "" {
				return 0
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
			return 0
		}
	}
	getInt := func(m map[string]interface{}, key string) int {
		v, ok := m[key]
		if !ok || v == nil {
			return 0
		}
		switch vv := v.(type) {
		case int:
			return vv
		case int64:
			return int(vv)
		case float64:
			return int(vv)
		case string:
			s := strings.TrimSpace(vv)
			if s == "" {
				return 0
			}
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return int(f)
			}
			return 0
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", vv))
			if s == "" {
				return 0
			}
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return int(f)
			}
			return 0
		}
	}

	orderOut := map[string]interface{}{
		"order_number_id": getString(order, "uuid"),
		"order_id":        getString(order, "order_id"),
		"tour_package_id": getString(order, "tour_package_id"),
		"pickup_address":  getString(order, "pickup_address"),
		"pickup_city_id":  getString(order, "pickup_city_id"),
		"pickup_city_label": func() string {
			if v, ok := order["pickup_city_label"]; ok && v != nil {
				switch vv := v.(type) {
				case string:
					return vv
				case []byte:
					return string(vv)
				default:
					return fmt.Sprintf("%v", vv)
				}
			}
			return ""
		}(),
		"discount_amount":   getFloat64(order, "discount_amount"),
		"additional_amount": getFloat64(order, "additional_amount"),
		"official_pax":      getInt(order, "official_pax"),
		"member_pax":        getInt(order, "member_pax"),
		"total_pax":         getInt(order, "total_pax"),
		"total_amount":      getFloat64(order, "total_amount"),
		"start_date":        order["start_date"],
		"end_date":          order["end_date"],
		"status":            getInt(order, "status"),
		"payment_status":    getInt(order, "payment_status"),
		"created_at":        order["created_at"],
		"created_by":        getString(order, "created_by"),
	}

	return map[string]interface{}{
		"order":     orderOut,
		"customers": customer,
		"addons":    addons,
	}, nil
}

func (s *TourPackageService) CreateTourPackage(ctx context.Context, req *model.CreateTourPackageRequest, orgID, userID string) error {
	packageID := helper.GenerateUUID()
	return s.repo.CreateTourPackage(ctx, req, packageID, orgID, userID)
}

func (s *TourPackageService) UpdateTourPackage(ctx context.Context, req *model.UpdateTourPackageRequest, orgID, userID string) error {
	return s.repo.UpdateTourPackage(ctx, req, orgID, userID)
}

func (s *TourPackageService) DeleteTourPackage(ctx context.Context, orgID, userID, packageID string) error {
	return s.repo.SoftDeleteTourPackage(ctx, orgID, userID, packageID)
}

func (s *TourPackageService) GetTourPackageDetail(ctx context.Context, orgID, packageID string) (*model.TourPackageDetailResponse, error) {
	res, err := s.repo.GetTourPackageDetail(ctx, orgID, packageID)
	if err != nil {
		return nil, err
	}

	switch res.Meta.PackageType {
	case 1:
		res.Meta.PackageTypeLabel = "Private Trip"
	case 2:
		res.Meta.PackageTypeLabel = "Open Trip"
	default:
		res.Meta.PackageTypeLabel = "Unknown"
	}

	s.ensureCitiesLoaded()
	for i := range res.PickupAreas {
		key := strconv.Itoa(res.PickupAreas[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.PickupAreas[i].CityName = name
		}
	}
	for i := range res.Itineraries {
		key := strconv.Itoa(res.Itineraries[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			res.Itineraries[i].CityName = name
		}
	}

	return res, nil
}

func (s *TourPackageService) ensureCitiesLoaded() {
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

func normalizeTourPackageDateTime(v string) (string, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02 15:04:05"), nil
	}
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t.Format("2006-01-02 15:04:05"), nil
	}
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return t.Format("2006-01-02 15:04:05"), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.Format("2006-01-02 15:04:05"), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02 15:04:05"), nil
	}
	return "", fmt.Errorf("invalid datetime")
}
