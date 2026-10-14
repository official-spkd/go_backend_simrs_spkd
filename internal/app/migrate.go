package app

import (
	"fmt"

	"gorm.io/gorm"

	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/models"
)

func migrate(db *gorm.DB) error {
	// The sequence must exist before the patients table, whose column default uses it.
	if err := db.Exec("CREATE SEQUENCE IF NOT EXISTS patient_medical_record_number_seq").Error; err != nil {
		return fmt.Errorf("create sequence: %w", err)
	}
	if err := db.AutoMigrate(&roles.Role{}, &models.User{}, &models.Patient{}, &models.Visit{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Visits created before penjamin used the lowercase vocabulary stored General/BPJS/Insurance.
	// Idempotent: rows already converted are left untouched.
	err := db.Exec(`UPDATE visits SET payer = CASE payer
		WHEN 'General' THEN 'umum' WHEN 'BPJS' THEN 'bpjs' WHEN 'Insurance' THEN 'asuransi' END
		WHERE payer IN ('General', 'BPJS', 'Insurance')`).Error
	if err != nil {
		return fmt.Errorf("normalize visit penjamin: %w", err)
	}
	return nil
}
