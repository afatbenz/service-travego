package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupFleetUnitRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewFleetUnitRepository(db, driver)
	partnerRepo := repository.NewPartnerRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	srv := service.NewFleetUnitService(repo, partnerRepo, orgRepo)
	h := handler.NewFleetUnitHandler(srv)

	services := api.Group("/services")
	units := services.Group("/fleet-units")

	units.Get("", helper.JWTAuthorizationMiddleware(), h.List)
	units.Post("/create", helper.JWTAuthorizationMiddleware(), h.Create)
	units.Post("/update", helper.JWTAuthorizationMiddleware(), h.Update)
	units.Get("/detail/:unit_id", helper.JWTAuthorizationMiddleware(), h.Detail)
	units.Post("/revenue", helper.JWTAuthorizationMiddleware(), h.UnitRevenue)

	fleetUnits := api.Group("/fleet-units")
	fleetUnits.Post("/order/history", helper.JWTAuthorizationMiddleware(), h.OrderHistory)
}
