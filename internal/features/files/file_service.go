package files

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/storage"
	"gin-quickstart/internal/platform/utils"
)

const (
	uploadURLTTL   = 15 * time.Minute
	downloadURLTTL = 5 * time.Minute
	sniffBytes     = 512
	maxNameRunes   = 100
)

var (
	ErrNotUploaded  = errors.New("the file has not been uploaded to storage yet")
	ErrNotPending   = errors.New("the upload was already completed or rejected")
	ErrNotReady     = errors.New("the file is not available: its upload was not completed")
	ErrTooManyFiles = errors.New("this record already has the maximum number of files")
)

// InputError is a request that is well-formed but not acceptable (maps to 400).
type InputError string

func (e InputError) Error() string { return string(e) }

// RejectedError means the stored object failed verification and was discarded (maps to 422).
type RejectedError string

func (e RejectedError) Error() string { return string(e) }

type UploadResult struct {
	File      *models.File
	UploadURL string
	Headers   map[string]string
	ExpiresAt time.Time
}

type DownloadResult struct {
	URL       string
	FileName  string
	ExpiresAt time.Time
}

type Service interface {
	// RequestUpload registers a pending file and returns a presigned URL to PUT its bytes to.
	RequestUpload(ctx context.Context, req UploadRequest, actor *utils.TokenClaims) (*UploadResult, error)
	// Complete verifies the uploaded object (size and real type) and makes the file available.
	Complete(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) (*models.File, error)
	List(ctx context.Context, resourceType string, resourceID uuid.UUID, actor *utils.TokenClaims) ([]models.File, error)
	Download(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) (*DownloadResult, error)
	Delete(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) error
}

type service struct {
	repo     Repository
	registry *Registry
	store    storage.ObjectStorage
}

func NewService(repo Repository, registry *Registry, store storage.ObjectStorage) Service {
	return &service{repo: repo, registry: registry, store: store}
}

func (s *service) RequestUpload(ctx context.Context, req UploadRequest, actor *utils.TokenClaims) (*UploadResult, error) {
	target, resourceID, err := s.access(ctx, req.ResourceType, req.ResourceID, actor, true)
	if err != nil {
		return nil, err
	}

	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	if !slices.Contains(target.AllowedContentTypes(), contentType) {
		return nil, InputError("content_type must be one of: " + strings.Join(target.AllowedContentTypes(), ", "))
	}
	if req.SizeBytes > target.MaxSizeBytes() {
		return nil, InputError(fmt.Sprintf("size_bytes must be at most %d bytes", target.MaxSizeBytes()))
	}

	now := time.Now()
	count, err := s.repo.CountActive(ctx, req.ResourceType, resourceID, now)
	if err != nil {
		return nil, err
	}
	if count >= int64(target.MaxFiles()) {
		return nil, ErrTooManyFiles
	}

	id := uuid.New()
	name := safeName(req.FileName)
	f := &models.File{
		ID:               id,
		ResourceType:     req.ResourceType,
		ResourceID:       resourceID,
		ObjectKey:        fmt.Sprintf("%s/%s/%s/%s", req.ResourceType, resourceID, id, name),
		FileName:         name,
		ContentType:      contentType,
		SizeBytes:        req.SizeBytes,
		Status:           models.FilePending,
		UploadedByUserID: actor.UserID,
		UploadExpiresAt:  now.Add(uploadURLTTL),
	}
	url, err := s.store.PresignUpload(ctx, f.ObjectKey, uploadURLTTL)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return &UploadResult{
		File:      f,
		UploadURL: url,
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: f.UploadExpiresAt,
	}, nil
}

func (s *service) Complete(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) (*models.File, error) {
	f, target, err := s.fileAccess(ctx, id, actor, true)
	if err != nil {
		return nil, err
	}
	if f.Status != models.FilePending {
		return nil, ErrNotPending
	}

	info, err := s.store.Stat(ctx, f.ObjectKey)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, ErrNotUploaded
	}
	if err != nil {
		return nil, err
	}

	if reason := s.verify(ctx, f, info, target); reason != "" {
		return nil, s.reject(ctx, f, reason)
	}

	now := time.Now()
	f.Status, f.SizeBytes, f.UploadedAt = models.FileUploaded, info.Size, &now
	ok, err := s.repo.Transition(ctx, f, models.FilePending)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotPending
	}
	return f, nil
}

// verify checks the stored object against what was declared and what the target allows. It returns
// why the object must be refused, or an empty string when it is fine.
func (s *service) verify(ctx context.Context, f *models.File, info *storage.ObjectInfo, target Target) string {
	if info.Size != f.SizeBytes {
		return "the uploaded size does not match the declared size"
	}
	if info.Size > target.MaxSizeBytes() {
		return "the uploaded file is too large"
	}
	head, err := s.store.Peek(ctx, f.ObjectKey, sniffBytes)
	if err != nil {
		return "the uploaded file could not be read"
	}
	// The declared type is only a claim; the leading bytes tell what the file really is.
	if http.DetectContentType(head) != f.ContentType {
		return "the file content does not match its declared type"
	}
	return ""
}

// reject discards a bad upload so it can never be served, and reports why.
func (s *service) reject(ctx context.Context, f *models.File, reason string) error {
	_ = s.store.Remove(ctx, f.ObjectKey)
	f.Status = models.FileRejected
	if _, err := s.repo.Transition(ctx, f, models.FilePending); err != nil {
		return err
	}
	return RejectedError(reason)
}

func (s *service) List(ctx context.Context, resourceType string, resourceID uuid.UUID, actor *utils.TokenClaims) ([]models.File, error) {
	if _, _, err := s.access(ctx, resourceType, resourceID.String(), actor, false); err != nil {
		return nil, err
	}
	return s.repo.ListUploaded(ctx, resourceType, resourceID)
}

func (s *service) Download(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) (*DownloadResult, error) {
	f, _, err := s.fileAccess(ctx, id, actor, false)
	if err != nil {
		return nil, err
	}
	if f.Status != models.FileUploaded {
		return nil, ErrNotReady
	}
	url, err := s.store.PresignDownload(ctx, f.ObjectKey, f.FileName, downloadURLTTL)
	if err != nil {
		return nil, err
	}
	return &DownloadResult{URL: url, FileName: f.FileName, ExpiresAt: time.Now().Add(downloadURLTTL)}, nil
}

// Delete hides the file from the record. Verified files stay in storage for the record's history;
// an upload that never completed is removed.
func (s *service) Delete(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims) error {
	f, _, err := s.fileAccess(ctx, id, actor, true)
	if err != nil {
		return err
	}
	if f.Status != models.FileUploaded {
		_ = s.store.Remove(ctx, f.ObjectKey)
	}
	return s.repo.Delete(ctx, id)
}

// access resolves the target of a resource type and checks that the caller may use it.
func (s *service) access(ctx context.Context, resourceType, rawID string, actor *utils.TokenClaims, write bool) (Target, uuid.UUID, error) {
	target, ok := s.registry.Get(resourceType)
	if !ok {
		return nil, uuid.Nil, InputError("resource_type is not supported: " + resourceType)
	}
	if !slices.Contains(target.Roles(), actor.Role) {
		return nil, uuid.Nil, &utils.AccessError{Kind: utils.AccessForbidden, Message: "your role cannot use files of this record"}
	}
	resourceID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, uuid.Nil, InputError("resource_id must be a valid UUID")
	}
	if err := target.Authorize(ctx, resourceID, actor, write); err != nil {
		return nil, uuid.Nil, err
	}
	return target, resourceID, nil
}

// fileAccess loads a file and checks access to the record it is attached to.
func (s *service) fileAccess(ctx context.Context, id uuid.UUID, actor *utils.TokenClaims, write bool) (*models.File, Target, error) {
	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	target, _, err := s.access(ctx, f.ResourceType, f.ResourceID.String(), actor, write)
	if err != nil {
		return nil, nil, err
	}
	return f, target, nil
}

// safeName reduces a client-supplied file name to a harmless one: no directories, no control or
// unusual characters, bounded length, extension kept.
func safeName(name string) string {
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	var b strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || r == ' ' || r == '(' || r == ')':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	clean := strings.Trim(strings.TrimSpace(b.String()), ".")
	runes := []rune(clean)
	if len(runes) > maxNameRunes {
		ext := []rune(path.Ext(clean))
		if len(ext) > 10 {
			ext = nil
		}
		runes = append(runes[:maxNameRunes-len(ext)], ext...)
	}
	if len(runes) == 0 {
		return "file"
	}
	return string(runes)
}
