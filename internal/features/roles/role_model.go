package roles

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id" swaggertype:"string" format:"uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name        string    `gorm:"size:50;uniqueIndex;not null" json:"name" example:"doctor"`
	DisplayName string    `gorm:"size:100;not null" json:"display_name" example:"Doctor"`
	Description string    `gorm:"size:255" json:"description" example:"Medical doctor"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system" example:"true"` // built-in roles cannot be deleted
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	Doctor      = "doctor"
	FrontOffice = "front_office"
	IT          = "it"
	Pharmacist  = "pharmacist"
)

func Defaults() []Role {
	return []Role{
		{Name: Doctor, DisplayName: "Doctor", Description: "Medical doctor", IsSystem: true},
		{Name: FrontOffice, DisplayName: "Front Office", Description: "Registration and reception staff", IsSystem: true},
		{Name: IT, DisplayName: "IT", Description: "System administrator", IsSystem: true},
		{Name: Pharmacist, DisplayName: "Pharmacist", Description: "Pharmacy staff", IsSystem: true},
	}
}
