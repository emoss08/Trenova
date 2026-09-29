package extractionrollout

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ExtractionRollout)(nil)

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
	bun.BaseModel `bun:"table:extraction_rollouts,alias:exro" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Enabled    bool      `json:"enabled"    bun:"enabled,type:BOOLEAN,notnull,default:false"`
	ProviderID *pulid.ID `json:"providerId" bun:"provider_id,type:VARCHAR(100),nullzero"`
	Percent    int       `json:"percent"    bun:"percent,type:INTEGER,notnull,default:5"`

	MaxAccuracyDropPoints      int `json:"maxAccuracyDropPoints"      bun:"max_accuracy_drop_points,type:INTEGER,notnull,default:5"`
	MaxRejectionIncreasePoints int `json:"maxRejectionIncreasePoints" bun:"max_rejection_increase_points,type:INTEGER,notnull,default:10"`

	StartedAt         *int64     `json:"startedAt"         bun:"started_at,type:BIGINT,nullzero"`
	HaltedAt          *int64     `json:"haltedAt"          bun:"halted_at,type:BIGINT,nullzero"`
	HaltReason        HaltReason `json:"haltReason"        bun:"halt_reason,type:VARCHAR(30),nullzero"`
	HaltCandidateRate float64    `json:"haltCandidateRate" bun:"halt_candidate_rate,type:DOUBLE PRECISION,notnull,default:0"`
	HaltBaselineRate  float64    `json:"haltBaselineRate"  bun:"halt_baseline_rate,type:DOUBLE PRECISION,notnull,default:0"`

	UpdatedByID *pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version     int64     `json:"version"     bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt   int64     `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64     `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (r *ExtractionRollout) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("exro_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = now
		}
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
