package extractionrollout

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	DefaultPercent                    = 5
	MinPercent                        = 1
	MaxPercent                        = 100
	DefaultMaxAccuracyDropPoints      = 5
	MinMaxAccuracyDropPoints          = 1
	MaxMaxAccuracyDropPoints          = 50
	DefaultMaxRejectionIncreasePoints = 10
	MinMaxRejectionIncreasePoints     = 1
	MaxMaxRejectionIncreasePoints     = 100
)

type ExtractionRollout struct {
	keel.Table `keel:"extraction_rollouts"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:exro_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Enabled    bool      `json:"enabled" keel:"enabled"`
	ProviderID *pulid.ID `json:"providerId" keel:"provider_id"`
	Percent    int       `json:"percent" keel:"percent"`

	MaxAccuracyDropPoints      int `json:"maxAccuracyDropPoints" keel:"max_accuracy_drop_points"`
	MaxRejectionIncreasePoints int `json:"maxRejectionIncreasePoints" keel:"max_rejection_increase_points"`

	StartedAt         *int64     `json:"startedAt" keel:"started_at"`
	HaltedAt          *int64     `json:"haltedAt" keel:"halted_at"`
	HaltReason        HaltReason `json:"haltReason" keel:"halt_reason,nullzero"`
	HaltCandidateRate float64    `json:"haltCandidateRate" keel:"halt_candidate_rate"`
	HaltBaselineRate  float64    `json:"haltBaselineRate" keel:"halt_baseline_rate"`

	UpdatedByID *pulid.ID `json:"updatedById" keel:"updated_by_id"`
	Version     int64     `json:"version" keel:"version,version"`
	CreatedAt   int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt   int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func Default(orgID, buID pulid.ID) *ExtractionRollout {
	return &ExtractionRollout{
		OrganizationID:             orgID,
		BusinessUnitID:             buID,
		Percent:                    DefaultPercent,
		MaxAccuracyDropPoints:      DefaultMaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: DefaultMaxRejectionIncreasePoints,
	}
}

func (r *ExtractionRollout) IsHalted() bool { return r != nil && r.HaltedAt != nil }

func (r *ExtractionRollout) CandidateID() pulid.ID {
	if r == nil || !r.Enabled || r.IsHalted() || r.ProviderID == nil {
		return pulid.Nil
	}

	return *r.ProviderID
}

func (r *ExtractionRollout) Assigns(documentID pulid.ID) bool {
	return hashutils.InPercentSample("rollout:"+documentID.String(), r.Percent)
}

func (r *ExtractionRollout) Halt(breach Breach, now int64) {
	r.HaltedAt = &now
	r.HaltReason = breach.Reason
	r.HaltCandidateRate = breach.CandidateRate
	r.HaltBaselineRate = breach.BaselineRate
}

func (r *ExtractionRollout) ClearHalt() {
	r.HaltedAt = nil
	r.HaltReason = ""
	r.HaltCandidateRate = 0
	r.HaltBaselineRate = 0
}

type Change struct {
	Enabled                    bool
	ProviderID                 pulid.ID
	Percent                    int
	MaxAccuracyDropPoints      int
	MaxRejectionIncreasePoints int
}

func (r *ExtractionRollout) Apply(change *Change, now int64) {
	startsOver := change.Enabled &&
		(!r.Enabled || r.IsHalted() || r.ProviderID == nil || *r.ProviderID != change.ProviderID)

	r.Enabled = change.Enabled
	r.ProviderID = pulid.PtrOrNil(change.ProviderID)
	r.Percent = change.Percent
	r.MaxAccuracyDropPoints = change.MaxAccuracyDropPoints
	r.MaxRejectionIncreasePoints = change.MaxRejectionIncreasePoints

	if startsOver {
		r.StartedAt = &now
		r.ClearHalt()
	}
}

func (r *ExtractionRollout) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.Percent, bounds(
			MinPercent,
			MaxPercent,
			fmt.Sprintf(
				"Send between %d and %d percent of documents to the candidate",
				MinPercent,
				MaxPercent,
			),
		)...),
		validation.Field(&r.MaxAccuracyDropPoints, bounds(
			MinMaxAccuracyDropPoints, MaxMaxAccuracyDropPoints,
			fmt.Sprintf(
				"The accuracy guard must allow between %d and %d points",
				MinMaxAccuracyDropPoints, MaxMaxAccuracyDropPoints,
			),
		)...),
		validation.Field(&r.MaxRejectionIncreasePoints, bounds(
			MinMaxRejectionIncreasePoints, MaxMaxRejectionIncreasePoints,
			fmt.Sprintf(
				"The rejection guard must allow between %d and %d points",
				MinMaxRejectionIncreasePoints, MaxMaxRejectionIncreasePoints,
			),
		)...),
	))

	if r.Enabled && (r.ProviderID == nil || r.ProviderID.IsNil()) {
		multiErr.Add(
			"providerId",
			errortypes.ErrRequired,
			"Choose the AI provider to roll out",
		)
	}
}

func bounds(minimum, maximum int, message string) []validation.Rule {
	return []validation.Rule{
		validation.Min(minimum).Error(message),
		validation.Max(maximum).Error(message),
	}
}

func (r *ExtractionRollout) GetID() pulid.ID { return r.ID }

func (r *ExtractionRollout) GetTableName() string { return "extraction_rollouts" }

func (r *ExtractionRollout) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ExtractionRollout) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ExtractionRollout) BeforeCreate(_ context.Context) error {
	if r.Percent == 0 {
		r.Percent = 5
	}
	if r.MaxAccuracyDropPoints == 0 {
		r.MaxAccuracyDropPoints = 5
	}
	if r.MaxRejectionIncreasePoints == 0 {
		r.MaxRejectionIncreasePoints = 10
	}
	if r.Percent == 0 {
		r.Percent = DefaultPercent
	}
	if r.MaxAccuracyDropPoints == 0 {
		r.MaxAccuracyDropPoints = DefaultMaxAccuracyDropPoints
	}
	if r.MaxRejectionIncreasePoints == 0 {
		r.MaxRejectionIncreasePoints = DefaultMaxRejectionIncreasePoints
	}

	return nil
}
