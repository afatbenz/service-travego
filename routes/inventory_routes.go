package routes

import (
	"database/sql"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

func SetupInventoryRoutes(api fiber.Router, db *sql.DB, driver string, notificationService *service.NotificationService) {
	repo := repository.NewInventoryRepository(db, driver)
	srv := service.NewInventoryService(repo, notificationService)
	h := handler.NewInventoryHandler(srv)

	inventories := api.Group("/inventories")

	items := inventories.Group("/items")
	items.Get("/", helper.JWTAuthorizationMiddleware(), h.GetItems)
	items.Get("/generate-sku", helper.JWTAuthorizationMiddleware(), h.GenerateSKU)
	items.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateItem)
	items.Post("/update", helper.JWTAuthorizationMiddleware(), h.UpdateItem)
	items.Post("/delete", helper.JWTAuthorizationMiddleware(), h.DeleteItem)
	items.Post("/transfer", helper.JWTAuthorizationMiddleware(), h.TransferItem)
	items.Post("/detail", helper.JWTAuthorizationMiddleware(), h.GetItemDetail)
	items.Post("/movement", helper.JWTAuthorizationMiddleware(), h.GetItemMovements)

	request := inventories.Group("/request")
	request.Get("/list", helper.JWTAuthorizationMiddleware(), h.GetRequests)
	request.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateRequest)
	request.Post("/detail", helper.JWTAuthorizationMiddleware(), h.GetRequestDetail)
	request.Post("/update", helper.JWTAuthorizationMiddleware(), h.UpdateRequest)
	request.Post("/submit-orders", helper.JWTAuthorizationMiddleware(), h.SubmitRequestOrders)

	supliers := inventories.Group("/supliers")
	supliers.Get("/list", helper.JWTAuthorizationMiddleware(), h.GetSuppliers)
	supliers.Post("/create", helper.JWTAuthorizationMiddleware(), h.CreateSupplier)
	supliers.Post("/detail", helper.JWTAuthorizationMiddleware(), h.GetSupplierDetail)
	supliers.Post("/delete", helper.JWTAuthorizationMiddleware(), h.DeleteSupplier)

	orders := inventories.Group("/orders")
	orders.Get("/list", helper.JWTAuthorizationMiddleware(), h.GetOrders)
	orders.Post("/submit", helper.JWTAuthorizationMiddleware(), h.SubmitOrder)
	orders.Post("/detail", helper.JWTAuthorizationMiddleware(), h.GetOrderDetail)
	orders.Post("/received", helper.JWTAuthorizationMiddleware(), h.ReceiveOrder)
}
