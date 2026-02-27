package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupDashboardRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewDashboardRepository(db, driver)
	srv := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(srv)

	dashboard := api.Group("/dashboard") // This is inside /api group because it's passed 'api' router which is app.Group("/api")
	partner := dashboard.Group("/partner")
	
	// GET /api/dashboard/partner/summary
	partner.Get("/summary", helper.JWTAuthorizationMiddleware(), h.GetPartnerSummary)
}
