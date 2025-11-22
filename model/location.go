package model

// Location represents location data
type Location struct {
	Cities    []City     `json:"cities"`
	Provinces []Province `json:"provinces"`
}

// City represents a city
type City struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Province string `json:"province"`
}

// Province represents a province
type Province struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
