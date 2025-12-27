package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupCheckoutRoutes(api fiber.Router, db *sql.DB, driver string) {
	fleetRepo := repository.NewFleetRepository(db, driver)
	checkoutService := service.NewCheckoutService(fleetRepo)
	checkoutHandler := handler.NewCheckoutHandler(checkoutService)

	checkoutGroup := api.Group("/checkout")
	checkoutGroup.Use(helper.DualAuthMiddleware())
	checkoutGroup.Post("/fleet/summary", checkoutHandler.GetFleetCheckoutSummary)
}
