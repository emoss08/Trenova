package driverpay

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RecurringEarning)(nil)
	_ pagination.CursorEntity            = (*RecurringEarning)(nil)
	_ validationframework.TenantedEntity = (*RecurringEarning)(nil)
)

type RecurringEarning struct {
	bun.BaseModel             `bun:"table:recurring_earnings,alias:rern" json:"-"`
	keel.Table                `keel:"recurring_earnings"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID              pulid.ID         `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:rern_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	WorkerID        pulid.ID         `json:"workerId"        bun:"worker_id,type:VARCHAR(100),notnull" keel:"worker_id"`
	PayCodeID       pulid.ID         `json:"payCodeId"       bun:"pay_code_id,type:VARCHAR(100),notnull" keel:"pay_code_id"`
	Status          EarningStatus    `json:"status"          bun:"status,type:VARCHAR(50),notnull,default:'Active'" keel:"status"`
	Kind            EarningKind      `json:"kind"            bun:"kind,type:VARCHAR(50),notnull,default:'Standard'" keel:"kind"`
	Frequency       EarningFrequency `json:"frequency"       bun:"frequency,type:VARCHAR(50),notnull,default:'EverySettlement'" keel:"frequency"`
	Description     string           `json:"description"     bun:"description,type:VARCHAR(255),notnull" keel:"description"`
	AmountMinor     int64            `json:"amountMinor"     bun:"amount_minor,type:BIGINT,notnull" keel:"amount_minor"`
	TotalCapMinor   *int64           `json:"totalCapMinor"   bun:"total_cap_minor,type:BIGINT,nullzero" keel:"total_cap_minor"`
	PaidToDateMinor int64            `json:"paidToDateMinor" bun:"paid_to_date_minor,type:BIGINT,notnull" keel:"paid_to_date_minor"`
	StartDate       int64            `json:"startDate"       bun:"start_date,type:BIGINT,notnull" keel:"start_date"`
	EndDate         *int64           `json:"endDate"         bun:"end_date,type:BIGINT,nullzero" keel:"end_date"`
	CurrencyCode    string           `json:"currencyCode"    bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	CreatedByID     pulid.ID         `json:"createdById"     bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	Version         int64            `json:"version"         bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt       int64            `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Worker  *worker.Worker `json:"worker,omitempty"  bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:worker_id"`
	PayCode *PayCode       `json:"payCode,omitempty" bun:"rel:belongs-to,join:pay_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:pay_code_id"`
}

func (r *RecurringEarning) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.PayCodeID, validation.Required.Error("Pay code is required")),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 255).Error("Description must be between 1 and 255 characters"),
		),
		validation.Field(&r.StartDate, validation.Required.Error("Start date is required")),
		validation.Field(&r.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

	if !r.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Earning status is invalid")
	}
	if !r.Frequency.IsValid() {
		multiErr.Add("frequency", errortypes.ErrInvalid, "Earning frequency is invalid")
	}
	if r.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Earning amount must be greater than zero",
		)
	}
	if r.TotalCapMinor != nil && *r.TotalCapMinor <= 0 {
		multiErr.Add(
			"totalCapMinor",
			errortypes.ErrInvalid,
			"Total cap must be greater than zero when provided",
		)
	}
	if r.EndDate != nil && *r.EndDate <= r.StartDate {
		multiErr.Add(
			"endDate",
			errortypes.ErrInvalid,
			"End date must be after the start date",
		)
	}
}

func (r *RecurringEarning) RemainingCapMinor() *int64 {
	if r.TotalCapMinor == nil {
		return nil
	}
	remaining := max(*r.TotalCapMinor-r.PaidToDateMinor, 0)
	return &remaining
}

func (r *RecurringEarning) NextAmountMinor() int64 {
	if r.Status != EarningStatusActive {
		return 0
	}
	if remaining := r.RemainingCapMinor(); remaining != nil {
		return min(r.AmountMinor, *remaining)
	}
	return r.AmountMinor
}

func (r *RecurringEarning) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rern",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   searchFieldDescription,
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (r *RecurringEarning) GetID() pulid.ID { return r.ID }

func (r *RecurringEarning) GetCreatedAt() int64 { return r.CreatedAt }

func (r *RecurringEarning) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RecurringEarning) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RecurringEarning) GetTableName() string { return "recurring_earnings" }

func (r *RecurringEarning) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rern_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}

func (r *RecurringEarning) BeforeCreate(ctx context.Context) error {
	return r.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
