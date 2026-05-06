package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupLeaveManagementRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewLeaveManagementRepository(db, driver)
	srv := service.NewLeaveManagementService(repo)
	h := handler.NewLeaveManagementHandler(srv)

	services := api.Group("/services")
	leave := services.Group("/leave-management")

	leave.Get("/types", helper.JWTAuthorizationMiddleware(), h.GetLeaveTypes)
	leave.Get("/list", helper.JWTAuthorizationMiddleware(), h.GetLeaveList)
}
