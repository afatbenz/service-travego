package routes

import (
	"database/sql"
	"os"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func serviceBaseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return os.Getenv("APP_HOST")
}

func SetupServiceRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewFleetRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	srv := service.NewFleetService(repo)
	h := handler.NewServiceHandler(srv)

	// Route: /api/service/fleet

	svcGroup := api.Group("/service")
	svcGroup.Use(helper.DualAuthMiddleware(orgRepo))
	svcGroup.Get("/fleet", h.GetServiceFleets)
	svcGroup.Post("/fleet/detail", h.GetServiceFleetDetail)
	svcGroup.Post("/fleet/availibility", h.GetServiceFleetAvailibility)
	svcGroup.Get("/fleet/addon/:fleetid", h.GetServiceFleetAddons)
	svcGroup.Get("/available-city", h.GetAvailableCities)

	// tour packages
	tourRepo := repository.NewTourPackageRepository(db, driver)
	tourSrv := service.NewTourPackageService(tourRepo, serviceBaseURL())
	tourH := handler.NewTourPackageHandler(tourSrv)
	svcGroup.Get("/tour-packages", tourH.GetTourPackages)
	svcGroup.Post("/tour-packages/detail", tourH.TourPackageDetail)
}
