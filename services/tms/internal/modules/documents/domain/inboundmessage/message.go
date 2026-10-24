package inboundmessage

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxSubjectLength = 500
	// MaxMatchReasonLength is the match_reason column's width, checked before
	// the write so a long reason is a field error rather than a database error.
	MaxMatchReasonLength = 500
	// MaxBodyBytes is how much of a message body is kept in the row.
	//
	// A quoted thread can run to megabytes, and the classifier reads the top
	// of it anyway. The whole message goes to object storage; this is the part
	// that has to be searchable and has to be there when storage is not.
	MaxBodyBytes = 32 * 1024
)

// InboundMessage is one email that arrived.
//
// It is kept whether or not anything could be made of it. A tender nobody
// matched is still the thing a person has to go and look at, and a message
// that was dropped because it did not classify is a message the customer
// believes was received.
type InboundMessage struct {
	bun.BaseModel             `bun:"table:inbound_messages,alias:imsg" json:"-"`
	keel.Table                `keel:"inbound_messages"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:imsg_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	MailboxID      pulid.ID `json:"mailboxId"      bun:"mailbox_id,type:VARCHAR(100),notnull" keel:"mailbox_id"`

	// ProviderMessageID is what the provider called it. It is unique per
	// mailbox, which is what makes a redelivered webhook a no-op rather than a
	// second shipment.
	ProviderMessageID string `json:"providerMessageId" bun:"provider_message_id,type:VARCHAR(255),notnull" keel:"provider_message_id"`
	// MessageID, InReplyTo and References are the mail headers a reply has to
	// carry to land in the same thread rather than starting a new one.
	MessageID  string   `json:"messageId"  bun:"message_id,type:VARCHAR(500),nullzero" keel:"message_id,nullzero"`
	InReplyTo  string   `json:"inReplyTo"  bun:"in_reply_to,type:VARCHAR(500),nullzero" keel:"in_reply_to,nullzero"`
	References []string `json:"references" bun:"references,type:JSONB,nullzero" keel:"references,jsonb,nullzero"`

	FromAddress string   `json:"fromAddress" bun:"from_address,type:VARCHAR(255),notnull" keel:"from_address"`
	FromName    string   `json:"fromName"    bun:"from_name,type:VARCHAR(255),nullzero" keel:"from_name,nullzero"`
	ToAddresses []string `json:"toAddresses" bun:"to_addresses,type:JSONB,nullzero" keel:"to_addresses,jsonb,nullzero"`
	CcAddresses []string `json:"ccAddresses" bun:"cc_addresses,type:JSONB,nullzero" keel:"cc_addresses,jsonb,nullzero"`
	Subject     string   `json:"subject"     bun:"subject,type:VARCHAR(500),nullzero" keel:"subject,nullzero"`
	// TextBody is the bounded plain text. HTMLKey points at the whole message
	// in object storage.
	TextBody   string  `json:"textBody"   bun:"text_body,type:TEXT,nullzero" keel:"text_body,nullzero"`
	HTMLKey    string  `json:"htmlKey"    bun:"html_key,type:VARCHAR(512),nullzero" keel:"html_key,nullzero"`
	ReceivedAt int64   `json:"receivedAt" bun:"received_at,type:BIGINT,notnull" keel:"received_at"`
	SpamScore  float64 `json:"spamScore"  bun:"spam_score,type:NUMERIC(6,3),nullzero" keel:"spam_score,nullzero"`

	Classification Classification `json:"classification" bun:"classification,type:VARCHAR(30),nullzero" keel:"classification,nullzero"`
	Confidence     float64        `json:"confidence"     bun:"confidence,type:NUMERIC(4,3),nullzero" keel:"confidence,nullzero"`
	Status         Status         `json:"status"         bun:"status,type:VARCHAR(20),notnull" keel:"status"`

	// The records the message turned out to be about, with why. The reason is
	// stored because a match nobody can check is a match nobody will trust: a
	// shipment matched on a pro number in the subject is a different claim
	// from one matched on the sender's domain.
	MatchedCustomerID pulid.ID `json:"matchedCustomerId" bun:"matched_customer_id,type:VARCHAR(100),nullzero" keel:"matched_customer_id,nullzero"`
	MatchedCarrierID  pulid.ID `json:"matchedCarrierId"  bun:"matched_carrier_id,type:VARCHAR(100),nullzero" keel:"matched_carrier_id,nullzero"`
	MatchedShipmentID pulid.ID `json:"matchedShipmentId" bun:"matched_shipment_id,type:VARCHAR(100),nullzero" keel:"matched_shipment_id,nullzero"`
	MatchReason       string   `json:"matchReason"       bun:"match_reason,type:VARCHAR(500),nullzero" keel:"match_reason,nullzero"`

	RunID       pulid.ID `json:"runId"       bun:"run_id,type:VARCHAR(100),nullzero" keel:"run_id,nullzero"`
	ReviewedBy  pulid.ID `json:"reviewedBy"  bun:"reviewed_by,type:VARCHAR(100),nullzero" keel:"reviewed_by,nullzero"`
	ReviewedAt  int64    `json:"reviewedAt"  bun:"reviewed_at,type:BIGINT,nullzero" keel:"reviewed_at,nullzero"`
	ReviewNote  string   `json:"reviewNote"  bun:"review_note,type:TEXT,nullzero" keel:"review_note,nullzero"`
	FailureCode string   `json:"failureCode" bun:"failure_code,type:VARCHAR(100),nullzero" keel:"failure_code,nullzero"`
	FailureText string   `json:"failureText" bun:"failure_text,type:TEXT,nullzero" keel:"failure_text,nullzero"`

	MessageKey              string      `json:"messageKey"              bun:"message_key,type:VARCHAR(500),nullzero" keel:"message_key,nullzero"`
	ThreadKey               string      `json:"threadKey"               bun:"thread_key,type:VARCHAR(500),notnull" keel:"thread_key"`
	SenderDomain            string      `json:"senderDomain"            bun:"sender_domain,type:VARCHAR(255),notnull" keel:"sender_domain"`
	OwnerID                 pulid.ID    `json:"ownerId"                 bun:"owner_id,type:VARCHAR(100),nullzero" keel:"owner_id,nullzero"`
	DueAt                   int64       `json:"dueAt"                   bun:"due_at,type:BIGINT,nullzero" keel:"due_at,nullzero"`
	Parked                  Parking     `json:"parked"                  bun:"parked,type:VARCHAR(20),nullzero" keel:"parked,nullzero"`
	SnoozedUntil            int64       `json:"snoozedUntil"            bun:"snoozed_until,type:BIGINT,nullzero" keel:"snoozed_until,nullzero"`
	SnoozeEvent             SnoozeEvent `json:"snoozeEvent"             bun:"snooze_event,type:VARCHAR(30),nullzero" keel:"snooze_event,nullzero"`
	SnoozeWatchID           pulid.ID    `json:"snoozeWatchId"           bun:"snooze_watch_id,type:VARCHAR(100),nullzero" keel:"snooze_watch_id,nullzero"`
	SnoozeDocumentTypeID    pulid.ID    `json:"snoozeDocumentTypeId"    bun:"snooze_document_type_id,type:VARCHAR(100),nullzero" keel:"snooze_document_type_id,nullzero"`
	SnoozedAt               int64       `json:"snoozedAt"               bun:"snoozed_at,type:BIGINT,nullzero" keel:"snoozed_at,nullzero"`
	SnoozedBy               pulid.ID    `json:"snoozedBy"               bun:"snoozed_by,type:VARCHAR(100),nullzero" keel:"snoozed_by,nullzero"`
	AwaitingSince           int64       `json:"awaitingSince"           bun:"awaiting_since,type:BIGINT,nullzero" keel:"awaiting_since,nullzero"`
	AwaitingUntil           int64       `json:"awaitingUntil"           bun:"awaiting_until,type:BIGINT,nullzero" keel:"awaiting_until,nullzero"`
	NovaRead                bool        `json:"novaRead"                bun:"nova_read,type:BOOLEAN,notnull,default:false" keel:"nova_read"`
	HandledByRuleID         pulid.ID    `json:"handledByRuleId"         bun:"handled_by_rule_id,type:VARCHAR(100),nullzero" keel:"handled_by_rule_id,nullzero"`
	HandledAction           NovaAction  `json:"handledAction"           bun:"handled_action,type:VARCHAR(30),nullzero" keel:"handled_action,nullzero"`
	ClassificationCorrected bool        `json:"classificationCorrected" bun:"classification_corrected,type:BOOLEAN,notnull,default:false" keel:"classification_corrected"`
	Facts                   []Fact      `json:"facts"                   bun:"facts,type:JSONB,nullzero" keel:"facts,jsonb,nullzero"`
	Conflicts               []Conflict  `json:"conflicts"               bun:"conflicts,type:JSONB,nullzero" keel:"conflicts,jsonb,nullzero"`
	ActionState             ActionState `json:"actionState"             bun:"action_state,type:JSONB,notnull,default:'{}'" keel:"action_state,jsonb"`
	LoadDraft               *LoadDraft  `json:"loadDraft"               bun:"load_draft,type:JSONB,nullzero" keel:"load_draft,jsonb"`

	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Mailbox      *Mailbox             `json:"mailbox,omitempty"      bun:"rel:belongs-to,join:mailbox_id=id" keel:"belongs:mailbox_id"`
	Attachments  []*InboundAttachment `json:"attachments,omitempty"  bun:"rel:has-many,join:id=message_id" keel:"many:message_id"`
}

func (m *InboundMessage) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.MailboxID, validation.Required.Error("Mailbox is required")),
		validation.Field(&m.ProviderMessageID,
			validation.Required.Error("Provider message id is required"),
		),
		validation.Field(&m.FromAddress, validation.Required.Error("Sender is required")),
		validation.Field(&m.Subject, validation.Length(0, maxSubjectLength)),
		validation.Field(&m.MatchReason, validation.Length(0, MaxMatchReasonLength).Error(
			fmt.Sprintf("A match reason may be at most %d characters", MaxMatchReasonLength),
		)),
		validation.Field(&m.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is not a known one"),
		),
	))

	if m.Classification != "" && !m.Classification.IsValid() {
		multiErr.Add("classification", errortypes.ErrInvalid,
			"Classification is not a known one")
	}
	if m.Confidence < 0 || m.Confidence > 1 {
		multiErr.Add("confidence", errortypes.ErrInvalid, "Confidence must be between 0 and 1")
	}
}

// NeedsReview is whether the message is waiting on a person. It is the lane
// the inbox splits on, and the one number anybody wants from this table.
func (m *InboundMessage) NeedsReview() bool {
	return m.Status == StatusInReview || m.Status == StatusQuarantined
}

func (m *InboundMessage) IsWaiting() bool {
	return m.NeedsReview() && m.Parked == ""
}

func (m *InboundMessage) IsSnoozed() bool {
	return m.NeedsReview() && m.Parked == ParkingSnoozed
}

func (m *InboundMessage) IsAwaitingReply() bool {
	return m.NeedsReview() && m.Parked == ParkingAwaitingReply
}

func (m *InboundMessage) Snooze(snooze Snooze, now int64) {
	m.Parked = ParkingSnoozed
	m.SnoozedUntil = snooze.Until
	m.SnoozeEvent = snooze.Event
	m.SnoozeWatchID = snooze.WatchID
	m.SnoozeDocumentTypeID = snooze.DocumentTypeID
	m.SnoozedAt = now
	m.SnoozedBy = snooze.By
	m.AwaitingSince = 0
	m.AwaitingUntil = 0
}

func (m *InboundMessage) AwaitReply(until, now int64) {
	m.Unpark()
	m.Parked = ParkingAwaitingReply
	m.AwaitingSince = now
	m.AwaitingUntil = until
}

func (m *InboundMessage) Unpark() {
	m.Parked = ""
	m.SnoozedUntil = 0
	m.SnoozeEvent = ""
	m.SnoozeWatchID = pulid.Nil
	m.SnoozeDocumentTypeID = pulid.Nil
	m.SnoozedAt = 0
	m.SnoozedBy = pulid.Nil
	m.AwaitingSince = 0
	m.AwaitingUntil = 0
}

func (m *InboundMessage) ApplyResponseTarget(minutes int) {
	if minutes <= 0 {
		m.DueAt = 0

		return
	}
	m.DueAt = m.ReceivedAt + int64(minutes)*60
}

func (m *InboundMessage) HasRecord() bool {
	return m.MatchedShipmentID.IsNotNil() || m.MatchedCustomerID.IsNotNil() ||
		m.MatchedCarrierID.IsNotNil()
}

func (m *InboundMessage) BeforeAppendModel(_ context.Context, query bun.Query) error {
	m.MessageID = stringutils.StripNUL(m.MessageID)
	m.InReplyTo = stringutils.StripNUL(m.InReplyTo)
	m.FromName = stringutils.StripNUL(m.FromName)
	m.Subject = stringutils.StripNUL(m.Subject)
	m.TextBody = stringutils.StripNUL(m.TextBody)
	m.FailureText = stringutils.StripNUL(m.FailureText)
	if _, ok := query.(*bun.InsertQuery); ok && m.ID.IsNil() {
		m.ID = pulid.MustNew("imsg_")
	}
	m.SenderDomain = emailutils.Domain(m.FromAddress)
	m.MessageKey = NormalizeMessageID(m.MessageID)
	if m.ThreadKey == "" {
		m.ThreadKey = DeriveThreadKey(m.MessageID, m.ID)
	}

	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.Status == "" {
			m.Status = StatusReceived
		}
		if m.ReceivedAt == 0 {
			m.ReceivedAt = now
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

func (m *InboundMessage) BeforeCreate(ctx context.Context) error {
	return m.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (m *InboundMessage) BeforeUpdate(ctx context.Context) error {
	return m.BeforeAppendModel(ctx, (*bun.UpdateQuery)(nil))
}

// GetPostgresSearchConfig covers what a person actually remembers about a
// message they are hunting for: who sent it and what the subject said. The body
// is left out on purpose — it is bounded to 32 KB of quoted thread, and
// indexing that would make every message match every search.
func (m *InboundMessage) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "imsg",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "subject", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "from_address",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "from_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "classification",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightC,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (m *InboundMessage) GetID() pulid.ID      { return m.ID }
func (m *InboundMessage) GetCreatedAt() int64  { return m.CreatedAt }
func (m *InboundMessage) GetTableName() string { return "inbound_messages" }

// InboundAttachment is one file that came with a message, and what became of it.
type InboundAttachment struct {
	bun.BaseModel `bun:"table:inbound_message_attachments,alias:imsga" json:"-"`
	keel.Table    `keel:"inbound_message_attachments"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:imsga_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	MessageID      pulid.ID `json:"messageId"      bun:"message_id,type:VARCHAR(100),notnull" keel:"message_id"`

	FileName    string         `json:"fileName"    bun:"file_name,type:VARCHAR(255),notnull" keel:"file_name"`
	ContentType string         `json:"contentType" bun:"content_type,type:VARCHAR(100),nullzero" keel:"content_type,nullzero"`
	ByteSize    int64          `json:"byteSize"    bun:"byte_size,type:BIGINT,nullzero" keel:"byte_size,nullzero"`
	Kind        AttachmentKind `json:"kind"        bun:"kind,type:VARCHAR(30),notnull" keel:"kind"`
	// ProviderAttachmentID is the provider's own id for the file. Content
	// fetched after the webhook is matched to its row by it.
	ProviderAttachmentID string `json:"providerAttachmentId" bun:"provider_attachment_id,type:VARCHAR(255),nullzero" keel:"provider_attachment_id,nullzero"`

	// UploadSessionID and DocumentID trace the file through the document
	// pipeline, so an attachment that failed extraction says where it stopped
	// rather than only that it is not attached to anything.
	UploadSessionID pulid.ID `json:"uploadSessionId" bun:"upload_session_id,type:VARCHAR(100),nullzero" keel:"upload_session_id,nullzero"`
	DocumentID      pulid.ID `json:"documentId"      bun:"document_id,type:VARCHAR(100),nullzero" keel:"document_id,nullzero"`
	DraftID         pulid.ID `json:"draftId"         bun:"draft_id,type:VARCHAR(100),nullzero" keel:"draft_id,nullzero"`
	FailureText     string   `json:"failureText"     bun:"failure_text,type:TEXT,nullzero" keel:"failure_text,nullzero"`

	FiledShipmentID pulid.ID         `json:"filedShipmentId" bun:"filed_shipment_id,type:VARCHAR(100),nullzero" keel:"filed_shipment_id,nullzero"`
	FiledAs         string           `json:"filedAs"         bun:"filed_as,type:VARCHAR(100),nullzero" keel:"filed_as,nullzero"`
	FiledAt         int64            `json:"filedAt"         bun:"filed_at,type:BIGINT,nullzero" keel:"filed_at,nullzero"`
	FiledBy         pulid.ID         `json:"filedBy"         bun:"filed_by,type:VARCHAR(100),nullzero" keel:"filed_by,nullzero"`
	PageCount       int              `json:"pageCount"       bun:"page_count,type:INTEGER,notnull,default:0" keel:"page_count"`
	Pages           []AttachmentPage `json:"pages"           bun:"pages,type:JSONB,nullzero" keel:"pages,jsonb,nullzero"`
	SplitDocuments  []SplitDocument  `json:"splitDocuments"  bun:"split_documents,type:JSONB,nullzero" keel:"split_documents,jsonb,nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (a *InboundAttachment) Readable() bool {
	return a.FailureText == "" && a.DocumentID.IsNotNil()
}

func (a *InboundAttachment) Filed() bool {
	return a.FiledShipmentID.IsNotNil() || len(a.SplitDocuments) > 0
}

func (a *InboundAttachment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.MessageID, validation.Required.Error("Message is required")),
		validation.Field(&a.FileName, validation.Required.Error("File name is required")),
	))

	if a.Kind != "" && !a.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind is not a known one")
	}
}

func (a *InboundAttachment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("imsga_")
		}
		if a.Kind == "" {
			a.Kind = AttachmentUnknown
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *InboundAttachment) BeforeCreate(ctx context.Context) error {
	return a.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (a *InboundAttachment) GetID() pulid.ID      { return a.ID }
func (a *InboundAttachment) GetCreatedAt() int64  { return a.CreatedAt }
func (a *InboundAttachment) GetTableName() string { return "inbound_message_attachments" }
