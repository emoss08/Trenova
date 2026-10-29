package protectedfield

import (
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// Path is where a protected field sits in a record's JSON, for the
// places that hold a record as JSON rather than as its Go type: an audit
// entry's states and changes. The entity's WithholdFields is the rule for a
// record in a response; these paths are the same rule for its stored JSON.
//
// Path is a dot path; a step ending in [] walks every element of a list. A
// path is matched at any depth, so a worker's profile is withheld inside a
// shipment as on the worker, unless Resource names the one kind of record it
// is rooted at (a credential's number, a rate quote's cost). When lists the
// values the object holding the field must carry for it to be protected (a
// detention trace's driver pay step).
type Path struct {
	Path     string
	Group    Group
	Resource permission.Resource
	When     map[string]any
}

const (
	traceStepKindField = "kind"
	credentialNumber   = "number"
)

var driverPayStep = map[string]any{traceStepKindField: "DriverPay"}

var protectedPaths = [...]Path{
	{Path: "profile.dob", Group: Personal},
	{Path: "profile.licenseNumber", Group: Personal},
	{Path: "profile.twicCardNumber", Group: Personal},
	{Path: "profile.lastDrugTest", Group: Personal},
	{Path: "evidenceCredential.number", Group: Personal},
	{Path: credentialNumber, Group: Personal, Resource: permission.ResourceWorkerCredential},

	{Path: "marginFloorPercent", Group: Margins},
	{Path: "maxPayPercentOfSell", Group: Margins},
	{Path: "targetMarginPercent", Group: Margins},
	{Path: "tenderCounterMarginFloorPercent", Group: Margins},
	{Path: "counterMargin", Group: Margins},
	{Path: "counterMarginPercent", Group: Margins},
	{Path: "marginAmount", Group: Margins},
	{Path: "marginPercent", Group: Margins},
	{Path: "netMargin", Group: Margins},
	{Path: "driverPayAmount", Group: Margins},
	{Path: "proposedNetMargin", Group: Margins},
	{Path: "proposedDriverPay", Group: Margins},
	{Path: "negativeMarginStops", Group: Margins},
	{Path: "sellTotal", Group: Margins},
	{Path: "trace.totals.cost", Group: Margins},
	{Path: "trace.totals.margin", Group: Margins},
	{Path: "trace.totals.marginPercent", Group: Margins},
	{Path: "calculationTrace.steps[].amount", Group: Margins, When: driverPayStep},
	{Path: "costAmount", Group: Margins, Resource: permission.ResourceRateQuote},
}

// Paths lists every declared protected path.
func Paths() []Path {
	return protectedPaths[:]
}

const listStep = "[]"

func pathSteps(path string) []string {
	parts := strings.Split(path, ".")
	steps := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if name, isList := strings.CutSuffix(part, listStep); isList {
			steps = append(steps, name, listStep)
			continue
		}
		steps = append(steps, part)
	}
	return steps
}

// matches reports whether a field reached by steps, held by holder, of a
// record of resource is this protected path.
func (p *Path) matches(resource permission.Resource, steps []string, holder map[string]any) bool {
	want := pathSteps(p.Path)
	if p.Resource != "" {
		if p.Resource != resource || len(steps) != len(want) {
			return false
		}
	} else if len(steps) < len(want) {
		return false
	}
	tail := steps[len(steps)-len(want):]
	for i := range want {
		if tail[i] != want[i] {
			return false
		}
	}
	for key, value := range p.When {
		if holder == nil || holder[key] != value {
			return false
		}
	}
	return true
}

// GroupAt is the group that protects the field reached by steps in a
// record of resource, held by holder, if any does.
func GroupAt(resource permission.Resource, steps []string, holder map[string]any) (Group, bool) {
	for i := range protectedPaths {
		if protectedPaths[i].matches(resource, steps, holder) {
			return protectedPaths[i].Group, true
		}
	}
	return "", false
}

// WithholdJSON returns a record's JSON with every protected field a
// reader without its group may not see answered null. The value is copied
// before anything is withheld; one with nothing to withhold is returned as is.
func WithholdJSON(
	resource permission.Resource,
	value map[string]any,
	visible GroupSet,
) map[string]any {
	if value == nil || visible.HasAll() {
		return value
	}
	out, _ := withholdValue(resource, nil, value, visible).(map[string]any)
	return out
}

// WithholdValue is WithholdJSON for a value found at steps
// inside a record, such as one side of an audit change.
func WithholdValue(
	resource permission.Resource,
	steps []string,
	value any,
	visible GroupSet,
) any {
	if visible.HasAll() {
		return value
	}
	return withholdValue(resource, steps, value, visible)
}

func withholdValue(resource permission.Resource, steps []string, value any, visible GroupSet) any {
	switch typed := value.(type) {
	case map[string]any:
		out := maps.Clone(typed)
		for key, field := range typed {
			at := append(append(make([]string, 0, len(steps)+1), steps...), key)
			if group, ok := GroupAt(resource, at, typed); ok && !visible.Has(group) {
				out[key] = nil
				continue
			}
			out[key] = withholdValue(resource, at, field, visible)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		at := append(append(make([]string, 0, len(steps)+1), steps...), listStep)
		for i, element := range typed {
			out[i] = withholdValue(resource, at, element, visible)
		}
		return out
	default:
		return value
	}
}
