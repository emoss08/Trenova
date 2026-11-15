package managedfeatures

import (
	"errors"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

const CurrentStateID = "current"

var ErrNoState = errors.New("the control plane has not set a feature list yet")

type State struct {
	keel.Table `keel:"managed_feature_states"`

	ID         string                       `json:"id"         keel:"id,pk"`
	Features   []platformcatalog.FeatureKey `json:"features"   keel:"features,jsonb"`
	ReceivedAt int64                        `json:"receivedAt" keel:"received_at"`
}

func (s *State) Has(feature platformcatalog.FeatureKey) bool {
	if s == nil {
		return false
	}
	return slices.Contains(s.Features, feature)
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
