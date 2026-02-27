package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"service-travego/configs"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
)

type FleetService struct {
	repo       *repository.FleetRepository
	citiesName map[string]string
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

func (s *FleetService) GetPartnerOrderList(orgID string) ([]model.PartnerOrderListItem, error) {
	return s.repo.GetPartnerOrderList(orgID)
}

func (s *FleetService) GetPartnerOrderDetail(orderID, orgID string) (*model.OrderDetailResponse, error) {
	return s.repo.GetPartnerOrderDetail(orderID, orgID)
}

func (s *FleetService) ListFleets(req *model.ListFleetRequest) ([]model.FleetListItem, error) {
	items, err := s.repo.ListFleets(req)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to list fleets")
	}
	return items, nil
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

func intToString(n int) string { return strconv.Itoa(n) }
