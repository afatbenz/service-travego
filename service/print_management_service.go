package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"service-travego/model"
	"service-travego/repository"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/skip2/go-qrcode"
)

type PrintManagementService struct {
	repo *repository.PrintManagementRepository

	locationOnce sync.Once
	cities       map[string]string
	provinces    map[string]string

	bankOnce  sync.Once
	bankNames map[string]string

	paymentOnce         sync.Once
	paymentTypeLabels   map[int]string
	paymentMethodLabels map[int]string
}

func NewPrintManagementService(repo *repository.PrintManagementRepository) *PrintManagementService {
	return &PrintManagementService{repo: repo}
}

func (s *PrintManagementService) ensureLocationsLoaded() {
	s.locationOnce.Do(func() {
		f, err := os.Open("config/location.json")
		if err != nil {
			s.cities = map[string]string{}
			s.provinces = map[string]string{}
			return
		}
		defer f.Close()

		var loc model.Location
		if err := json.NewDecoder(f).Decode(&loc); err != nil {
			s.cities = map[string]string{}
			s.provinces = map[string]string{}
			return
		}

		cm := make(map[string]string, len(loc.Cities))
		for _, c := range loc.Cities {
			cm[c.ID] = c.Name
		}
		pm := make(map[string]string, len(loc.Provinces))
		for _, p := range loc.Provinces {
			pm[p.ID] = p.Name
		}
		s.cities = cm
		s.provinces = pm
	})
}

func (s *PrintManagementService) ensureBankLoaded() {
	s.bankOnce.Do(func() {
		raw, err := os.ReadFile(filepath.FromSlash("configs/json/bank.json"))
		if err != nil {
			s.bankNames = map[string]string{}
			return
		}
		var items []struct {
			Name string `json:"name"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(raw, &items); err != nil {
			s.bankNames = map[string]string{}
			return
		}
		m := make(map[string]string, len(items))
		for _, it := range items {
			if it.Code != "" {
				m[it.Code] = it.Name
			}
		}
		s.bankNames = m
	})
}

func (s *PrintManagementService) ensurePaymentCommonLoaded() {
	s.paymentOnce.Do(func() {
		f, err := os.Open("config/common.json")
		if err != nil {
			s.paymentTypeLabels = map[int]string{}
			s.paymentMethodLabels = map[int]string{}
			return
		}
		defer f.Close()

		var cfg model.CommonConfig
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			s.paymentTypeLabels = map[int]string{}
			s.paymentMethodLabels = map[int]string{}
			return
		}

		pt := make(map[int]string, len(cfg.PaymentStatus))
		for _, it := range cfg.PaymentStatus {
			pt[it.ID] = it.Label
		}
		pm := make(map[int]string, len(cfg.PaymentMethod))
		for _, it := range cfg.PaymentMethod {
			pm[it.ID] = it.Label
		}
		s.paymentTypeLabels = pt
		s.paymentMethodLabels = pm
	})
}

func (s *PrintManagementService) GenerateOrderFleetPDF(organizationID, orderID string) ([]byte, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}
	if strings.TrimSpace(organizationID) == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "missing organization context")
	}

	org, err := s.repo.GetOrganizationInfo(organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "organization not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch organization")
	}

	order, err := s.repo.GetFleetOrderInfo(orderID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "order not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch order")
	}

	customer, err := s.repo.GetCustomerInfo(orderID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch customer")
	}
	if err == sql.ErrNoRows {
		customer = &repository.PrintCustomerInfo{}
	}

	items, err := s.repo.GetFleetOrderItems(orderID, organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch order items")
	}

	addons, err := s.repo.GetFleetOrderAddons(orderID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch addons")
	}
	addonsByItem := make(map[string][]repository.PrintFleetOrderAddon)
	for _, a := range addons {
		if strings.TrimSpace(a.OrderItemID) == "" {
			continue
		}
		addonsByItem[a.OrderItemID] = append(addonsByItem[a.OrderItemID], a)
	}
	if len(items) == 1 {
		ordKey := strings.TrimSpace(orderID)
		itemKey := strings.TrimSpace(items[0].OrderItemID)
		if ordKey != "" && itemKey != "" && ordKey != itemKey {
			if v, ok := addonsByItem[ordKey]; ok && len(v) > 0 {
				addonsByItem[itemKey] = append(addonsByItem[itemKey], v...)
				delete(addonsByItem, ordKey)
			}
		}
	}

	bank, err := s.repo.GetOrganizationBankAccount(organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch bank account")
	}
	if err == sql.ErrNoRows {
		bank = &repository.PrintOrganizationBank{}
	}

	s.ensureLocationsLoaded()
	s.ensureBankLoaded()

	companyCityLabel := s.cities[org.CompanyCity]
	if companyCityLabel == "" {
		companyCityLabel = org.CompanyCity
	}
	companyProvinceLabel := s.provinces[org.CompanyProvince]
	if companyProvinceLabel == "" {
		companyProvinceLabel = org.CompanyProvince
	}
	customerCityLabel := s.cities[customer.CustomerCity]
	if customerCityLabel == "" {
		customerCityLabel = customer.CustomerCity
	}
	pickupCityLabel := s.cities[order.PickupCityID]
	if pickupCityLabel == "" {
		pickupCityLabel = order.PickupCityID
	}

	var subtotalFleet float64
	var totalAdditionalFee float64
	var totalDiscount float64
	fleetRows, rowSubtotals := buildFleetRows(items, addonsByItem)
	for _, v := range rowSubtotals {
		subtotalFleet += v
	}
	for _, it := range items {
		totalAdditionalFee += it.AdditionalAmount
		totalDiscount += it.FleetDiscount
	}
	totalLineItems := len(items)
	termsPageBreak := ""
	paymentPageBreak := ""
	pageClass := ""
	bottomPackStart := ""
	bottomPackEnd := ""
	if totalLineItems < 3 {
		termsPageBreak = `<div class="page-break"></div>`
		pageClass = "bottom-pack"
		bottomPackStart = `<div class="bottom-anchor">`
		bottomPackEnd = `</div>`
	}
	if totalLineItems > 3 && totalLineItems < 7 {
		paymentPageBreak = `<div class="page-break"></div>`
	}
	if totalLineItems >= 7 {
		termsPageBreak = `<div class="page-break"></div>`
	}

	totalAmount := subtotalFleet + totalAdditionalFee - totalDiscount
	if totalAmount < 0 {
		totalAmount = 0
	}
	minimumPayment := totalAmount * 0.2
	remainingAmount := totalAmount - minimumPayment
	if remainingAmount < 0 {
		remainingAmount = 0
	}

	fullPaymentDue, dpDue := computeDueDates(order.CreatedAt, order.StartDate)

	invoiceID := order.InvoiceIDCandidate
	ts := time.Now().Unix()
	qrPayload := fmt.Sprintf("%s|%s|%d", orderID, invoiceID, ts)
	qrPNG, err := qrcode.Encode(qrPayload, qrcode.Medium, 256)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to generate qr")
	}
	qrDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)

	bankName := s.bankNames[bank.BankCode]
	if bankName == "" {
		bankName = bank.BankCode
	}

	tplPath := filepath.FromSlash("docs/print/template/order.html")
	rawTpl, err := os.ReadFile(tplPath)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to read template")
	}

	companyName := org.CompanyName
	if strings.TrimSpace(companyName) == "" {
		companyName = org.OrganizationName
	}
	customerCompany := strings.TrimSpace(customer.CustomerCompany)
	if customerCompany == "" {
		customerCompany = "-"
	}

	additionalRequest := strings.TrimSpace(order.AdditionalRequest)
	if additionalRequest == "" {
		additionalRequest = "Tidak ada permintaan khusus"
	}

	companyLogoURL, companyLogoBase := resolveAssetURL(org.CompanyWebsite, org.CompanyLogo)
	if shouldLogDev() {
		log.Printf("[PRINT] company_logo raw=%q base=%q resolved=%q", strings.TrimSpace(org.CompanyLogo), companyLogoBase, companyLogoURL)
	}
	if dataURL, ok, err := fetchImageAsDataURL(companyLogoURL); ok {
		companyLogoURL = dataURL
	} else if shouldLogDev() && err != nil {
		log.Printf("[PRINT] company_logo fetch failed resolved=%q err=%v", companyLogoURL, err)
	}

	vars := map[string]string{
		"company_logo":           html.EscapeString(companyLogoURL),
		"company_name":           html.EscapeString(companyName),
		"company_address":        html.EscapeString(org.CompanyAddress),
		"company_city":           html.EscapeString(org.CompanyCity),
		"company_city_label":     html.EscapeString(companyCityLabel),
		"company_province":       html.EscapeString(org.CompanyProvince),
		"company_province_label": html.EscapeString(companyProvinceLabel),
		"company_postal_code":    html.EscapeString(org.CompanyPostal),
		"company_phone":          html.EscapeString(org.CompanyPhone),
		"company_email":          html.EscapeString(org.CompanyEmail),
		"company_website":        html.EscapeString(org.CompanyWebsite),
		"customer_name":          html.EscapeString(customer.CustomerName),
		"customer_company":       html.EscapeString(customerCompany),
		"customer_address":       html.EscapeString(customer.CustomerAddress),
		"customer_city_label":    html.EscapeString(customerCityLabel),
		"customer_phone":         html.EscapeString(customer.CustomerPhone),
		"order_id":               html.EscapeString(order.OrderID),
		"invoice_id":             html.EscapeString(invoiceID),
		"order_date":             html.EscapeString(formatDateLong(order.CreatedAt)),
		"start_date":             html.EscapeString(formatDateTravel(order.StartDate)),
		"end_date":               html.EscapeString(formatDateTravel(order.EndDate)),
		"pickup_address":         html.EscapeString(order.PickupAddress),
		"pickup_city":            html.EscapeString(pickupCityLabel),
		"destination":            html.EscapeString(pickupCityLabel),
		"pickup_time":            html.EscapeString(formatTimeHHmm(order.StartDate)),
		"additional_request":     html.EscapeString(additionalRequest),
		"special_request":        html.EscapeString(additionalRequest),
		"subtotal_fleet":         html.EscapeString(formatIDR(subtotalFleet)),
		"total_additional_fee":   html.EscapeString(formatIDR(totalAdditionalFee)),
		"total_discount":         html.EscapeString(formatIDRNegative(totalDiscount)),
		"addon_rows":             "",
		"terms_page_break":       termsPageBreak,
		"payment_page_break":     paymentPageBreak,
		"page_class":             html.EscapeString(pageClass),
		"bottom_pack_start":      bottomPackStart,
		"bottom_pack_end":        bottomPackEnd,
		"total_amount":           html.EscapeString(formatIDR(totalAmount)),
		"minimum_payment":        html.EscapeString(formatIDR(minimumPayment)),
		"remaining_amount":       html.EscapeString(formatIDR(remainingAmount)),
		"dp_due_date":            html.EscapeString(formatDateLong(dpDue)),
		"full_payment_due_date":  html.EscapeString(formatDateLong(fullPaymentDue)),
		"bank_name":              html.EscapeString(bankName),
		"bank_code":              html.EscapeString(bank.BankCode),
		"bank_account":           html.EscapeString(bank.BankAccount),
		"bank_account_name":      html.EscapeString(bank.BankAccountName),
		"fleet_items_rows":       fleetRows,
		"qr_code":                html.EscapeString(qrDataURL),
		"customer_city":          html.EscapeString(customer.CustomerCity),
		"pickup_city_id":         html.EscapeString(order.PickupCityID),
		"company_postal":         html.EscapeString(org.CompanyPostal),
		"organization_name":      html.EscapeString(org.OrganizationName),
		"company_organization":   html.EscapeString(org.OrganizationName),
		"addon_item":             "",
		"addon_price":            "",
		"fleet_name":             "",
		"fleet_qty":              "",
		"fleet_price":            "",
		"fleet_subtotal":         "",
		"fleet_facilities":       "",
	}

	htmlDoc := applyTemplateVars(string(rawTpl), vars)

	pdf, err := renderHTMLToPDF(htmlDoc)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to render pdf")
	}
	return pdf, nil
}

func (s *PrintManagementService) GenerateFleetInvoicePDF(organizationID, orderID string, invoiceNumber *string) ([]byte, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "order_id is required")
	}
	if strings.TrimSpace(organizationID) == "" {
		return nil, NewServiceError(ErrUnauthorized, http.StatusUnauthorized, "missing organization context")
	}

	org, err := s.repo.GetOrganizationInfo(organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "organization not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch organization")
	}

	order, err := s.repo.GetFleetOrderInfo(orderID, organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "order not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch order")
	}

	customer, err := s.repo.GetCustomerInfo(orderID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch customer")
	}
	if err == sql.ErrNoRows {
		customer = &repository.PrintCustomerInfo{}
	}

	items, err := s.repo.GetFleetOrderItems(orderID, organizationID)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch order items")
	}

	addons, err := s.repo.GetFleetOrderAddons(orderID, organizationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch addons")
	}
	addonsByItem := make(map[string][]repository.PrintFleetOrderAddon)
	for _, a := range addons {
		if strings.TrimSpace(a.OrderItemID) == "" {
			continue
		}
		addonsByItem[a.OrderItemID] = append(addonsByItem[a.OrderItemID], a)
	}
	if len(items) == 1 {
		ordKey := strings.TrimSpace(orderID)
		itemKey := strings.TrimSpace(items[0].OrderItemID)
		if ordKey != "" && itemKey != "" && ordKey != itemKey {
			if v, ok := addonsByItem[ordKey]; ok && len(v) > 0 {
				addonsByItem[itemKey] = append(addonsByItem[itemKey], v...)
				delete(addonsByItem, ordKey)
			}
		}
	}

	pay, err := s.repo.GetPaymentOrderForInvoice(organizationID, orderID, invoiceNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewServiceError(ErrNotFound, http.StatusNotFound, "payment not found")
		}
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to fetch payment")
	}

	s.ensureLocationsLoaded()
	s.ensurePaymentCommonLoaded()

	companyCityLabel := s.cities[org.CompanyCity]
	if companyCityLabel == "" {
		companyCityLabel = org.CompanyCity
	}
	companyProvinceLabel := s.provinces[org.CompanyProvince]
	if companyProvinceLabel == "" {
		companyProvinceLabel = org.CompanyProvince
	}
	customerCityLabel := s.cities[customer.CustomerCity]
	if customerCityLabel == "" {
		customerCityLabel = customer.CustomerCity
	}
	pickupCityLabel := s.cities[order.PickupCityID]
	if pickupCityLabel == "" {
		pickupCityLabel = order.PickupCityID
	}

	fleetRows, rowSubtotals := buildFleetInvoiceRows(items, addonsByItem)
	var subtotalFleet float64
	var totalAdditionalFee float64
	var totalDiscount float64
	var totalAddon float64
	for _, v := range rowSubtotals {
		subtotalFleet += v
	}
	for _, it := range items {
		totalAdditionalFee += it.AdditionalAmount
		totalDiscount += it.FleetDiscount
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok && len(v) > 0 {
			for _, a := range v {
				totalAddon += a.AddonPrice * float64(it.FleetQty)
			}
		}
	}

	totalAmount := subtotalFleet + totalAdditionalFee - totalDiscount
	if totalAmount < 0 {
		totalAmount = 0
	}

	paymentTypeLabel := s.paymentTypeLabels[pay.PaymentType]
	if paymentTypeLabel == "" {
		paymentTypeLabel = strconv.Itoa(pay.PaymentType)
	}
	paymentMethodLabel := s.paymentMethodLabels[pay.PaymentMethod]
	if paymentMethodLabel == "" {
		paymentMethodLabel = strconv.Itoa(pay.PaymentMethod)
	}

	paymentStatus := "BELUM LUNAS"
	if pay.RemainingAmount == 0 {
		paymentStatus = "LUNAS"
	}

	companyName := org.CompanyName
	if strings.TrimSpace(companyName) == "" {
		companyName = org.OrganizationName
	}

	customerCompany := strings.TrimSpace(customer.CustomerCompany)
	if customerCompany == "" {
		customerCompany = "-"
	}

	companyLogoURL, companyLogoBase := resolveAssetURL(org.CompanyWebsite, org.CompanyLogo)
	if shouldLogDev() {
		log.Printf("[PRINT] company_logo raw=%q base=%q resolved=%q", strings.TrimSpace(org.CompanyLogo), companyLogoBase, companyLogoURL)
	}
	if dataURL, ok, err := fetchImageAsDataURL(companyLogoURL); ok {
		companyLogoURL = dataURL
	} else if shouldLogDev() && err != nil {
		log.Printf("[PRINT] company_logo fetch failed resolved=%q err=%v", companyLogoURL, err)
	}

	inv := strings.TrimSpace(pay.InvoiceNumber)
	if inv == "" {
		cnt, err := s.repo.CountPaymentOrdersByOrganization(organizationID)
		if err == nil {
			inv = generateInvoiceNumber(1, time.Now(), cnt+1)
		}
		if inv == "" {
			inv = "-"
		}
	}

	tplPath := filepath.FromSlash("docs/print/template/fleet_invoice.html")
	rawTpl, err := os.ReadFile(tplPath)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to read template")
	}

	vars := map[string]string{
		"company_logo":       html.EscapeString(companyLogoURL),
		"company_name":       html.EscapeString(companyName),
		"company_address":    html.EscapeString(org.CompanyAddress),
		"company_city":       html.EscapeString(companyCityLabel),
		"company_province":   html.EscapeString(companyProvinceLabel),
		"company_phone":      html.EscapeString(org.CompanyPhone),
		"company_email":      html.EscapeString(org.CompanyEmail),
		"company_website":    html.EscapeString(org.CompanyWebsite),
		"invoice_number":     html.EscapeString(inv),
		"invoice_date":       html.EscapeString(formatDateTimeLong(pay.CreatedAt)),
		"order_number":       html.EscapeString(order.OrderID),
		"order_date":         html.EscapeString(formatDateLong(order.CreatedAt)),
		"payment_status":     html.EscapeString(paymentStatus),
		"customer_name":      html.EscapeString(customer.CustomerName),
		"customer_company":   html.EscapeString(customerCompany),
		"customer_phone":     html.EscapeString(customer.CustomerPhone),
		"customer_email":     html.EscapeString(customer.CustomerEmail),
		"start_date":         html.EscapeString(formatDateTravel(order.StartDate)),
		"end_date":           html.EscapeString(formatDateTravel(order.EndDate)),
		"destination":        html.EscapeString(pickupCityLabel),
		"pickup_address":     html.EscapeString(order.PickupAddress),
		"pickup_city":        html.EscapeString(pickupCityLabel),
		"fleet_items_rows":   fleetRows,
		"additional_charges": html.EscapeString(formatNumberIDR(totalAdditionalFee)),
		"total_addon":        html.EscapeString(formatNumberIDR(totalAddon)),
		"total_discount":     html.EscapeString(formatNumberIDR(totalDiscount)),
		"total_amount":       html.EscapeString(formatNumberIDR(totalAmount)),
		"payment_type":       html.EscapeString(paymentTypeLabel),
		"payment_amount":     html.EscapeString(formatNumberIDR(pay.PaymentAmount)),
		"remaining_amount":   html.EscapeString(formatNumberIDR(pay.RemainingAmount)),
		"payment_method":     html.EscapeString(paymentMethodLabel),
		"current_date":       html.EscapeString(formatDateLong(time.Now())),
	}

	htmlDoc := applyTemplateVars(string(rawTpl), vars)
	pdf, err := renderHTMLToPDF(htmlDoc)
	if err != nil {
		return nil, NewServiceError(ErrInternalServer, http.StatusInternalServerError, "failed to render pdf")
	}
	return pdf, nil
}

func applyTemplateVars(tpl string, vars map[string]string) string {
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
	return re.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		key := sub[1]
		if v, ok := vars[key]; ok {
			return v
		}
		return ""
	})
}

func buildFleetRows(items []repository.PrintFleetOrderItem, addonsByItem map[string][]repository.PrintFleetOrderAddon) (string, []float64) {
	if len(items) == 0 {
		return `<tr><td class="c" style="color:var(--muted)">1</td><td><strong>-</strong></td><td class="c">0</td><td class="r">Rp 0</td><td class="r"><strong>Rp 0</strong></td></tr>`, []float64{0}
	}

	var b strings.Builder
	subtotals := make([]float64, 0, len(items))
	for i, it := range items {
		addonSum := 0.0
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok {
			for _, a := range v {
				addonSum += a.AddonPrice
			}
		}
		sub := (it.FleetPrice * float64(it.FleetQty)) + (addonSum * float64(it.FleetQty))
		if sub < 0 {
			sub = 0
		}
		subtotals = append(subtotals, sub)
		b.WriteString("<tr>")
		b.WriteString(`<td class="c" style="color:var(--muted)">`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("</td>")
		b.WriteString("<td><strong>")
		b.WriteString(html.EscapeString(it.FleetName))
		b.WriteString("</strong>")
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok && len(v) > 0 {
			for _, a := range v {
				label := strings.TrimSpace(a.AddonName)
				desc := strings.TrimSpace(a.AddonDesc)
				if desc != "" {
					label = label + " - " + desc
				}
				if strings.TrimSpace(label) == "" {
					continue
				}
				b.WriteString(`<div style="font-size:9px;opacity:0.6;margin-top:2px;">`)
				b.WriteString(html.EscapeString(label))
				b.WriteString("</div>")
			}
		}
		b.WriteString("</td>")
		b.WriteString(`<td class="c">`)
		b.WriteString(strconv.Itoa(it.FleetQty))
		b.WriteString(" unit </td>")
		b.WriteString(`<td class="r">`)
		b.WriteString(html.EscapeString(formatIDR(it.FleetPrice)))
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok && len(v) > 0 {
			for _, a := range v {
				b.WriteString(`<div style="font-size:9px;opacity:0.6;margin-top:2px;">`)
				b.WriteString(html.EscapeString(formatIDR(a.AddonPrice)))
				b.WriteString("</div>")
			}
		}
		b.WriteString("</td>")
		b.WriteString(`<td class="r"><strong>`)
		b.WriteString(html.EscapeString(formatIDR(sub)))
		b.WriteString("</strong></td>")
		b.WriteString("</tr>")
	}
	return b.String(), subtotals
}

func buildFleetInvoiceRows(items []repository.PrintFleetOrderItem, addonsByItem map[string][]repository.PrintFleetOrderAddon) (string, []float64) {
	if len(items) == 0 {
		return `<tr><td class="c">1</td><td>-</td><td class="c">0 unit</td><td class="r">Rp 0</td><td class="r"><strong>Rp 0</strong></td></tr>`, []float64{0}
	}

	var b strings.Builder
	subtotals := make([]float64, 0, len(items))
	for i, it := range items {
		addonSum := 0.0
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok {
			for _, a := range v {
				addonSum += a.AddonPrice
			}
		}
		sub := (it.FleetPrice * float64(it.FleetQty)) + (addonSum * float64(it.FleetQty))
		if sub < 0 {
			sub = 0
		}
		subtotals = append(subtotals, sub)
		b.WriteString("<tr>")
		b.WriteString(`<td class="c">`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("</td>")
		b.WriteString("<td>")
		b.WriteString(html.EscapeString(it.FleetName))
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok && len(v) > 0 {
			for _, a := range v {
				label := strings.TrimSpace(a.AddonName)
				desc := strings.TrimSpace(a.AddonDesc)
				if desc != "" {
					label = label + " | " + desc
				}
				if strings.TrimSpace(label) == "" {
					continue
				}
				b.WriteString(`<div style="font-size:11px;opacity:0.6;margin-top:2px;">`)
				b.WriteString(html.EscapeString(label))
				b.WriteString("</div>")
			}
		}
		b.WriteString("</td>")
		b.WriteString(`<td class="c">`)
		b.WriteString(strconv.Itoa(it.FleetQty))
		b.WriteString(" unit</td>")
		b.WriteString(`<td class="r">Rp `)
		b.WriteString(html.EscapeString(formatNumberIDR(it.FleetPrice)))
		if v, ok := addonsByItem[strings.TrimSpace(it.OrderItemID)]; ok && len(v) > 0 {
			for _, a := range v {
				b.WriteString(`<div style="font-size:11px;opacity:0.6;margin-top:2px;">Rp `)
				b.WriteString(html.EscapeString(formatNumberIDR(a.AddonPrice)))
				b.WriteString("</div>")
			}
		}
		b.WriteString("</td>")
		b.WriteString(`<td class="r"><strong>Rp `)
		b.WriteString(html.EscapeString(formatNumberIDR(sub)))
		b.WriteString("</strong></td>")
		b.WriteString("</tr>")
	}
	return b.String(), subtotals
}

func computeDueDates(createdAt, startDate time.Time) (time.Time, time.Time) {
	daysUntilStart := startDate.Sub(createdAt).Hours() / 24
	if daysUntilStart < 7 {
		return startDate.AddDate(0, 0, -1), createdAt.AddDate(0, 0, 1)
	}
	if daysUntilStart < 30 {
		return startDate.AddDate(0, 0, -7), createdAt.AddDate(0, 0, 3)
	}
	return startDate.AddDate(0, 0, -7), createdAt.AddDate(0, 0, 7)
}

func renderHTMLToPDF(htmlDoc string) ([]byte, error) {
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer cancelAllocator()

	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, 45*time.Second)
	defer cancelTimeout()

	var frameTree *page.FrameTree
	var pdfBuf []byte

	err := chromedp.Run(
		ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			ft, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			frameTree = ft
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if frameTree == nil {
				return fmt.Errorf("missing frame tree")
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlDoc).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return pdfBuf, nil
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006")
}

func formatDateLong(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	months := [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	mm := ""
	if int(t.Month()) >= 1 && int(t.Month()) <= 12 {
		mm = months[int(t.Month())]
	}
	if mm == "" {
		mm = t.Month().String()
	}
	return fmt.Sprintf("%02d %s %04d", t.Day(), mm, t.Year())
}

func formatDateTimeLong(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	months := [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	mm := ""
	if int(t.Month()) >= 1 && int(t.Month()) <= 12 {
		mm = months[int(t.Month())]
	}
	if mm == "" {
		mm = t.Month().String()
	}
	return fmt.Sprintf("%02d %s %04d %s", t.Day(), mm, t.Year(), t.Format("15:04"))
}

func formatDateTravel(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	months := [...]string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	mm := ""
	if int(t.Month()) >= 1 && int(t.Month()) <= 12 {
		mm = months[int(t.Month())]
	}
	if mm == "" {
		mm = t.Month().String()[:3]
	}
	return fmt.Sprintf("%02d %s %04d", t.Day(), mm, t.Year())
}

func formatTimeHHmm(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04")
}

func shouldLogDev() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return env == "" || env == "development" || env == "dev" || env == "local"
}

func normalizeBaseURL(base string) string {
	b := strings.TrimSpace(base)
	b = strings.TrimSuffix(b, "/")
	if b == "" {
		return ""
	}
	if strings.HasPrefix(b, "http://") || strings.HasPrefix(b, "https://") {
		return b
	}
	return "http://" + b
}

func resolveAssetURL(baseURL, path string) (string, string) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", ""
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "data:") {
		return p, ""
	}

	base := normalizeBaseURL(os.Getenv("APP_HOST"))
	if base == "" {
		base = normalizeBaseURL(baseURL)
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.HasPrefix(p, "/assets") {
	} else {
		p = "/assets" + p
	}

	if base == "" {
		return p, ""
	}
	return base + p, base
}

func fetchImageAsDataURL(url string) (string, bool, error) {
	u := strings.TrimSpace(url)
	if u == "" {
		return "", false, nil
	}
	if strings.HasPrefix(u, "data:") {
		return u, true, nil
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "", false, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", false, fmt.Errorf("status %d", resp.StatusCode)
	}

	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct == "" {
		ct = "image/png"
	} else if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	const maxBytes = 5 * 1024 * 1024
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", false, err
	}
	if len(b) == 0 || len(b) > maxBytes {
		return "", false, fmt.Errorf("invalid size")
	}

	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(b), true, nil
}

func formatIDR(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "Rp 0"
	}
	v := int64(math.Round(amount))
	if v < 0 {
		v = 0
	}
	return "Rp " + formatThousand(v)
}

func formatIDRNegative(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "Rp 0"
	}
	v := int64(math.Round(amount))
	if v <= 0 {
		return "Rp 0"
	}
	return "- Rp " + formatThousand(v)
}

func formatNumberIDR(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "0"
	}
	v := int64(math.Round(amount))
	if v < 0 {
		v = 0
	}
	return formatThousand(v)
}

func formatThousand(v int64) string {
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteString(".")
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func generateInvoiceNumber(orderType int, now time.Time, sequence int) string {
	if sequence < 1 {
		sequence = 1
	}
	return fmt.Sprintf("INV-%d%s-000%d", orderType, now.Format("01200602"), sequence)
}
