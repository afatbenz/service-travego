package handler

import (
	"encoding/json"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type InventoryHandler struct {
	service *service.InventoryService
}

func NewInventoryHandler(s *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{service: s}
}

func (h *InventoryHandler) GetItems(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	items, err := h.service.GetItems(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Items loaded", items)
}

func (h *InventoryHandler) GenerateSKU(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	itemSKU, err := h.service.GenerateItemSKU(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "SKU generated", fiber.Map{
		"item_sku": itemSKU,
	})
}

func (h *InventoryHandler) CreateItem(c *fiber.Ctx) error {
	raw := c.Body()
	var req model.CreateInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
			if v, ok := m["item_name"].(string); ok {
				req.ItemName = v
			}
			if v, ok := m["item_uom"].(string); ok {
				req.ItemUOM = v
			}
			if v, ok := m["item_category"]; ok {
				req.ItemCategory = helper.ToInt(v)
			}
			if v, ok := m["garage_id"].(string); ok {
				req.GarageID = v
			}
			if v, ok := m["stock"]; ok {
				req.Stock = helper.ToInt(v)
			}
			if v, ok := m["movement_type"]; ok {
				req.MovementType = helper.ToInt(v)
			}
			if v, ok := m["item_sku"].(string); ok {
				req.ItemSKU = v
			}
			if v, ok := m["item_price"]; ok {
				req.ItemPrice = helper.ToFloat64(v)
			}
			if v, ok := m["transaction_type"].(string); ok {
				req.TransactionType = v
			}
			if v, ok := m["supplier_id"].(string); ok {
				req.SupplierID = v
			}
			if v, ok := m["supplier_name"].(string); ok {
				req.SupplierName = v
			}
			if v, ok := m["supplier_phone"].(string); ok {
				req.SupplierPhone = v
			}
			if v, ok := m["supplier_url"].(string); ok {
				req.SupplierURL = v
			}
			if v, ok := m["transaction_date"].(string); ok {
				req.TransactionDate = v
			}
			if v, ok := m["supplier_price"]; ok {
				req.SupplierPrice = helper.ToFloat64(v)
			}
			if v, ok := m["notes"].(string); ok {
				req.Notes = v
			}
		}
	}

	if req.GarageID == "" {
		return helper.BadRequestResponse(c, "garage_id is required")
	}
	if req.ItemCategory == 0 {
		return helper.BadRequestResponse(c, "item_category is required")
	}
	if req.ItemSKU == "" {
		return helper.BadRequestResponse(c, "item_sku is required")
	}
	if req.ItemUOM == "" {
		return helper.BadRequestResponse(c, "item_uom is required")
	}
	if req.Stock <= 0 {
		return helper.BadRequestResponse(c, "stock is required")
	}
	if req.ItemPrice <= 0 {
		return helper.BadRequestResponse(c, "item_price is required")
	}
	if req.TransactionType == "" {
		return helper.BadRequestResponse(c, "transaction_type is required")
	}
	if req.TransactionDate == "" {
		return helper.BadRequestResponse(c, "transaction_date is required")
	}

	if req.TransactionType == "2" {
		if req.SupplierID == "" && req.SupplierName == "" {
			return helper.BadRequestResponse(c, "supplier_id or supplier_name is required when transaction_type is 2")
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	item, err := h.service.CreateItem(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Item created", fiber.Map{
		"item_id": item.ItemID,
	})
}

func (h *InventoryHandler) handleCreateItemWithID(c *fiber.Ctx, req model.CreateInventoryItemRequest, userID, orgID string) error {
	if req.MovementType == 0 {
		return helper.BadRequestResponse(c, "movement_type is required when item_id is provided")
	}
	if req.GarageID == "" {
		return helper.BadRequestResponse(c, "garage_id is required when item_id is provided")
	}

	item, err := h.service.CreateItem(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Item created", fiber.Map{
		"item_id": item.ItemID,
	})
}

func (h *InventoryHandler) UpdateItem(c *fiber.Ctx) error {
	var req model.UpdateInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
			if v, ok := m["item_name"].(string); ok {
				req.ItemName = v
			}
			if v, ok := m["item_uom"].(string); ok {
				req.ItemUOM = v
			}
			if v, ok := m["item_category"]; ok {
				req.ItemCategory = helper.ToInt(v)
			}
			if v, ok := m["garage_id"].(string); ok {
				req.GarageID = v
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	item, err := h.service.UpdateItem(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Item updated", fiber.Map{
		"item": item,
	})
}

func (h *InventoryHandler) DeleteItem(c *fiber.Ctx) error {
	var req model.DeleteInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
		}
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	if err := h.service.DeleteItem(orgID, req.ItemID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Item deleted", nil)
}

func (h *InventoryHandler) TransferItem(c *fiber.Ctx) error {
	var req model.TransferInventoryItemRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
			if v, ok := m["garage_from"].(string); ok {
				req.GarageFrom = v
			}
			if v, ok := m["garage_destination"].(string); ok {
				req.GarageDestination = v
			}
			if v, ok := m["stock"]; ok {
				req.Stock = helper.ToInt(v)
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if err := h.service.TransferItem(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Stock transferred", fiber.Map{
		"item_id":            req.ItemID,
		"garage_from":        req.GarageFrom,
		"garage_destination": req.GarageDestination,
		"stock":              req.Stock,
	})
}

func (h *InventoryHandler) GetItemDetail(c *fiber.Ctx) error {
	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
		}
	}

	if req.ItemID == "" {
		return helper.BadRequestResponse(c, "item_id is required")
	}

	item, err := h.service.GetItemDetail(req.ItemID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Item detail loaded", item)
}

func (h *InventoryHandler) GetItemMovements(c *fiber.Ctx) error {
	var req model.GetItemMovementRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
			if v, ok := m["start_date"].(string); ok {
				req.StartDate = v
			}
			if v, ok := m["end_date"].(string); ok {
				req.EndDate = v
			}
		}
	}

	if req.ItemID == "" {
		return helper.BadRequestResponse(c, "item_id is required")
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	movements, err := h.service.GetItemMovements(orgID, req.ItemID, req.StartDate, req.EndDate)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Movements loaded", movements)
}

func (h *InventoryHandler) GetRequests(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	requests, err := h.service.GetRequests(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Requests loaded", requests)
}

func (h *InventoryHandler) CreateRequest(c *fiber.Ctx) error {
	var req model.CreateInventoryRequestRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["request_id"].(string); ok {
				req.RequestID = v
			}
			if v, ok := m["item_id"].(string); ok {
				req.ItemID = v
			}
			if v, ok := m["item_name"].(string); ok {
				req.ItemName = v
			}
			if v, ok := m["item_phone"].(string); ok {
				req.ItemPhone = v
			}
			if v, ok := m["item_url"].(string); ok {
				req.ItemURL = v
			}
			if v, ok := m["garage_id"].(string); ok {
				req.GarageID = v
			}
			if v, ok := m["quantity"]; ok {
				req.Quantity = helper.ToInt(v)
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if req.ItemID == "" && req.ItemName == "" {
		return helper.BadRequestResponse(c, "item_id or item_name is required")
	}
	if req.GarageID == "" {
		return helper.BadRequestResponse(c, "garage_id is required")
	}
	if req.Quantity <= 0 {
		return helper.BadRequestResponse(c, "quantity is required")
	}

	request, err := h.service.CreateRequest(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Request created", fiber.Map{
		"request_id": request.RequestID,
	})
}

func (h *InventoryHandler) GetRequestDetail(c *fiber.Ctx) error {
	var req model.UpdateInventoryRequestRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["request_id"].(string); ok {
				req.RequestID = v
			}
		}
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	request, err := h.service.GetRequest(req.RequestID, orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Request detail loaded", request)
}

func (h *InventoryHandler) UpdateRequest(c *fiber.Ctx) error {
	var req model.UpdateInventoryRequestRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["request_id"].(string); ok {
				req.RequestID = v
			}
			if v, ok := m["action"].(string); ok {
				req.Action = v
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if err := h.service.UpdateRequest(orgID, userID, &req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	msg := "Request updated successfully"
	if strings.ToLower(req.Action) == "delete" {
		msg = "Request deleted successfully"
	} else if strings.ToLower(req.Action) == "approve" {
		msg = "Request approved successfully"
	}
	return helper.SuccessResponse(c, fiber.StatusOK, msg, nil)
}

func (h *InventoryHandler) SubmitRequestOrders(c *fiber.Ctx) error {
	var req model.SubmitRequestOrderRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["request_id"].(string); ok {
				req.RequestID = v
			}
			if v, ok := m["suplier_id"].(string); ok {
				req.SupplierID = v
			}
			if v, ok := m["suplier_name"].(string); ok {
				req.SupplierName = v
			}
			if v, ok := m["suplier_phone"].(string); ok {
				req.SupplierPhone = v
			}
			if v, ok := m["item_price"]; ok {
				req.ItemPrice = helper.ToFloat64(v)
			}
			if v, ok := m["quantity"]; ok {
				req.Quantity = helper.ToInt(v)
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if req.RequestID == "" {
		return helper.BadRequestResponse(c, "request_id is required")
	}
	if req.ItemPrice <= 0 {
		return helper.BadRequestResponse(c, "item_price is required")
	}
	if req.Quantity <= 0 {
		return helper.BadRequestResponse(c, "quantity is required")
	}
	if req.SupplierID == "" && req.SupplierName == "" {
		return helper.BadRequestResponse(c, "suplier_id or suplier_name is required")
	}

	order, err := h.service.SubmitRequestOrder(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order created from request", fiber.Map{
		"purchase_id": order.PurchaseID,
	})
}

func (h *InventoryHandler) ReceiveOrder(c *fiber.Ctx) error {
	var req struct {
		PurchaseID string `json:"purchase_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["purchase_id"].(string); ok {
				req.PurchaseID = v
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	if req.PurchaseID == "" {
		return helper.BadRequestResponse(c, "purchase_id is required")
	}

	if err := h.service.ReceiveRequest(orgID, userID, &model.ReceiveInventoryOrderRequest{PurchaseID: req.PurchaseID}); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Order received successfully", nil)
}

func (h *InventoryHandler) GetOrders(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	orders, err := h.service.GetOrders(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Orders loaded", orders)
}

func (h *InventoryHandler) GetOrderDetail(c *fiber.Ctx) error {
	var req struct {
		PurchaseID string `json:"purchase_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["purchase_id"].(string); ok {
				req.PurchaseID = v
			}
		}
	}

	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	order, err := h.service.GetOrder(req.PurchaseID, orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order detail loaded", order)
}

func (h *InventoryHandler) SubmitOrder(c *fiber.Ctx) error {
	var req model.SubmitInventoryOrderRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["purchase_id"].(string); ok {
				req.PurchaseID = v
			}
			if v, ok := m["suplier_name"].(string); ok {
				req.SupplierName = v
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	order, err := h.service.SubmitOrder(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Order submitted", order)
}

func (h *InventoryHandler) GetSuppliers(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	suppliers, err := h.service.GetSuppliers(orgID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Suppliers loaded", suppliers)
}

func (h *InventoryHandler) CreateSupplier(c *fiber.Ctx) error {
	var req model.CreateSupplierRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["suplier_name"].(string); ok {
				req.SupplierName = v
			}
			if v, ok := m["suplier_address"].(string); ok {
				req.Address = v
			}
			if v, ok := m["suplier_city"]; ok {
				req.City = helper.ToInt(v)
			}
			if v, ok := m["suplier_phone"].(string); ok {
				req.Phone = v
			}
			if v, ok := m["supliter_email"].(string); ok {
				req.Email = v
			}
		}
	}

	userID, _ := c.Locals("user_id").(string)
	orgID, _ := c.Locals("organization_id").(string)
	if userID == "" || orgID == "" {
		return helper.BadRequestResponse(c, "missing user or organization context")
	}

	supplier, err := h.service.CreateSupplier(orgID, userID, &req)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Supplier created", fiber.Map{
		"suplier_id": supplier.SupplierID,
	})
}

func (h *InventoryHandler) GetSupplierDetail(c *fiber.Ctx) error {
	var req struct {
		SupplierID string `json:"suplier_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["suplier_id"].(string); ok {
				req.SupplierID = v
			}
		}
	}

	supplier, err := h.service.GetSupplier(req.SupplierID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Supplier detail loaded", supplier)
}

func (h *InventoryHandler) DeleteSupplier(c *fiber.Ctx) error {
	var req model.DeleteSupplierRequest
	if err := c.BodyParser(&req); err != nil {
		raw := c.Body()
		var m map[string]interface{}
		if err2 := json.Unmarshal(raw, &m); err2 == nil {
			if v, ok := m["suplier_id"].(string); ok {
				req.SupplierID = v
			}
		}
	}

	if err := h.service.DeleteSupplier("", req.SupplierID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Supplier deleted", nil)
}
