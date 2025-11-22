package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupUserRoutes configures user routes including profile update
func SetupUserRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db, driver)

	// Initialize services
	userService := service.NewUserService(userRepo)

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
	user := api.Group("/user")
	user.Put("/profile", userHandler.UpdateProfile)
}
