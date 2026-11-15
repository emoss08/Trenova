package managedfeatures

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const ChangeHistoryLimit = 200

type Change struct {
	keel.Table `keel:"managed_feature_changes"`

	ID        pulid.ID                     `json:"id"        keel:"id,pk,autoid:mfc_"`
	Added     []platformcatalog.FeatureKey `json:"added"     keel:"added,jsonb"`
	Removed   []platformcatalog.FeatureKey `json:"removed"   keel:"removed,jsonb"`
	Starting  bool                         `json:"starting"  keel:"starting"`
	ChangedAt int64                        `json:"changedAt" keel:"changed_at"`
}

type Replacement struct {
	Previous *State
	Current  *State
	Change   *Change
	Unknown  []platformcatalog.FeatureKey
}

func (r *Replacement) Changed() bool {
	return r != nil && r.Change != nil
}

func Replace(held *State, received []platformcatalog.FeatureKey, now int64) *Replacement {
	next := Normalize(received)
	if held != nil && slices.Equal(held.Features, next.Features) {
		return &Replacement{Previous: held, Current: held, Unknown: next.Unknown}
	}

	var before []platformcatalog.FeatureKey
	if held != nil {
		before = held.Features
	}
	added, removed := Changes(before, next.Features)

	return &Replacement{
		Previous: held,
		Current: &State{
			ID:         CurrentStateID,
			Features:   next.Features,
			ReceivedAt: now,
		},
		Change: &Change{
			Added:     added,
			Removed:   removed,
			Starting:  held == nil,
			ChangedAt: now,
		},
		Unknown: next.Unknown,
	}
}
