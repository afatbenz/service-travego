package model

// UploadPhotoRequest represents upload photo request payload
type UploadPhotoRequest struct {
	FilePath   string `json:"filepath" validate:"required"`
	UploadType string `json:"upload-type" validate:"required,oneof=profile-user icon-company content-thumbnail"`
}

