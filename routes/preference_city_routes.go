package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupPreferenceCityRoutes(api fiber.Router, db *sql.DB, driver string) {
	prefRepo := repository.NewPreferenceCityRepository(db, driver)
	prefService := service.NewPreferenceCityService(prefRepo, "config/location.json")
	prefHandler := handler.NewPreferenceCityHandler(prefService)

	prefs := api.Group("/services/preferences")
	prefs.Use(helper.JWTAuthorizationMiddleware())

	prefs.Get("/cities", prefHandler.GetCities)
	prefs.Post("/cities/create", prefHandler.CreateCity)
	prefs.Post("/cities/update", prefHandler.UpdateCity)
	prefs.Post("/cities/delete", prefHandler.DeleteCity)
	prefs.Post("/cities/delete-types", prefHandler.DeleteTypes)
}
