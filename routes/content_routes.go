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

	// Initialize org repo for auth
	orgRepo := repository.NewOrganizationRepository(db, driver)

	// Content routes
	content := api.Group("/content")
	content.Delete("/delete-list/:uuid", helper.JWTAuthorizationMiddleware(), contentHandler.DeleteListByUUID)
	content.Post("/upload", helper.JWTAuthorizationMiddleware(), contentHandler.UploadContent)
	content.Post("/update", helper.JWTAuthorizationMiddleware(), contentHandler.UpsertGeneralContent)
	content.Get("/:parent/:section_tag", helper.JWTAuthorizationMiddleware(), contentHandler.GetContentDetailByParentAndTag)
	content.Get("/:parent", helper.JWTAuthorizationMiddleware(), contentHandler.GetContentByParent)
	content.Get("", helper.DualAuthMiddleware(orgRepo), contentHandler.GetAllGeneralContent)
}
