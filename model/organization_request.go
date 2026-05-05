package model

// CreateOrganizationRequest represents create organization request payload
type CreateOrganizationRequest struct {
	OrganizationCode string `json:"organization_code" validate:"omitempty"` // Optional, will be auto-generated if not provided
	OrganizationName string `json:"organization_name" validate:"required"`
	CompanyName      string `json:"company_name" validate:"required"`
	Address          string `json:"address" validate:"required"`
	City             int    `json:"city" validate:"required"`
	Province         int    `json:"province" validate:"required"`
	Phone            string `json:"phone" validate:"required"`
	Email            string `json:"email" validate:"required,email"`
	NPWPNumber       string `json:"npwp_number" validate:"omitempty"`
	OrganizationType int    `json:"organization_type" validate:"required"`
	PostalCode       string `json:"postal_code" validate:"omitempty"`
}

// JoinOrganizationRequest represents join organization request payload
type JoinOrganizationRequest struct {
	OrganizationCode string `json:"organization_code" validate:"required"`
}

// UpdateOrganizationBankAccountRequest represents update organization bank account request payload
type UpdateOrganizationBankAccountRequest struct {
	BankAccountID string `json:"bank_account_id" validate:"required"`
	Active        *bool  `json:"active"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}

// DeleteOrganizationBankAccountRequest represents delete organization bank account request payload
type DeleteOrganizationBankAccountRequest struct {
	BankAccountID string `json:"bank_account_id" validate:"required"`
}

type CreateOrganizationDivisionRequest struct {
	DivisionName string `json:"division_name" validate:"required"`
	Description  string `json:"description"`
}

type UpdateOrganizationDivisionRequest struct {
	DivisionID   string `json:"division_id" validate:"required"`
	DivisionName string `json:"division_name" validate:"required"`
	Description  string `json:"description"`
}

type DeleteOrganizationDivisionRequest struct {
	DivisionID string `json:"division_id" validate:"required"`
}

type CreateOrganizationRoleRequest struct {
	RoleName    string `json:"role_name" validate:"required"`
	Description string `json:"description"`
	DivisionID  string `json:"division_id" validate:"required"`
}

type UpdateOrganizationRoleRequest struct {
	RoleID      string `json:"role_id" validate:"required"`
	RoleName    string `json:"role_name" validate:"required"`
	Description string `json:"description"`
	DivisionID  string `json:"division_id" validate:"required"`
}

type DeleteOrganizationRoleRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}

type CreateEmployeeRequest struct {
	EmployeeID     string  `json:"employee_id" validate:"required"`
	NIK            string  `json:"nik"`
	Fullname       string  `json:"fullname" validate:"required"`
	Avatar         string  `json:"avatar"`
	Photo          string  `json:"photo"`
	Phone          string  `json:"phone"`
	BirthDate      string  `json:"birth_date"`
	DateOfBirth    string  `json:"date_of_birth"`
	Email          string  `json:"email"`
	Address        string  `json:"address"`
	AddressCity    int     `json:"address_city"`
	CityID         string  `json:"city_id"`
	JoinDate       string  `json:"join_date"`
	RoleID         string  `json:"role_id" validate:"required"`
	ContractStatus *int    `json:"contract_status"`
	ContractTypeID string  `json:"contract_type_id"`
	ResignDate     *string `json:"resign_date"`
}

type UpdateEmployeeRequest struct {
	UUID           string  `json:"uuid" validate:"required"`
	EmployeeID     string  `json:"employee_id" validate:"required"`
	NIK            string  `json:"nik"`
	Fullname       string  `json:"fullname" validate:"required"`
	Avatar         string  `json:"avatar"`
	Photo          string  `json:"photo"`
	Phone          string  `json:"phone"`
	BirthDate      string  `json:"birth_date"`
	DateOfBirth    string  `json:"date_of_birth"`
	Email          string  `json:"email"`
	Address        string  `json:"address"`
	AddressCity    int     `json:"address_city"`
	CityID         string  `json:"city_id"`
	JoinDate       string  `json:"join_date"`
	RoleID         string  `json:"role_id" validate:"required"`
	ContractStatus *int    `json:"contract_status"`
	ContractTypeID string  `json:"contract_type_id"`
	ResignDate     *string `json:"resign_date"`
	Status         *int    `json:"status"`
}

type EmployeeShiftScheduleRequest struct {
	RoleID     string `json:"role_id"`
	DivisionID string `json:"division_id"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

type EmployeeShiftSubmitItem struct {
	EmployeeID string `json:"employee_id" validate:"required"`
	ShiftDate  string `json:"shift_date" validate:"required"`
	ShiftType  int    `json:"shift_type"`
}

type EmployeeShiftSetScheduleRequest struct {
	Type       string                    `json:"type" validate:"required"`
	ShiftID    string                    `json:"shift_id"`
	EmployeeID string                    `json:"employee_id"`
	ShiftDate  string                    `json:"shift_date"`
	ShiftType  int                       `json:"shift_type"`
	Schedules  []EmployeeShiftSubmitItem `json:"schedules"`
}
