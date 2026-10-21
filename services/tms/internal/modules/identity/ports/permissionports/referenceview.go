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
	// Person is one person's access for the length of one request. check,
	// when set, is how the surface already asks for a kind's read, so the two
	// share its memo.
	Person(
		actor *RequestActor,
		check func(ctx context.Context, kind permission.Resource) (bool, error),
	) LinkedFields
}

// LinkedFields answers for the records one person reaches through records
// they may read.
type LinkedFields interface {
	Reads(ctx context.Context, kind permission.Resource) bool
	Decide(ctx context.Context, kind permission.Resource, field string) LinkedFieldDecision
}
