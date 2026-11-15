package managedfeatures

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

const PlanHistoryLimit = 20

type AddOnFeatures struct {
	AddOn    platformcatalog.AddOn
	Features []platformcatalog.Feature
}

type PlanView struct {
	Base      []platformcatalog.Feature
	Included  []AddOnFeatures
	Available []AddOnFeatures
}

type AddOnShift struct {
	Left   []platformcatalog.AddOn
	Joined []platformcatalog.AddOn
}

func (s AddOnShift) Empty() bool {
	return len(s.Left) == 0 && len(s.Joined) == 0
}

type HistoryEntry struct {
	ChangedAt int64
	Starting  bool
	Left      []platformcatalog.AddOn
	Joined    []platformcatalog.AddOn
}

func View(held []platformcatalog.FeatureKey) *PlanView {
	byAddOn := featuresByAddOn()
	on := addOnsOn(held)
	view := &PlanView{
		Base:      byAddOn[platformcatalog.AddOnBase],
		Included:  []AddOnFeatures{},
		Available: []AddOnFeatures{},
	}
	for _, addOn := range platformcatalog.AddOns() {
		entry := AddOnFeatures{AddOn: addOn, Features: byAddOn[addOn.Key]}
		if on[addOn.Key] {
			view.Included = append(view.Included, entry)
			continue
		}
		view.Available = append(view.Available, entry)
	}
	return view
}

func Shift(before, after []platformcatalog.FeatureKey) AddOnShift {
	was, is := addOnsOn(before), addOnsOn(after)
	shift := AddOnShift{Left: []platformcatalog.AddOn{}, Joined: []platformcatalog.AddOn{}}
	for _, addOn := range platformcatalog.AddOns() {
		switch {
		case was[addOn.Key] && !is[addOn.Key]:
			shift.Left = append(shift.Left, addOn)
		case !was[addOn.Key] && is[addOn.Key]:
			shift.Joined = append(shift.Joined, addOn)
		}
	}
	return shift
}

func History(current []platformcatalog.FeatureKey, changes []*Change) []HistoryEntry {
	out := make([]HistoryEntry, 0, len(changes))
	after := slices.Clone(current)
	for _, change := range changes {
		if change == nil {
			continue
		}
		before := undo(after, change)
		if change.Starting {
			before = nil
		}
		shift := Shift(before, after)
		if change.Starting || !shift.Empty() {
			out = append(out, HistoryEntry{
				ChangedAt: change.ChangedAt,
				Starting:  change.Starting,
				Left:      shift.Left,
				Joined:    shift.Joined,
			})
		}
		after = before
	}
	return out
}

func undo(after []platformcatalog.FeatureKey, change *Change) []platformcatalog.FeatureKey {
	before := make([]platformcatalog.FeatureKey, 0, len(after)+len(change.Removed))
	for _, key := range after {
		if !slices.Contains(change.Added, key) {
			before = append(before, key)
		}
	}
	for _, key := range change.Removed {
		if !slices.Contains(before, key) {
			before = append(before, key)
		}
	}
	return before
}

func addOnsOn(features []platformcatalog.FeatureKey) map[platformcatalog.AddOnKey]bool {
	on := make(map[platformcatalog.AddOnKey]bool, len(features))
	for _, key := range features {
		if addOn, ok := platformcatalog.AddOnFor(key); ok {
			on[addOn.Key] = true
		}
	}
	return on
}

func featuresByAddOn() map[platformcatalog.AddOnKey][]platformcatalog.Feature {
	out := make(map[platformcatalog.AddOnKey][]platformcatalog.Feature)
	for _, feature := range platformcatalog.Features() {
		out[feature.AddOn] = append(out[feature.AddOn], feature)
	}
	return out
}
