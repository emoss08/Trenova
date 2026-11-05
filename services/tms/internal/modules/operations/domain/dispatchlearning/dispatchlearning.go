package dispatchlearning

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Weight struct {
	Multiplier float64 `json:"multiplier"`
	Samples    int64   `json:"samples"`
}

type DispatchStyle struct {
	keel.Table `keel:"dispatch_styles"`

	ID                pulid.ID                               `json:"id"                keel:"id,pk,autoid:dsty_"`
	BusinessUnitID    pulid.ID                               `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID                               `json:"organizationId"    keel:"organization_id,pk,tenant"`
	Weights           map[dispatcheconomics.Component]Weight `json:"weights"           keel:"weights,jsonb"`
	Decisions         int64                                  `json:"decisions"         keel:"decisions"`
	Approvals         int64                                  `json:"approvals"         keel:"approvals"`
	Overrides         int64                                  `json:"overrides"         keel:"overrides"`
	Declines          int64                                  `json:"declines"          keel:"declines"`
	LearningSince     int64                                  `json:"learningSince"     keel:"learning_since"`
	ObservedThroughAt int64                                  `json:"observedThroughAt" keel:"observed_through_at"`
	ObservedThroughID pulid.ID                               `json:"observedThroughId" keel:"observed_through_id,nullzero"`
	Version           int64                                  `json:"version"           keel:"version,version"`
	CreatedAt         int64                                  `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64                                  `json:"updatedAt"         keel:"updated_at,autoupdate"`
}

type LearnedWeight struct {
	Component  dispatcheconomics.Component `json:"component"`
	Multiplier float64                     `json:"multiplier"`
	Samples    int64                       `json:"samples"`
}

func New(orgID, buID pulid.ID, now int64) *DispatchStyle {
	return &DispatchStyle{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Weights:        map[dispatcheconomics.Component]Weight{},
		LearningSince:  now,
	}
}

func (s *DispatchStyle) Multipliers() map[dispatcheconomics.Component]float64 {
	out := make(map[dispatcheconomics.Component]float64, len(s.Weights))
	for component, weight := range s.Weights {
		if !component.Soft() || weight.Samples <= 0 {
			continue
		}
		out[component] = clampMultiplier(weight.Multiplier)
	}

	return out
}

func (s *DispatchStyle) Learned() []LearnedWeight {
	out := make([]LearnedWeight, 0, len(s.Weights))
	for _, component := range dispatcheconomics.SoftComponents() {
		weight, ok := s.Weights[component]
		if !ok || weight.Samples <= 0 {
			continue
		}
		out = append(out, LearnedWeight{
			Component:  component,
			Multiplier: clampMultiplier(weight.Multiplier),
			Samples:    weight.Samples,
		})
	}
	slices.SortStableFunc(out, func(a, b LearnedWeight) int {
		return cmp.Compare(math.Abs(math.Log(b.Multiplier)), math.Abs(math.Log(a.Multiplier)))
	})

	return out
}

func (s *DispatchStyle) Reset(now int64) {
	s.Weights = map[dispatcheconomics.Component]Weight{}
	s.Decisions = 0
	s.Approvals = 0
	s.Overrides = 0
	s.Declines = 0
	s.LearningSince = now
}

func (s *DispatchStyle) Advance(at int64, id pulid.ID) {
	if at < s.ObservedThroughAt {
		return
	}
	s.ObservedThroughAt = at
	s.ObservedThroughID = id
}

func (s *DispatchStyle) Validate(multiErr *errortypes.MultiError) {
	for component, weight := range s.Weights {
		field := "weights." + string(component)
		if !component.Soft() {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"Only a preference can carry a learned weight",
			)
			continue
		}
		if math.IsNaN(weight.Multiplier) || math.IsInf(weight.Multiplier, 0) ||
			weight.Multiplier < dispatcheconomics.MinMultiplier ||
			weight.Multiplier > dispatcheconomics.MaxMultiplier {
			multiErr.Add(
				field+".multiplier",
				errortypes.ErrInvalid,
				"A learned weight must be between 0.25 and 4 times the default",
			)
		}
		if weight.Samples < 0 {
			multiErr.Add(field+".samples", errortypes.ErrInvalid, "Samples cannot be negative")
		}
	}
	if s.Approvals < 0 || s.Overrides < 0 || s.Declines < 0 ||
		s.Decisions != s.Approvals+s.Overrides+s.Declines {
		multiErr.Add(
			"decisions",
			errortypes.ErrInvalid,
			"Decisions must be the sum of approvals, overrides and declines",
		)
	}
}

func clampMultiplier(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return 1
	}

	return math.Min(
		math.Max(value, dispatcheconomics.MinMultiplier),
		dispatcheconomics.MaxMultiplier,
	)
}
