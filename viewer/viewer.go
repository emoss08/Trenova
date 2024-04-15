package viewer //nolint:goimports // Path: viewer/viewer.go

import (
	"context"

	"github.com/emoss08/trenova/internal/ent"
)

// Role for viewer actions.
type Role int

// List of roles.
const (
	_ Role = 1 << iota
	Admin
	Edit
	View
)

// Viewer describes the query/mutation viewer-context.
type Viewer interface {
	Organization(context.Context) (string, error) // Organization to query (tenant == organization).
	Admin() bool                                  // If viewer is admin.
	Can(Role) bool                                // If viewer is able to apply role action.
}

// UserViewer describes a user-viewer.
type UserViewer struct {
	User *ent.User // Actual user.
	Role Role      // Attached roles.
}

func (v UserViewer) Organization(context.Context) (ent.Organization, error) {
	org, err := v.User.QueryOrganization().Only(context.Background())
	if err != nil {
		return ent.Organization{}, err
	}
	return *org, nil
}

func (v UserViewer) Can(r Role) bool {
	if v.Admin() {
		return true
	}
	return v.Role%r != 0
}

func (v UserViewer) Admin() bool {
	return v.Role&Admin != 0
}

// AppViewer describes an app-viewer.
type AppViewer struct {
	Role Role // Attached roles.
}

func (v AppViewer) Teams(context.Context) ([]string, error) {
	return nil, nil
}

func (v AppViewer) Can(r Role) bool {
	if v.Admin() {
		return true
	}
	return v.Role&r != 0
}

func (v AppViewer) Admin() bool {
	return v.Role&Admin != 0
}

type ctxKey struct{}

// FromContext returns the Viewer stored in a context.
func FromContext(ctx context.Context) Viewer {
	v, _ := ctx.Value(ctxKey{}).(Viewer)
	return v
}

// NewContext returns a copy of parent context with the given Viewer attached with it.
func NewContext(parent context.Context, v Viewer) context.Context {
	return context.WithValue(parent, ctxKey{}, v)
}
