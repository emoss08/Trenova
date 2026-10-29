package recurringpay

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*RecurringEarning)(nil)
	_ validationframework.TenantedEntity = (*RecurringEarning)(nil)
)

// Standing earnings such as per diem and retention bonuses
type RecurringEarning struct {
	keel.Table `keel:"recurring_earnings,report=Payroll"`

	ID              pulid.ID         `json:"id"              keel:"id,pk,autoid:rern_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId"  keel:"organization_id,pk,tenant"`
	WorkerID        pulid.ID         `json:"workerId"        keel:"worker_id"`
	PayCodeID       pulid.ID         `json:"payCodeId"       keel:"pay_code_id"`
	Status          EarningStatus    `json:"status"          keel:"status"`
	Kind            EarningKind      `json:"kind"            keel:"kind"`
	Frequency       EarningFrequency `json:"frequency"       keel:"frequency"`
	Description     string           `json:"description"     keel:"description"`
	AmountMinor     int64            `json:"amountMinor"     keel:"amount_minor,label=Amount (Minor)"`
	TotalCapMinor   *int64           `json:"totalCapMinor"   keel:"total_cap_minor,label=Total Cap (Minor)"`
	PaidToDateMinor int64            `json:"paidToDateMinor" keel:"paid_to_date_minor,label=Paid to Date (Minor)"`
	StartDate       int64            `json:"startDate"       keel:"start_date,label=Starts"`
	EndDate         *int64           `json:"endDate"         keel:"end_date,label=Ends"`
	CurrencyCode    string           `json:"currencyCode"    keel:"currency_code,label=Currency"`
	CreatedByID     pulid.ID         `json:"createdById"     keel:"created_by_id,nullzero"`
	Version         int64            `json:"version"         keel:"version,version"`
	CreatedAt       int64            `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64            `json:"updatedAt"       keel:"updated_at,autoupdate"`

	Worker  *worker.Worker   `json:"worker,omitempty"  keel:"belongs:worker_id,label=Driver"`
	PayCode *paycode.PayCode `json:"payCode,omitempty" keel:"belongs:pay_code_id"`
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
				Name:   "description",
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

func (r *RecurringEarning) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = EarningStatusActive
	}
	if r.Kind == "" {
		r.Kind = EarningKindStandard
	}
	if r.Frequency == "" {
		r.Frequency = EarningFrequencyEverySettlement
	}
	if r.CurrencyCode == "" {
		r.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (RecurringEarning) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_recurring_earnings_pay_code",
			Message: "Pay code does not exist in your organization",
		},
	}
}
