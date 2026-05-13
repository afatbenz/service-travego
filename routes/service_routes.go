package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

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
	svcGroup.Get("/fleet/addon/:fleetid", h.GetServiceFleetAddons)
	svcGroup.Get("/available-city", h.GetAvailableCities)
}
