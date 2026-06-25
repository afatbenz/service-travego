package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupPrintManagementRoutes(api fiber.Router, db *sql.DB, driver string) {
	orgRepo := repository.NewOrganizationRepository(db, driver)
	repo := repository.NewPrintManagementRepository(db, driver)
	srv := service.NewPrintManagementService(repo)
	h := handler.NewPrintManagementHandler(srv)

	services := api.Group("/services")
	services.Use(helper.DualAuthMiddleware(orgRepo))
	pm := services.Group("/print-management")
	pm.Post("/fleet/invoice", h.GenerateFleetInvoiceDocument)
	pm.Post("/fleet/order", h.GenerateOrderFleetDocument)
	pm.Get("/fleet/trips/:schedule_number", h.GenerateFleetTripsDocument)

	pm.Post("/subscription", h.GenerateSubscriptionDocument)
}
