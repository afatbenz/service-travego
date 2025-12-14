package service

import (
	"encoding/json"
	"os"
	"service-travego/model"
	"strings"
)

type GeneralService struct {
	configPath   string
	menuPath     string
	locationPath string
}

func NewGeneralService(configPath, menuPath, locationPath string) *GeneralService {
	return &GeneralService{
		configPath:   configPath,
		menuPath:     menuPath,
		locationPath: locationPath,
	}
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

    // Determine province filter name (from ID or provided name)
    filterProvinceLower := ""
    if strings.TrimSpace(provinceName) != "" {
        filterProvinceLower = strings.ToLower(strings.TrimSpace(provinceName))
    } else if provinceID != "" {
        for _, province := range location.Provinces {
            if province.ID == provinceID {
                filterProvinceLower = strings.ToLower(province.Name)
                break
            }
        }
    }

	// Filter cities
	for _, city := range location.Cities {
        // Filter by province (if provided via name or id)
        if filterProvinceLower != "" {
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
