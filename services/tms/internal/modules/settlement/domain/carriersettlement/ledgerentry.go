package carriersettlement

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*LedgerEntry)(nil)
)

type LedgerEntry struct {
	keel.Table `keel:"carrier_ledger_entries"`

	ID                  pulid.ID        `json:"id"                  keel:"id,pk,autoid:carldg_"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID        `json:"organizationId"      keel:"organization_id,pk,tenant"`
	CarrierID           pulid.ID        `json:"carrierId"           keel:"carrier_id"`
	EntryType           LedgerEntryType `json:"entryType"           keel:"entry_type"`
	SourceObjectType    string          `json:"sourceObjectType"    keel:"source_object_type"`
	SourceObjectID      string          `json:"sourceObjectId"      keel:"source_object_id"`
	SourceEventType     string          `json:"sourceEventType"     keel:"source_event_type"`
	RelatedSettlementID *pulid.ID       `json:"relatedSettlementId" keel:"related_settlement_id"`
	JournalBatchID      *pulid.ID       `json:"journalBatchId"      keel:"journal_batch_id"`
	DocumentNumber      string          `json:"documentNumber"      keel:"document_number,nullzero"`
	TransactionDate     int64           `json:"transactionDate"     keel:"transaction_date"`
	LineNumber          int             `json:"lineNumber"          keel:"line_number"`
	AmountMinor         int64           `json:"amountMinor"         keel:"amount_minor"`
	CreatedByID         pulid.ID        `json:"createdById"         keel:"created_by_id"`
	CreatedAt           int64           `json:"createdAt"           keel:"created_at,autocreate"`

	Carrier *carrier.Carrier `json:"carrier,omitempty" keel:"belongs:carrier_id"`
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
