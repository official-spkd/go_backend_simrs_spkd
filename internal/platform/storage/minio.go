package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"gin-quickstart/internal/platform/config"
)

type minioStorage struct {
	api    *minio.Client // used by the server to talk to MinIO
	public *minio.Client // only signs URLs for the host browsers use
	bucket string
}

// NewMinIO connects to MinIO and makes sure the bucket exists.
func NewMinIO(ctx context.Context, cfg config.Storage) (ObjectStorage, error) {
	newClient := func(endpoint string, ssl bool) (*minio.Client, error) {
		return minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: ssl,
			Region: cfg.Region, // fixed region: signing never needs a network round trip
		})
	}

	api, err := newClient(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	public, err := newClient(cfg.PublicEndpoint, cfg.PublicUseSSL)
	if err != nil {
		return nil, fmt.Errorf("minio public client: %w", err)
	}

	exists, err := api.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := api.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", cfg.Bucket, err)
		}
	}
	return &minioStorage{api: api, public: public, bucket: cfg.Bucket}, nil
}

func (s *minioStorage) PresignUpload(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.public.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}
	return u.String(), nil
}

func (s *minioStorage) PresignDownload(ctx context.Context, key, filename string, expiry time.Duration) (string, error) {
	params := url.Values{}
	params.Set("response-content-disposition", `attachment; filename="`+filename+`"`)
	u, err := s.public.PresignedGetObject(ctx, s.bucket, key, expiry, params)
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return u.String(), nil
}

func (s *minioStorage) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	info, err := s.api.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return &ObjectInfo{Size: info.Size, ContentType: info.ContentType}, nil
}

func (s *minioStorage) Peek(ctx context.Context, key string, n int64) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(0, n-1); err != nil {
		return nil, err
	}
	obj, err := s.api.GetObject(ctx, s.bucket, key, opts)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer obj.Close()

	buf, err := io.ReadAll(io.LimitReader(obj, n))
	if err != nil {
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("read object: %w", err)
	}
	return buf, nil
}

func (s *minioStorage) Remove(ctx context.Context, key string) error {
	if err := s.api.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

func isNotFound(err error) bool {
	var resp minio.ErrorResponse
	return errors.As(err, &resp) && (resp.Code == "NoSuchKey" || resp.Code == "NotFound")
}
