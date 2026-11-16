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
	Ending    []AddOnEnding                `json:"ending"    keel:"ending,jsonb"`
	Kept      []platformcatalog.AddOnKey   `json:"kept"      keel:"kept,jsonb"`
	Starting  bool                         `json:"starting"  keel:"starting"`
	ChangedAt int64                        `json:"changedAt" keel:"changed_at"`
}

type Replacement struct {
	Previous      *State
	Current       *State
	Change        *Change
	Unknown       []platformcatalog.FeatureKey
	UnknownAddOns []platformcatalog.AddOnKey
	PlanEnd       *PlanEndNotice
}

func (r *Replacement) Updated() bool {
	return r != nil && r.Current != r.Previous
}

func (r *Replacement) Changed() bool {
	return r != nil && r.Change != nil
}

func (r *Replacement) FeaturesChanged() bool {
	if !r.Updated() {
		return false
	}
	return r.Previous == nil || !slices.Equal(r.Previous.Features, r.Current.Features)
}

func Replace(held *State, received *Received, now int64) *Replacement {
	next := Normalize(received.Features)
	out := &Replacement{
		Previous:      held,
		Unknown:       next.Unknown,
		UnknownAddOns: []platformcatalog.AddOnKey{},
	}

	var endings []AddOnEnding
	var planURL string
	backupDays := held.BackupDays()
	if held != nil {
		endings, planURL = held.AddOnEndings, held.PlanURL
	}
	if received.FinalBackupDays > 0 {
		backupDays = received.FinalBackupDays
	}
	if received.AddOnEndings != nil {
		normalized := NormalizeEndings(*received.AddOnEndings)
		endings, out.UnknownAddOns = normalized.Endings, normalized.Unknown
	}
	if received.PlanURL != nil {
		planURL = *received.PlanURL
	}
	if endings == nil {
		endings = []AddOnEnding{}
	}
	marks, notice := announcePlanEnd(marksOf(held), received.PlanEndsAt, now)

	if held != nil &&
		slices.Equal(held.Features, next.Features) &&
		slices.Equal(held.AddOnEndings, endings) &&
		held.PlanURL == planURL &&
		held.BackupDays() == backupDays &&
		marks == marksOf(held) {
		out.Current = held
		return out
	}

	out.Current = &State{
		ID:              CurrentStateID,
		Features:        next.Features,
		AddOnEndings:    endings,
		PlanURL:         planURL,
		FinalBackupDays: backupDays,
		ReceivedAt:      now,
	}
	marks.apply(out.Current)
	if notice != nil {
		notice.FinalBackupDays = backupDays
	}
	out.PlanEnd = notice
	out.Change = planChange(held, out.Current, now)
	return out
}

func planChange(held, current *State, now int64) *Change {
	var beforeFeatures []platformcatalog.FeatureKey
	var beforeEndings []AddOnEnding
	if held != nil {
		beforeFeatures, beforeEndings = held.Features, held.AddOnEndings
	}
	added, removed := Changes(beforeFeatures, current.Features)
	on := addOnsOn(current.Features)

	ending := []AddOnEnding{}
	for _, e := range current.AddOnEndings {
		if on[e.AddOn] && !slices.Contains(beforeEndings, e) {
			ending = append(ending, e)
		}
	}
	kept := []platformcatalog.AddOnKey{}
	for _, e := range beforeEndings {
		if on[e.AddOn] && endsAt(current.AddOnEndings, e.AddOn) == 0 {
			kept = append(kept, e.AddOn)
		}
	}

	if held != nil && len(added) == 0 && len(removed) == 0 && len(ending) == 0 && len(kept) == 0 {
		return nil
	}
	return &Change{
		Added:     added,
		Removed:   removed,
		Ending:    ending,
		Kept:      kept,
		Starting:  held == nil,
		ChangedAt: now,
	}
}
