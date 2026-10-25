package inboundmessage

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table     `keel:"inbound_messages"`
	ID             pulid.ID `keel:"id,pk,autoid:imsg_"         json:"id"`
	BusinessUnitID pulid.ID `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID `keel:"organization_id,pk,tenant"  json:"organizationId"`
	MailboxID      pulid.ID `keel:"mailbox_id"                 json:"mailboxId"`

	// ProviderMessageID is what the provider called it. It is unique per
	// mailbox, which is what makes a redelivered webhook a no-op rather than a
	// second shipment.
	ProviderMessageID string `json:"providerMessageId" keel:"provider_message_id"`
	// MessageID, InReplyTo and References are the mail headers a reply has to
	// carry to land in the same thread rather than starting a new one.
	MessageID  string   `json:"messageId"  keel:"message_id,nullzero"`
	InReplyTo  string   `json:"inReplyTo"  keel:"in_reply_to,nullzero"`
	References []string `json:"references" keel:"references,jsonb,nullzero"`

	FromAddress string   `json:"fromAddress" keel:"from_address"`
	FromName    string   `json:"fromName"    keel:"from_name,nullzero"`
	ToAddresses []string `json:"toAddresses" keel:"to_addresses,jsonb,nullzero"`
	CcAddresses []string `json:"ccAddresses" keel:"cc_addresses,jsonb,nullzero"`
	Subject     string   `json:"subject"     keel:"subject,nullzero"`
	// TextBody is the bounded plain text. HTMLKey points at the whole message
	// in object storage.
	TextBody   string  `json:"textBody"   keel:"text_body,nullzero"`
	HTMLKey    string  `json:"htmlKey"    keel:"html_key,nullzero"`
	ReceivedAt int64   `json:"receivedAt" keel:"received_at"`
	SpamScore  float64 `json:"spamScore"  keel:"spam_score,nullzero"`

	Classification Classification `json:"classification" keel:"classification,nullzero"`
	Confidence     float64        `json:"confidence"     keel:"confidence,nullzero"`
	Status         Status         `json:"status"         keel:"status"`

	// The records the message turned out to be about, with why. The reason is
	// stored because a match nobody can check is a match nobody will trust: a
	// shipment matched on a pro number in the subject is a different claim
	// from one matched on the sender's domain.
	MatchedCustomerID pulid.ID `json:"matchedCustomerId" keel:"matched_customer_id,nullzero"`
	MatchedCarrierID  pulid.ID `json:"matchedCarrierId"  keel:"matched_carrier_id,nullzero"`
	MatchedShipmentID pulid.ID `json:"matchedShipmentId" keel:"matched_shipment_id,nullzero"`
	MatchReason       string   `json:"matchReason"       keel:"match_reason,nullzero"`

	RunID       pulid.ID `json:"runId"       keel:"run_id,nullzero"`
	ReviewedBy  pulid.ID `json:"reviewedBy"  keel:"reviewed_by,nullzero"`
	ReviewedAt  int64    `json:"reviewedAt"  keel:"reviewed_at,nullzero"`
	ReviewNote  string   `json:"reviewNote"  keel:"review_note,nullzero"`
	FailureCode string   `json:"failureCode" keel:"failure_code,nullzero"`
	FailureText string   `json:"failureText" keel:"failure_text,nullzero"`

	MessageKey              string      `json:"messageKey"              keel:"message_key,nullzero"`
	ThreadKey               string      `json:"threadKey"               keel:"thread_key"`
	SenderDomain            string      `json:"senderDomain"            keel:"sender_domain"`
	OwnerID                 pulid.ID    `json:"ownerId"                 keel:"owner_id,nullzero"`
	DueAt                   int64       `json:"dueAt"                   keel:"due_at,nullzero"`
	Parked                  Parking     `json:"parked"                  keel:"parked,nullzero"`
	SnoozedUntil            int64       `json:"snoozedUntil"            keel:"snoozed_until,nullzero"`
	SnoozeEvent             SnoozeEvent `json:"snoozeEvent"             keel:"snooze_event,nullzero"`
	SnoozeWatchID           pulid.ID    `json:"snoozeWatchId"           keel:"snooze_watch_id,nullzero"`
	SnoozeDocumentTypeID    pulid.ID    `json:"snoozeDocumentTypeId"    keel:"snooze_document_type_id,nullzero"`
	SnoozedAt               int64       `json:"snoozedAt"               keel:"snoozed_at,nullzero"`
	SnoozedBy               pulid.ID    `json:"snoozedBy"               keel:"snoozed_by,nullzero"`
	AwaitingSince           int64       `json:"awaitingSince"           keel:"awaiting_since,nullzero"`
	AwaitingUntil           int64       `json:"awaitingUntil"           keel:"awaiting_until,nullzero"`
	NovaRead                bool        `json:"novaRead"                keel:"nova_read"`
	HandledByRuleID         pulid.ID    `json:"handledByRuleId"         keel:"handled_by_rule_id,nullzero"`
	HandledAction           NovaAction  `json:"handledAction"           keel:"handled_action,nullzero"`
	ClassificationCorrected bool        `json:"classificationCorrected" keel:"classification_corrected"`
	Facts                   []Fact      `json:"facts"                   keel:"facts,jsonb,nullzero"`
	Conflicts               []Conflict  `json:"conflicts"               keel:"conflicts,jsonb,nullzero"`
	ActionState             ActionState `json:"actionState"             keel:"action_state,jsonb"`
	LoadDraft               *LoadDraft  `json:"loadDraft"               keel:"load_draft,jsonb"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Mailbox      *Mailbox             `json:"mailbox,omitempty"      keel:"belongs:mailbox_id"`
	Attachments  []*InboundAttachment `json:"attachments,omitempty"  keel:"many:message_id"`
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

func (m *InboundMessage) BeforeCreate(ctx context.Context) error {
	m.MessageID = stringutils.StripNUL(m.MessageID)
	m.InReplyTo = stringutils.StripNUL(m.InReplyTo)
	m.FromName = stringutils.StripNUL(m.FromName)
	m.Subject = stringutils.StripNUL(m.Subject)
	m.TextBody = stringutils.StripNUL(m.TextBody)
	m.FailureText = stringutils.StripNUL(m.FailureText)
	m.SenderDomain = emailutils.Domain(m.FromAddress)
	m.MessageKey = NormalizeMessageID(m.MessageID)
	if m.ThreadKey == "" {
		m.ThreadKey = DeriveThreadKey(m.MessageID, m.ID)
	}
	now := timeutils.NowUnix()
	if m.Status == "" {
		m.Status = StatusReceived
	}
	if m.ReceivedAt == 0 {
		m.ReceivedAt = now
	}

	return nil
}

func (m *InboundMessage) BeforeUpdate(ctx context.Context) error {
	m.MessageID = stringutils.StripNUL(m.MessageID)
	m.InReplyTo = stringutils.StripNUL(m.InReplyTo)
	m.FromName = stringutils.StripNUL(m.FromName)
	m.Subject = stringutils.StripNUL(m.Subject)
	m.TextBody = stringutils.StripNUL(m.TextBody)
	m.FailureText = stringutils.StripNUL(m.FailureText)
	m.SenderDomain = emailutils.Domain(m.FromAddress)
	m.MessageKey = NormalizeMessageID(m.MessageID)
	if m.ThreadKey == "" {
		m.ThreadKey = DeriveThreadKey(m.MessageID, m.ID)
	}

	return nil
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
	keel.Table `keel:"inbound_message_attachments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:imsga_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	MessageID      pulid.ID `json:"messageId"      keel:"message_id"`

	FileName    string         `json:"fileName"    keel:"file_name"`
	ContentType string         `json:"contentType" keel:"content_type,nullzero"`
	ByteSize    int64          `json:"byteSize"    keel:"byte_size,nullzero"`
	Kind        AttachmentKind `json:"kind"        keel:"kind"`
	// ProviderAttachmentID is the provider's own id for the file. Content
	// fetched after the webhook is matched to its row by it.
	ProviderAttachmentID string `json:"providerAttachmentId" keel:"provider_attachment_id,nullzero"`

	// UploadSessionID and DocumentID trace the file through the document
	// pipeline, so an attachment that failed extraction says where it stopped
	// rather than only that it is not attached to anything.
	UploadSessionID pulid.ID `json:"uploadSessionId" keel:"upload_session_id,nullzero"`
	DocumentID      pulid.ID `json:"documentId"      keel:"document_id,nullzero"`
	DraftID         pulid.ID `json:"draftId"         keel:"draft_id,nullzero"`
	FailureText     string   `json:"failureText"     keel:"failure_text,nullzero"`

	FiledShipmentID pulid.ID         `json:"filedShipmentId" keel:"filed_shipment_id,nullzero"`
	FiledAs         string           `json:"filedAs"         keel:"filed_as,nullzero"`
	FiledAt         int64            `json:"filedAt"         keel:"filed_at,nullzero"`
	FiledBy         pulid.ID         `json:"filedBy"         keel:"filed_by,nullzero"`
	PageCount       int              `json:"pageCount"       keel:"page_count"`
	Pages           []AttachmentPage `json:"pages"           keel:"pages,jsonb,nullzero"`
	SplitDocuments  []SplitDocument  `json:"splitDocuments"  keel:"split_documents,jsonb,nullzero"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
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

func (a *InboundAttachment) BeforeCreate(ctx context.Context) error {
	if a.Kind == "" {
		a.Kind = AttachmentUnknown
	}

	return nil
}

func (a *InboundAttachment) GetID() pulid.ID      { return a.ID }
func (a *InboundAttachment) GetCreatedAt() int64  { return a.CreatedAt }
func (a *InboundAttachment) GetTableName() string { return "inbound_message_attachments" }
