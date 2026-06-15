package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FleetUnitService struct {
	repo                      *repository.FleetUnitRepository
	partnerRepo               *repository.PartnerRepository
	orgRepo                   *repository.OrganizationRepository
	citiesName                map[string]string
	transmissionLabel         map[string]string
	commonOnce                sync.Once
	paymentMethodLabels       map[int]string
	paymentStatusLabels       map[int]string
	transactionCategoryLabels map[string]string
	transactionItemLabels     map[string]string
}

func NewFleetUnitService(repo *repository.FleetUnitRepository, partnerRepo *repository.PartnerRepository, orgRepo *repository.OrganizationRepository) *FleetUnitService {
	return &FleetUnitService{repo: repo, partnerRepo: partnerRepo, orgRepo: orgRepo}
}

func (s *FleetUnitService) ensureCitiesLoaded() {
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

func (s *FleetUnitService) ensureTransmissionLoaded() {
	if s.transmissionLabel != nil {
		return
	}
	f, err := os.Open("config/fleet-config.json")
	if err != nil {
		s.transmissionLabel = map[string]string{}
		return
	}
	defer f.Close()
	var cfg model.FleetConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		s.transmissionLabel = map[string]string{}
		return
	}
	m := make(map[string]string, len(cfg.FleetTransmission))
	for _, it := range cfg.FleetTransmission {
		if it.ID != "" && it.Label != "" {
			m[it.ID] = it.Label
		}
	}
	s.transmissionLabel = m
}

func (s *FleetUnitService) ensureCommonLoaded() {
	s.commonOnce.Do(func() {
		s.paymentMethodLabels = map[int]string{}
		s.paymentStatusLabels = map[int]string{}
		s.transactionCategoryLabels = map[string]string{}
		s.transactionItemLabels = map[string]string{}

		f, err := os.Open("config/common.json")
		if err != nil {
			return
		}
		defer f.Close()

		var cfg struct {
			PaymentMethod         []model.CommonItem `json:"payment-method"`
			PaymentStatus         []model.CommonItem `json:"payment-status"`
			TransactionCategories []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"transaction-categories"`
			TransactionItems []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"transaction-items"`
		}
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			return
		}
		for _, it := range cfg.PaymentMethod {
			s.paymentMethodLabels[it.ID] = it.Label
		}
		for _, it := range cfg.PaymentStatus {
			s.paymentStatusLabels[it.ID] = it.Label
		}
		for _, it := range cfg.TransactionCategories {
			k := strings.ToUpper(strings.TrimSpace(it.ID))
			if k == "" {
				continue
			}
			s.transactionCategoryLabels[k] = it.Label
		}
		for _, it := range cfg.TransactionItems {
			k := strings.ToUpper(strings.TrimSpace(it.ID))
			if k == "" {
				continue
			}
			s.transactionItemLabels[k] = it.Label
		}
	})
}

func (s *FleetUnitService) List(orgID, fleetId, orderID, search string) ([]model.FleetUnitListItem, error) {
	items, err := s.repo.List(orgID, fleetId, orderID, search)
	if err != nil {
		msg := "failed to get fleet units"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return items, nil
}

func (s *FleetUnitService) Create(orgID, userID string, req *model.FleetUnitCreateRequest) (string, error) {
	req.OrganizationID = orgID
	req.CreatedBy = userID

	var partnerID *string
	if req.PartnerID == nil && req.PartnerName != nil && req.PartnerPhone != nil {
		partnerIDStr, err := s.partnerRepo.GetOrCreateByNamePhone(orgID, userID, *req.PartnerName, *req.PartnerPhone, req.PartnerEmail)
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to handle partner")
		}
		partnerID = &partnerIDStr
	} else if req.PartnerID != nil {
		partnerID = req.PartnerID
	}

	vehicleID := strings.ToUpper(strings.TrimSpace(req.VehicleID))
	plateNumber := strings.ToUpper(strings.TrimSpace(req.PlateNumber))
	if vehicleID != "" {
		existing, err := s.repo.FindExistingVehicleIDs(orgID, []string{vehicleID})
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
		}
		if _, ok := existing[vehicleID]; ok {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
		}
	}
	if plateNumber != "" {
		existing, err := s.repo.FindExistingPlateNumbers(orgID, []string{plateNumber})
		if err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
		}
		if _, ok := existing[plateNumber]; ok {
			return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
		}
	}

	id, err := s.repo.Create(req)
	if err != nil {
		msg := "failed to create fleet unit"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}

	if req.OwnershipType != nil && *req.OwnershipType == 1 && partnerID != nil {
		if err := s.repo.SetUnitOwnership(id, *partnerID, orgID, userID); err != nil {
			return "", NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to set unit ownership")
		}
	}

	return id, nil
}

func (s *FleetUnitService) CreateBatch(orgID, userID, fleetID string, units []model.FleetUnitCreateUnit) ([]string, error) {
	seenVehicle := map[string]struct{}{}
	seenPlate := map[string]struct{}{}

	vehicleIDs := make([]string, 0, len(units))
	plateNumbers := make([]string, 0, len(units))
	for _, u := range units {
		vid := strings.ToUpper(strings.TrimSpace(u.VehicleID))
		pn := strings.ToUpper(strings.TrimSpace(u.PlateNumber))

		if vid != "" {
			if _, ok := seenVehicle[vid]; ok {
				return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
			}
			seenVehicle[vid] = struct{}{}
			vehicleIDs = append(vehicleIDs, vid)
		}
		if pn != "" {
			if _, ok := seenPlate[pn]; ok {
				return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
			}
			seenPlate[pn] = struct{}{}
			plateNumbers = append(plateNumbers, pn)
		}
	}

	existingVehicles, err := s.repo.FindExistingVehicleIDs(orgID, vehicleIDs)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
	}
	if len(existingVehicles) > 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_VEHICLE_ID")
	}

	existingPlates, err := s.repo.FindExistingPlateNumbers(orgID, plateNumbers)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create fleet unit")
	}
	if len(existingPlates) > 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "DUPLICATE_PLATE_NUMBER")
	}

	ids := make([]string, 0, len(units))
	for _, u := range units {
		req := &model.FleetUnitCreateRequest{
			VehicleID:      u.VehicleID,
			PlateNumber:    u.PlateNumber,
			FleetID:        fleetID,
			Engine:         u.Engine,
			Transmission:   u.Transmission,
			Capacity:       u.Capacity,
			ProductionYear: u.ProductionYear,
			OwnershipType:  u.OwnershipType,
			PartnerID:      u.PartnerID,
			PartnerName:    u.PartnerName,
			PartnerPhone:   u.PartnerPhone,
			PartnerEmail:   u.PartnerEmail,
		}
		id, err := s.Create(orgID, userID, req)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *FleetUnitService) Update(orgID, userID string, req *model.FleetUnitUpdateRequest) error {
	req.OrganizationID = orgID
	req.UpdatedBy = userID

	var partnerID *string
	if req.PartnerID == nil && req.PartnerName != nil && req.PartnerPhone != nil {
		partnerIDStr, err := s.partnerRepo.GetOrCreateByNamePhone(orgID, userID, *req.PartnerName, *req.PartnerPhone, req.PartnerPic)
		if err != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to handle partner")
		}
		partnerID = &partnerIDStr
	} else if req.PartnerID != nil {
		partnerID = req.PartnerID
	}

	if req.OwnershipType != nil && *req.OwnershipType == 0 {
		if err := s.repo.DeleteUnitOwnership(orgID, req.UnitID); err != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to delete unit ownership")
		}
	}

	if req.PartnerID != nil && req.PartnerName != nil && req.PartnerPhone != nil {
		fmt.Println("---- masuk kondisi ini")
		if errUpdatePartner := s.repo.UpdateOwnerInformation(orgID, req.UnitID, *partnerID, *req.PartnerName, *req.PartnerPhone, *req.PartnerPic); errUpdatePartner != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update partner information")
		}
	}

	if err := s.repo.Update(req); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "fleet unit not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update fleet unit")
	}

	if req.OwnershipType != nil && *req.OwnershipType == 1 && partnerID != nil {
		if err := s.repo.SetUnitOwnership(req.UnitID, *partnerID, orgID, userID); err != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to set unit ownership")
		}
	}

	return nil
}

func (s *FleetUnitService) Detail(orgID, uuid string) (*model.FleetUnitDetailResponse, error) {
	res, err := s.repo.Detail(orgID, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "fleet unit not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get fleet unit detail")
	}

	s.ensureTransmissionLoaded()
	res.TransmissionID = res.Transmission
	if label, ok := s.transmissionLabel[strings.TrimSpace(res.Transmission)]; ok && label != "" {
		res.Transmission = label
	}

	if strings.TrimSpace(res.FleetID) != "" {
		cityIDs, err := s.repo.GetFleetPickupCityIDs(orgID, res.FleetID)
		if err == nil && len(cityIDs) > 0 {
			s.ensureCitiesLoaded()
			out := make([]string, 0, len(cityIDs))
			seen := map[string]struct{}{}
			for _, id := range cityIDs {
				key := strconv.Itoa(id)
				name := s.citiesName[key]
				if name == "" {
					continue
				}
				if _, ok := seen[name]; ok {
					continue
				}
				seen[name] = struct{}{}
				out = append(out, name)
			}
			res.PickupPoint = out
		}
	}

	if res.OwnershipType != nil {
		if *res.OwnershipType == 1 {
			ownershipInfo, err := s.repo.GetOwnershipInformation(orgID, uuid)
			if err == nil && ownershipInfo != nil {
				res.OwnershipInformation = ownershipInfo
			}
		} else if *res.OwnershipType == 2 {
			org, err := s.orgRepo.FindByID(orgID)
			if err == nil && org != nil {
				res.OwnershipInformation = &model.FleetUnitOwnershipInformation{
					PartnerName: org.OrganizationName,
				}
			}
		}
	}

	return res, nil
}

func (s *FleetUnitService) UnitOrderHistory(orgID, unitID, startDate, endDate string) ([]model.FleetUnitOrderHistoryItem, error) {
	items, err := s.repo.UnitOrderHistory(orgID, unitID, startDate, endDate)
	if err != nil {
		msg := "failed to get fleet unit order history"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}

	s.ensureCitiesLoaded()
	orderIDs := make([]string, 0, len(items))
	seenOrders := map[string]struct{}{}
	for i := range items {
		items[i].PickupCityLabel = s.citiesName[strings.TrimSpace(items[i].PickupCityID)]
		oid := strings.TrimSpace(items[i].OrderID)
		if oid != "" {
			if _, ok := seenOrders[oid]; !ok {
				seenOrders[oid] = struct{}{}
				orderIDs = append(orderIDs, oid)
			}
		}
	}

	destCityIDs, err := s.repo.GetOrderDestinationCityIDs(orderIDs)
	if err != nil {
		msg := "failed to get fleet unit order history"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	for i := range items {
		labels := make([]string, 0)
		seenLabel := map[string]struct{}{}

		if strings.TrimSpace(items[i].DestinationIDs) != "" {
			rawIDs := strings.Split(items[i].DestinationIDs, ",")
			for _, raw := range rawIDs {
				id := strings.TrimSpace(raw)
				if id == "" {
					continue
				}
				name := s.citiesName[id]
				if name == "" {
					continue
				}
				if _, ok := seenLabel[name]; ok {
					continue
				}
				seenLabel[name] = struct{}{}
				labels = append(labels, name)
			}
		} else {
			cityIDs := destCityIDs[strings.TrimSpace(items[i].OrderID)]
			for _, id := range cityIDs {
				name := s.citiesName[strings.TrimSpace(id)]
				if name == "" {
					continue
				}
				if _, ok := seenLabel[name]; ok {
					continue
				}
				seenLabel[name] = struct{}{}
				labels = append(labels, name)
			}
		}

		if len(labels) > 0 {
			items[i].DestinationCity = strings.Join(labels, ", ")
			if strings.TrimSpace(items[i].Destinations) == "" {
				items[i].Destinations = items[i].DestinationCity
			}
		}
	}
	return items, nil
}

func (s *FleetUnitService) GetUnitRevenue(orgID, unitID, startDate, endDate string) (*model.FleetUnitRevenue, error) {
	revenue, err := s.repo.GetUnitRevenue(orgID, unitID, startDate, endDate)
	if err != nil {
		fmt.Println(err)
		return &model.FleetUnitRevenue{TotalRevenue: 0, TotalBooking: 0}, nil
	}
	if revenue == nil {
		return &model.FleetUnitRevenue{TotalRevenue: 0, TotalBooking: 0}, nil
	}
	return revenue, nil
}

func (s *FleetUnitService) GetUnitRevenueHistory(orgID, unitID, startDate, endDate string) ([]model.FleetUnitRevenueHistoryItem, error) {
	rows, err := s.repo.ListUnitRevenueHistory(orgID, strings.TrimSpace(unitID), startDate, endDate)
	if err != nil {
		fmt.Println(err)
		return []model.FleetUnitRevenueHistoryItem{}, nil
	}

	s.ensureCommonLoaded()

	for i := range rows {
		if label, ok := s.paymentStatusLabels[rows[i].PaymentType]; ok && label != "" {
			rows[i].PaymentTypeLabel = label
		} else if rows[i].PaymentType != 0 {
			rows[i].PaymentTypeLabel = strconv.Itoa(rows[i].PaymentType)
		}

		if label, ok := s.paymentMethodLabels[rows[i].PaymentMethod]; ok && label != "" {
			rows[i].PaymentMethodLabel = label
		} else if rows[i].PaymentMethod != 0 {
			rows[i].PaymentMethodLabel = strconv.Itoa(rows[i].PaymentMethod)
		}
	}

	return rows, nil
}

func (s *FleetUnitService) UnitExpenses(orgID, unitID, period string) ([]model.FleetUnitExpenseItem, error) {
	t, err := time.Parse("2006-01", strings.TrimSpace(period))
	if err != nil {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "Invalid period format. Use YYYY-MM")
	}

	startDate := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	rows, err := s.repo.ListUnitExpenses(orgID, strings.TrimSpace(unitID), startDate, endDate)
	if err != nil {
		msg := "failed to get fleet unit expenses"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}

	s.ensureCommonLoaded()

	for i := range rows {
		if label, ok := s.paymentMethodLabels[rows[i].PaymentType]; ok && label != "" {
			rows[i].PaymentTypeLabel = label
		} else if rows[i].PaymentType != 0 {
			rows[i].PaymentTypeLabel = strconv.Itoa(rows[i].PaymentType)
		}

		catKey := strings.ToUpper(strings.TrimSpace(rows[i].TransactionCategory))
		rows[i].TransactionCategory = catKey
		if catKey != "" {
			if label, ok := s.transactionCategoryLabels[catKey]; ok && label != "" {
				rows[i].TransactionCategoryLabel = label
			} else {
				rows[i].TransactionCategoryLabel = catKey
			}
		}

		itemKey := strings.ToUpper(strings.TrimSpace(rows[i].TransactionItem))
		rows[i].TransactionItem = itemKey
		if itemKey != "" {
			if label, ok := s.transactionItemLabels[itemKey]; ok && label != "" {
				rows[i].TransactionItemLabel = label
			} else {
				rows[i].TransactionItemLabel = itemKey
			}
		}
	}

	return rows, nil
}

func (s *FleetUnitService) UnitRating(orgID, unitID string) (float64, error) {
	rating, err := s.repo.UnitRating(orgID, unitID)
	if err != nil {
		msg := "failed to get fleet unit rating"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return 0, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return rating, nil
}

func (s *FleetUnitService) UnitReviews(orgID, unitID string) ([]model.OrderReviewItem, error) {
	items, err := s.repo.UnitReviews(orgID, unitID)
	if err != nil {
		msg := "failed to get fleet unit reviews"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	return items, nil
}

func (s *FleetUnitService) UnitScheduleStats(orgID, unitID string) (int64, *model.FleetUnitScheduleRange, *model.FleetUnitScheduleRange, error) {
	total, err := s.repo.UnitTotalSchedules(orgID, unitID)
	if err != nil {
		msg := "failed to get fleet unit schedules"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return 0, nil, nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}

	now := time.Now()
	latest, err := s.repo.UnitLatestSchedule(orgID, unitID, now)
	if err != nil {
		msg := "failed to get fleet unit schedules"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return 0, nil, nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}
	upcoming, err := s.repo.UnitUpcomingSchedule(orgID, unitID, now)
	if err != nil {
		msg := "failed to get fleet unit schedules"
		if env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))); env != "production" && env != "prod" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}
		return 0, nil, nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, msg)
	}

	return total, latest, upcoming, nil
}
