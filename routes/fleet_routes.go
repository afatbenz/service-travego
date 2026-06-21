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
	orgRepo := repository.NewOrganizationRepository(db, driver)
	srv := service.NewFleetService(repo)
	h := handler.NewFleetHandler(srv, orgRepo)

	services := api.Group("/services")
	fleet := services.Group("/fleet")

	fleet.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateFleet)
	fleet.Post("/delete", helper.JWTAuthorizationMiddleware(), h.DeleteFleet)
	fleet.Post("/update", helper.JWTAuthorizationMiddleware(), h.UpdateFleet)
	fleet.Post("/activate", helper.JWTAuthorizationMiddleware(), h.SetFleetActiveStatus)
	fleet.Get("/list", helper.JWTAuthorizationMiddleware(), h.ListFleets)
	fleet.Get("/availibility", helper.JWTAuthorizationMiddleware(), h.FleetAvailibility)
	fleet.Post("/detail", helper.JWTAuthorizationMiddleware(), h.FleetDetail)
	fleet.Post("/revenue", helper.JWTAuthorizationMiddleware(), h.FleetRevenue)
	fleet.Get("/addon/:fleetid", helper.JWTAuthorizationMiddleware(), h.GetFleetAddonList)
	fleet.Get("/prices/:fleetid/:typeid", helper.JWTAuthorizationMiddleware(), h.GetFleetPricesByFleetID)
	fleet.Get("/facilities", helper.JWTAuthorizationMiddleware(), h.GetFacilityList)

	// Orders
	fleet.Post("/orders/create", helper.JWTAuthorizationMiddleware(), h.CreatePartnerOrder)
	fleet.Get("/order/detail/:order_id", helper.JWTAuthorizationMiddleware(), h.GetPartnerOrderDetail)
	fleet.Post("/order/update", helper.JWTAuthorizationMiddleware(), h.UpdatePartnerOrder)
	fleet.Post("/order/delete-addon", helper.JWTAuthorizationMiddleware(), h.DeleteFleetOrderAddon)
	fleet.Post("/order/process/:processType/:order_id", helper.JWTAuthorizationMiddleware(), h.ProcessFleetOrder)
	fleet.Post("/order/cancel", helper.JWTAuthorizationMiddleware(), h.CancelPartnerOrder)
	fleet.Post("/order/cancelation-detail", helper.JWTAuthorizationMiddleware(), h.CancelPartnerOrderDetail)
}
