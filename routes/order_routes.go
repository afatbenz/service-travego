package routes

import (
	"database/sql"
	"service-travego/configs"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupOrderRoutes(api fiber.Router, db *sql.DB, driver string, cfg *configs.Config) {
	fleetRepo := repository.NewFleetRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	contentRepo := repository.NewContentRepository(db, driver)
	orderService := service.NewOrderService(fleetRepo, contentRepo, orgRepo, &cfg.Email)
	orderHandler := handler.NewOrderHandler(orderService)

	orderGroup := api.Group("/order")
	orderGroup.Use(helper.DualAuthMiddleware(orgRepo))
	orderGroup.Post("/fleet/summary", orderHandler.GetFleetOrderSummary)
	orderGroup.Post("/fleet/create", orderHandler.CreateOrder)
	orderGroup.Post("/fleet/payment", orderHandler.CreateOrderPayment)
	orderGroup.Get("/fleet/list", orderHandler.GetOrderList)
	orderGroup.Get("/fleet/detail/:encryptOrderId", orderHandler.GetOrderDetail)
	orderGroup.Get("/payment-method", orderHandler.GetPaymentMethods)
}
