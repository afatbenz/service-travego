package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupTeamRoutes(api fiber.Router, db *sql.DB, driver string) {
	orgRepo := repository.NewOrganizationRepository(db, driver)
	userRepo := repository.NewUserRepository(db, driver)
	orgService := service.NewOrganizationService(orgRepo, userRepo)
	orgHandler := handler.NewOrganizationHandler(orgService)

	services := api.Group("/services")
	team := services.Group("/team")

	team.Get("/divisions", helper.JWTAuthorizationMiddleware(), orgHandler.TeamListDivisions)
	team.Post("/divisions/create", helper.JWTAuthorizationMiddleware(), orgHandler.TeamCreateDivision)
	team.Post("/divisions/update", helper.JWTAuthorizationMiddleware(), orgHandler.TeamUpdateDivision)
	team.Post("/divisions/delete", helper.JWTAuthorizationMiddleware(), orgHandler.TeamDeleteDivision)

	team.Get("/roles", helper.JWTAuthorizationMiddleware(), orgHandler.TeamListRoles)
	team.Post("/roles/create", helper.JWTAuthorizationMiddleware(), orgHandler.TeamCreateRole)
	team.Post("/roles/update", helper.JWTAuthorizationMiddleware(), orgHandler.TeamUpdateRole)
	team.Post("/roles/delete", helper.JWTAuthorizationMiddleware(), orgHandler.TeamDeleteRole)
}
