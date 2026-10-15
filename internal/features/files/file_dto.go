package files

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type UploadRequest struct {
	ResourceType string `json:"resource_type" binding:"required,max=50" example:"soap_note"`
	ResourceID   string `json:"resource_id" binding:"required,uuid" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	FileName     string `json:"file_name" binding:"required,notblank,max=255" example:"lab-result.pdf"`
	ContentType  string `json:"content_type" binding:"required,max=100" example:"application/pdf"`
	SizeBytes    int64  `json:"size_bytes" binding:"required,gt=0" example:"482113"`
}

type ListRequest struct {
	ResourceType string `form:"resource_type" binding:"required,max=50"`
	ResourceID   string `form:"resource_id" binding:"required,uuid"`
}

type FileResponse struct {
	ID               string     `json:"id" example:"6e1f4b2a-0d3c-4a58-8c1d-7b3e9d1f6c2a"`
	ResourceType     string     `json:"resource_type" example:"soap_note"`
	ResourceID       string     `json:"resource_id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	FileName         string     `json:"file_name" example:"lab-result.pdf"`
	ContentType      string     `json:"content_type" example:"application/pdf"`
	SizeBytes        int64      `json:"size_bytes" example:"482113"`
	Status           string     `json:"status" enums:"pending,uploaded,rejected" example:"uploaded"`
	UploadedByUserID string     `json:"uploaded_by_user_id" example:"5b815a95-e4d9-40c1-bfc0-dccbe1f38722"`
	UploadedAt       *time.Time `json:"uploaded_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

// UploadTarget tells the client where and how to send the file bytes.
type UploadTarget struct {
	URL       string            `json:"url" example:"http://localhost:9000/simrs-files/soap_note/..."`
	Method    string            `json:"method" example:"PUT"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type UploadResponse struct {
	File   FileResponse `json:"file"`
	Upload UploadTarget `json:"upload"`
}

type DownloadResponse struct {
	URL       string    `json:"url"`
	FileName  string    `json:"file_name" example:"lab-result.pdf"`
	ExpiresAt time.Time `json:"expires_at"`
}

func toFileResponse(f *models.File) FileResponse {
	return FileResponse{
		ID:               f.ID.String(),
		ResourceType:     f.ResourceType,
		ResourceID:       f.ResourceID.String(),
		FileName:         f.FileName,
		ContentType:      f.ContentType,
		SizeBytes:        f.SizeBytes,
		Status:           f.Status,
		UploadedByUserID: f.UploadedByUserID.String(),
		UploadedAt:       f.UploadedAt,
		CreatedAt:        f.CreatedAt,
	}
}
