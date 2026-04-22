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

	services := api.Group("/services")
	fleet := services.Group("/fleet")
	fleet.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateFleet)
	fleet.Post("/delete", helper.JWTAuthorizationMiddleware(), h.DeleteFleet)
	fleet.Post("/update", helper.JWTAuthorizationMiddleware(), h.UpdateFleet)
	fleet.Get("/list", helper.JWTAuthorizationMiddleware(), h.ListFleets)
	fleet.Post("/detail", helper.JWTAuthorizationMiddleware(), h.FleetDetail)
	fleet.Get("/addon/:fleetid", helper.JWTAuthorizationMiddleware(), h.GetFleetAddonList)
	fleet.Get("/prices/:fleetid/:typeid", helper.JWTAuthorizationMiddleware(), h.GetFleetPricesByFleetID)
	fleet.Post("/orders/create", helper.JWTAuthorizationMiddleware(), h.CreatePartnerOrder)
	fleet.Get("/order/detail/:order_id", helper.JWTAuthorizationMiddleware(), h.GetPartnerOrderDetail)
	fleet.Post("/order/detail/:order_id", helper.JWTAuthorizationMiddleware(), h.GetPartnerOrderDetail)
}
