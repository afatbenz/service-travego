package handler

import (
	"encoding/json"
	"fmt"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/service"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type CustomersHandler struct {
	service *service.CustomersService
}

func NewCustomersHandler(s *service.CustomersService) *CustomersHandler {
	return &CustomersHandler{service: s}
}

func (h *CustomersHandler) ListCustomers(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	customerName := c.Query("customer_name")

	items, err := h.service.ListCustomers(orgID, customerName)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Customers loaded", items)
}

func (h *CustomersHandler) CreateCustomer(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	getString := func(keys ...string) string {
		for _, k := range keys {
			v, ok := payload[k]
			if !ok || v == nil {
				continue
			}
			switch vv := v.(type) {
			case string:
				s := strings.TrimSpace(vv)
				if s != "" {
					return s
				}
			case float64:
				if vv == float64(int64(vv)) {
					return strconv.FormatInt(int64(vv), 10)
				}
				return fmt.Sprintf("%v", vv)
			default:
				s := strings.TrimSpace(fmt.Sprintf("%v", vv))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
		return ""
	}

	req := &model.CustomerCreateRequest{
		CustomerName:      getString("customer_name"),
		CustomerPhone:     getString("customer_phone", "phone"),
		CustomerTelephone: getString("customer_telephone", "telephone"),
		CustomerAddress:   getString("customer_address", "address"),
		CustomerCity:      getString("customer_city", "city_id"),
		CustomerEmail:     getString("customer_email", "email"),
		CustomerCompany:   getString("customer_company", "company_name"),
		CustomerBOD:       getString("customer_bod", "date_of_birth"),
	}

	if req.CustomerName == "" || req.CustomerPhone == "" || req.CustomerAddress == "" || req.CustomerCity == "" {
		return helper.BadRequestResponse(c, "customer_name, customer_phone, customer_address, customer_city is required")
	}

	customerID := helper.GenerateUUID()
	if err := h.service.CreateCustomer(orgID, req, customerID); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Customer created", fiber.Map{
		"customer_id": customerID,
	})
}

func (h *CustomersHandler) CustomerDetail(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}
	customerID := c.Params("customerid")
	if customerID == "" {
		return helper.BadRequestResponse(c, "customerid is required")
	}

	data, err := h.service.GetCustomerDetail(orgID, customerID)
	if err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}
	return helper.SuccessResponse(c, fiber.StatusOK, "Customer detail loaded", data)
}

func (h *CustomersHandler) UpdateCustomer(c *fiber.Ctx) error {
	orgID, _ := c.Locals("organization_id").(string)
	if orgID == "" {
		return helper.BadRequestResponse(c, "missing organization context")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return helper.BadRequestResponse(c, "invalid payload")
	}

	getString := func(keys ...string) string {
		for _, k := range keys {
			v, ok := payload[k]
			if !ok || v == nil {
				continue
			}
			switch vv := v.(type) {
			case string:
				s := strings.TrimSpace(vv)
				if s != "" {
					return s
				}
			case float64:
				if vv == float64(int64(vv)) {
					return strconv.FormatInt(int64(vv), 10)
				}
				return fmt.Sprintf("%v", vv)
			default:
				s := strings.TrimSpace(fmt.Sprintf("%v", vv))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
		return ""
	}

	customerID := getString("customer_id")
	if customerID == "" {
		return helper.BadRequestResponse(c, "customer_id is required")
	}

	req := &model.CustomerCreateRequest{
		CustomerName:      getString("customer_name"),
		CustomerPhone:     getString("customer_phone", "phone"),
		CustomerTelephone: getString("customer_telephone", "telephone"),
		CustomerAddress:   getString("customer_address", "address"),
		CustomerCity:      getString("customer_city", "city_id"),
		CustomerEmail:     getString("customer_email", "email"),
		CustomerCompany:   getString("customer_company", "company_name"),
		CustomerBOD:       getString("customer_bod", "date_of_birth"),
	}

	if req.CustomerName == "" || req.CustomerPhone == "" || req.CustomerAddress == "" || req.CustomerCity == "" {
		return helper.BadRequestResponse(c, "customer_name, customer_phone, customer_address, customer_city is required")
	}

	if err := h.service.UpdateCustomer(orgID, customerID, req); err != nil {
		code := service.GetStatusCode(err)
		return helper.SendErrorResponse(c, code, err.Error())
	}

	return helper.SuccessResponse(c, fiber.StatusOK, "Customer updated", nil)
}
