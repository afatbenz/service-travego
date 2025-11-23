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

// SetupAuthRoutes configures authentication routes
func SetupAuthRoutes(api fiber.Router, db *sql.DB, driver string, cfg *configs.Config) {
	// Initialize user repository
	userRepo := repository.NewUserRepository(db, driver)

	// Initialize organization user repository
	orgUserRepo := repository.NewOrganizationUserRepository(db, driver)

	// Initialize auth service and handler
	authService := service.NewAuthService(userRepo, &cfg.Email)
	authService.SetOrganizationUserRepository(orgUserRepo)
	authHandler := handler.NewAuthHandler(authService)

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/verify-otp", authHandler.VerifyOTP)
	auth.Post("/resend-otp", authHandler.ResendOTP)
	auth.Post("/login", authHandler.Login)
	auth.Post("/reset-password", authHandler.RequestResetPassword)
	auth.Post("/update-password", authHandler.UpdatePassword)

	auth.Post("/logout", func(c *fiber.Ctx) error {
		return helper.SuccessResponse(c, fiber.StatusOK, "Auth logout endpoint - to be implemented", nil)
	})

	auth.Post("/refresh", func(c *fiber.Ctx) error {
		return helper.SuccessResponse(c, fiber.StatusOK, "Auth refresh token endpoint - to be implemented", nil)
	})
}
