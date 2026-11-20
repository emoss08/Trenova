package managedfeatures

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const ChangeHistoryLimit = 200

type Change struct {
	keel.Table `keel:"managed_feature_changes"`

	ID        pulid.ID                     `json:"id"        keel:"id,pk,autoid:mfc_"`
	ScopeKey  string                       `json:"scopeKey"  keel:"scope_key"`
	Added     []platformcatalog.FeatureKey `json:"added"     keel:"added,jsonb"`
	Removed   []platformcatalog.FeatureKey `json:"removed"   keel:"removed,jsonb"`
	Ending    []AddOnEnding                `json:"ending"    keel:"ending,jsonb"`
	Kept      []platformcatalog.AddOnKey   `json:"kept"      keel:"kept,jsonb"`
	Starting  bool                         `json:"starting"  keel:"starting"`
	ChangedAt int64                        `json:"changedAt" keel:"changed_at"`
}

type Replacement struct {
	Scope         managedscope.Scope
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

func (r *Replacement) LifecycleChanged() bool {
	if !r.Updated() {
		return false
	}
	return r.Previous.lifecycle() != r.Current.lifecycle()
}

func (r *Replacement) FeaturesChanged() bool {
	if !r.Updated() {
		return false
	}
	return !r.Previous.HasList() || !slices.Equal(r.Previous.Features, r.Current.Features)
}

func Replace(
	scope managedscope.Scope,
	held *State,
	received *Received,
	now int64,
) *Replacement {
	next := Normalize(received.Features)
	out := &Replacement{
		Scope:         scope,
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

	if held.HasList() &&
		slices.Equal(held.Features, next.Features) &&
		slices.Equal(held.AddOnEndings, endings) &&
		held.PlanURL == planURL &&
		held.BackupDays() == backupDays &&
		marks == marksOf(held) {
		out.Current = held
		return out
	}

	out.Current = &State{
		ScopeKey:            scope.Key(),
		ScopeBusinessUnitID: scope.BusinessUnitID(),
		Lifecycle:           held.lifecycle(),
		Features:            next.Features,
		AddOnEndings:        endings,
		PlanURL:             planURL,
		FinalBackupDays:     backupDays,
		ReceivedAt:          now,
	}
	marks.apply(out.Current)
	if notice != nil {
		notice.FinalBackupDays = backupDays
	}
	out.PlanEnd = notice
	out.Change = planChange(held, out.Current, now)
	return out
}

func SetLifecycle(
	scope managedscope.Scope,
	held *State,
	lifecycle Lifecycle,
) *Replacement {
	out := &Replacement{Scope: scope, Previous: held, Current: held}
	if held != nil && held.lifecycle() == lifecycle {
		return out
	}
	next := State{
		ScopeKey:            scope.Key(),
		ScopeBusinessUnitID: scope.BusinessUnitID(),
		Features:            []platformcatalog.FeatureKey{},
		AddOnEndings:        []AddOnEnding{},
		FinalBackupDays:     held.BackupDays(),
	}
	if held != nil {
		next = *held
	}
	next.Lifecycle = lifecycle
	out.Current = &next
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

	starting := !held.HasList()
	if !starting && len(added) == 0 && len(removed) == 0 && len(ending) == 0 && len(kept) == 0 {
		return nil
	}
	return &Change{
		ScopeKey:  current.ScopeKey,
		Added:     added,
		Removed:   removed,
		Ending:    ending,
		Kept:      kept,
		Starting:  starting,
		ChangedAt: now,
	}
}
