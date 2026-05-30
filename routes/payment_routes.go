package routes

import (
	"database/sql"
	"service-travego/config"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupPaymentRoutes mendaftarkan route untuk integrasi payment
func SetupPaymentRoutes(api fiber.Router, db *sql.DB, driver string, midtransCfg *config.MidtransConfig) {
	repo := repository.NewPaymentRepository(db, driver)
	svc := service.NewPaymentService(repo, midtransCfg)
	h := handler.NewPaymentHandler(svc)

	orgRepo := repository.NewOrganizationRepository(db, driver)

	serviceGroup := api.Group("/services")
	paymentGroup := serviceGroup.Group("/payment/order")

	paymentGroup.Use(helper.DualAuthMiddleware(orgRepo))

	paymentGroup.Post("/submit", h.CreatePayment)
	paymentGroup.Post("/notification", h.PaymentNotifications)
}
