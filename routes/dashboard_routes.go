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

	dashboard.Get("/", helper.JWTAuthorizationMiddleware(), h.GetDashboard)
	dashboard.Get("/finance", helper.JWTAuthorizationMiddleware(), h.GetFinance)

	// GET /api/dashboard/summary
	dashboard.Get("/summary", helper.JWTAuthorizationMiddleware(), h.GetPartnerSummary)

	dashboard.Get("/top/destinations", helper.JWTAuthorizationMiddleware(), h.GetTopDestinations)
	dashboard.Get("/top/pickup_city", helper.JWTAuthorizationMiddleware(), h.GetTopPickupCity)
	dashboard.Get("/top/fleets", helper.JWTAuthorizationMiddleware(), h.GetTopFleets)
	dashboard.Get("/top/tour_packages", helper.JWTAuthorizationMiddleware(), h.GetTopTourPackages)
	dashboard.Get("/top/drivers", helper.JWTAuthorizationMiddleware(), h.GetTopDrivers)
	dashboard.Get("/top/customers", helper.JWTAuthorizationMiddleware(), h.GetTopCustomers)
}
