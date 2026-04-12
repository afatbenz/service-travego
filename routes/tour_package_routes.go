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

	services := api.Group("/services")
	tourPackages := services.Group("/tour-packages")
	tourPackages.Get("/list", helper.JWTAuthorizationMiddleware(), h.GetTourPackages)
	tourPackages.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateTourPackage)
	tourPackages.Post("/update", helper.JWTAuthorizationMiddleware(), h.UpdateTourPackage)
	tourPackages.Post("/detail", helper.JWTAuthorizationMiddleware(), h.TourPackageDetail)
	tourPackages.Post("/:packageid", helper.JWTAuthorizationMiddleware(), h.DeleteTourPackage)
}
