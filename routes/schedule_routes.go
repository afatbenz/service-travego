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
	h := handler.NewScheduleHandler(srv, db, driver)

	services := api.Group("/services")
	schedule := services.Group("/schedule")
	schedule.Post("/create", helper.JWTAuthorizationMiddleware(), h.Create)
	schedule.Post("/update", helper.JWTAuthorizationMiddleware(), h.Update)
	schedule.Get("/fleet", helper.JWTAuthorizationMiddleware(), h.GetFleetSchedule)
	schedule.Get("/fleet-trip/detail/:schedule_number", helper.JWTAuthorizationMiddleware(), h.GetFleetTripDetail)
	schedule.Post("/fleet-trip/update", helper.JWTAuthorizationMiddleware(), h.UpdateFleetTrip)
	schedule.Post("/fleet/availibility", helper.JWTAuthorizationMiddleware(), h.GetFleetAvailability)
	schedule.Post("/daily-availibility/fleet", helper.JWTAuthorizationMiddleware(), h.GetDailyAvailabilityFleet)
	schedule.Post("/daily-availibility/fleet-unit", helper.JWTAuthorizationMiddleware(), h.GetDailyAvailabilityFleetUnit)
	schedule.Get("/fleet-units/availibility", helper.JWTAuthorizationMiddleware(), h.GetScheduleFleetUnitAvailability)
	schedule.Get("/operations/availibility", helper.JWTAuthorizationMiddleware(), h.GetScheduleOperationAvailability)
	schedule.Get("/fleet/detail", helper.JWTAuthorizationMiddleware(), h.GetScheduleDetailByDate)
	schedule.Get("/detail/:order_id", helper.JWTAuthorizationMiddleware(), h.GetScheduleDetail)
	schedule.Get("/types", helper.JWTAuthorizationMiddleware(), h.GetScheduleTypes)
}
