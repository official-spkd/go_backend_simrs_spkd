package roles

import "gin-quickstart/internal/platform/models"

// Built-in role names.
const (
	Superadmin  = "superadmin"
	Doctor      = "doctor"
	FrontOffice = "front_office"
	IT          = "it"
	Pharmacist  = "pharmacist"
	Cashier     = "cashier"
	Management  = "management"
)

func Defaults() []models.Role {
	return []models.Role{
		{Name: Superadmin, DisplayName: "Superadmin", Description: "Highest privilege; manages users and roles", IsSystem: true},
		{Name: Doctor, DisplayName: "Doctor", Description: "Medical doctor", IsSystem: true},
		{Name: FrontOffice, DisplayName: "Front Office", Description: "Registration and reception staff", IsSystem: true},
		{Name: IT, DisplayName: "IT", Description: "IT support staff", IsSystem: true},
		{Name: Pharmacist, DisplayName: "Pharmacist", Description: "Pharmacy staff", IsSystem: true},
		{Name: Cashier, DisplayName: "Cashier", Description: "Outpatient billing and payment staff", IsSystem: true},
		{Name: Management, DisplayName: "Management", Description: "Hospital management; reads reports", IsSystem: true},
	}
}
