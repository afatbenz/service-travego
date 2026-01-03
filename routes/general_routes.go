package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupGeneralRoutes configures general routes
func SetupGeneralRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize general service and handler
	generalRepo := repository.NewGeneralRepository(db, driver)
	generalService := service.NewGeneralService("config/general-config.json", "config/web-menu.json", "config/location.json", generalRepo)
	generalHandler := handler.NewGeneralHandler(generalService)

	// Fleet types service (DB-backed)
	ftRepo := repository.NewFleetTypeRepository(db)
	ftService := service.NewFleetTypeService(ftRepo)
	generalHandler.SetFleetTypeService(ftService)

	fmRepo := repository.NewFleetMetaRepository(db, driver)
	fmService := service.NewFleetMetaService(fmRepo)
	generalHandler.SetFleetMetaService(fmService)

	// General routes
	general := api.Group("/general")
	general.Get("/config", generalHandler.GetGeneralConfig)
	general.Get("/bank-list", generalHandler.GetBankList)
	general.Get("/web-menu", generalHandler.GetWebMenu)
	general.Get("/provinces", generalHandler.GetProvinces)
	general.Get("/cities", generalHandler.GetCities)
	general.Get("/fleet-types", generalHandler.GetFleetTypes)
	general.Get("/fleet-body", helper.JWTAuthorizationMiddleware(), generalHandler.GetFleetBodies)
	general.Get("/fleet-engine", helper.JWTAuthorizationMiddleware(), generalHandler.GetFleetEngines)
}
