package service

import (
	"encoding/json"
	"os"
	"service-travego/model"
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
func (s *GeneralService) GetProvinces() ([]model.Province, error) {
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

	return location.Provinces, nil
}

// GetCities reads and returns cities from location JSON file
func (s *GeneralService) GetCities() ([]model.City, error) {
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

	return location.Cities, nil
}
