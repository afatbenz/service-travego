package service

import (
	"encoding/json"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"time"
)

type SystemService struct {
	repo       *repository.SystemRepository
	citiesName map[string]string
	provinceMap map[string]string
}

func NewSystemService(repo *repository.SystemRepository) *SystemService {
	return &SystemService{
		repo: repo,
	}
}

func (s *SystemService) ensureLocationLoaded() {
	if s.citiesName != nil {
		return
	}
	f, err := os.Open("config/location.json")
	if err != nil {
		s.citiesName = map[string]string{}
		s.provinceMap = map[string]string{}
		return
	}
	defer f.Close()
	var loc model.Location
	if err := json.NewDecoder(f).Decode(&loc); err != nil {
		s.citiesName = map[string]string{}
		s.provinceMap = map[string]string{}
		return
	}
	cities := make(map[string]string, len(loc.Cities))
	provinces := make(map[string]string, len(loc.Provinces))
	for _, c := range loc.Cities {
		cities[c.ID] = c.Name
	}
	for _, p := range loc.Provinces {
		provinces[p.ID] = p.Name
	}
	s.citiesName = cities
	s.provinceMap = provinces
}

func (s *SystemService) GetSystemSummarize(period string) (*model.SystemSummarymarizeResponse, error) {
	return s.repo.GetSummarize(period)
}

func (s *SystemService) GetDeviceList(search, status string) ([]model.DeviceListItem, error) {
	return s.repo.GetDeviceList(search, status)
}

func (s *SystemService) UpdateDevice(account string, action string, enableData *model.DeviceEnableRequest) error {
	return s.repo.UpdateDevice(account, action, enableData)
}

func (s *SystemService) GetOrganizations(search string, status string) ([]model.SystemOrganizationItem, error) {
	s.ensureLocationLoaded()

	raw, err := s.repo.GetOrganizations(search, status)
	if err != nil {
		return nil, err
	}

	out := make([]model.SystemOrganizationItem, 0, len(raw))
	for _, t := range raw {
		item := model.SystemOrganizationItem{
			OrganizationID:   t.OrganizationID.String,
			OrganizationCode: t.OrganizationCode.String,
			OrganizationName: t.OrganizationName.String,
			CompanyName:      t.CompanyName.String,
			CompanyAddress:   t.Address.String,
			Phone:            t.Phone.String,
			Logo:             t.Logo.String,
			PackageID:        t.PackageID.String,
		}

		// Map city
		if cityName, ok := s.citiesName[t.City.String]; ok {
			item.CompanyCity = cityName
		} else {
			item.CompanyCity = t.City.String
		}

		// Map province
		if provName, ok := s.provinceMap[t.Province.String]; ok {
			item.CompanyProvince = provName
		} else {
			item.CompanyProvince = t.Province.String
		}

		// Map package name from packages.json
		item.PackageName = s.repo.GetPackageName(t.PackageID.String)

		// Format expiry date
		if t.ExpiryDate.Valid {
			item.ExpiryDate = t.ExpiryDate.Time.Format("2006-01-02")
		}

		// Set status
		if t.ExpiryDate.Valid && !t.ExpiryDate.Time.Before(time.Now()) {
			item.Status = "active"
		} else {
			item.Status = "inactive"
		}

		out = append(out, item)
	}

	return out, nil
}

func (s *SystemService) GetUsers(search string, isActive string) ([]model.SystemUserItem, error) {
	raw, err := s.repo.GetUsers(search, isActive)
	if err != nil {
		return nil, err
	}

	out := make([]model.SystemUserItem, 0, len(raw))
	for _, t := range raw {
		item := model.SystemUserItem{
			Fullname:         t.Fullname.String,
			Phone:            t.Phone.String,
			Email:            t.Email.String,
			Avatar:           t.Avatar.String,
			OrganizationName: t.OrganizationName.String,
			OrganizationRole: int(t.OrganizationRole.Int64),
			IsActive:         t.IsActive.Valid && t.IsActive.Bool,
		}
		out = append(out, item)
	}

	return out, nil
}

