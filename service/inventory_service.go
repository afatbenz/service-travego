package service

import (
	"database/sql"
	"net/http"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"service-travego/utils"
	"time"

	"github.com/google/uuid"
)

type InventoryService struct {
	repo *repository.InventoryRepository
}

func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) GetMovementNotes(movementType int) string {
	if movementType == 1 {
		return "Item ditambahkan dari menu inventory items"
	}
	if movementType == 3 {
		return "Item dikoreksi dari menu inventory items"
	}
	return ""
}

func (s *InventoryService) GetItems(organizationID string) ([]model.InventoryItemWithLabel, error) {
	return s.repo.GetAllItems(organizationID)
}

func (s *InventoryService) GenerateItemSKU(organizationID string) (string, error) {
	return s.repo.GenerateItemSKU(organizationID)
}

func (s *InventoryService) GetItem(itemID, organizationID string) (*model.InventoryItem, error) {
	item, err := s.repo.GetItemByID(itemID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "item not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get item")
	}
	return item, nil
}

func (s *InventoryService) CreateItem(organizationID, createdBy string, req *model.CreateInventoryItemRequest) (*model.InventoryItem, error) {
	if req.ItemID != "" {
		return s.handleCreateItemWithItemID(organizationID, createdBy, req)
	}

	if req.ItemName == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "item_name is required")
	}
	if req.ItemUOM == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "item_uom is required")
	}

	itemID, err := s.repo.GetItemIDByName(req.ItemName, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			item := &model.InventoryItem{
				OrganizationID: organizationID,
				ItemName:       req.ItemName,
				ItemUOM:        req.ItemUOM,
				ItemCategory:   req.ItemCategory,
				CreatedBy:      createdBy,
				UpdatedBy:      createdBy,
			}

			if err := s.repo.CreateItem(item); err != nil {
				return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create item")
			}

			itemID = item.ItemID

			if req.GarageID != "" {
				s.repo.UpsertItemGarage(itemID, req.GarageID, organizationID, createdBy, req.Stock)

				movementType := 1
				s.repo.CreateInventoryMovement(organizationID, &model.InventoryMovement{
					MovementID:   uuid.New().String(),
					ItemID:       itemID,
					GarageID:     req.GarageID,
					Quantity:     req.Stock,
					StockBefore:  0,
					StockFinal:   req.Stock,
					MovementType: movementType,
					Notes:        s.GetMovementNotes(movementType),
					CreatedAt:    time.Now(),
					CreatedBy:    createdBy,
				})
			}

			return item, nil
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to find item")
	}

	if req.GarageID != "" {
		s.repo.UpsertItemGarage(itemID, req.GarageID, organizationID, createdBy, req.Stock)

		s.repo.CreateInventoryMovement(organizationID, &model.InventoryMovement{
			MovementID:   uuid.New().String(),
			ItemID:       itemID,
			GarageID:     req.GarageID,
			Quantity:     req.Stock,
			StockBefore:  0,
			StockFinal:   req.Stock,
			MovementType: 1,
			Notes:        s.GetMovementNotes(1),
			CreatedAt:    time.Now(),
			CreatedBy:    createdBy,
		})
	}

	item, err := s.repo.GetItemByID(itemID, organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get item")
	}

	return item, nil
}

func (s *InventoryService) handleCreateItemWithItemID(organizationID, createdBy string, req *model.CreateInventoryItemRequest) (*model.InventoryItem, error) {
	if req.MovementType == 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "movement_type is required when item_id is provided")
	}

	if req.GarageID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "garage_id is required when item_id is provided")
	}

	currentStock, err := s.repo.GetItemGarageStock(req.ItemID, req.GarageID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get garage stock")
	}

	var finalStock int
	var stockBefore int
	if err == sql.ErrNoRows {
		stockBefore = 0
		finalStock = req.Stock
	} else {
		stockBefore = currentStock
		if req.MovementType == 1 {
			finalStock = currentStock + req.Stock
		} else if req.MovementType == 2 {
			finalStock = req.Stock
		}
	}

	if err := s.repo.UpsertItemGarage(req.ItemID, req.GarageID, organizationID, createdBy, finalStock); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to upsert garage stock")
	}

	movementType := req.MovementType
	if movementType == 2 {
		movementType = 3
	}

	s.repo.CreateInventoryMovement(organizationID, &model.InventoryMovement{
		MovementID:   uuid.New().String(),
		ItemID:       req.ItemID,
		GarageID:     req.GarageID,
		Quantity:     req.Stock,
		StockBefore:  stockBefore,
		StockFinal:   finalStock,
		MovementType: movementType,
		Notes:        s.GetMovementNotes(movementType),
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
	})

	item, err := s.repo.GetItemByID(req.ItemID, organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get item")
	}

	return item, nil
}

func (s *InventoryService) UpdateItem(organizationID, updatedBy string, req *model.UpdateInventoryItemRequest) (*model.InventoryItem, error) {
	if req.ItemID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "item_id is required")
	}

	existing, err := s.repo.GetItemByID(req.ItemID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "item not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get item")
	}

	updates := map[string]interface{}{}
	if req.ItemName != "" {
		updates["item_name"] = req.ItemName
	}
	if req.ItemUOM != "" {
		updates["item_uom"] = req.ItemUOM
	}
	if req.ItemCategory != 0 {
		updates["item_category"] = req.ItemCategory
	}

	if req.GarageID != "" {
		updates["garage_id"] = req.GarageID
	}

	if len(updates) == 0 {
		return existing, nil
	}

	if err := s.repo.UpdateItem(req.ItemID, organizationID, updatedBy, updates); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update item")
	}

	existing.UpdatedBy = updatedBy
	for k, v := range updates {
		switch k {
		case "item_name":
			existing.ItemName = v.(string)
		case "item_uom":
			existing.ItemUOM = v.(string)
		case "item_category":
			existing.ItemCategory = helper.ToInt(v)
		case "garage_id":
			existing.GarageID = v.(string)
		}
	}

	return existing, nil
}

func (s *InventoryService) DeleteItem(organizationID, itemID string) error {
	return s.repo.DeleteItem(itemID, organizationID)
}

func (s *InventoryService) TransferItem(organizationID, createdBy string, req *model.TransferInventoryItemRequest) error {
	if req.ItemID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "item_id is required")
	}
	if req.GarageFrom == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "garage_from is required")
	}
	if req.GarageDestination == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "garage_destination is required")
	}
	if req.GarageFrom == req.GarageDestination {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "garage_from and garage_destination must be different")
	}
	if req.Stock <= 0 {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "stock is required")
	}

	currentStockFrom, err := s.repo.GetItemGarageStockWithGarageName(req.ItemID, req.GarageFrom, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "garage_from not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get garage_from stock")
	}

	currentStockDest, err := s.repo.GetItemGarageStockWithGarageName(req.ItemID, req.GarageDestination, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "garage_destination not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get garage_destination stock")
	}

	if req.Stock > currentStockFrom.Stock {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "Stok yang ditransfer melebihi jumlah stok yang tersedia")
	}

	if err := s.repo.TransferItemStock(organizationID, createdBy, req, currentStockFrom, currentStockDest); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "garage stock not found")
		}
		return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to transfer stock")
	}

	return nil
}

func (s *InventoryService) GetItemDetail(itemID string) (*model.InventoryItemDetail, error) {
	item, err := s.repo.GetItemDetail(itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "item not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get item detail")
	}
	return item, nil
}

func (s *InventoryService) GetItemMovements(organizationID, itemID, startDate, endDate string) ([]model.InventoryItemMovement, error) {
	return s.repo.GetItemMovements(organizationID, itemID, startDate, endDate)
}

func (s *InventoryService) GetRequests(organizationID string) ([]model.InventoryRequestWithLabel, error) {
	return s.repo.GetRequestList(organizationID)
}

func (s *InventoryService) GetRequest(requestID, organizationID string) (*model.InventoryRequestWithLabel, error) {
	req, err := s.repo.GetRequestByID(requestID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "request not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get request")
	}
	return req, nil
}

func (s *InventoryService) CreateRequest(organizationID, createdBy string, req *model.CreateInventoryRequestRequest) (*model.InventoryRequest, error) {
	if req.ItemName == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "item_name is required")
	}
	if req.GarageID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "garage_id is required")
	}
	if req.Quantity <= 0 {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "quantity is required")
	}

	itemID, err := s.repo.GetItemIDByName(req.ItemName, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			newItem := &model.InventoryItem{
				OrganizationID: organizationID,
				ItemName:       req.ItemName,
				CreatedBy:      createdBy,
				UpdatedBy:      createdBy,
			}
			if err := s.repo.CreateItem(newItem); err != nil {
				return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create item")
			}
			itemID = newItem.ItemID
		} else {
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to find item")
		}
	}

	request := &model.InventoryRequest{
		RequestNumber:  utils.GenerateRequestNumber(1),
		ItemID:         itemID,
		GarageID:       req.GarageID,
		Quantity:       req.Quantity,
		OrganizationID: organizationID,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}

	if err := s.repo.CreateRequest(request); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create request")
	}

	return request, nil
}

func (s *InventoryService) UpdateRequest(organizationID, updatedBy string, req *model.UpdateInventoryRequestRequest) error {
	if req.RequestID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "request_id is required")
	}

	action := req.Action
	if action == "" {
		action = "delete"
	}

	switch action {
	case "delete":
		return s.repo.UpdateRequestStatus(req.RequestID, organizationID, updatedBy, 0)
	case "approve":
		if err := s.repo.UpdateRequestApprove(req.RequestID, organizationID, updatedBy, time.Now()); err != nil {
			return NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to approve request")
		}
		return nil
	default:
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid action")
	}
}

func (s *InventoryService) GetOrders(organizationID string) ([]model.InventoryOrderWithDetail, error) {
	return s.repo.GetOrdersList(organizationID)
}

func (s *InventoryService) GetOrder(purchaseID, organizationID string) (*model.InventoryOrderWithDetail, error) {
	order, err := s.repo.GetOrderByPurchaseID(purchaseID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "order not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order")
	}
	return order, nil
}

func (s *InventoryService) SubmitOrder(organizationID, userID string, req *model.SubmitInventoryOrderRequest) (*model.InventoryOrderWithDetail, error) {
	if req.PurchaseID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "purchase_id is required")
	}

	order, err := s.repo.GetOrderByPurchaseID(req.PurchaseID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "order not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get order")
	}

	if req.SupplierName != "" {
		supplierID, err := s.repo.GetSupplierIDByName(req.SupplierName)
		if err != nil {
			if err == sql.ErrNoRows {
				newSupplier := &model.Supplier{
					SupplierName: req.SupplierName,
					CreatedBy:    userID,
					UpdatedBy:    userID,
				}
				if err := s.repo.CreateSupplier(newSupplier); err != nil {
					return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create supplier")
				}
				supplierID = newSupplier.SupplierID
			} else {
				return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to find supplier")
			}
		}
		if err := s.repo.UpdateOrderSupplier(req.PurchaseID, organizationID, supplierID, userID); err != nil {
			return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to update supplier")
		}
		order.SupplierID = supplierID
	}

	return order, nil
}

func (s *InventoryService) GetSuppliers(organizationID string) ([]model.Supplier, error) {
	return s.repo.GetSuppliers(organizationID)
}

func (s *InventoryService) GetSupplier(supplierID string) (*model.Supplier, error) {
	supplier, err := s.repo.GetSupplierByID(supplierID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "supplier not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to get supplier")
	}
	return supplier, nil
}

func (s *InventoryService) CreateSupplier(organizationID, createdBy string, req *model.CreateSupplierRequest) (*model.Supplier, error) {
	if req.SupplierName == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "suplier_name is required")
	}

	supplier := &model.Supplier{
		SupplierName:    req.SupplierName,
		SupplierAddress: req.Address,
		SupplierCity:    req.City,
		SupplierPhone:   req.Phone,
		SupplierEmail:   req.Email,
		CreatedBy:       createdBy,
		UpdatedBy:       createdBy,
	}

	if err := s.repo.CreateSupplier(supplier); err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to create supplier")
	}

	return supplier, nil
}

func (s *InventoryService) DeleteSupplier(organizationID, supplierID string) error {
	return s.repo.DeleteSupplier(supplierID)
}
