package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupCustomersRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewCustomersRepository(db, driver)
	srv := service.NewCustomersService(repo)
	h := handler.NewCustomersHandler(srv)

	services := api.Group("/services")
	services.Get("/customers", helper.JWTAuthorizationMiddleware(), h.ListCustomers)
}

