package managedfeatures

import (
	"errors"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultFinalBackupDays = 30
	MaxFinalBackupDays     = 3650
)

var (
	ErrNoState          = errors.New("the control plane has not set a feature list yet")
	ErrUnknownLifecycle = errors.New("a scope's lifecycle is active or suspended")
)

type Lifecycle string

const (
	LifecycleActive    = Lifecycle("active")
	LifecycleSuspended = Lifecycle("suspended")
)

func (l Lifecycle) IsValid() bool {
	return l == LifecycleActive || l == LifecycleSuspended
}

type State struct {
	keel.Table `keel:"managed_feature_states"`

	ScopeKey            string                       `json:"scopeKey"            keel:"scope_key,pk"`
	ScopeBusinessUnitID pulid.ID                     `json:"scopeBusinessUnitId" keel:"scope_business_unit_id,nullzero"`
	Lifecycle           Lifecycle                    `json:"lifecycle"           keel:"lifecycle"`
	Features            []platformcatalog.FeatureKey `json:"features"            keel:"features,jsonb"`
	AddOnEndings        []AddOnEnding                `json:"addOnEndings"        keel:"addon_endings,jsonb"`
	PlanURL             string                       `json:"planUrl"             keel:"plan_url,nullzero"`
	PlanEndsAt          int64                        `json:"planEndsAt"          keel:"plan_ends_at,nullzero"`
	PlanEndAnnounced    int64                        `json:"planEndAnnounced"    keel:"plan_end_announced,nullzero"`
	PlanEndReminded     int64                        `json:"planEndReminded"     keel:"plan_end_reminded,nullzero"`
	FinalBackupDays     int                          `json:"finalBackupDays"     keel:"final_backup_days"`
	ReceivedAt          int64                        `json:"receivedAt"          keel:"received_at"`
}

func (s *State) Scope() managedscope.Scope {
	return managedscope.FromKey(s.ScopeKey, s.ScopeBusinessUnitID)
}

func (s *State) Suspended() bool {
	return s != nil && s.Lifecycle == LifecycleSuspended
}

func (s *State) HasList() bool {
	return s != nil && s.ReceivedAt > 0
}

func (s *State) lifecycle() Lifecycle {
	if s == nil || !s.Lifecycle.IsValid() {
		return LifecycleActive
	}
	return s.Lifecycle
}

func (s *State) BackupDays() int {
	if s == nil || s.FinalBackupDays < 1 {
		return DefaultFinalBackupDays
	}
	return s.FinalBackupDays
}

func (s *State) Has(feature platformcatalog.FeatureKey) bool {
	if s == nil || s.Suspended() {
		return false
	}
	return slices.Contains(s.Features, feature)
}

func (s *State) EndsAt(addOn platformcatalog.AddOnKey) int64 {
	if s == nil {
		return 0
	}
	return endsAt(s.AddOnEndings, addOn)
}

func endsAt(endings []AddOnEnding, addOn platformcatalog.AddOnKey) int64 {
	for _, ending := range endings {
		if ending.AddOn == addOn {
			return ending.EndsAt
		}
	}
	return 0
}

type Normalized struct {
	Features []platformcatalog.FeatureKey
	Unknown  []platformcatalog.FeatureKey
}

func Normalize(received []platformcatalog.FeatureKey) Normalized {
	out := Normalized{
		Features: make([]platformcatalog.FeatureKey, 0, len(received)),
		Unknown:  []platformcatalog.FeatureKey{},
	}
	for _, key := range received {
		feature, known := platformcatalog.Lookup(key)
		switch {
		case !known:
			out.Unknown = append(out.Unknown, key)
		case !feature.IsBase():
			out.Features = append(out.Features, key)
		}
	}
	slices.Sort(out.Features)
	out.Features = slices.Compact(out.Features)
	slices.Sort(out.Unknown)
	out.Unknown = slices.Compact(out.Unknown)
	return out
}

func Changes(
	held, next []platformcatalog.FeatureKey,
) (added, removed []platformcatalog.FeatureKey) {
	added = []platformcatalog.FeatureKey{}
	removed = []platformcatalog.FeatureKey{}
	for _, key := range next {
		if !slices.Contains(held, key) {
			added = append(added, key)
		}
	}
	for _, key := range held {
		if !slices.Contains(next, key) {
			removed = append(removed, key)
		}
	}
	return added, removed
}
