package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*LedgerEntry)(nil)
	_ validationframework.TenantedEntity = (*LedgerEntry)(nil)
)

type LedgerEntry struct {
	bun.BaseModel `bun:"table:carrier_ledger_entries,alias:carldg" json:"-"`

	ID                  pulid.ID        `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID        `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	CarrierID           pulid.ID        `json:"carrierId"           bun:"carrier_id,type:VARCHAR(100),notnull"`
	EntryType           LedgerEntryType `json:"entryType"           bun:"entry_type,type:VARCHAR(50),notnull"`
	SourceObjectType    string          `json:"sourceObjectType"    bun:"source_object_type,type:VARCHAR(50),notnull"`
	SourceObjectID      string          `json:"sourceObjectId"      bun:"source_object_id,type:VARCHAR(100),notnull"`
	SourceEventType     string          `json:"sourceEventType"     bun:"source_event_type,type:VARCHAR(100),notnull"`
	RelatedSettlementID *pulid.ID       `json:"relatedSettlementId" bun:"related_settlement_id,type:VARCHAR(100),nullzero"`
	JournalBatchID      *pulid.ID       `json:"journalBatchId"      bun:"journal_batch_id,type:VARCHAR(100),nullzero"`
	DocumentNumber      string          `json:"documentNumber"      bun:"document_number,type:VARCHAR(100),nullzero"`
	TransactionDate     int64           `json:"transactionDate"     bun:"transaction_date,type:BIGINT,notnull"`
	LineNumber          int             `json:"lineNumber"          bun:"line_number,type:INTEGER,notnull"`
	AmountMinor         int64           `json:"amountMinor"         bun:"amount_minor,type:BIGINT,notnull"`
	CreatedByID         pulid.ID        `json:"createdById"         bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Carrier *carrier.Carrier `json:"carrier,omitempty" bun:"rel:belongs-to,join:carrier_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (e *LedgerEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(
			&e.SourceObjectType,
			validation.Required.Error("Source object type is required"),
		),
		validation.Field(
			&e.SourceObjectID,
			validation.Required.Error("Source object is required"),
		),
		validation.Field(
			&e.SourceEventType,
			validation.Required.Error("Source event type is required"),
		),
		validation.Field(
			&e.TransactionDate,
			validation.Required.Error("Transaction date is required"),
		),
	))

	if !e.EntryType.IsValid() {
		multiErr.Add("entryType", errortypes.ErrInvalid, "Ledger entry type is invalid")
	}
	if e.AmountMinor == 0 {
		multiErr.Add("amountMinor", errortypes.ErrInvalid, "Amount cannot be zero")
	}
}

func (e *LedgerEntry) GetID() pulid.ID { return e.ID }

func (e *LedgerEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *LedgerEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *LedgerEntry) GetTableName() string { return "carrier_ledger_entries" }

func (e *LedgerEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("carldg_")
		}
		e.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
