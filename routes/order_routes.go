package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupOrderRoutes(api fiber.Router, db *sql.DB, driver string) {
	fleetRepo := repository.NewFleetRepository(db, driver)
	orderService := service.NewOrderService(fleetRepo)
	orderHandler := handler.NewOrderHandler(orderService)

	orderGroup := api.Group("/order")
	orderGroup.Use(helper.DualAuthMiddleware())
	orderGroup.Post("/fleet/summary", orderHandler.GetFleetOrderSummary)
}
