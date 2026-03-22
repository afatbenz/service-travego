package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"time"
)

type CustomersService struct {
	repo       *repository.CustomersRepository
	citiesName map[string]string
}

func NewCustomersService(repo *repository.CustomersRepository) *CustomersService {
	return &CustomersService{repo: repo}
}

func (s *CustomersService) ListCustomers(orgID, customerName string) ([]model.CustomerListItem, error) {
	items, err := s.repo.ListCustomers(orgID, customerName)
	if err != nil {
		return nil, err
	}
	s.ensureLocationsLoaded()
	if len(s.citiesName) == 0 {
		return items, nil
	}
	for i := range items {
		if items[i].CustomerCityID == "" {
			continue
		}
		if name, ok := s.citiesName[items[i].CustomerCityID]; ok && name != "" {
			items[i].CustomerCity = name
		} else {
			items[i].CustomerCity = items[i].CustomerCityID
		}
		items[i].CityName = items[i].CustomerCity
	}
	return items, nil
}

func (s *CustomersService) CreateCustomer(orgID string, req *model.CustomerCreateRequest, customerID string) error {
	return s.repo.CreateCustomer(orgID, req, customerID)
}

func (s *CustomersService) UpdateCustomer(orgID, customerID string, req *model.CustomerCreateRequest) error {
	if err := s.repo.UpdateCustomer(orgID, customerID, req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "customer not found")
		}
		return err
	}
	return nil
}

func (s *CustomersService) GetCustomerDetail(orgID, customerID string) (map[string]interface{}, error) {
	data, err := s.repo.GetCustomerDetail(orgID, customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "customer not found")
		}
		return nil, err
	}
	s.ensureLocationsLoaded()

	if raw, ok := data["customer_city"]; ok && raw != nil {
		cityID := ""
		switch v := raw.(type) {
		case string:
			cityID = strings.TrimSpace(v)
		case []byte:
			cityID = strings.TrimSpace(string(v))
		case int:
			cityID = strconv.Itoa(v)
		case int32:
			cityID = strconv.FormatInt(int64(v), 10)
		case int64:
			cityID = strconv.FormatInt(v, 10)
		case float64:
			cityID = strconv.FormatInt(int64(v), 10)
		default:
			cityID = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if cityID != "" {
			if name, ok := s.citiesName[cityID]; ok && name != "" {
				data["customer_city_name"] = name
			} else {
				data["customer_city_name"] = cityID
			}
		}
	}

	if raw, ok := data["customer_bod"]; ok && raw != nil {
		switch v := raw.(type) {
		case time.Time:
			data["customer_bod"] = v.Format("2006-01-02")
		case string:
			s := strings.TrimSpace(v)
			if s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					data["customer_bod"] = t.Format("2006-01-02")
				} else if t, err := time.Parse("2006-01-02", s); err == nil {
					data["customer_bod"] = t.Format("2006-01-02")
				}
			}
		case []byte:
			s := strings.TrimSpace(string(v))
			if s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					data["customer_bod"] = t.Format("2006-01-02")
				} else if t, err := time.Parse("2006-01-02", s); err == nil {
					data["customer_bod"] = t.Format("2006-01-02")
				}
			}
		}
	}
	return data, nil
}

func (s *CustomersService) ensureLocationsLoaded() {
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
	cm := make(map[string]string, len(loc.Cities))
	for _, c := range loc.Cities {
		cm[c.ID] = c.Name
	}
	s.citiesName = cm
}
