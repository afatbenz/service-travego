package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupSystemRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewSystemRepository(db, driver)
	srv := service.NewSystemService(repo)
	h := handler.NewSystemHandler(srv)

	dashboard := api.Group("/system") // This is inside /api group because it's passed 'api' router which is app.Group("/api")

	dashboard.Get("/summarize", helper.JWTAuthorizationMiddleware(), h.GetSystemSummarymarize)
	dashboard.Get("/assistant/device", helper.JWTAuthorizationMiddleware(), h.GetDeviceList)
	dashboard.Put("/assistant/device/:action", helper.JWTAuthorizationMiddleware(), h.UpdateDevice)
	dashboard.Get("/organizations", helper.JWTAuthorizationMiddleware(), h.GetOrganizations)
	dashboard.Get("/users", helper.JWTAuthorizationMiddleware(), h.GetUsers)
}
