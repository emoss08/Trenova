package refview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

var _ permissionports.ReferenceViews = (*Port)(nil)

// Port answers permissionports.ReferenceViews for surfaces in other modules,
// which reach refview only through that port.
type Port struct {
	views  *Views
	engine permissionports.PermissionEngine
}

func NewPort(views *Views, engine permissionports.PermissionEngine) *Port {
	if views == nil {
		views = Default()
	}
	return &Port{views: views, engine: engine}
}

func (p *Port) Person(
	actor *permissionports.RequestActor,
	check func(ctx context.Context, kind permission.Resource) (bool, error),
) permissionports.LinkedFields {
	return linkedFields{views: p.views, access: NewPersonAccess(p.engine, actor, check)}
}

type linkedFields struct {
	views  *Views
	access *PersonAccess
}

func (l linkedFields) Reads(ctx context.Context, kind permission.Resource) bool {
	return l.access.Reads(ctx, kind)
}

func (l linkedFields) Decide(
	ctx context.Context,
	kind permission.Resource,
	field string,
) permissionports.LinkedFieldDecision {
	switch l.views.Decide(ctx, l.access, kind, field, Nested) {
	case Visible:
		return permissionports.LinkedFieldVisible
	case WithheldKind:
		return permissionports.LinkedFieldWithheldKind
	case WithheldSensitivity:
		return permissionports.LinkedFieldWithheldSensitivity
	default:
		return permissionports.LinkedFieldWithheldKind
	}
}
