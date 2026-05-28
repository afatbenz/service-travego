package routes

import (
	"database/sql"
	"os"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func serviceBaseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return os.Getenv("APP_HOST")
}

func SetupServiceRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewFleetRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)
	custRepo := repository.NewCustomersRepository(db, driver)
	tourRepo := repository.NewTourPackageRepository(db, driver)

	srv := service.NewFleetService(repo)
	custSrv := service.NewCustomersService(custRepo)
	tourSrv := service.NewTourPackageService(tourRepo, serviceBaseURL())

	h := handler.NewServiceHandler(srv, tourSrv, custSrv)
	tourH := handler.NewTourPackageHandler(tourSrv)

	// Print Management for Public
	pmRepo := repository.NewPrintManagementRepository(db, driver)
	pmSrv := service.NewPrintManagementService(pmRepo)
	pmH := handler.NewPrintManagementHandler(pmSrv)

	// Route: /api/service/fleet

	svcGroup := api.Group("/service")
	svcGroup.Use(helper.DualAuthMiddleware(orgRepo))
	svcGroup.Get("/fleet", h.GetServiceFleets)
	svcGroup.Post("/fleet/detail", h.GetServiceFleetDetail)
	svcGroup.Post("/fleet/availibility", h.GetServiceFleetAvailibility)
	svcGroup.Post("/fleet/order/availibility", h.OrderAvailability)
	svcGroup.Get("/fleet/addon/:fleetid", h.GetServiceFleetAddons)
	svcGroup.Get("/available-city", h.GetAvailableCities)

	// customers
	svcGroup.Post("/customer/availibility", h.CheckCustomerAvailibility)

	svcGroup.Post("/review/submit", h.SubmitReview)

	// tour packages
	svcGroup.Get("/tour-packages", tourH.GetTourPackages)
	svcGroup.Post("/tour-packages/detail", tourH.TourPackageDetail)

	// Public Print Document
	svcGroup.Post("/print/fleet/order", pmH.GenerateOrderFleetDocument)
	svcGroup.Post("/print/fleet/invoice", pmH.GenerateFleetInvoiceDocument)
}
