package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupUserRoutes configures user routes including profile update
func SetupUserRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db, driver)
	orgUserRepo := repository.NewOrganizationUserRepository(db, driver)
	orgRepo := repository.NewOrganizationRepository(db, driver)

	// Initialize services
	userService := service.NewUserService(userRepo)
	userService.SetOrganizationUserRepository(orgUserRepo)
	userService.SetOrganizationRepository(orgRepo)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)

	// User routes
	users := api.Group("/users")
	users.Get("/", userHandler.GetAllUsers)
	users.Get("/:id", userHandler.GetUserByID)
	users.Post("/", userHandler.CreateUser)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", userHandler.DeleteUser)

	// User profile routes
	profile := api.Group("/profile")
	profile.Post("/update", helper.JWTAuthorizationMiddleware(), userHandler.UpdateProfile)
	profile.Post("/update-password", helper.JWTAuthorizationMiddleware(), userHandler.UpdatePassword)
	profile.Get("/detail", helper.JWTAuthorizationMiddleware(), userHandler.GetProfile)
}
