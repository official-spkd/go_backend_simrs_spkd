package utils

// AccessKind says why an action on a resource was refused.
type AccessKind int

const (
	AccessNotFound  AccessKind = iota // the resource does not exist
	AccessForbidden                   // the caller may not touch it
	AccessConflict                    // its current state does not allow the action
)

// AccessError lets a feature explain a refusal to code that does not know that feature, such as
// the generic file upload feature asking a feature whether a user may attach files to something.
type AccessError struct {
	Kind    AccessKind
	Message string
}

func (e *AccessError) Error() string { return e.Message }
