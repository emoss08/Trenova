package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*WorkerPTOLedgerEntry)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTOLedgerEntry)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerPTOLedgerEntry)(nil)
)

type WorkerPTOLedgerEntry struct {
	bun.BaseModel `bun:"table:worker_pto_ledger,alias:wpl" json:"-"`

	ID               pulid.ID           `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID           `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID           `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID         pulid.ID           `json:"workerId"         bun:"worker_id,type:VARCHAR(100),notnull"`
	PTOType          PTOType            `json:"ptoType"          bun:"pto_type,type:worker_pto_type_enum,notnull"`
	EntryType        PTOLedgerEntryType `json:"entryType"        bun:"entry_type,type:pto_ledger_entry_type_enum,notnull"`
	AmountDays       decimal.Decimal    `json:"amountDays"       bun:"amount_days,type:NUMERIC(8,2),notnull"`
	BalanceAfterDays decimal.Decimal    `json:"balanceAfterDays" bun:"balance_after_days,type:NUMERIC(8,2),notnull"`
	Sequence         int64              `json:"sequence"         bun:"sequence,type:BIGINT,notnull"`
	EffectiveAt      int64              `json:"effectiveAt"      bun:"effective_at,type:BIGINT,notnull"`
	PeriodKey        string             `json:"periodKey"        bun:"period_key,type:VARCHAR(64),nullzero"`
	SourcePTOID      pulid.ID           `json:"sourcePtoId"      bun:"source_pto_id,type:VARCHAR(100),nullzero"`
	AssignmentID     pulid.ID           `json:"assignmentId"     bun:"assignment_id,type:VARCHAR(100),nullzero"`
	PTOPolicyID      pulid.ID           `json:"ptoPolicyId"      bun:"pto_policy_id,type:VARCHAR(100),nullzero"`
	ActorType        PTOLedgerActorType `json:"actorType"        bun:"actor_type,type:pto_ledger_actor_type_enum,notnull,default:'User'"`
	CreatedByID      pulid.ID           `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),nullzero"`
	Note             string             `json:"note"             bun:"note,type:VARCHAR(255),nullzero"`
	CreatedAt        int64              `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker    *Worker    `json:"worker,omitempty"    bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SourcePTO *WorkerPTO `json:"sourcePto,omitempty" bun:"rel:belongs-to,join:source_pto_id=id,join:worker_id=worker_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (e *WorkerPTOLedgerEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wpl_")
		}
		e.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
