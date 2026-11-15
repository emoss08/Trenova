package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// PlanFieldKeeper keeps a write from changing what the organization's plan
// withholds. A base record's form reads null for a field that belongs to an
// add-on the plan leaves out (ResourceDefinition.FieldFeatures) and sends the
// record back; Keep puts the stored value of every such field back on the
// incoming record, so dropping an add-on loses nothing and buying it back
// shows the data again.
type PlanFieldKeeper interface {
	// Keep sets, on incoming (a pointer to a record of the kind), every field
	// whose feature the plan leaves out to its value on stored: the record as
	// it is stored on an update, or on a create the record's defaults, nil for
	// zero values. Fields of plain structs the record holds (a worker's
	// profile, a customer's billing profile) are kept by their own name; a
	// nested struct the incoming record does not send is left alone. Lists
	// are not walked: a caller keeps each element against its stored match.
	// With every feature on it changes nothing; a plan that cannot be read is
	// an error, and the write must fail with it.
	Keep(ctx context.Context, kind permission.Resource, incoming, stored any) error
}
