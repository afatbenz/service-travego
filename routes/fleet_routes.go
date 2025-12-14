package routes

import (
    "database/sql"
    "service-travego/handler"
    "service-travego/helper"
    "service-travego/repository"
    "service-travego/service"

    "github.com/gofiber/fiber/v2"
)

func SetupFleetRoutes(api fiber.Router, db *sql.DB, driver string) {
    repo := repository.NewFleetRepository(db, driver)
    srv := service.NewFleetService(repo)
    h := handler.NewFleetHandler(srv)

    partner := api.Group("/partner")
    services := partner.Group("/services")
    fleet := services.Group("/fleet")
    fleet.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateFleet)
}
