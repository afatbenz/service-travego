package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupPricingRoutes configures pricing routes (public endpoints - requires static API key for landing page)
func SetupPricingRoutes(app *fiber.App, db *sql.DB, driver string) {
	repo := repository.NewPricingRepository(db, driver)
	srv := service.NewPricingService(repo)
	h := handler.NewPricingHandler(srv)

	app.Get("/api/services/packages/pricing", helper.StaticApiKeyMiddleware(), h.GetPackages)
	app.Get("/api/services/reviews", helper.StaticApiKeyMiddleware(), h.GetReviews)
	app.Post("/api/services/contact/submit", helper.StaticApiKeyMiddleware(), h.SubmitContact)
}
