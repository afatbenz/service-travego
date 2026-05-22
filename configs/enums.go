package configs

import "service-travego/model"

// OrganizationRole represents the role of a user in an organization
type OrganizationRole int

const (
	OrganizationRoleAdmin OrganizationRole = 1 // ADMIN
	OrganizationRoleStaff OrganizationRole = 2 // Staff
)

// String returns the string representation of OrganizationRole
func (r OrganizationRole) String() string {
	switch r {
	case OrganizationRoleAdmin:
		return "ADMIN"
	case OrganizationRoleStaff:
		return "Staff"
	default:
		return "Unknown"
	}
}

// IsValid checks if the organization role is valid
func (r OrganizationRole) IsValid() bool {
	return r == OrganizationRoleAdmin || r == OrganizationRoleStaff
}

// Gender represents the gender of a user
type Gender int

const (
	GenderMale   Gender = 1 // Laki Laki
	GenderFemale Gender = 2 // Perempuan
)

// String returns the string representation of Gender
func (g Gender) String() string {
	switch g {
	case GenderMale:
		return "Laki Laki"
	case GenderFemale:
		return "Perempuan"
	default:
		return "Unknown"
	}
}

// IsValid checks if the gender is valid
func (g Gender) IsValid() bool {
	return g == GenderMale || g == GenderFemale
}

// UploadType represents the type of file upload
type UploadType string

const (
	UploadTypeProfileUser      UploadType = "profile-user"      // PROFILE_USER
	UploadTypeIconCompany      UploadType = "icon-company"      // ICON_COMPANY
	UploadTypeContentThumbnail UploadType = "content-thumbnail" // CONTENT_THUMBNAIL
	UploadTypeArmada           UploadType = "armada"
	UploadTypePackage          UploadType = "package"
	UploadTypeOrder            UploadType = "order"
	UploadTypeContent          UploadType = "content"
	UploadTypeEmployeePhoto    UploadType = "employee_photo"
	UploadTypePayment          UploadType = "payment"
)

// String returns the string representation of UploadType
func (u UploadType) String() string {
	return string(u)
}

// IsValid checks if the upload type is valid
func (u UploadType) IsValid() bool {
	return u == UploadTypeProfileUser || u == UploadTypeIconCompany || u == UploadTypeContentThumbnail ||
		u == UploadTypeArmada || u == UploadTypePackage || u == UploadTypeOrder || u == UploadTypeContent || u == UploadTypeEmployeePhoto || u == UploadTypePayment
}

// GetStoragePath returns the storage path for the upload type
func (u UploadType) GetStoragePath() string {
	switch u {
	case UploadTypeProfileUser:
		return "/assets/avatar"
	case UploadTypeIconCompany:
		return "/assets/icon"
	case UploadTypeContentThumbnail:
		return "/assets/thumbnail"
	case UploadTypeArmada:
		return "/assets/armada"
	case UploadTypePackage:
		return "/assets/package"
	case UploadTypeOrder:
		return "/assets/order"
	case UploadTypeContent:
		return "/assets/content"
	case UploadTypeEmployeePhoto:
		return "/assets/employee"
	case UploadTypePayment:
		return "/assets/payment"
	default:
		return ""
	}
}

type RentType int

type PaymentStatus int

var TransactionTypeLabel = map[int]string{
	int(model.TransactionTypeIncomeRental):      "Income Rental",
	int(model.TransactionTypeIncomeTourPackage): "Income Tour Package",
	int(model.TransactionTypeIncomeComission):   "Income Commission",
	int(model.TransactionTypeIncomeOtherIncome): "Income Other Income",
	int(model.TransactionTypeIncomeAds):         "Income Ads",

	int(model.TransactionTypeExpenseFuel):               "Expense Fuel",
	int(model.TransactionTypeExpenseTol):                "Expense Toll",
	int(model.TransactionTypeExpenseDriverAllowance):    "Expense Driver Allowance",
	int(model.TransactionTypeExpenseGuideFee):           "Expense Guide Fee",
	int(model.TransactionTypeExpenseCrewMeal):           "Expense Crew Meal",
	int(model.TransactionTypeExpenseVehicleMaintenance): "Expense Vehicle Maintenance",
	int(model.TransactionTypeExpenseVehicleTax):         "Expense Vehicle Tax",
	int(model.TransactionTypeExpenseVehicleInsurance):   "Expense Vehicle Insurance",
	int(model.TransactionTypeExpenseHotel):              "Expense Hotel",
	int(model.TransactionTypeExpenseRestaurant):         "Expense Restaurant",
	int(model.TransactionTypeExpenseAttractionTicket):   "Expense Attraction Ticket",
	int(model.TransactionTypeExpenseSalary):             "Expense Salary",
	int(model.TransactionTypeExpenseOfficeRent):         "Expense Office Rent",
	int(model.TransactionTypeExpenseUtility):            "Expense Utility",
	int(model.TransactionTypeExpenseMarketing):          "Expense Marketing",
	int(model.TransactionTypeExpenseBankCharge):         "Expense Bank Charge",
	int(model.TransactionTypeExpenseOtherExpenses):      "Expense Other Expenses",
	int(model.TransactionTypeExpenseCommission):         "Expense Commission",
}

const (
	PaymentStatusPaid            PaymentStatus = 1
	PaymentStatusWaitingPayment  PaymentStatus = 2
	PaymentStatusWaitingApproval PaymentStatus = 3
	PaymentStatusPartiallyPaid   PaymentStatus = 4
	PaymentStatusCancelled       PaymentStatus = 5
)

func (s PaymentStatus) String() string {
	switch s {
	case PaymentStatusPaid:
		return "PAID"
	case PaymentStatusWaitingPayment:
		return "WAITING PAYMENT"
	case PaymentStatusWaitingApproval:
		return "WAITING APPROVAL"
	case PaymentStatusPartiallyPaid:
		return "PARTIALLY PAID"
	case PaymentStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

const (
	RentTypeCityTour       RentType = 1
	RentTypeOverland       RentType = 2
	RentTypePickupDropOnly RentType = 3
)

func (r RentType) String() string {
	switch r {
	case RentTypeCityTour:
		return "City Tour"
	case RentTypeOverland:
		return "Overland"
	case RentTypePickupDropOnly:
		return "Pickup / Drop Only"
	default:
		return "Unknown"
	}
}

// OrderStatus represents the status of an order
type OrderStatus int

const (
	OrderStatusCancelled    OrderStatus = 0 // Pesanan Dibatalkan
	OrderStatusConfirmed    OrderStatus = 1 // Pesanan Dikonfirmasi
	OrderStatusNotConfirmed OrderStatus = 2 // Pesanan Belum Dikonfirmasi
)

// String returns the string representation of OrderStatus
func (s OrderStatus) String() string {
	switch s {
	case OrderStatusCancelled:
		return "Pesanan Dibatalkan"
	case OrderStatusConfirmed:
		return "Pesanan Dikonfirmasi"
	case OrderStatusNotConfirmed:
		return "Pesanan Belum Dikonfirmasi"
	default:
		return "Unknown"
	}
}

// IsValid checks if the order status is valid
func (s OrderStatus) IsValid() bool {
	return s == OrderStatusCancelled || s == OrderStatusConfirmed || s == OrderStatusNotConfirmed
}
