package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"
)

type FleetService struct {
	repo                *repository.FleetRepository
	citiesName          map[string]string
	paymentMethodLabels map[int]string
}

func NewFleetService(repo *repository.FleetRepository) *FleetService {
	return &FleetService{repo: repo}
}

func (s *FleetService) CreateFleet(createdBy, organizationID string, req *model.CreateFleetRequest) (string, error) {
	if req.FleetName == "" || req.FleetType == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleet_name and fleet_type are required")
	}
	req.CreatedBy = createdBy
	req.OrganizationID = organizationID
	id, err := s.repo.CreateFleet(req)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet")
	}
	return id, nil
}

func (s *FleetService) UpdateFleet(updatedBy, organizationID string, req *model.UpdateFleetRequest) error {
	if req.FleetID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleet_id is required")
	}
	req.OrganizationID = organizationID
	req.UpdatedBy = updatedBy
	if err := s.repo.UpdateFleet(req); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update fleet")
	}
	return nil
}

func (s *FleetService) GetServiceFleets(page, perPage int) ([]model.ServiceFleetItem, error) {
	items, err := s.repo.GetServiceFleets(page, perPage)
	if err != nil {
		fmt.Println("Error fetching service fleets:", err)
		return nil, err
	}

	s.ensureCitiesLoaded()
	for i := range items {
		item := &items[i]
		item.Price = item.OriginalPrice // Default

		if item.DiscountType != nil && item.DiscountValue != nil {
			switch *item.DiscountType {
			case "PERCENT":
				// assuming discount_value is percentage e.g. 10 for 10%
				item.Price = item.OriginalPrice - (item.OriginalPrice * *item.DiscountValue / 100)
			case "AMOUNT":
				item.Price = item.OriginalPrice - *item.DiscountValue
			case "FLAT":
				item.Price = *item.DiscountValue
			}
		}

		// Convert City IDs to City Names
		var cityNames []string
		for _, cityID := range item.Cities {
			// item.Cities currently holds IDs as strings
			// Check if we need to convert to int for map lookup?
			// ensureCitiesLoaded uses map[string]string where key is ID string.
			// location.json likely has IDs as strings.
			// fleet_pickup has city_id as int. GROUP_CONCAT returns string "1,2,3".
			// strings.Split gives ["1", "2", "3"].
			// So key lookup should work directly.
			if name, ok := s.citiesName[cityID]; ok {
				cityNames = append(cityNames, name)
			} else {
				// Fallback to ID if name not found? Or skip? User asked for "list kota".
				// Let's include ID if name missing or maybe just ignore.
				// Better to include name if found.
				cityNames = append(cityNames, cityID)
			}
		}
		item.Cities = cityNames
	}
	return items, nil
}

func (s *FleetService) GetAvailableCities(orgID string) ([]model.ServiceFleetPickupItem, error) {
	cityIDs, err := s.repo.GetAvailableCities(orgID)
	if err != nil {
		return nil, err
	}

	s.ensureCitiesLoaded()

	var cities []model.ServiceFleetPickupItem
	for _, id := range cityIDs {
		key := intToString(id)
		name := ""
		if val, ok := s.citiesName[key]; ok {
			name = val
		}
		// Only include if name found? User said "tampilkan data kota... lalu cari nama kota... response city_id, city_name".
		// Assuming we include it even if name is missing (though unlikely if location.json is source of truth).
		// But let's filter to only those found in location.json if that's implied "from location.json array cities[]".
		// Actually, if ID is in DB but not in JSON, name will be empty.
		if name != "" {
			cities = append(cities, model.ServiceFleetPickupItem{
				CityID:   id,
				CityName: name,
			})
		}
	}

	// Sort by CityName
	// Need to import "sort"
	// But first let's add the method. I'll add sort import in a separate edit if needed or use bubble sort for small list.
	// Since I can't see imports easily, I'll use a simple sort or rely on subsequent edit.
	// Actually, I should check imports.
	// Let's implement a simple sort here to be safe without adding imports if possible, or assume sort is available?
	// `sort` is standard.
	// Let's check imports first or just add it.
	// Wait, I can't add import easily with SearchReplace unless I read the top.
	// I'll use a simple insertion sort for now, assuming list is small (cities).
	for i := 1; i < len(cities); i++ {
		j := i
		for j > 0 && cities[j].CityName < cities[j-1].CityName {
			cities[j], cities[j-1] = cities[j-1], cities[j]
			j--
		}
	}

	return cities, nil
}

func (s *FleetService) GetServiceFleetDetail(fleetID string) (*model.ServiceFleetDetailResponse, error) {
	// First resolve OrgID
	orgID, err := s.repo.GetFleetOrgID(fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet org ID:", err)
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
	}

	meta, err := s.repo.GetFleetDetailMeta(orgID, fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet detail meta:", err)
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
	}
	fac, err := s.repo.GetFleetFacilities(fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet facilities:", err)
		fac = []string{}
	}
	pickup, err := s.repo.GetFleetPickup(orgID, fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet pickup:", err)
		pickup = []model.FleetPickupItem{}
	}
	addon, err := s.repo.GetFleetAddon(orgID, fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet addon:", err)
		addon = []model.FleetAddonItem{}
	}
	prices, err := s.repo.GetFleetPrices(orgID, fleetID)
	if err != nil {
		fmt.Println("Error fetching fleet prices:", err)
		prices = []model.FleetPriceItem{}
	}
	images, err := s.repo.GetFleetImages(fleetID)
	if err != nil {
		images = []model.FleetImageItem{}
	}

	s.ensureCitiesLoaded()

	// Convert Pickup
	svcPickup := make([]model.ServiceFleetPickupItem, len(pickup))
	for i, p := range pickup {
		svcPickup[i] = model.ServiceFleetPickupItem{
			CityID: p.CityID,
		}
		key := intToString(p.CityID)
		if name, ok := s.citiesName[key]; ok {
			svcPickup[i].CityName = name
		} else {
			svcPickup[i].CityName = ""
		}
	}

	// Convert Pricing
	svcPrices := make([]model.ServiceFleetPriceItem, len(prices))
	for i, p := range prices {
		svcPrices[i] = model.ServiceFleetPriceItem{
			UUID:          p.UUID,
			Duration:      p.Duration,
			RentType:      p.RentType,
			RentTypeLabel: configs.RentType(p.RentType).String(),
			Price:         p.Price,
			DiscAmount:    p.DiscAmount,
			DiscPrice:     p.DiscPrice,
			Uom:           p.Uom,
		}
	}

	resp := &model.ServiceFleetDetailResponse{
		Meta:       *meta,
		Facilities: fac,
		Pickup:     svcPickup,
		Addon:      addon,
		Pricing:    svcPrices,
		Images:     images,
	}
	return resp, nil
}

func (s *FleetService) GetPartnerOrderList(orgID string, filter *model.PartnerOrderListFilter) ([]model.PartnerOrderListItem, error) {
	items, err := s.repo.GetPartnerOrderList(orgID, filter)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if token, err := helper.EncryptString(items[i].OrderID); err == nil {
			items[i].TransactionID = token
		}
	}
	return items, nil
}

func (s *FleetService) GetPartnerOrdersWithSummary(orgID string, filter *model.PartnerOrderListFilter) (*model.PartnerOrderListResponse, error) {
	items, err := s.repo.GetPartnerOrderList(orgID, filter)
	if err != nil {
		msg := "failed to get order list"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	for i := range items {
		if token, err := helper.EncryptString(items[i].OrderID); err == nil {
			items[i].TransactionID = token
		}
	}
	summary, err := s.repo.GetPartnerOrderSummary(orgID, filter)
	if err != nil {
		msg := "failed to get order summary"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return &model.PartnerOrderListResponse{
		Summary: *summary,
		Orders:  items,
	}, nil
}

func (s *FleetService) GetPartnerOrderDetail(orderID, orgID string) (*model.OrderDetailResponse, error) {
	res, err := s.repo.GetPartnerOrderDetail(orderID, orgID)
	if err != nil {
		return nil, err
	}
	res.RentTypeLabel = configs.RentType(res.RentType).String()
	s.ensureCitiesLoaded()

	// Map Customer City
	if res.Customer.CustomerCity != "" {
		if name, ok := s.citiesName[res.Customer.CustomerCity]; ok && name != "" {
			res.Customer.CityLabel = name
		}
	}

	// Map Pickup City
	if res.Pickup.PickupCity != "" {
		if name, ok := s.citiesName[res.Pickup.PickupCity]; ok && name != "" {
			res.Pickup.CityLabel = name
		}
	}

	// Map Destination City
	for i := range res.Destination {
		if res.Destination[i].City != "" {
			if name, ok := s.citiesName[res.Destination[i].City]; ok && name != "" {
				res.Destination[i].CityLabel = name
			}
		}
	}

	// Map Itinerary City
	for i := range res.Itinerary {
		if res.Itinerary[i].CityID != "" {
			if name, ok := s.citiesName[res.Itinerary[i].CityID]; ok && name != "" {
				res.Itinerary[i].CityLabel = name
			}
		}
	}

	return res, nil
}

func (s *FleetService) GetPartnerOrderPaymentSummary(orderID, orgID string, totalAmount float64) (*model.PaymentSummary, error) {
	row, err := s.repo.GetLatestPaymentOrder(orderID, 1, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &model.PaymentSummary{
				PaidAmount:       0,
				PaymentRemaining: totalAmount,
				PaymentStatus:    "unpaid",
			}, nil
		}
		return nil, err
	}

	s.ensurePaymentMethodsLoaded()
	baseTotal := row.TotalAmount
	if baseTotal <= 0 {
		baseTotal = totalAmount
	}
	status := "pending"
	if row.RemainingAmount == 0 {
		status = "paid"
	}
	return &model.PaymentSummary{
		PaymentAmount:      row.PaymentAmount,
		PaymentRemaining:   row.RemainingAmount,
		PaidAmount:         baseTotal - row.RemainingAmount,
		PaymentMethod:      row.PaymentMethod,
		PaymentMethodLabel: s.paymentMethodLabels[row.PaymentMethod],
		PaymentStatus:      status,
		PaymentDate:        row.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *FleetService) GetFleetAddonList(orgID, fleetID string) ([]model.FleetAddonListItem, error) {
	if fleetID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleetid is required")
	}
	addons, err := s.repo.GetFleetAddon(orgID, fleetID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get fleet addon")
	}
	items := make([]model.FleetAddonListItem, len(addons))
	for i, a := range addons {
		items[i] = model.FleetAddonListItem{
			AddonID:    a.UUID,
			AddonName:  a.AddonName,
			AddonPrice: float64(a.AddonPrice),
		}
	}
	return items, nil
}

func (s *FleetService) GetFleetPricesByFleetID(orgID, fleetID, typeID string) ([]model.FleetPriceListItem, error) {
	if fleetID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleetid is required")
	}
	if typeID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "typeid is required")
	}
	rentType, err := strconv.Atoi(typeID)
	if err != nil {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid typeid")
	}
	_ = orgID

	items, err := s.repo.GetFleetPriceListByRentType(fleetID, rentType)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get fleet prices")
	}

	for i := range items {
		items[i].RentTypeLabel = configs.RentType(items[i].RentType).String()
	}
	return items, nil
}

func (s *FleetService) CreatePartnerOrder(orgID, userID string, req *model.FleetOrderCreateRequest) (string, error) {
	if req.FleetID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleet_id is required")
	}
	if req.CustomerID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "customer_id is required")
	}
	if req.PickupDatetime == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "pickup_datetime is required")
	}
	if req.DropoffDatetime == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "dropoff_datetime is required")
	}
	if req.PickupCityID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "pickup_city_id is required")
	}
	if req.PriceID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "price_id is required")
	}

	qty := req.FleetQty
	if qty <= 0 {
		qty = req.Quantity
	}
	if qty <= 0 {
		qty = 1
	}

	pickupLoc := strings.TrimSpace(req.PickupLocation)
	if pickupLoc == "" {
		pickupLoc = strings.TrimSpace(req.PickupAddress)
	}
	if pickupLoc == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "pickup_location is required")
	}

	startDate, err := normalizeDateTime(req.PickupDatetime)
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid pickup_datetime")
	}
	endDate, err := normalizeDateTime(req.DropoffDatetime)
	if err != nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid dropoff_datetime")
	}

	price := req.Price
	if price <= 0 {
		p, _, err := s.repo.GetPriceByID(req.PriceID)
		if err != nil {
			return "", NewServiceError(ErrNotFound, http.StatusNotFound, "price not found")
		}
		price = p
	}
	addonTotal := 0.0
	if len(req.Addons) > 0 {
		addonIDs := make([]string, 0, len(req.Addons))
		qtyByID := make(map[string]int, len(req.Addons))
		for _, a := range req.Addons {
			if a.AddonID == "" {
				continue
			}
			addonIDs = append(addonIDs, a.AddonID)
			q := a.Quantity
			if q <= 0 {
				q = 1
			}
			qtyByID[a.AddonID] += q
		}
		prices, err := s.repo.GetAddonPrices(addonIDs)
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to calc addons")
		}
		for id, q := range qtyByID {
			addonTotal += prices[id] * float64(q)
		}
	}
	totalAmount := (float64(qty) * price) + addonTotal - req.DiscountAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	if orgID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}
	orgCode, err := s.repo.GetOrganizationCodeByOrgID(orgID)
	if err != nil || strings.TrimSpace(orgCode) == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "organization context missing")
	}

	count, err := s.repo.GetOrderCountByOrgID(orgID)
	if err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order count")
	}

	truncatedCode := orgCode
	if len(orgCode) >= 5 {
		truncatedCode = orgCode[:3] + orgCode[len(orgCode)-2:]
	}
	timePart := time.Now().Format("06020115")
	orderID := fmt.Sprintf("%s%s%d-FRT", truncatedCode, timePart, count+1)

	if err := s.repo.CreatePartnerOrder(orderID, req.FleetID, startDate, endDate, req.PickupCityID, pickupLoc, qty, req.PriceID, totalAmount, req.CustomerID, orgID, userID, req.Itinerary, req.Addons, req.AdditionalRequest); err != nil {
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create order")
	}
	return orderID, nil
}

func (s *FleetService) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	items, err := s.repo.ListFleets(req)
	if err != nil {
		msg := "failed to list fleets"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return items, nil
}

func (s *FleetService) ListFleetsForUnit(orgID, searchFor string) ([]model.FleetUnitSearchItem, error) {
	items, err := s.repo.ListFleetsForUnit(orgID, searchFor)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to list fleets")
	}
	return items, nil
}

func (s *FleetService) DeleteFleet(orgID, userID, fleetID string) error {
	if strings.TrimSpace(fleetID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "fleet_id is required")
	}
	if err := s.repo.SoftDeleteFleet(orgID, userID, fleetID); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to delete fleet")
	}
	return nil
}

func normalizeDateTime(v string) (string, error) {
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
	return "", fmt.Errorf("invalid datetime")
}

func (s *FleetService) GetFleetDetail(orgID, fleetID string) (*model.FleetDetailResponse, error) {
	meta, err := s.repo.GetFleetDetailMeta(orgID, fleetID)
	if err != nil {
		return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet not found")
	}
	fac, err := s.repo.GetFleetFacilities(fleetID)
	if err != nil {
		fac = []string{}
	}
	pickup, err := s.repo.GetFleetPickup(orgID, fleetID)
	if err != nil {
		pickup = []model.FleetPickupItem{}
	}
	addon, err := s.repo.GetFleetAddon(orgID, fleetID)
	if err != nil {
		addon = []model.FleetAddonItem{}
	}
	prices, err := s.repo.GetFleetPrices(orgID, fleetID)
	if err != nil {
		prices = []model.FleetPriceItem{}
	}
	images, err := s.repo.GetFleetImages(fleetID)
	if err != nil {
		images = []model.FleetImageItem{}
	}

	s.ensureCitiesLoaded()

	for i := range pickup {
		key := intToString(pickup[i].CityID)
		if name, ok := s.citiesName[key]; ok {
			pickup[i].CityName = name
		} else {
			pickup[i].CityName = ""
		}
	}

	for i := range prices {
		prices[i].RentTypeLabel = configs.RentType(prices[i].RentType).String()
	}

	resp := &model.FleetDetailResponse{
		Meta:       *meta,
		Facilities: fac,
		Pickup:     pickup,
		Addon:      addon,
		Pricing:    prices,
		Images:     images,
	}
	return resp, nil
}

func (s *FleetService) GetServiceFleetAddons(orgID, fleetID string) ([]model.ServiceFleetAddonItem, error) {
	addons, err := s.repo.GetFleetAddon(orgID, fleetID)
	if err != nil {
		return nil, err
	}

	items := make([]model.ServiceFleetAddonItem, len(addons))
	for i, a := range addons {
		items[i] = model.ServiceFleetAddonItem{
			AddonID:    a.UUID,
			AddonName:  a.AddonName,
			AddonDesc:  a.AddonDesc,
			AddonPrice: a.AddonPrice,
		}
	}
	return items, nil
}

func (s *FleetService) ensureCitiesLoaded() {
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

func (s *FleetService) ensurePaymentMethodsLoaded() {
	if s.paymentMethodLabels != nil {
		return
	}
	f, err := os.Open("config/common.json")
	if err != nil {
		s.paymentMethodLabels = map[int]string{}
		return
	}
	defer f.Close()

	var cfg model.CommonConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		s.paymentMethodLabels = map[int]string{}
		return
	}

	m := make(map[int]string, len(cfg.PaymentMethod))
	for _, it := range cfg.PaymentMethod {
		m[it.ID] = it.Label
	}
	s.paymentMethodLabels = m
}

func intToString(n int) string { return strconv.Itoa(n) }
