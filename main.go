package main

import (
	"log"
	"os"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load environment variables first
	// You can set APP_ENV environment variable to determine which .env file to load
	// APP_ENV=production, APP_ENV=preprod, APP_ENV=development (default)
	err := helper.LoadEnv()
	if err != nil {
		log.Printf("Warning: Failed to load .env file: %v. Continuing with system environment variables.", err)
	}

	// Load configuration from JSON (will be overridden by env vars if present)
	cfg, err := configs.LoadConfig("config/app.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override config with environment variables if they exist
	configs.OverrideWithEnv(cfg)

	// Validate email configuration
	if err := configs.ValidateEmailConfig(&cfg.Email); err != nil {
		log.Fatalf("Email configuration error: %v", err)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ServerHeader: "Fiber",
		ErrorHandler: helper.ErrorHandler,
	})

	// Middleware
	app.Use(helper.TransactionIDMiddleware())
	app.Use(helper.BodyCaptureMiddleware())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		TimeFormat: "15:04:05",
		Output:     os.Stdout,
	}))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.App.AllowOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	// Setup routes
	routes.SetupRoutes(app, cfg)

	// Start server - prioritize PORT env variable (common in cloud platforms)
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.App.Port
	}
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
