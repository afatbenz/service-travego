package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupTourPackageRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewTourPackageRepository(db, driver)
	srv := service.NewTourPackageService(repo)
	h := handler.NewTourPackageHandler(srv)

	partner := api.Group("/partner")
	partner.Get("/tour-packages", helper.JWTAuthorizationMiddleware(), h.GetTourPackages)
}
