package worker

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidPTOLedgerEntryType = errors.New("invalid PTO ledger entry type")
	ErrInvalidPTOLedgerActorType = errors.New("invalid PTO ledger actor type")
)

type PTOLedgerEntryType string

const (
	PTOLedgerEntryOpeningBalance = PTOLedgerEntryType("OpeningBalance")
	PTOLedgerEntryAccrual        = PTOLedgerEntryType("Accrual")
	PTOLedgerEntryUsage          = PTOLedgerEntryType("Usage")
	PTOLedgerEntryReversal       = PTOLedgerEntryType("Reversal")
	PTOLedgerEntryAdjustment     = PTOLedgerEntryType("Adjustment")
	PTOLedgerEntryCarryover      = PTOLedgerEntryType("Carryover")
	PTOLedgerEntryExpiry         = PTOLedgerEntryType("Expiry")
	PTOLedgerEntryPayout         = PTOLedgerEntryType("Payout")
	PTOLedgerEntryForfeiture     = PTOLedgerEntryType("Forfeiture")
)

func (t PTOLedgerEntryType) String() string { return string(t) }

func (t PTOLedgerEntryType) IsValid() bool {
	switch t {
	case PTOLedgerEntryOpeningBalance, PTOLedgerEntryAccrual, PTOLedgerEntryUsage,
		PTOLedgerEntryReversal, PTOLedgerEntryAdjustment, PTOLedgerEntryCarryover,
		PTOLedgerEntryExpiry, PTOLedgerEntryPayout, PTOLedgerEntryForfeiture:
		return true
	default:
		return false
	}
}

func (t PTOLedgerEntryType) RequiredSign() int {
	switch t {
	case PTOLedgerEntryOpeningBalance, PTOLedgerEntryAccrual, PTOLedgerEntryReversal,
		PTOLedgerEntryCarryover:
		return 1
	case PTOLedgerEntryUsage, PTOLedgerEntryExpiry, PTOLedgerEntryPayout, PTOLedgerEntryForfeiture:
		return -1
	case PTOLedgerEntryAdjustment:
		return 0
	default:
		return 0
	}
}

func (t PTOLedgerEntryType) CountsAsAccrual() bool {
	return t == PTOLedgerEntryAccrual || t == PTOLedgerEntryOpeningBalance
}

type PTOLedgerActorType string

const (
	PTOLedgerActorUser   = PTOLedgerActorType("User")
	PTOLedgerActorSystem = PTOLedgerActorType("System")
)

func (a PTOLedgerActorType) String() string { return string(a) }

func (a PTOLedgerActorType) IsValid() bool {
	switch a {
	case PTOLedgerActorUser, PTOLedgerActorSystem:
		return true
	default:
		return false
	}
}

var (
	_ validationframework.TenantedEntity = (*WorkerPTOLedgerEntry)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerPTOLedgerEntry)(nil)
)

type WorkerPTOLedgerEntry struct {
	keel.Table `                                                 keel:"worker_pto_ledger"`

	ID               pulid.ID           `json:"id" keel:"id,pk,autoid:wpl_"`
	BusinessUnitID   pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID         pulid.ID           `json:"workerId" keel:"worker_id"`
	PTOType          PTOType            `json:"ptoType" keel:"pto_type"`
	EntryType        PTOLedgerEntryType `json:"entryType" keel:"entry_type"`
	AmountDays       decimal.Decimal    `json:"amountDays" keel:"amount_days"`
	BalanceAfterDays decimal.Decimal    `json:"balanceAfterDays" keel:"balance_after_days"`
	Sequence         int64              `json:"sequence" keel:"sequence"`
	EffectiveAt      int64              `json:"effectiveAt" keel:"effective_at"`
	PeriodKey        string             `json:"periodKey" keel:"period_key,nullzero"`
	SourcePTOID      pulid.ID           `json:"sourcePtoId" keel:"source_pto_id,nullzero"`
	AssignmentID     pulid.ID           `json:"assignmentId" keel:"assignment_id,nullzero"`
	PTOPolicyID      pulid.ID           `json:"ptoPolicyId" keel:"pto_policy_id,nullzero"`
	ActorType        PTOLedgerActorType `json:"actorType" keel:"actor_type"`
	CreatedByID      pulid.ID           `json:"createdById" keel:"created_by_id,nullzero"`
	Note             string             `json:"note" keel:"note,nullzero"`
	CreatedAt        int64              `json:"createdAt" keel:"created_at,autocreate"`

	Worker    *Worker    `json:"worker,omitempty" keel:"belongs:worker_id"`
	SourcePTO *WorkerPTO `json:"sourcePto,omitempty" keel:"belongs:source_pto_id|worker_id"`
}

func (e *WorkerPTOLedgerEntry) BeforeCreate(_ context.Context) error {
	if e.ActorType == "" {
		e.ActorType = PTOLedgerActorUser
	}

	return nil
}

func (e *WorkerPTOLedgerEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.PTOType,
			validation.Required.Error("PTO type is required"),
			domainvalidation.ValidEnum[PTOType](
				"ptoType must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			),
		),
		validation.Field(&e.EntryType,
			validation.Required.Error("Entry type is required"),
			domainvalidation.ValidEnum[PTOLedgerEntryType](
				"entryType must be one of: OpeningBalance, Accrual, Usage, Reversal, Adjustment, Carryover, Expiry",
			),
		),
		validation.Field(
			&e.ActorType,
			validation.Required.Error("Actor type is required"),
			domainvalidation.ValidEnum[PTOLedgerActorType](
				"actorType must be one of: User, System",
			),
		),
		validation.Field(&e.EffectiveAt,
			validation.Required.Error("Effective date is required"),
			validation.Min(int64(1)).Error("Effective date must be a positive value"),
		),
		validation.Field(&e.Note,
			validation.Length(0, 255).Error("Note must be 255 characters or fewer"),
		),
	))

	if e.AmountDays.IsZero() {
		multiErr.Add("amountDays", errortypes.ErrInvalid, "Amount cannot be zero")
	}
	switch e.EntryType.RequiredSign() {
	case 1:
		if !e.AmountDays.IsPositive() {
			multiErr.Add("amountDays", errortypes.ErrInvalid, "This entry type must add days")
		}
	case -1:
		if !e.AmountDays.IsNegative() {
			multiErr.Add("amountDays", errortypes.ErrInvalid, "This entry type must remove days")
		}
	}
	if e.EntryType == PTOLedgerEntryAdjustment && e.Note == "" {
		multiErr.Add("note", errortypes.ErrRequired, "A note is required for manual adjustments")
	}
}

func (e *WorkerPTOLedgerEntry) GetID() pulid.ID { return e.ID }

func (e *WorkerPTOLedgerEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *WorkerPTOLedgerEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *WorkerPTOLedgerEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *WorkerPTOLedgerEntry) GetTableName() string { return "worker_pto_ledger" }

func (e *WorkerPTOLedgerEntry) GetResourceType() string { return "worker_pto_ledger" }

func (e *WorkerPTOLedgerEntry) GetResourceID() string { return e.ID.String() }

func (e *WorkerPTOLedgerEntry) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wpl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "note", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "period_key",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}
