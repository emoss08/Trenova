package audit

import (
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
)

const (
	changeFrom = "from"
	changeTo   = "to"
)

// WithholdFields answers null, in the entry's states and changes, every
// protected field whose group the reader does not hold: an audit entry is the
// record as it stood, and shows no more of it than the record would.
// Nothing is changed in place: each withheld map is a copy, so a reader with
// the group, or a cached entry, still has the whole diff.
func (e *Entry) WithholdFields(visible protectedfield.GroupSet) {
	if e == nil || visible.HasAll() {
		return
	}
	e.PreviousState = protectedfield.WithholdJSON(e.Resource, e.PreviousState, visible)
	e.CurrentState = protectedfield.WithholdJSON(e.Resource, e.CurrentState, visible)
	e.Changes = WithholdChanges(e.Resource, e.Changes, visible)
}

// WithheldCopy is the entry as a reader holding only these groups may see it.
func (e *Entry) WithheldCopy(visible protectedfield.GroupSet) *Entry {
	if e == nil || visible.HasAll() {
		return e
	}
	out := *e
	out.WithholdFields(visible)
	return &out
}

// WithholdChanges withholds a diff keyed by dotted path. A change to a
// protected field keeps its path and type and loses both values; a change to
// an object or a list keeps its values with the protected fields inside them
// withheld.
func WithholdChanges(
	resource permission.Resource,
	changes map[string]any,
	visible protectedfield.GroupSet,
) map[string]any {
	if changes == nil || visible.HasAll() {
		return changes
	}
	out := make(map[string]any, len(changes))
	for path, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			out[path] = raw
			continue
		}
		steps := strings.Split(path, ".")
		withheld := maps.Clone(change)
		if group, protected := protectedfield.GroupAt(resource, steps, nil); protected &&
			!visible.Has(group) {
			withheld[changeFrom] = nil
			withheld[changeTo] = nil
			out[path] = withheld
			continue
		}
		for _, side := range []string{changeFrom, changeTo} {
			if value, has := change[side]; has {
				withheld[side] = protectedfield.WithholdValue(resource, steps, value, visible)
			}
		}
		out[path] = withheld
	}
	return out
}
