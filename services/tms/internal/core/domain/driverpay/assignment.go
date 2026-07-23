package driverpay

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPayAssignment)(nil)
	_ validationframework.TenantedEntity = (*WorkerPayAssignment)(nil)
)

type RateOverride struct {
	ComponentID pulid.ID        `json:"componentId"`
	Rate        decimal.Decimal `json:"rate"`
}

type WorkerPayAssignment struct {
	bun.BaseModel `bun:"table:worker_pay_assignments,alias:wpa" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID        `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	PayProfileID   pulid.ID        `json:"payProfileId"   bun:"pay_profile_id,type:VARCHAR(100),notnull"`
	EffectiveFrom  int64           `json:"effectiveFrom"  bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo    *int64          `json:"effectiveTo"    bun:"effective_to,type:BIGINT,nullzero"`
	SplitPercent   decimal.Decimal `json:"splitPercent"   bun:"split_percent,type:NUMERIC(7,4),notnull,default:100"`
	RateOverrides  []RateOverride  `json:"rateOverrides"  bun:"rate_overrides,type:JSONB,nullzero"`
	Notes          string          `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	CreatedByID    pulid.ID        `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker     *worker.Worker `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PayProfile *PayProfile    `json:"payProfile,omitempty" bun:"rel:belongs-to,join:pay_profile_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *WorkerPayAssignment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.PayProfileID, validation.Required.Error("Pay profile is required")),
		validation.Field(
			&a.EffectiveFrom,
			validation.Required.Error("Effective from date is required"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if a.EffectiveTo != nil && *a.EffectiveTo <= a.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to date must be after the effective from date",
		)
	}
	if a.SplitPercent.IsNegative() || a.SplitPercent.GreaterThan(decimal.NewFromInt(100)) {
		multiErr.Add(
			"splitPercent",
			errortypes.ErrInvalid,
			"Split percent must be between 0 and 100",
		)
	}
	if a.SplitPercent.IsZero() {
		multiErr.Add(
			"splitPercent",
			errortypes.ErrInvalid,
			"Split percent must be greater than zero",
		)
	}
	seen := make(map[pulid.ID]struct{}, len(a.RateOverrides))
	for idx, override := range a.RateOverrides {
		overrideErr := multiErr.WithIndex("rateOverrides", idx)
		if override.ComponentID.IsNil() {
			overrideErr.Add(
				"componentId",
				errortypes.ErrRequired,
				"Override component is required",
			)
			continue
		}
		if _, dup := seen[override.ComponentID]; dup {
			overrideErr.Add(
				"componentId",
				errortypes.ErrDuplicate,
				"Each component may only be overridden once",
			)
			continue
		}
		seen[override.ComponentID] = struct{}{}
		if override.Rate.IsNegative() {
			overrideErr.Add("rate", errortypes.ErrInvalid, "Override rate cannot be negative")
		}
	}
}

func (a *WorkerPayAssignment) OverrideFor(componentID pulid.ID) (decimal.Decimal, bool) {
	for _, override := range a.RateOverrides {
		if override.ComponentID == componentID {
			return override.Rate, true
		}
	}
	return decimal.Zero, false
}

func (a *WorkerPayAssignment) IsEffectiveAt(ts int64) bool {
	if ts < a.EffectiveFrom {
		return false
	}
	return a.EffectiveTo == nil || ts < *a.EffectiveTo
}

func (a *WorkerPayAssignment) GetID() pulid.ID { return a.ID }

func (a *WorkerPayAssignment) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *WorkerPayAssignment) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *WorkerPayAssignment) GetTableName() string { return "worker_pay_assignments" }

func (a *WorkerPayAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("wpa_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}
