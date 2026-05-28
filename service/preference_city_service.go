package service

import (
	"encoding/json"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"sort"
	"strconv"
)

type PreferenceCityService struct {
	prefRepo     *repository.PreferenceCityRepository
	locationPath string
}

func NewPreferenceCityService(prefRepo *repository.PreferenceCityRepository, locationPath string) *PreferenceCityService {
	return &PreferenceCityService{
		prefRepo:     prefRepo,
		locationPath: locationPath,
	}
}

func (s *PreferenceCityService) Create(cityIDs []int, minimalDay int, organizationID string, createdBy string, serviceTypes []int) error {
	for _, cityID := range cityIDs {
		if err := s.prefRepo.Create(cityID, minimalDay, organizationID, createdBy, serviceTypes); err != nil {
			return err
		}
	}
	return nil
}

func (s *PreferenceCityService) Update(preferenceID string, cityID int, minimalDay int, organizationID string, serviceTypeIDs []int) error {
	return s.prefRepo.Update(preferenceID, cityID, minimalDay, organizationID, serviceTypeIDs)
}

func (s *PreferenceCityService) Delete(preferenceID, organizationID string) error {
	return s.prefRepo.Delete(preferenceID, organizationID)
}

func (s *PreferenceCityService) DeleteByCityAndServiceType(cityID int, serviceType int, organizationID string) error {
	return s.prefRepo.DeleteByCityAndServiceType(cityID, serviceType, organizationID)
}

func (s *PreferenceCityService) GetAll(organizationID string, cityID *int) ([]model.PreferenceCityWithLabels, error) {
	prefs, err := s.prefRepo.GetAll(organizationID, cityID)
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
			provinceIDStr := city.ProvinceID
			if province, ok := provinceMap[provinceIDStr]; ok {
				withLabels.ProvinceLabel = province.Name
			}
		}

		if types, err := s.prefRepo.GetTypesByCityIDAndOrganizationID(pref.CityID, pref.OrganizationID); err == nil {
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
