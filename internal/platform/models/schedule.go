package models

import (
	"time"

	"github.com/google/uuid"
)

// DoctorSchedule is the weekly template of one doctor: one row per weekday. Daily sessions are
// derived from it, so they are never stored.
type DoctorSchedule struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DoctorID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_doctor_schedules_doctor_weekday,priority:1"`
	Doctor   *Doctor   `gorm:"foreignKey:DoctorID;constraint:OnDelete:CASCADE"`
	Weekday  int       `gorm:"not null;uniqueIndex:idx_doctor_schedules_doctor_weekday,priority:2"` // time.Weekday: 0 = Sunday

	StartTime string `gorm:"size:5;not null"` // HH:MM
	EndTime   string `gorm:"size:5;not null"` // HH:MM

	// The total quota is JKNQuota + NonJKNQuota; every registration channel draws from the same pool.
	JKNQuota    int `gorm:"column:jkn_quota;not null"`
	NonJKNQuota int `gorm:"column:non_jkn_quota;not null"`

	EffectiveDate time.Time `gorm:"type:date;not null"` // the template applies to sessions on or after this date
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (s *DoctorSchedule) TotalQuota() int {
	return s.JKNQuota + s.NonJKNQuota
}

// ScheduleException closes a doctor's session on one date (leave), optionally handing it to a
// substitute doctor.
type ScheduleException struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DoctorID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_schedule_exceptions_doctor_date,priority:1"`
	Doctor   *Doctor   `gorm:"foreignKey:DoctorID;constraint:OnDelete:CASCADE"`
	Date     time.Time `gorm:"type:date;not null;uniqueIndex:idx_schedule_exceptions_doctor_date,priority:2;index"`

	SubstituteDoctorID *uuid.UUID `gorm:"type:uuid;index"`
	SubstituteDoctor   *Doctor    `gorm:"foreignKey:SubstituteDoctorID"`

	Reason    string `gorm:"size:200;not null"`
	CreatedAt time.Time
}
