// Package refview decides whether one field of a record may be shown to a
// caller, for every surface that shows a record of one kind carried by
// another: GraphQL for people and agents, REST, reports, exports, search and
// analytics. A record reached through a parent the caller may read always
// shows its kind's reference view; every other field needs the kind's own
// read, and the sensitivity ceiling applies on top.
package refview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// Reach is how the record holding a field was reached.
type Reach uint8

const (
	// Root is the record a root resolver returned, which it authorized.
	Root Reach = iota
	// Nested is a record carried by another record the caller may read.
	Nested
	// SelfScoped is a record under a resolver that acts only on the
	// caller's own records.
	SelfScoped
)

type Decision uint8

const (
	Visible Decision = iota
	// WithheldKind is a field outside the reference view of a kind the
	// caller cannot read.
	WithheldKind
	// WithheldSensitivity is a field ranked above the caller's ceiling.
	WithheldSensitivity
)

func (d Decision) Visible() bool { return d == Visible }

// Access is one caller's grants, as each surface knows them.
type Access interface {
	// Reads reports whether the caller may read a kind at all. A failed
	// check reads as no.
	Reads(ctx context.Context, kind permission.Resource) bool
	// Ceiling is the most sensitive tier the caller may be shown on a kind
	// reached this way, and false where no ceiling applies.
	Ceiling(
		ctx context.Context,
		kind permission.Resource,
		reach Reach,
	) (permission.FieldSensitivity, bool)
}

type Views struct {
	registry *permission.Registry
}

func New(registry *permission.Registry) *Views {
	if registry == nil {
		registry = fieldsensitivity.DefaultRegistry()
	}
	return &Views{registry: registry}
}

// Default is the views of the process's registry.
func Default() *Views {
	return New(fieldsensitivity.DefaultRegistry())
}

func (v *Views) Registry() *permission.Registry { return v.registry }

// InView reports whether a field is in a kind's reference view.
func (v *Views) InView(kind permission.Resource, field string) bool {
	return v.registry.InReferenceView(kind.String(), field)
}

// PartOf reports whether a kind carried by another is part of that record,
// so it follows its carrier's access rather than its own.
func (v *Views) PartOf(kind, carrier permission.Resource) bool {
	return v.registry.PartOf(kind.String(), carrier.String())
}

// Decide is whether one field of a record of a kind, reached as given, may
// be shown to the caller.
func (v *Views) Decide(
	ctx context.Context,
	access Access,
	kind permission.Resource,
	field string,
	reach Reach,
) Decision {
	if reach == Nested && !v.InView(kind, field) && !access.Reads(ctx, kind) {
		return WithheldKind
	}
	ceiling, applies := access.Ceiling(ctx, kind, reach)
	if !applies || field == "id" {
		return Visible
	}
	if !fieldsensitivity.VisibleAt(v.registry.GetFieldSensitivity(kind.String(), field), ceiling) {
		return WithheldSensitivity
	}

	return Visible
}

// DecidePart is Decide for a field of a record that is part of a record of a
// kind, such as a stop of a linked shipment: it has no reference view of its
// own, so reached through a linked record all of it needs the kind's read.
func (v *Views) DecidePart(
	ctx context.Context,
	access Access,
	kind permission.Resource,
	field string,
	reach Reach,
) Decision {
	if reach == Nested && !access.Reads(ctx, kind) {
		return WithheldKind
	}

	return v.Decide(ctx, access, kind, field, reach)
}

// Filter splits a record's fields into those the caller may be shown and
// those withheld, in the order given.
func (v *Views) Filter(
	ctx context.Context,
	access Access,
	kind permission.Resource,
	fields []string,
	reach Reach,
) (visible, withheld []string) {
	visible = make([]string, 0, len(fields))
	for _, field := range fields {
		if v.Decide(ctx, access, kind, field, reach).Visible() {
			visible = append(visible, field)
			continue
		}
		withheld = append(withheld, field)
	}

	return visible, withheld
}
