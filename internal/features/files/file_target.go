package files

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/utils"
)

// Target is implemented by a feature whose records can have files attached (SOAP notes, patients,
// and so on). The files feature knows nothing else about that feature: the target names the
// resource type, says who may use it and what is accepted, and decides access per resource.
type Target interface {
	// Type is the resource_type value clients send, e.g. "soap_note".
	Type() string
	// Roles are the roles that may use this resource type at all.
	Roles() []string
	AllowedContentTypes() []string
	MaxSizeBytes() int64
	// MaxFiles is how many files one resource can hold.
	MaxFiles() int
	// Authorize checks that the caller may read (write false) or change (write true) the files of
	// the resource. It returns *utils.AccessError to refuse.
	Authorize(ctx context.Context, resourceID uuid.UUID, actor *utils.TokenClaims, write bool) error
}

// Registry holds the targets that opted in to file attachments.
type Registry struct {
	targets map[string]Target
}

func NewRegistry(targets ...Target) (*Registry, error) {
	r := &Registry{targets: make(map[string]Target, len(targets))}
	for _, t := range targets {
		if _, dup := r.targets[t.Type()]; dup {
			return nil, fmt.Errorf("file target %q registered twice", t.Type())
		}
		r.targets[t.Type()] = t
	}
	return r, nil
}

func (r *Registry) Get(resourceType string) (Target, bool) {
	t, ok := r.targets[resourceType]
	return t, ok
}

// Roles is the union of the roles of all targets, for the route middleware.
func (r *Registry) Roles() []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range r.targets {
		for _, role := range t.Roles() {
			if !seen[role] {
				seen[role] = true
				out = append(out, role)
			}
		}
	}
	sort.Strings(out)
	return out
}
