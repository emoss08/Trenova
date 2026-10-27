package driverpay

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*WorkerPayAssignment)(nil)
)

type RateOverride struct {
	ComponentID pulid.ID        `json:"componentId"`
	Rate        decimal.Decimal `json:"rate"`
}

type WorkerPayAssignment struct {
	keel.Table `keel:"worker_pay_assignments"`

	ID             pulid.ID        `json:"id"             keel:"id,pk,autoid:wpa_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID        `json:"workerId"       keel:"worker_id"`
	PayProfileID   pulid.ID        `json:"payProfileId"   keel:"pay_profile_id"`
	EffectiveFrom  int64           `json:"effectiveFrom"  keel:"effective_from"`
	EffectiveTo    *int64          `json:"effectiveTo"    keel:"effective_to"`
	SplitPercent   decimal.Decimal `json:"splitPercent"   keel:"split_percent"`
	RateOverrides  []RateOverride  `json:"rateOverrides"  keel:"rate_overrides,jsonb,nullzero"`
	Notes          string          `json:"notes"          keel:"notes,nullzero"`
	CreatedByID    pulid.ID        `json:"createdById"    keel:"created_by_id,nullzero"`
	Version        int64           `json:"version"        keel:"version,version"`
	CreatedAt      int64           `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Worker     *worker.Worker `json:"worker,omitempty"     keel:"belongs:worker_id"`
	PayProfile *PayProfile    `json:"payProfile,omitempty" keel:"belongs:pay_profile_id"`
}

func (a *WorkerPayAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.PayProfileID, validation.Required.Error("Pay profile is required")),
		validation.Field(
			&a.EffectiveFrom,
			validation.Required.Error("Effective from date is required"),
		),
	))

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

func (a *WorkerPayAssignment) BeforeCreate(ctx context.Context) error {
	if a.SplitPercent.IsZero() {
		a.SplitPercent = decimal.RequireFromString("100")
	}

	return nil
}
