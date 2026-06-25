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

// SetupSubscriptionRoutes configures subscription routes
func SetupSubscriptionRoutes(api fiber.Router, db *sql.DB, driver string, midtransCfg *config.MidtransConfig) {
	orgRepo := repository.NewOrganizationRepository(db, driver)
	orgUserRepo := repository.NewOrganizationUserRepository(db, driver)
	subscriptionRepo := repository.NewSubscriptionRepository(db, driver)
	paymentRepo := repository.NewPaymentRepository(db, driver)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo)
	subscriptionService.SetOrganizationUserRepository(orgUserRepo)
	subscriptionService.SetOrganizationRepository(orgRepo)
	subscriptionService.SetPaymentRepository(&paymentRepo)
	subscriptionService.SetMidtransConfig(midtransCfg)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)

	// account routes
	account := api.Group("/account")
	account.Get("/subscription", helper.JWTAuthorizationMiddleware(), subscriptionHandler.GetSubscription)
	account.Get("/subscription/history", helper.JWTAuthorizationMiddleware(), subscriptionHandler.GetSubscriptionHistory)

	subscriptionGroup := api.Group("/subscription")
	subscriptionGroup.Post("/submit", helper.JWTAuthorizationMiddleware(), subscriptionHandler.SubmitSubscription)
	subscriptionGroup.Post("/summary", helper.JWTAuthorizationMiddleware(), subscriptionHandler.GetSubscriptionSummary)
	subscriptionGroup.Get("/detail/:invoicenumber", helper.JWTAuthorizationMiddleware(), subscriptionHandler.GetSubscriptionDetailByInvoice)
}
