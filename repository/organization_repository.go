package repository

import (
	"database/sql"
	"fmt"
	"os"
	"service-travego/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrganizationRepository struct {
	db     *sql.DB
	driver string
}

func NewOrganizationRepository(db *sql.DB, driver string) *OrganizationRepository {
	return &OrganizationRepository{
		db:     db,
		driver: driver,
	}
}

// getAssetURL returns the full URL for an asset path
func (r *OrganizationRepository) getAssetURL(path string) string {
	if path == "" {
		return path
	}

	// Only process paths that start with /assets
	if !strings.HasPrefix(path, "/assets") {
		return path
	}

	// Get APP_HOST from environment
	appHost := os.Getenv("APP_HOST")
	if appHost == "" {
		return path
	}

	// Remove trailing slash from APP_HOST if present
	appHost = strings.TrimSuffix(appHost, "/")

	// Return full URL
	return appHost + path
}

// getPlaceholder returns the appropriate placeholder for the database driver
func (r *OrganizationRepository) getPlaceholder(pos int) string {
	if r.driver == "mysql" {
		return "?"
	}
	return fmt.Sprintf("$%d", pos)
}

// FindByID retrieves an organization by ID from database
func (r *OrganizationRepository) FindByID(id string) (*model.Organization, error) {
	query := fmt.Sprintf(`
        SELECT organization_id, organization_code, organization_name, company_name, address, city, province,
               phone, email, npwp_number, organization_type, postal_code, domain_url, COALESCE(logo, ''), created_by, created_at, updated_at
        FROM organizations
        WHERE organization_id = %s
    `, r.getPlaceholder(1))

	var org model.Organization
	var npwpNumber sql.NullString
	var postalCode sql.NullString
	var domainURL sql.NullString
	var logo sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&org.ID,
		&org.OrganizationCode,
		&org.OrganizationName,
		&org.CompanyName,
		&org.Address,
		&org.City,
		&org.Province,
		&org.Phone,
		&org.Email,
		&npwpNumber,
		&org.OrganizationType,
		&postalCode,
		&domainURL,
		&logo,
		&org.CreatedBy,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err == nil {
		if npwpNumber.Valid {
			org.NPWPNumber = npwpNumber.String
		}
		if postalCode.Valid {
			org.PostalCode = postalCode.String
		}
		if domainURL.Valid {
			org.DomainURL = domainURL.String
		}
		if logo.Valid {
			org.Logo = r.getAssetURL(logo.String)
		}
	}
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No organization found with ID:", id)
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &org, nil
}

// FindByCode retrieves an organization by code from database
func (r *OrganizationRepository) FindByCode(code string) (*model.Organization, error) {
	query := fmt.Sprintf(`
        SELECT organization_id, organization_code, organization_name, company_name, address, city, province,
               phone, email, created_at, updated_at
        FROM organizations
        WHERE organization_code = %s
    `, r.getPlaceholder(1))

	var org model.Organization
	err := r.db.QueryRow(query, code).Scan(
		&org.ID,
		&org.OrganizationCode,
		&org.OrganizationName,
		&org.CompanyName,
		&org.Address,
		&org.City,
		&org.Province,
		&org.Phone,
		&org.Email,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &org, nil
}

// FindByUsername retrieves all organizations by username from database
func (r *OrganizationRepository) FindByUsername(username string) ([]model.Organization, error) {
	query := fmt.Sprintf(`
		SELECT organization_id, organization_code, organization_name, company_name, address, city, province,
		       phone, email, npwp_number, organization_type, postal_code, domain_url, created_by, created_at, updated_at
		FROM organizations
		WHERE created_by = (SELECT user_id FROM users WHERE username = %s)
		ORDER BY created_at DESC
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []model.Organization
	for rows.Next() {
		var org model.Organization
		var npwpNumber sql.NullString
		var postalCode sql.NullString
		var domainURL sql.NullString
		err := rows.Scan(
			&org.ID,
			&org.OrganizationCode,
			&org.OrganizationName,
			&org.CompanyName,
			&org.Address,
			&org.City,
			&org.Province,
			&org.Phone,
			&org.Email,
			&npwpNumber,
			&org.OrganizationType,
			&postalCode,
			&domainURL,
			&org.CreatedBy,
			&org.CreatedAt,
			&org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if npwpNumber.Valid {
			org.NPWPNumber = npwpNumber.String
		}
		if postalCode.Valid {
			org.PostalCode = postalCode.String
		}
		if domainURL.Valid {
			org.DomainURL = domainURL.String
		}
		orgs = append(orgs, org)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orgs, nil
}

// Create inserts a new organization into database
func (r *OrganizationRepository) Create(org *model.Organization) (*model.Organization, error) {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
            INSERT INTO organizations (
                organization_id, organization_code, organization_name, company_name, address,
                city, province, phone, email, npwp_number, organization_type, postal_code,
                created_by, created_at, updated_at
            )
            SELECT %s, %s, %s, %s, %s,
                   %s, %s, %s, %s, %s, %s, %s,
                   u.user_id, %s, %s
            FROM users u
            WHERE u.user_id = %s
            RETURNING created_at, updated_at
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
			r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
			r.getPlaceholder(14), r.getPlaceholder(15),
			r.getPlaceholder(13),
		)

		err := r.db.QueryRow(
			query,
			org.ID,
			org.OrganizationCode,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			sql.NullString{String: org.NPWPNumber, Valid: org.NPWPNumber != ""},
			org.OrganizationType,
			sql.NullString{String: org.PostalCode, Valid: org.PostalCode != ""},
			org.CreatedBy, // used in WHERE u.user_id = $13
			org.CreatedAt,
			org.UpdatedAt,
		).Scan(&org.CreatedAt, &org.UpdatedAt)

		if err != nil {
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
            INSERT INTO organizations (
                organization_id, organization_code, organization_name, company_name, address,
                city, province, phone, email, npwp_number, organization_type, postal_code,
                created_by, created_at, updated_at
            )
            SELECT %s, %s, %s, %s, %s,
                   %s, %s, %s, %s, %s, %s, %s,
                   u.user_id, %s, %s
            FROM users u
            WHERE u.user_id = %s
        `,
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5),
			r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9), r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12),
			r.getPlaceholder(14), r.getPlaceholder(15),
			r.getPlaceholder(13),
		)

		_, err := r.db.Exec(
			query,
			org.ID,
			org.OrganizationCode,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			sql.NullString{String: org.NPWPNumber, Valid: org.NPWPNumber != ""},
			org.OrganizationType,
			sql.NullString{String: org.PostalCode, Valid: org.PostalCode != ""},
			org.CreatedBy, // used in WHERE u.user_id = ?
			org.CreatedAt,
			org.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	}

	return org, nil
}

// GetBankAccountByID retrieves a bank account by ID and organization ID
func (r *OrganizationRepository) GetBankAccountByID(bankAccountID, organizationID string) (*model.OrganizationBankAccountResponse, error) {
	query := fmt.Sprintf(`
		SELECT 
			oba.bank_account_id, oba.bank_code, oba.account_number, oba.account_name,
			oba.merchant_id, oba.merchant_nmid, oba.merchant_mcc,
			oba.merchant_address, oba.merchant_city, oba.merchant_postal_code,
			oba.account_type, oba.created_at, oba.created_by,
			COALESCE(u.fullname, '') as created_by_fullname,
			COALESCE(bl.name, '') as bank_name,
			COALESCE(bl.icon, '') as bank_icon,
			oba.active
		FROM organization_bank_accounts oba
		LEFT JOIN users u ON oba.created_by = u.user_id
		LEFT JOIN bank_list bl ON oba.bank_code = bl.code
		WHERE oba.bank_account_id = %s AND oba.organization_id = %s
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var acc model.OrganizationBankAccountResponse
	var merchantID, merchantNMID, merchantMCC, merchantAddress, merchantCity, merchantPostalCode sql.NullString
	var accountType sql.NullInt32

	err := r.db.QueryRow(query, bankAccountID, organizationID).Scan(
		&acc.BankAccountID,
		&acc.BankCode,
		&acc.AccountNumber,
		&acc.AccountName,
		&merchantID,
		&merchantNMID,
		&merchantMCC,
		&merchantAddress,
		&merchantCity,
		&merchantPostalCode,
		&accountType,
		&acc.CreatedAt,
		&acc.CreatedBy,
		&acc.CreatedByFullName,
		&acc.BankName,
		&acc.BankIcon,
		&acc.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if merchantID.Valid && merchantID.String != "" {
		acc.BankIcon = "/assets/bank-icon/qris.png"
	}

	acc.BankIcon = r.getAssetURL(acc.BankIcon)

	if merchantID.Valid {
		acc.MerchantID = merchantID.String
	}
	if merchantNMID.Valid {
		acc.MerchantNMID = merchantNMID.String
	}
	if merchantMCC.Valid {
		acc.MerchantMCC = merchantMCC.String
	}
	if merchantAddress.Valid {
		acc.MerchantAddress = merchantAddress.String
	}
	if merchantCity.Valid {
		acc.MerchantCity = merchantCity.String
	}
	if merchantPostalCode.Valid {
		acc.MerchantPostalCode = merchantPostalCode.String
	}
	if accountType.Valid {
		acc.AccountType = model.AccountType(accountType.Int32)
	}

	return &acc, nil
}

// GetBankAccounts retrieves bank accounts for an organization
func (r *OrganizationRepository) GetBankAccounts(organizationID string) ([]model.OrganizationBankAccountResponse, error) {
	query := fmt.Sprintf(`
		SELECT 
			oba.bank_account_id, oba.bank_code, oba.account_number, oba.account_name,
			oba.merchant_id, oba.merchant_nmid, oba.merchant_mcc,
			oba.merchant_address, oba.merchant_city, oba.merchant_postal_code,
			oba.account_type, oba.created_at, oba.created_by,
			COALESCE(u.fullname, '') as created_by_fullname,
			COALESCE(bl.name, '') as bank_name,
			COALESCE(bl.icon, '') as bank_icon,
			oba.active
		FROM organization_bank_accounts oba
		LEFT JOIN users u ON oba.created_by = u.user_id
		LEFT JOIN bank_list bl ON oba.bank_code = bl.code
		WHERE oba.organization_id = %s AND oba.status = 1
		ORDER BY oba.created_at DESC
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.OrganizationBankAccountResponse
	for rows.Next() {
		var acc model.OrganizationBankAccountResponse
		var merchantID, merchantNMID, merchantMCC, merchantAddress, merchantCity, merchantPostalCode sql.NullString
		var accountType sql.NullInt32

		err := rows.Scan(
			&acc.BankAccountID,
			&acc.BankCode,
			&acc.AccountNumber,
			&acc.AccountName,
			&merchantID,
			&merchantNMID,
			&merchantMCC,
			&merchantAddress,
			&merchantCity,
			&merchantPostalCode,
			&accountType,
			&acc.CreatedAt,
			&acc.CreatedBy,
			&acc.CreatedByFullName,
			&acc.BankName,
			&acc.BankIcon,
			&acc.Active,
		)
		if err != nil {
			return nil, err
		}

		if merchantID.Valid && merchantID.String != "" {
			acc.BankIcon = "/assets/bank-icon/qris.png"
		}

		acc.BankIcon = r.getAssetURL(acc.BankIcon)

		if merchantID.Valid {
			acc.MerchantID = merchantID.String
		}
		if merchantNMID.Valid {
			acc.MerchantNMID = merchantNMID.String
		}
		if merchantMCC.Valid {
			acc.MerchantMCC = merchantMCC.String
		}
		if merchantAddress.Valid {
			acc.MerchantAddress = merchantAddress.String
		}
		if merchantCity.Valid {
			acc.MerchantCity = merchantCity.String
		}
		if merchantPostalCode.Valid {
			acc.MerchantPostalCode = merchantPostalCode.String
		}
		if accountType.Valid {
			acc.AccountType = model.AccountType(accountType.Int32)
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// CreateBankAccount inserts a new bank account into database
func (r *OrganizationRepository) CreateBankAccount(req *model.CreateOrganizationBankAccountRequest, organizationID, createdBy, createdProxy, createdIP string) error {
	bankAccountID := uuid.New().String()
	createdAt := time.Now()

	query := fmt.Sprintf(`
		INSERT INTO organization_bank_accounts (
			bank_account_id, bank_code, account_number, account_name,
			merchant_name, merchant_mcc, merchant_address, merchant_city, merchant_postal_code,
			account_type, organization_id, created_at, created_by, created_proxy, created_ip, status, active
		) VALUES (
			%s, %s, %s, %s,
			%s, %s, %s, %s, %s,
			%s, %s, %s, %s, %s, %s, 1, true
		)
	`,
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
		r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8), r.getPlaceholder(9),
		r.getPlaceholder(10), r.getPlaceholder(11), r.getPlaceholder(12), r.getPlaceholder(13), r.getPlaceholder(14), r.getPlaceholder(15),
	)

	// Helper to handle empty strings as NULL
	toNullString := func(s string) sql.NullString {
		return sql.NullString{
			String: s,
			Valid:  s != "",
		}
	}

	// Helper to handle empty strings as NULL for potential integer columns
	// Since we don't know for sure which one is integer, we can try to pass NULL if empty
	// Note: sql.NullString with Valid=false passes NULL, which is compatible with Integer columns in Postgres (if nullable)

	_, err := r.db.Exec(query,
		bankAccountID,
		req.BankCode,
		req.AccountNumber,
		req.AccountHolder, // Mapping AccountHolder to account_name
		toNullString(req.MerchantName),
		toNullString(req.MerchantMCC),
		toNullString(req.MerchantAddress),
		toNullString(req.MerchantCity),
		toNullString(req.MerchantPostalCode),
		req.AccountType,
		organizationID,
		createdAt,
		createdBy,
		createdProxy,
		createdIP,
	)

	return err
}

// Update updates an existing organization in database
func (r *OrganizationRepository) Update(org *model.Organization) (*model.Organization, error) {
	org.UpdatedAt = time.Now()

	if r.driver == "postgres" {
		query := fmt.Sprintf(`
            UPDATE organizations
            SET organization_name = %s, company_name = %s, address = %s, city = %s, province = %s,
                phone = %s, email = %s, updated_at = %s
            WHERE organization_id = %s
            RETURNING organization_code, created_at
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9))

		err := r.db.QueryRow(
			query,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UpdatedAt,
			org.ID,
		).Scan(&org.OrganizationCode, &org.CreatedAt)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, sql.ErrNoRows
			}
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`
            UPDATE organizations
            SET organization_name = %s, company_name = %s, address = %s, city = %s, province = %s,
                phone = %s, email = %s, updated_at = %s
            WHERE organization_id = %s
        `, r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4),
			r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7), r.getPlaceholder(8),
			r.getPlaceholder(9))

		_, err := r.db.Exec(
			query,
			org.OrganizationName,
			org.CompanyName,
			org.Address,
			org.City,
			org.Province,
			org.Phone,
			org.Email,
			org.UpdatedAt,
			org.ID,
		)
		if err != nil {
			return nil, err
		}
	}

	return org, nil
}

// UpdateByIDAndCode updates organization fields with mandatory and optional values, matching by organization_id and organization_code
func (r *OrganizationRepository) UpdateByIDAndCode(orgID, orgCode string, name, company, phone, address, email string, province, city *string, npwpNumber, postalCode *string, organizationType *int) error {
	updatedAt := time.Now()

	setParts := []string{
		fmt.Sprintf("organization_name = %s", r.getPlaceholder(1)),
		fmt.Sprintf("company_name = %s", r.getPlaceholder(2)),
		fmt.Sprintf("phone = %s", r.getPlaceholder(3)),
		fmt.Sprintf("address = %s", r.getPlaceholder(4)),
		fmt.Sprintf("email = %s", r.getPlaceholder(5)),
		fmt.Sprintf("updated_at = %s", r.getPlaceholder(6)),
	}

	args := []interface{}{name, company, phone, address, email, updatedAt}
	pos := 7

	if province != nil {
		setParts = append(setParts, fmt.Sprintf("province = %s", r.getPlaceholder(pos)))
		args = append(args, sql.NullString{String: *province, Valid: *province != ""})
		pos++
	}
	if city != nil {
		setParts = append(setParts, fmt.Sprintf("city = %s", r.getPlaceholder(pos)))
		args = append(args, sql.NullString{String: *city, Valid: *city != ""})
		pos++
	}
	if npwpNumber != nil {
		setParts = append(setParts, fmt.Sprintf("npwp_number = %s", r.getPlaceholder(pos)))
		args = append(args, sql.NullString{String: *npwpNumber, Valid: *npwpNumber != ""})
		pos++
	}
	if postalCode != nil {
		setParts = append(setParts, fmt.Sprintf("postal_code = %s", r.getPlaceholder(pos)))
		args = append(args, sql.NullString{String: *postalCode, Valid: *postalCode != ""})
		pos++
	}
	if organizationType != nil {
		setParts = append(setParts, fmt.Sprintf("organization_type = %s", r.getPlaceholder(pos)))
		args = append(args, sql.NullInt32{Int32: int32(*organizationType), Valid: true})
		pos++
	}

	query := fmt.Sprintf("UPDATE organizations SET %s WHERE organization_id = %s AND organization_code = %s",
		strings.Join(setParts, ", "), r.getPlaceholder(pos), r.getPlaceholder(pos+1))

	args = append(args, orgID, orgCode)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetDomainURL retrieves the domain_url for an organization
func (r *OrganizationRepository) GetDomainURL(orgID string) (string, error) {
	query := fmt.Sprintf("SELECT domain_url FROM organizations WHERE organization_id = %s", r.getPlaceholder(1))
	var domainURL sql.NullString
	err := r.db.QueryRow(query, orgID).Scan(&domainURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if domainURL.Valid {
		return domainURL.String, nil
	}
	return "", nil
}

// UpdateDomainURL updates the domain_url for an organization
func (r *OrganizationRepository) UpdateDomainURL(orgID string, domainURL string) error {
	query := fmt.Sprintf("UPDATE organizations SET domain_url = %s, updated_at = %s WHERE organization_id = %s", r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	_, err := r.db.Exec(query, domainURL, time.Now(), orgID)
	return err
}

// UpdateLogo updates the logo path for an organization
func (r *OrganizationRepository) UpdateLogo(orgID string, logoPath string) error {
	query := fmt.Sprintf("UPDATE organizations SET logo = %s, updated_at = %s WHERE organization_id = %s",
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))
	_, err := r.db.Exec(query, logoPath, time.Now(), orgID)
	return err
}

// UpdateBankAccount updates an existing bank account for an organization
func (r *OrganizationRepository) UpdateBankAccount(bankAccountID, organizationID string, active *bool, accountNumber, accountName, updatedProxy, updatedIP string) error {
	var query string
	var args []interface{}

	if active != nil {
		query = fmt.Sprintf("UPDATE organization_bank_accounts SET active = %s, updated_at = %s, updated_proxy = %s, updated_ip = %s WHERE bank_account_id = %s AND organization_id = %s",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6))
		args = append(args, *active, time.Now(), updatedProxy, updatedIP, bankAccountID, organizationID)
	} else {
		query = fmt.Sprintf("UPDATE organization_bank_accounts SET account_number = %s, account_name = %s, updated_at = %s, updated_proxy = %s, updated_ip = %s WHERE bank_account_id = %s AND organization_id = %s",
			r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3), r.getPlaceholder(4), r.getPlaceholder(5), r.getPlaceholder(6), r.getPlaceholder(7))
		args = append(args, accountNumber, accountName, time.Now(), updatedProxy, updatedIP, bankAccountID, organizationID)
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetPaymentMethods retrieves active payment methods for an organization
func (r *OrganizationRepository) GetPaymentMethods(organizationID string) (*model.PaymentMethodGroupedResponse, error) {
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(bl.icon, '') as icon,
			oba.bank_code,
			COALESCE(bl.name, '') as bank_name,
			oba.bank_account_id,
			oba.merchant_id,
			oba.merchant_name
		FROM organization_bank_accounts oba
		LEFT JOIN bank_list bl ON oba.bank_code = bl.code
		WHERE oba.organization_id = %s AND oba.status = 1 AND oba.active = true
	`, r.getPlaceholder(1))

	rows, err := r.db.Query(query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	response := &model.PaymentMethodGroupedResponse{
		Transfer: []model.PaymentMethodResponse{},
		Qris:     []model.PaymentMethodResponse{},
	}

	for rows.Next() {
		var m model.PaymentMethodResponse
		var merchantID, merchantName sql.NullString

		if err := rows.Scan(&m.Icon, &m.BankCode, &m.BankName, &m.BankAccountID, &merchantID, &merchantName); err != nil {
			return nil, err
		}

		isQris := false
		if (merchantID.Valid && merchantID.String != "") || (merchantName.Valid && merchantName.String != "") {
			isQris = true
			m.Icon = "/assets/bank-icon/qris.png"
		}

		// Process icon URL
		m.Icon = r.getAssetURL(m.Icon)

		if isQris {
			response.Qris = append(response.Qris, m)
		} else {
			response.Transfer = append(response.Transfer, m)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

// CheckBankAccountExists checks if a bank account already exists for the organization and bank code
// Returns the bank name if it exists, otherwise empty string
func (r *OrganizationRepository) CheckBankAccountExists(organizationID, bankCode string) (string, error) {
	query := fmt.Sprintf(`
		SELECT bl.name 
		FROM organization_bank_accounts oba
		JOIN bank_list bl ON oba.bank_code = bl.code
		WHERE oba.organization_id = %s AND oba.bank_code = %s AND oba.status = 1
	`, r.getPlaceholder(1), r.getPlaceholder(2))

	var bankName string
	err := r.db.QueryRow(query, organizationID, bankCode).Scan(&bankName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return bankName, nil
}

// DeleteBankAccount soft deletes a bank account (sets status to 0)
func (r *OrganizationRepository) DeleteBankAccount(bankAccountID, organizationID string) error {
	query := fmt.Sprintf("UPDATE organization_bank_accounts SET status = 0, updated_at = %s WHERE bank_account_id = %s AND organization_id = %s",
		r.getPlaceholder(1), r.getPlaceholder(2), r.getPlaceholder(3))

	result, err := r.db.Exec(query, time.Now(), bankAccountID, organizationID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
