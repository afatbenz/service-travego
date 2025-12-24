package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupContentRoutes configures content routes
func SetupContentRoutes(api fiber.Router, db *sql.DB, driver string) {
	// Initialize repository
	contentRepo := repository.NewContentRepository(db, driver)

	// Initialize service
	contentService := service.NewContentService(contentRepo)

	// Initialize handler
	contentHandler := handler.NewContentHandler(contentService)

	// Content routes
	content := api.Group("/content")
	content.Post("/general/create", helper.JWTAuthorizationMiddleware(), contentHandler.UpsertGeneralContent)
	content.Get("/general/detail/:section_tag", helper.JWTAuthorizationMiddleware(), contentHandler.GetGeneralContent)
}
