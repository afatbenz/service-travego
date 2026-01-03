package model

// GeneralConfig represents general configuration
type GeneralConfig struct {
	CompanyName string `json:"companyName"`
	Address     string `json:"address"`
	Email       string `json:"email"`
}

// MenuItem represents a menu item
type MenuItem struct {
	Title    string     `json:"title"`
	Desc     string     `json:"desc"`
	URL      string     `json:"url"`
	SubMenus []MenuItem `json:"subMenus,omitempty"`
}

// Bank represents bank information
type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// WebMenu represents web menu structure
type WebMenu struct {
	Dashboard   []MenuItem `json:"dashboard"`
	LandingPage []MenuItem `json:"landingPage"`
}
