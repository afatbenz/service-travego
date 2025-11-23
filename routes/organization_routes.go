package routes

import (
	"database/sql"
	"service-travego/configs"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupOrganizationRoutes configures organization routes
func SetupOrganizationRoutes(api fiber.Router, db *sql.DB, driver string, cfg *configs.Config) {
	// Initialize repositories
	orgRepo := repository.NewOrganizationRepository(db, driver)
	orgUserRepo := repository.NewOrganizationUserRepository(db, driver)
	userRepo := repository.NewUserRepository(db, driver)
	orgTypeRepo := repository.NewOrganizationTypeRepository(db, driver)

	// Initialize services
	orgService := service.NewOrganizationService(orgRepo, userRepo)
	orgService.SetOrganizationUserRepository(orgUserRepo)
	orgJoinService := service.NewOrganizationJoinService(orgRepo, orgUserRepo, userRepo, &cfg.Email)
	orgTypeService := service.NewOrganizationTypeService(orgTypeRepo)

	// Initialize handlers
	orgHandler := handler.NewOrganizationHandler(orgService)
	orgHandler.SetJoinService(orgJoinService)
	orgHandler.SetOrganizationTypeService(orgTypeService)

	// Organization routes
	organization := api.Group("/organization")
	organization.Post("/create", helper.JWTAuthorizationMiddleware(), orgHandler.CreateOrganization)
	organization.Post("/join", helper.JWTAuthorizationMiddleware(), orgHandler.JoinOrganization)
	organization.Get("/types", orgHandler.GetOrganizationTypes)
}
