package service

import (
	"encoding/json"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"sort"
	"strconv"
	"strings"
)

type GeneralService struct {
	configPath   string
	menuPath     string
	locationPath string
	generalRepo  *repository.GeneralRepository
}

func NewGeneralService(configPath, menuPath, locationPath string, generalRepo *repository.GeneralRepository) *GeneralService {
	s := &GeneralService{
		configPath:   configPath,
		menuPath:     menuPath,
		locationPath: locationPath,
		generalRepo:  generalRepo,
	}
	s.ensureLocationProvinceIDs()
	return s
}

// GetGeneralConfig reads and returns general configuration from JSON file
func (s *GeneralService) GetGeneralConfig() (*model.GeneralConfig, error) {
	file, err := os.Open(s.configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config model.GeneralConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetWebMenu reads and returns web menu from JSON file
func (s *GeneralService) GetWebMenu() (*model.WebMenu, error) {
	file, err := os.Open(s.menuPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var menu model.WebMenu
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&menu)
	if err != nil {
		return nil, err
	}

	return &menu, nil
}

func (s *GeneralService) GetFuelTypes() ([]model.FleetFuelType, error) {
	file, err := os.Open("config/fleet-config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg model.FleetConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.FuelType == nil {
		return []model.FleetFuelType{}, nil
	}
	return cfg.FuelType, nil
}

func (s *GeneralService) GetFleetTransmissions() ([]model.FleetTransmission, error) {
	file, err := os.Open("config/fleet-config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg model.FleetConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.FleetTransmission == nil {
		return []model.FleetTransmission{}, nil
	}
	return cfg.FleetTransmission, nil
}

func (s *GeneralService) GetContractTypes() ([]model.CommonItem, error) {
	file, err := os.Open("config/common.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg model.CommonConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.ContractType == nil {
		return []model.CommonItem{}, nil
	}
	return cfg.ContractType, nil
}

func (s *GeneralService) GetPaymentStatuses() ([]model.CommonItem, error) {
	file, err := os.Open("config/common.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg model.CommonConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.PaymentStatus == nil {
		return []model.CommonItem{}, nil
	}
	return cfg.PaymentStatus, nil
}

func (s *GeneralService) GetPaymentMethods() ([]model.CommonItem, error) {
	file, err := os.Open("config/common.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg model.CommonConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.PaymentMethod == nil {
		return []model.CommonItem{}, nil
	}
	return cfg.PaymentMethod, nil
}

// GetBankList reads and returns bank list from database sorted by name
func (s *GeneralService) GetBankList() ([]model.Bank, error) {
	return s.generalRepo.GetBankList()
}

func (s *GeneralService) GetPreferenceCities(organizationID string, cityID *int, serviceType *int) ([]model.PreferenceCityWithLabels, error) {
	prefs, err := s.generalRepo.GetPreferenceCities(organizationID, cityID, serviceType)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(s.locationPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var location model.Location
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&location); err != nil {
		return nil, err
	}

	cityMap := make(map[string]model.City)
	for _, city := range location.Cities {
		cityMap[city.ID] = city
	}

	provinceMap := make(map[string]model.Province)
	for _, province := range location.Provinces {
		provinceMap[province.ID] = province
	}

	var result []model.PreferenceCityWithLabels
	for _, pref := range prefs {
		withLabels := model.PreferenceCityWithLabels{
			PreferenceID:   pref.PreferenceID,
			CityID:         pref.CityID,
			MinimalDay:     pref.MinimalDay,
			OrganizationID: pref.OrganizationID,
			CreatedAt:      pref.CreatedAt,
			CreatedBy:      pref.CreatedBy,
		}

		cityIDStr := strconv.Itoa(pref.CityID)
		if city, ok := cityMap[cityIDStr]; ok {
			withLabels.CityLabel = city.Name
			if province, ok := provinceMap[city.ProvinceID]; ok {
				withLabels.ProvinceLabel = province.Name
			}
		}

		if types, err := s.generalRepo.GetPreferenceCityTypesByCityIDAndOrganizationID(pref.CityID, pref.OrganizationID); err == nil {
			for _, t := range types {
				if label, ok := model.ServiceTypeLabels[t]; ok {
					withLabels.ServiceTypes = append(withLabels.ServiceTypes, label)
				}
			}
		}

		result = append(result, withLabels)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CityLabel < result[j].CityLabel
	})

	return result, nil
}

// GetProvinces reads and returns provinces from location JSON file
// If searchText is provided, it filters provinces by name containing the search text (case-insensitive)
func (s *GeneralService) GetProvinces(searchText string) ([]model.Province, error) {
	file, err := os.Open(s.locationPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var location model.Location
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&location)
	if err != nil {
		return nil, err
	}

	if searchText == "" {
		return location.Provinces, nil
	}

	searchLower := strings.ToLower(strings.TrimSpace(searchText))
	var filtered []model.Province
	for _, p := range location.Provinces {
		if strings.Contains(strings.ToLower(p.Name), searchLower) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

// GetCities reads and returns cities from location JSON file
// Filters supported:
// - provinceID: map to province name and filter by exact name (case-insensitive)
// - provinceName: filter by exact province name (case-insensitive)
// - searchText: filter by city name contains (case-insensitive)
func (s *GeneralService) GetCities(provinceID, provinceName, searchText string) ([]model.City, error) {
	file, err := os.Open(s.locationPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var location model.Location
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&location)
	if err != nil {
		return nil, err
	}

	var filteredCities []model.City

	// Build helper maps for enrichment and filtering
	nameToID := make(map[string]string)
	for _, p := range location.Provinces {
		nameToID[strings.ToLower(p.Name)] = p.ID
	}

	// Determine province filter name (from ID or provided name)
	filterProvinceLower := ""
	if strings.TrimSpace(provinceName) != "" {
		filterProvinceLower = strings.ToLower(strings.TrimSpace(provinceName))
	} else if provinceID != "" {
		// If provinceID provided, we will filter directly by city.ProvinceID
	}

	// Filter and enrich cities
	for _, city := range location.Cities {
		// Enrich ProvinceID from province name if empty
		if city.ProvinceID == "" {
			if id, ok := nameToID[strings.ToLower(city.Province)]; ok {
				city.ProvinceID = id
			}
		}

		// Filter by provinceID (if provided)
		if provinceID != "" {
			if city.ProvinceID == "" || city.ProvinceID != provinceID {
				continue
			}
		} else if filterProvinceLower != "" {
			// Filter by province name (if provided)
			if strings.ToLower(city.Province) != filterProvinceLower {
				continue
			}
		}

		// Filter by search text (if provided) - case-insensitive partial match
		if searchText != "" {
			searchLower := strings.ToLower(strings.TrimSpace(searchText))
			cityNameLower := strings.ToLower(city.Name)
			if !strings.Contains(cityNameLower, searchLower) {
				continue
			}
		}

		filteredCities = append(filteredCities, city)
	}

	return filteredCities, nil
}

func (s *GeneralService) ensureLocationProvinceIDs() {
	f, err := os.Open(s.locationPath)
	if err != nil {
		return
	}
	defer f.Close()

	var location model.Location
	d := json.NewDecoder(f)
	if err = d.Decode(&location); err != nil {
		return
	}

	nameToID := make(map[string]string)
	for _, p := range location.Provinces {
		nameToID[strings.ToLower(strings.TrimSpace(p.Name))] = p.ID
	}

	changed := false
	for i := range location.Cities {
		if strings.TrimSpace(location.Cities[i].ProvinceID) == "" {
			if id, ok := nameToID[strings.ToLower(strings.TrimSpace(location.Cities[i].Province))]; ok && id != "" {
				location.Cities[i].ProvinceID = id
				changed = true
			}
		}
	}

	if !changed {
		return
	}

	b, err := json.MarshalIndent(location, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.locationPath, b, 0644)
}
