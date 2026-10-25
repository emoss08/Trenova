package driverpay

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*RecurringDeduction)(nil)
	_ validationframework.TenantedEntity = (*RecurringDeduction)(nil)
)

// Standing deductions, including garnishments and escrow contributions
type RecurringDeduction struct {
	keel.Table `keel:"recurring_deductions,report=Payroll"`

	ID                  pulid.ID           `json:"id"                  keel:"id,pk,autoid:rded_"`
	BusinessUnitID      pulid.ID           `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID           `json:"organizationId"      keel:"organization_id,pk,tenant"`
	WorkerID            pulid.ID           `json:"workerId"            keel:"worker_id"`
	PayCodeID           pulid.ID           `json:"payCodeId"           keel:"pay_code_id"`
	EscrowAccountID     *pulid.ID          `json:"escrowAccountId"     keel:"escrow_account_id"`
	Status              DeductionStatus    `json:"status"              keel:"status"`
	Kind                DeductionKind      `json:"kind"                keel:"kind"`
	CourtOrderNumber    string             `json:"courtOrderNumber"    keel:"court_order_number,nullzero"`
	CaseNumber          string             `json:"caseNumber"          keel:"case_number,nullzero"`
	IssuingAgency       string             `json:"issuingAgency"       keel:"issuing_agency,nullzero"`
	Priority            int16              `json:"priority"            keel:"priority,dimension"`
	Frequency           DeductionFrequency `json:"frequency"           keel:"frequency"`
	Description         string             `json:"description"         keel:"description"`
	AmountMinor         int64              `json:"amountMinor"         keel:"amount_minor,label=Amount (Minor)"`
	TotalCapMinor       *int64             `json:"totalCapMinor"       keel:"total_cap_minor,label=Total Cap (Minor)"`
	DeductedToDateMinor int64              `json:"deductedToDateMinor" keel:"deducted_to_date_minor,label=Deducted to Date (Minor)"`
	StartDate           int64              `json:"startDate"           keel:"start_date,label=Starts"`
	EndDate             *int64             `json:"endDate"             keel:"end_date,label=Ends"`
	CurrencyCode        string             `json:"currencyCode"        keel:"currency_code,label=Currency"`
	CreatedByID         pulid.ID           `json:"createdById"         keel:"created_by_id,nullzero"`
	Version             int64              `json:"version"             keel:"version,version"`
	CreatedAt           int64              `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64              `json:"updatedAt"           keel:"updated_at,autoupdate"`

	Worker        *worker.Worker `json:"worker,omitempty"        keel:"belongs:worker_id,label=Driver"`
	PayCode       *PayCode       `json:"payCode,omitempty"       keel:"belongs:pay_code_id"`
	EscrowAccount *EscrowAccount `json:"escrowAccount,omitempty" keel:"belongs:escrow_account_id"`
}

func (r *RecurringDeduction) IsEscrowContribution() bool {
	return r.EscrowAccountID != nil && !r.EscrowAccountID.IsNil()
}

func (r *RecurringDeduction) Validate(multiErr *errortypes.MultiError) {
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
		multiErr.Add("status", errortypes.ErrInvalid, "Deduction status is invalid")
	}
	if !r.Frequency.IsValid() {
		multiErr.Add("frequency", errortypes.ErrInvalid, "Deduction frequency is invalid")
	}
	if r.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Deduction amount must be greater than zero",
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
	// An empty kind is an ordinary deduction. The column defaults to Standard
	// and every deduction created before benefits existed has no kind on it, so
	// requiring one here would refuse rows the system itself writes.
	if r.Kind != "" && !r.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Deduction kind is invalid")
	}
	// Money taken from somebody's pay without a reference to the order
	// authorising it cannot be defended, so a garnishment is refused without one.
	if r.Kind == DeductionKindGarnishment && strings.TrimSpace(r.CourtOrderNumber) == "" {
		multiErr.Add(
			"courtOrderNumber",
			errortypes.ErrRequired,
			"A garnishment needs the order it is taken under",
		)
	}
	if r.Priority < 0 || r.Priority > 999 {
		multiErr.Add("priority", errortypes.ErrInvalid, "Priority is 0 to 999")
	}
}

func (r *RecurringDeduction) RemainingCapMinor() *int64 {
	if r.TotalCapMinor == nil {
		return nil
	}
	remaining := max(*r.TotalCapMinor-r.DeductedToDateMinor, 0)
	return &remaining
}

func (r *RecurringDeduction) NextAmountMinor() int64 {
	if r.Status != DeductionStatusActive {
		return 0
	}
	if remaining := r.RemainingCapMinor(); remaining != nil {
		return min(r.AmountMinor, *remaining)
	}
	return r.AmountMinor
}

func (r *RecurringDeduction) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rded",
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

func (r *RecurringDeduction) GetID() pulid.ID { return r.ID }

func (r *RecurringDeduction) GetCreatedAt() int64 { return r.CreatedAt }

func (r *RecurringDeduction) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RecurringDeduction) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RecurringDeduction) GetTableName() string { return "recurring_deductions" }

func (r *RecurringDeduction) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = DeductionStatusActive
	}
	if r.Kind == "" {
		r.Kind = DeductionKindStandard
	}
	if r.Priority == 0 {
		r.Priority = 100
	}
	if r.Frequency == "" {
		r.Frequency = DeductionFrequencyEverySettlement
	}
	if r.CurrencyCode == "" {
		r.CurrencyCode = money.DefaultCurrencyCode
	}
	if r.Priority <= 0 {
		r.Priority = r.Kind.DefaultPriority()
	}

	return nil
}
