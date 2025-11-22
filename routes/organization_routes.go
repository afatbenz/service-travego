package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupOrganizationRoutes configures organization routes
func SetupOrganizationRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize repositories
	orgRepo := repository.NewOrganizationRepository(db, driver)
	userRepo := repository.NewUserRepository(db, driver)

	// Initialize services
	orgService := service.NewOrganizationService(orgRepo, userRepo)

	// Initialize handlers
	orgHandler := handler.NewOrganizationHandler(orgService)

	// Organization routes
	organization := api.Group("/organization")
	organization.Post("/", orgHandler.CreateOrganization)
}
