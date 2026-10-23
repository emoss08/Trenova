package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// LinkedFieldDecision is whether one field of a record reached through
// another record may be shown.
type LinkedFieldDecision uint8

const (
	LinkedFieldVisible LinkedFieldDecision = iota
	// LinkedFieldWithheldKind is a field outside the reference view of a kind
	// the caller cannot read.
	LinkedFieldWithheldKind
	// LinkedFieldWithheldSensitivity is a field ranked above the caller's
	// ceiling.
	LinkedFieldWithheldSensitivity
)

// ReferenceViews is how a surface outside the identity module asks the
// reference view rule (refview) about records it shows inside others, such as
// the customer of a shipment on a report.
type ReferenceViews interface {
	// Person is one person's access for the length of one request, sharing
	// what the surface already knows of their grants.
	Person(actor *RequestActor, grants PersonGrants) LinkedFields
}

// PersonGrants is how a surface hands the reference view rule what it has
// already read of one person's grants, so the two share one read. A nil
// function is answered by the permission engine.
type PersonGrants struct {
	// Reads reports whether the person may read a kind.
	Reads func(ctx context.Context, kind permission.Resource) (bool, error)
	// Detail is the person's permission detail on a kind, which their
	// ceiling on a record of it reached through another is read from.
	Detail func(ctx context.Context, kind permission.Resource) (*ResourcePermissionDetail, error)
}

// LinkedFields answers for the records one person reaches through records
// they may read.
type LinkedFields interface {
	Reads(ctx context.Context, kind permission.Resource) bool
	Decide(ctx context.Context, kind permission.Resource, field string) LinkedFieldDecision
}
