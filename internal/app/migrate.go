package app

import (
	"fmt"

	"gorm.io/gorm"

	"gin-quickstart/internal/features/doctors"
	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/models"
)

func migrate(db *gorm.DB) error {
	// The sequence must exist before the patients table, whose column default uses it.
	if err := db.Exec("CREATE SEQUENCE IF NOT EXISTS patient_medical_record_number_seq").Error; err != nil {
		return fmt.Errorf("create sequence: %w", err)
	}
	if err := db.AutoMigrate(&roles.Role{}, &models.Poli{}, &models.Doctor{}, &models.User{}, &models.Patient{}, &models.Visit{}, &models.Counter{}, &models.Queue{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// GORM tags cannot express an expression index: names are unique regardless of case.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_polis_name_lower ON polis (LOWER(name))").Error; err != nil {
		return fmt.Errorf("create poli name index: %w", err)
	}

	// doctor_count is derived from active doctors, so refresh it for every poli.
	if err := doctors.RecountAllPoliDoctors(db); err != nil {
		return fmt.Errorf("recount poli doctors: %w", err)
	}

	return nil
}
