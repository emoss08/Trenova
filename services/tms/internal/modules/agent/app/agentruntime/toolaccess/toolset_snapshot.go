package toolaccess

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func (t *ToolSet) State() ToolSetState {
	return ToolSetState{
		Specs:      append([]kernelports.ToolSpec(nil), t.Specs...),
		Deferred:   slices.Clone(t.Deferred),
		Loaded:     t.loadedDeferred(),
		Pending:    slices.Clone(t.Pending),
		Allowed:    append([]string(nil), t.Allowed...),
		Disclosed:  t.Disclosed,
		Unattended: t.unattended,
		FindCalls:  t.FindCalls,
		Beyond:     slices.Clone(t.Beyond),
		Query:      packQueryVector(t.Query),
	}
}

func (t *ToolSet) loadedDeferred() []string {
	loaded := make([]string, 0, len(t.Loaded)-len(t.Specs))
	for _, name := range t.Deferred {
		if _, ok := t.Loaded[name]; ok {
			loaded = append(loaded, name)
		}
	}

	return loaded
}
