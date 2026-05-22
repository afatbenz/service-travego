package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func SetupMessagesRoutes(api fiber.Router, db *sql.DB, driver string) {
	orgRepo := repository.NewOrganizationRepository(db, driver)
	repo := repository.NewMessagesRepository(db, driver)
	srv := service.NewMessagesService(repo)
	h := handler.NewMessagesHandler(srv)

	group := api.Group("/messages")
	group.Use(func(c *fiber.Ctx) error {
		apiKey := c.Get("apikey")
		if apiKey == "" {
			apiKey = c.Get("ApiKey")
		}
		if apiKey == "" {
			apiKey = c.Get("APIKEY")
		}
		if apiKey == "" {
			apiKey = c.Get("api-key")
		}
		if strings.TrimSpace(apiKey) == "" {
			return helper.BadRequestResponse(c, "apikey header is required")
		}
		return c.Next()
	})
	group.Use(helper.DualAuthMiddleware(orgRepo))
	group.Post("/submit", h.SubmitMessage)
}

