package routes

import (
	"database/sql"
	"log"
	"service-travego/internal/waai"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// SetupAssistantRoutes registers assistant webhook routes under /api/assistant.
func SetupAssistantRoutes(api fiber.Router, db *sql.DB, driver string, rdb *redis.Client) {
	waaiCfg := waai.LoadConfig()
	if err := waaiCfg.Validate(); err != nil {
		log.Printf("Warning: Failed to register assistant routes: %v", err)
		return
	}

	handler := waai.NewHandler(waaiCfg, db, driver, rdb)

	assistant := api.Group("/assistant")
	assistant.Get("/webhook", handler.HandleWebhookGET)
	assistant.Post("/webhook", handler.HandleWebhookPOST)

	admin := assistant.Group("/admin")
	admin.Delete("/session/:phone", handler.ClearSessionHandler)
	admin.Get("/health", handler.HealthCheck)
}
