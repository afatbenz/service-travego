package configs

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
)

// String returns the string representation of UploadType
func (u UploadType) String() string {
	return string(u)
}

// IsValid checks if the upload type is valid
func (u UploadType) IsValid() bool {
    return u == UploadTypeProfileUser || u == UploadTypeIconCompany || u == UploadTypeContentThumbnail ||
        u == UploadTypeArmada || u == UploadTypePackage || u == UploadTypeOrder || u == UploadTypeContent
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
    default:
        return ""
    }
}

type RentType int

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
