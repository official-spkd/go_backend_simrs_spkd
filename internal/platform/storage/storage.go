package storage

import (
	"context"
	"errors"
	"time"
)

var ErrObjectNotFound = errors.New("object not found")

// ObjectInfo is what the storage reports about a stored object.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// ObjectStorage hides the object store behind the few operations the app needs. Clients upload
// and download straight to the store through presigned URLs; the API never carries file bytes.
type ObjectStorage interface {
	// PresignUpload returns a URL that accepts one HTTP PUT of the object until it expires.
	PresignUpload(ctx context.Context, key string, expiry time.Duration) (string, error)
	// PresignDownload returns a URL that serves the object as a download named filename.
	PresignDownload(ctx context.Context, key, filename string, expiry time.Duration) (string, error)
	Stat(ctx context.Context, key string) (*ObjectInfo, error)
	// Peek reads up to n leading bytes, used to check the real file type.
	Peek(ctx context.Context, key string, n int64) ([]byte, error)
	Remove(ctx context.Context, key string) error
}
