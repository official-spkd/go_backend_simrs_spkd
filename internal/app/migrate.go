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

	// Visits created before penjamin used the lowercase vocabulary stored General/BPJS/Insurance.
	// Idempotent: rows already converted are left untouched.
	err := db.Exec(`UPDATE visits SET payer = CASE payer
		WHEN 'General' THEN 'umum' WHEN 'BPJS' THEN 'bpjs' WHEN 'Insurance' THEN 'asuransi' END
		WHERE payer IN ('General', 'BPJS', 'Insurance')`).Error
	if err != nil {
		return fmt.Errorf("normalize visit penjamin: %w", err)
	}

	// Visits registered before the queue feature get a place in line, in registration order.
	// Idempotent: visits that already have a queue entry are skipped.
	err = db.Exec(`INSERT INTO queues (id, visit_id, queue_date, queue_sequence, queue_number, status, call_count, created_at, updated_at)
		SELECT gen_random_uuid(), v.id, v.visit_date,
			ROW_NUMBER() OVER (PARTITION BY v.visit_date ORDER BY v.created_at, v.id)
				+ COALESCE((SELECT MAX(q.queue_sequence) FROM queues q WHERE q.queue_date = v.visit_date), 0),
			'A' || LPAD((ROW_NUMBER() OVER (PARTITION BY v.visit_date ORDER BY v.created_at, v.id)
				+ COALESCE((SELECT MAX(q.queue_sequence) FROM queues q WHERE q.queue_date = v.visit_date), 0))::text, 3, '0'),
			'WAITING', 0, NOW(), NOW()
		FROM visits v
		WHERE v.visit_status = 'Registered' AND v.queue_status = 'Waiting'
			AND NOT EXISTS (SELECT 1 FROM queues q WHERE q.visit_id = v.id)`).Error
	if err != nil {
		return fmt.Errorf("backfill queues: %w", err)
	}
	return nil
}
