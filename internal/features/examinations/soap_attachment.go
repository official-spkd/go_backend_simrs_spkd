package examinations

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/utils"
)

// AttachmentResourceType is how clients name the files of a SOAP note. The resource ID is the
// visit ID, since a visit has exactly one SOAP note.
const AttachmentResourceType = "soap_note"

func (s *service) AuthorizeAttachment(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims, write bool) error {
	if _, err := s.authorize(ctx, visitID, actor, write); err != nil {
		return attachmentAccessError(err)
	}
	if write {
		if err := s.ensureNotFinalized(ctx, visitID); err != nil {
			return attachmentAccessError(err)
		}
	}
	return nil
}

// attachmentAccessError turns this feature's errors into the neutral form the file feature reads.
func attachmentAccessError(err error) error {
	kind := utils.AccessConflict
	switch {
	case errors.Is(err, ErrVisitNotFound):
		kind = utils.AccessNotFound
	case errors.Is(err, ErrDoctorNotLinked), errors.Is(err, ErrNotYourVisit):
		kind = utils.AccessForbidden
	case errors.Is(err, ErrNoDoctor), errors.Is(err, ErrVisitCancelled), errors.Is(err, ErrVisitNotToday), errors.Is(err, ErrFinalized):
		kind = utils.AccessConflict
	default:
		return err // unexpected: let it surface as a server error
	}
	return &utils.AccessError{Kind: kind, Message: err.Error()}
}

// AttachmentTarget lets doctors attach medical files (lab results, scans, referral letters) to the
// SOAP note of their own visits, until the note is finalized. It plugs into the file feature.
type AttachmentTarget struct {
	service Service
}

func NewAttachmentTarget(service Service) *AttachmentTarget {
	return &AttachmentTarget{service: service}
}

func (t *AttachmentTarget) Type() string    { return AttachmentResourceType }
func (t *AttachmentTarget) Roles() []string { return []string{roles.Doctor} }
func (t *AttachmentTarget) MaxFiles() int   { return 10 }

func (t *AttachmentTarget) MaxSizeBytes() int64 { return 10 << 20 } // 10 MB

func (t *AttachmentTarget) AllowedContentTypes() []string {
	return []string{"application/pdf", "image/jpeg", "image/png"}
}

func (t *AttachmentTarget) Authorize(ctx context.Context, resourceID uuid.UUID, actor *utils.TokenClaims, write bool) error {
	return t.service.AuthorizeAttachment(ctx, resourceID, actor, write)
}
