package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const recountSQL = `UPDATE polis SET doctor_count =
	(SELECT COUNT(*) FROM doctors WHERE doctors.poli_id = polis.id AND doctors.is_active)`

// RecountPoliDoctors recalculates doctor_count of the given polis from the doctors table
// (active doctors only). Call it inside the same transaction as the doctor change.
func RecountPoliDoctors(tx *gorm.DB, poliIDs ...uuid.UUID) error {
	if len(poliIDs) == 0 {
		return nil
	}
	return tx.Exec(recountSQL+" WHERE polis.id IN ?", poliIDs).Error
}

// RecountAllPoliDoctors recalculates doctor_count of every poli.
func RecountAllPoliDoctors(db *gorm.DB) error {
	return db.Exec(recountSQL).Error
}
