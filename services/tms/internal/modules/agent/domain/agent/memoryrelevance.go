package agent

import "github.com/emoss08/trenova/shared/pulid"

type MemoryRelevance struct {
	Judged bool
	IDs    map[pulid.ID]struct{}
}

func (r MemoryRelevance) Bears(id pulid.ID) bool {
	if !r.Judged {
		return true
	}
	_, ok := r.IDs[id]

	return ok
}
