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
	transactions.Post("/expenses/submit", helper.JWTAuthorizationMiddleware(), h.SubmitExpenseTransaction)
	transactions.Post("/expenses/delete", helper.JWTAuthorizationMiddleware(), h.DeleteExpenseTransaction)
	transactions.Post("/expenses/update", helper.JWTAuthorizationMiddleware(), h.UpdateExpenseTransaction)
	transactions.Get("/labels", helper.JWTAuthorizationMiddleware(), h.ListTransactionLabels)
	transactions.Get("/types", helper.JWTAuthorizationMiddleware(), h.GetTransactionTypes)
	transactions.Get("/fleet-trip", helper.JWTAuthorizationMiddleware(), h.GetFleetTripSummary)

	///
	transactions.Post("/expenses/fleet-trip/submit", helper.JWTAuthorizationMiddleware(), h.SubmitFleetTripExpenseForm)

	apiTransactions := api.Group("/transactions")
	apiTransactions.Post("/fleet-trip/expenses/submit", helper.JWTAuthorizationMiddleware(), h.SubmitFleetTripExpenseForm)
	apiTransactions.Post("/fleet-trip/expenses/delete", helper.JWTAuthorizationMiddleware(), h.DeleteFleetTripExpenseForm)
}
