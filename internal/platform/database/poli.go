package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const recountSQL = `UPDATE polis SET doctor_count =
	(SELECT COUNT(*) FROM users WHERE users.poli_id = polis.id AND users.is_active)`

// RecountPoliDoctors recalculates doctor_count of the given polis from the users table
// (active doctors only). Call it inside the same transaction as the user change.
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
