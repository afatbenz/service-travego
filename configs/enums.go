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
)

// String returns the string representation of UploadType
func (u UploadType) String() string {
	return string(u)
}

// IsValid checks if the upload type is valid
func (u UploadType) IsValid() bool {
	return u == UploadTypeProfileUser || u == UploadTypeIconCompany || u == UploadTypeContentThumbnail
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
	default:
		return ""
	}
}
