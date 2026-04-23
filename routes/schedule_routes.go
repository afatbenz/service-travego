package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupScheduleRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewScheduleRepository(db, driver)
	srv := service.NewScheduleService(repo)
	h := handler.NewScheduleHandler(srv)

	services := api.Group("/services")
	schedule := services.Group("/schedule")
	schedule.Post("/create", helper.JWTAuthorizationMiddleware(), h.Create)
	schedule.Get("/fleet", helper.JWTAuthorizationMiddleware(), h.GetFleetSchedule)
}
