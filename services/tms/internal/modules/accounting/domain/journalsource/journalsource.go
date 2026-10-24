package journalsource

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// What operational event produced each journal entry
type Source struct {
	keel.Table `                                              keel:"journal_sources,report=Accounting,label=Journal Source"`

	ID                   pulid.ID `json:"id" keel:"id,pk,autoid:jsrc_"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	SourceObjectType     string   `json:"sourceObjectType" keel:"source_object_type,label=Source Object"`
	SourceObjectID       string   `json:"sourceObjectId" keel:"source_object_id"`
	SourceEventType      string   `json:"sourceEventType" keel:"source_event_type,label=Source Event"`
	SourceDocumentNumber string   `json:"sourceDocumentNumber" keel:"source_document_number,nullzero,label=Source Document"`
	Status               string   `json:"status" keel:"status"`
	IdempotencyKey       string   `json:"idempotencyKey" keel:"idempotency_key,nullzero,noreport"`
	JournalBatchID       pulid.ID `json:"journalBatchId" keel:"journal_batch_id"`
	JournalEntryID       pulid.ID `json:"journalEntryId" keel:"journal_entry_id"`
	CreatedAt            int64    `json:"createdAt" keel:"created_at,autocreate"`

	JournalBatch *journalentry.JournalBatch `json:"journalBatch,omitempty" keel:"belongs:journal_batch_id"`
	JournalEntry *journalentry.JournalEntry `json:"journalEntry,omitempty" keel:"belongs:journal_entry_id"`
}

func (s *Source) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID, validation.Required),
		validation.Field(&s.BusinessUnitID, validation.Required),
		validation.Field(&s.SourceObjectType, validation.Required, validation.Length(1, 50)),
		validation.Field(&s.SourceObjectID, validation.Required, validation.Length(1, 100)),
		validation.Field(&s.SourceEventType, validation.Required, validation.Length(1, 100)),
		validation.Field(&s.Status, validation.Required, validation.Length(1, 50)),
		validation.Field(&s.JournalBatchID, validation.Required),
		validation.Field(&s.JournalEntryID, validation.Required),
	))
}

func (s *Source) GetTableName() string { return "journal_sources" }

func (s *Source) GetID() pulid.ID { return s.ID }

func (s *Source) GetCreatedAt() int64 { return s.CreatedAt }

func (s *Source) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *Source) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }
