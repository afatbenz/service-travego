package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupEmployeeRoutes(api fiber.Router, db *sql.DB, driver string) {
	orgRepo := repository.NewOrganizationRepository(db, driver)
	userRepo := repository.NewUserRepository(db, driver)
	orgService := service.NewOrganizationService(orgRepo, userRepo)
	orgHandler := handler.NewOrganizationHandler(orgService)

	services := api.Group("/services")
	employee := services.Group("/employee")

	employee.Get("/all", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeAll)
	employee.Get("/operations", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeOperations)
	employee.Post("/create", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeCreate)
	employee.Post("/update", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeUpdate)
	employee.Get("/detail/:uuid", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeDetail)
	employee.Delete("/delete/:uuid", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeDelete)
	employee.Post("/shift/schedule", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeShiftSchedule)
	employee.Post("/shift/set-schedule", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeShiftSetSchedule)
}
