package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupUploadRoutes configures upload routes
func SetupUploadRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize upload service and handler
	uploadService := service.NewUploadService()
	userRepo := repository.NewUserRepository(db, driver)
	uploadHandler := handler.NewUploadHandler(uploadService, userRepo)

	// Upload routes
	upload := api.Group("/upload")
	upload.Post("/photo", helper.JWTAuthorizationMiddleware(), uploadHandler.UploadPhoto)
	upload.Post("/avatar", helper.JWTAuthorizationMiddleware(), uploadHandler.UploadPhoto)

	// Common upload route
	common := api.Group("/common")
	common.Post("/upload", helper.JWTAuthorizationMiddleware(), uploadHandler.UploadCommon)
	common.Post("/delete-files", helper.JWTAuthorizationMiddleware(), uploadHandler.DeleteFilesCommon)
}
