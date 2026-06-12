package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupNotificationRoutes mendaftarkan route untuk webhook publik
func SetupNotificationRoutes(app *fiber.App, db *sql.DB, driver string) {
	paymentRepo := repository.NewPaymentRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	paymentSvc := service.NewPaymentService(paymentRepo, orgRepo, nil)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)
	notificationSvc := service.NewNotificationService(db, driver)
	notificationHandler := handler.NewNotificationHandler(notificationSvc)

	app.Post("/api/notification/payment", paymentHandler.HandlePaymentNotification)

	notifications := app.Group("/api/notifications")
	notifications.Get("/all", helper.JWTAuthorizationMiddleware(), notificationHandler.GetAllNotifications)
	notifications.Put("/read/:notification_id", helper.JWTAuthorizationMiddleware(), notificationHandler.MarkAsRead)
}
