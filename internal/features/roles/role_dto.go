package roles

type CreateRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50" example:"nurse"`
	DisplayName string `json:"display_name" binding:"required,max=100" example:"Nurse"`
	Description string `json:"description" binding:"max=255" example:"Ward nurse"`
}

type UpdateRequest struct {
	DisplayName string `json:"display_name" binding:"required,max=100" example:"Head Nurse"`
	Description string `json:"description" binding:"max=255" example:"Ward head nurse"`
}
