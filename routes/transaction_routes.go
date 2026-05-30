package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupTransactionRoutes(api fiber.Router, db *sql.DB, driver string) {
	repo := repository.NewTransactionRepository(db, driver)
	srv := service.NewTransactionService(repo)
	h := handler.NewTransactionHandler(srv)

	services := api.Group("/services")
	transactions := services.Group("/transactions")

	transactions.Get("/revenue", helper.JWTAuthorizationMiddleware(), h.ListAllRevenue)
	transactions.Get("/expenses", helper.JWTAuthorizationMiddleware(), h.ListAllExpenses)
	transactions.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateManualRevenue)
	transactions.Get("/labels", helper.JWTAuthorizationMiddleware(), h.ListTransactionLabels)
	transactions.Get("/types", helper.JWTAuthorizationMiddleware(), h.GetTransactionTypes)
}
