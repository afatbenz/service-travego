package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupPartnerRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewPartnerRepository(db, driver)
	srv := service.NewPartnerService(repo)
	h := handler.NewPartnerHandler(srv)

	services := api.Group("/services")
	partnership := services.Group("/partnership")
	operations := partnership.Group("/operations")

	operations.Get("", helper.JWTAuthorizationMiddleware(), h.List)
	operations.Post("/create", helper.JWTAuthorizationMiddleware(), h.Create)
	operations.Post("/update", helper.JWTAuthorizationMiddleware(), h.Update)
	operations.Post("/detail", helper.JWTAuthorizationMiddleware(), h.Detail)
}
