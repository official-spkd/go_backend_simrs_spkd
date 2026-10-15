package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FilePending  = "pending"  // upload URL issued, upload not confirmed yet
	FileUploaded = "uploaded" // confirmed: size and type verified
	FileRejected = "rejected" // uploaded but failed verification; object removed
)

// File is the metadata of a stored attachment. The bytes live in object storage. Any feature can
// own attachments: a file points to its owner with ResourceType + ResourceID.
type File struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ResourceType string    `gorm:"size:50;not null;index:idx_files_resource,priority:1"`
	ResourceID   uuid.UUID `gorm:"type:uuid;not null;index:idx_files_resource,priority:2"`

	ObjectKey   string `gorm:"size:500;not null;uniqueIndex"`
	FileName    string `gorm:"size:255;not null"`
	ContentType string `gorm:"size:100;not null"`
	SizeBytes   int64  `gorm:"not null"`
	Status      string `gorm:"size:10;not null;index"`

	UploadedByUserID uuid.UUID `gorm:"type:uuid;not null"`
	UploadExpiresAt  time.Time `gorm:"not null"`
	UploadedAt       *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
