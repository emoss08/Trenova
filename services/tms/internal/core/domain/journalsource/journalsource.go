package journalsource

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Source)(nil)

// Source is the bridge between an operational event and the journal it
// produced: it answers which shipment, settlement or payment is behind a given
// entry. SourceEventType and Status are varchar rather than database enums and
// their value sets are extended by later migrations, so they stay strings.
type Source struct {
	bun.BaseModel `bun:"table:journal_sources,alias:js" json:"-"`

	ID                   pulid.ID `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SourceObjectType     string   `json:"sourceObjectType"     bun:"source_object_type,type:VARCHAR(50),notnull"`
	SourceObjectID       string   `json:"sourceObjectId"       bun:"source_object_id,type:VARCHAR(100),notnull"`
	SourceEventType      string   `json:"sourceEventType"      bun:"source_event_type,type:VARCHAR(100),notnull"`
	SourceDocumentNumber string   `json:"sourceDocumentNumber" bun:"source_document_number,type:VARCHAR(100),nullzero"`
	Status               string   `json:"status"               bun:"status,type:VARCHAR(50),notnull"`
	IdempotencyKey       string   `json:"idempotencyKey"       bun:"idempotency_key,type:VARCHAR(200),nullzero"`
	JournalBatchID       pulid.ID `json:"journalBatchId"       bun:"journal_batch_id,type:VARCHAR(100),notnull"`
	JournalEntryID       pulid.ID `json:"journalEntryId"       bun:"journal_entry_id,type:VARCHAR(100),notnull"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	JournalBatch *journalentry.JournalBatch `json:"journalBatch,omitempty" bun:"rel:belongs-to,join:journal_batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	JournalEntry *journalentry.JournalEntry `json:"journalEntry,omitempty" bun:"rel:belongs-to,join:journal_entry_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (s *Source) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("jsrc_")
		}
		s.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
