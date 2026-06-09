package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupNotificationRoutes mendaftarkan route untuk webhook publik
func SetupNotificationRoutes(app *fiber.App, db *sql.DB, driver string) {
	paymentRepo := repository.NewPaymentRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	paymentSvc := service.NewPaymentService(paymentRepo, orgRepo, nil)
	h := handler.NewPaymentHandler(paymentSvc)

	app.Post("/api/notification/payment", h.HandlePaymentNotification)
}
