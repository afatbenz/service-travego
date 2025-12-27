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
	srv := service.NewFleetService(repo)
	h := handler.NewServiceHandler(srv)

	// Route: /api/service/fleet
	// Assuming 'api' passed here is already grounded at /api or we group it.
	// Looking at fleet_routes.go: partner := api.Group("/partner") -> services -> fleet
	// User asked for /api/service/fleet.
	// If 'api' is the root router, we can do:

	svcGroup := api.Group("/service")
	svcGroup.Use(helper.ApiKeyMiddleware())
	svcGroup.Get("/fleet", h.GetServiceFleets)
	svcGroup.Post("/fleet/detail", h.GetServiceFleetDetail)
	svcGroup.Get("/fleet/addon/:fleetid", h.GetServiceFleetAddons)
}
