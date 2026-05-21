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

	// Route: /api/services/order/payment
	// Sesuai permintaan user menggunakan 'services' (plural)
	serviceGroup := api.Group("/services")
	paymentGroup := serviceGroup.Group("/payment/order")

	// Apply DualAuthMiddleware to ensure organization_id and user_id are present
	paymentGroup.Use(helper.DualAuthMiddleware(orgRepo))

	paymentGroup.Post("/submit", h.CreatePayment)
	paymentGroup.Post("/notification", h.PaymentNotifications)
}
