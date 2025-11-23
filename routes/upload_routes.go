package routes

import (
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupUploadRoutes configures upload routes
func SetupUploadRoutes(api fiber.Router) {
	// Initialize upload service and handler
	uploadService := service.NewUploadService()
	uploadHandler := handler.NewUploadHandler(uploadService)

	// Upload routes
	upload := api.Group("/upload")
	upload.Post("/photo", helper.JWTAuthorizationMiddleware(), uploadHandler.UploadPhoto)
	upload.Post("/avatar", helper.JWTAuthorizationMiddleware(), uploadHandler.UploadPhoto)
}
