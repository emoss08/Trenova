// Package refview decides whether one field of a record may be shown to a
// caller, for every surface that shows a record of one kind carried by
// another: GraphQL for people and agents, REST, reports, exports, search and
// analytics. The organization's plan is asked first: a field of a kind whose
// feature the plan leaves out, or a field that belongs to such a feature, is
// never shown, however the record was reached. A record reached through a
// parent the caller may read always shows its kind's reference view; every
// other field needs the kind's own read, and the sensitivity ceiling applies
// on top.
package refview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
	// caller cannot read, or one the plan could not be read for.
	WithheldKind
	// WithheldSensitivity is a field ranked above the caller's ceiling.
	WithheldSensitivity
	// WithheldPlan is a field of a kind, or a field of its own feature, that
	// the organization's plan does not include. PlanDenial names it.
	WithheldPlan
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

// Catalog is the registry's static facts about reference views and parts,
// which the schema tooling reads without a plan or a caller.
type Catalog struct {
	registry *permissioncatalog.Registry
}

func NewCatalog(registry *permissioncatalog.Registry) *Catalog {
	if registry == nil {
		registry = fieldsensitivity.DefaultRegistry()
	}
	return &Catalog{registry: registry}
}

func (c *Catalog) Registry() *permissioncatalog.Registry { return c.registry }

// InView reports whether a field is in a kind's reference view.
func (c *Catalog) InView(kind permission.Resource, field string) bool {
	return c.registry.InReferenceView(kind.String(), field)
}

// PartOf reports whether a kind carried by another is part of that record,
// so it follows its carrier's access rather than its own.
func (c *Catalog) PartOf(kind, carrier permission.Resource) bool {
	return c.registry.PartOf(kind.String(), carrier.String())
}

// Views decides fields for callers: the catalog, and the organization's plan.
type Views struct {
	*Catalog

	lookup *onplan.Lookup
	plan   onplan.Plan
}

func New(registry *permissioncatalog.Registry, lookup *onplan.Lookup) *Views {
	return &Views{Catalog: NewCatalog(registry), lookup: lookup, plan: lookup}
}

// Request is the views for one request or operation: each feature is asked
// of the plan once, so every field it decides sees the same plan.
func (v *Views) Request() *Views {
	return &Views{Catalog: v.Catalog, lookup: v.lookup, plan: v.lookup.Memo()}
}

// Plan is the plan these views decide by.
func (v *Views) Plan() onplan.Plan { return v.plan }

// PlanDenial names the feature the plan withholds a field of a kind for; an
// empty field asks about the kind alone. A plan that could not be read is an
// error, which the caller withholds the field for.
func (v *Views) PlanDenial(
	ctx context.Context,
	kind permission.Resource,
	field string,
) (permissionports.PlanDenial, bool, error) {
	return v.plan.Field(ctx, kind.String(), field)
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
	if decision, withheld := v.byPlan(ctx, kind, field); withheld {
		return decision
	}

	return v.decideAccess(ctx, access, kind, field, reach)
}

// DecidePart is Decide for a field of a record of its own kind that is part
// of a record of the carrier's kind, such as a stop of a linked shipment.
// The plan is asked about both kinds: a part of a base record can belong to
// an add-on. It has no reference view of its own, so reached through a
// linked record all of it needs the carrier's read.
func (v *Views) DecidePart(
	ctx context.Context,
	access Access,
	own, carrier permission.Resource,
	field string,
	reach Reach,
) Decision {
	if decision, withheld := v.byPlan(ctx, own, field); withheld {
		return decision
	}
	if own != carrier {
		if decision, withheld := v.kindByPlan(ctx, carrier); withheld {
			return decision
		}
	}
	if reach == Nested && !access.Reads(ctx, carrier) {
		return WithheldKind
	}

	return v.decideAccess(ctx, access, carrier, field, reach)
}

// PartDenial names the feature the plan withholds a field of a part for:
// its own kind's, the field's, or its carrier's.
func (v *Views) PartDenial(
	ctx context.Context,
	own, carrier permission.Resource,
	field string,
) (permissionports.PlanDenial, bool, error) {
	if denial, off, err := v.PlanDenial(ctx, own, field); err != nil || off {
		return denial, off, err
	}
	return v.plan.Resource(ctx, carrier.String())
}

// PlanField is one field the plan withholds, with the feature it belongs to.
type PlanField struct {
	Field  string
	Denial permissionports.PlanDenial
}

// Filtered is a record's fields split by what may be shown: those the
// caller's access withholds apart from those the plan does.
type Filtered struct {
	Visible        []string
	Withheld       []string
	WithheldByPlan []PlanField
}

// Filter splits a record's fields into those the caller may be shown and
// those withheld, in the order given.
func (v *Views) Filter(
	ctx context.Context,
	access Access,
	kind permission.Resource,
	fields []string,
	reach Reach,
) Filtered {
	out := Filtered{Visible: make([]string, 0, len(fields))}
	for _, field := range fields {
		switch v.Decide(ctx, access, kind, field, reach) {
		case Visible:
			out.Visible = append(out.Visible, field)
		case WithheldPlan:
			denial, _, _ := v.PlanDenial(ctx, kind, field)
			out.WithheldByPlan = append(out.WithheldByPlan, PlanField{Field: field, Denial: denial})
		case WithheldKind, WithheldSensitivity:
			out.Withheld = append(out.Withheld, field)
		}
	}

	return out
}

func (v *Views) byPlan(
	ctx context.Context,
	kind permission.Resource,
	field string,
) (Decision, bool) {
	_, off, err := v.plan.Field(ctx, kind.String(), field)
	switch {
	case err != nil:
		return WithheldKind, true
	case off:
		return WithheldPlan, true
	default:
		return Visible, false
	}
}

func (v *Views) kindByPlan(ctx context.Context, kind permission.Resource) (Decision, bool) {
	_, off, err := v.plan.Resource(ctx, kind.String())
	switch {
	case err != nil:
		return WithheldKind, true
	case off:
		return WithheldPlan, true
	default:
		return Visible, false
	}
}

func (v *Views) decideAccess(
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
