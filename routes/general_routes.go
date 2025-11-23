package routes

import (
	"service-travego/handler"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupGeneralRoutes configures general routes
func SetupGeneralRoutes(api fiber.Router) {
	// Initialize general service and handler
	generalService := service.NewGeneralService("config/general-config.json", "config/web-menu.json", "config/location.json")
	generalHandler := handler.NewGeneralHandler(generalService)

	// General routes
	general := api.Group("/general")
	general.Get("/config", generalHandler.GetGeneralConfig)
	general.Get("/web-menu", generalHandler.GetWebMenu)
	general.Get("/provinces", generalHandler.GetProvinces)
	general.Get("/cities", generalHandler.GetCities)
}
