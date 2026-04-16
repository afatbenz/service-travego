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
	// Reuse fleet repository for partner order listing handler
	fleetService := service.NewFleetService(fleetRepo)
	fleetHandler := handler.NewFleetHandler(fleetService)

	orderGroup := api.Group("/order")
	orderGroup.Use(helper.DualAuthMiddleware(orgRepo))
	orderGroup.Post("/fleet/summary", orderHandler.GetFleetOrderSummary)
	orderGroup.Post("/fleet/create", orderHandler.CreateOrder)
	orderGroup.Post("/fleet/payment", orderHandler.CreateOrderPayment)
	orderGroup.Get("/fleet/list", orderHandler.GetOrderList)
	orderGroup.Get("/fleet/detail/:encryptOrderId", orderHandler.GetOrderDetail)
	orderGroup.Get("/fleet/find/:order_id", orderHandler.FindOrder)
	orderGroup.Post("/payment-confirmation", orderHandler.ConfirmPayment)
	orderGroup.Post("/payment/confirmation/upload", orderHandler.UploadPaymentEvidence)
	orderGroup.Get("/payment-method", orderHandler.GetPaymentMethods)

	// Move /api/services/fleet/orders registration here to keep path consistent
	services := api.Group("/services")
	fleet := services.Group("/fleet")
	fleet.Get("/orders", helper.JWTAuthorizationMiddleware(), fleetHandler.GetPartnerOrderList)
	orderServices := services.Group("/order")
	orderServices.Post("/payment", helper.JWTAuthorizationMiddleware(), orderHandler.CreateServiceOrderPayment)
	orderServices.Post("/payment-history", helper.JWTAuthorizationMiddleware(), orderHandler.GetServiceOrderPaymentHistory)
}
