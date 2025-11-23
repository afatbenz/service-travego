package routes

import (
	"service-travego/configs"
	"service-travego/helper"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(app *fiber.App, cfg *configs.Config) {
	// Initialize database
	db, err := helper.InitDatabase(cfg)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	// API routes
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return helper.SuccessResponse(c, fiber.StatusOK, "Service is running", fiber.Map{
			"status": "ok",
		})
	})

	// Initialize Redis
	_, err = helper.InitRedis(&cfg.Redis)
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	// Setup route groups
	SetupGeneralRoutes(api)
	SetupAuthRoutes(api, db, cfg.Database.Driver, cfg)
	SetupBookingRoutes(api)
	SetupOrganizationRoutes(api, db, cfg.Database.Driver, cfg)
	SetupUserRoutes(api, db, cfg.Database.Driver)
	SetupUploadRoutes(api)
}
