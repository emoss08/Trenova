package inboundmessage

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxSubjectLength = 500
	// MaxBodyBytes is how much of a message body is kept in the row.
	//
	// A quoted thread can run to megabytes, and the classifier reads the top
	// of it anyway. The whole message goes to object storage; this is the part
	// that has to be searchable and has to be there when storage is not.
	MaxBodyBytes = 32 * 1024
)

// Message is one email that arrived.
//
// It is kept whether or not anything could be made of it. A tender nobody
// matched is still the thing a person has to go and look at, and a message
// that was dropped because it did not classify is a message the customer
// believes was received.
type Message struct {
	bun.BaseModel             `bun:"table:inbound_messages,alias:imsg" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	MailboxID      pulid.ID `json:"mailboxId"      bun:"mailbox_id,type:VARCHAR(100),notnull"`

	// ProviderMessageID is what the provider called it. It is unique per
	// mailbox, which is what makes a redelivered webhook a no-op rather than a
	// second shipment.
	ProviderMessageID string `json:"providerMessageId" bun:"provider_message_id,type:VARCHAR(255),notnull"`
	// MessageID, InReplyTo and References are the mail headers a reply has to
	// carry to land in the same thread rather than starting a new one.
	MessageID  string   `json:"messageId"  bun:"message_id,type:VARCHAR(500),nullzero"`
	InReplyTo  string   `json:"inReplyTo"  bun:"in_reply_to,type:VARCHAR(500),nullzero"`
	References []string `json:"references" bun:"references,type:JSONB,nullzero"`

	FromAddress string   `json:"fromAddress" bun:"from_address,type:VARCHAR(255),notnull"`
	FromName    string   `json:"fromName"    bun:"from_name,type:VARCHAR(255),nullzero"`
	ToAddresses []string `json:"toAddresses" bun:"to_addresses,type:JSONB,nullzero"`
	CcAddresses []string `json:"ccAddresses" bun:"cc_addresses,type:JSONB,nullzero"`
	Subject     string   `json:"subject"     bun:"subject,type:VARCHAR(500),nullzero"`
	// TextBody is the bounded plain text. HTMLKey points at the whole message
	// in object storage.
	TextBody   string  `json:"textBody"   bun:"text_body,type:TEXT,nullzero"`
	HTMLKey    string  `json:"htmlKey"    bun:"html_key,type:VARCHAR(512),nullzero"`
	ReceivedAt int64   `json:"receivedAt" bun:"received_at,type:BIGINT,notnull"`
	SpamScore  float64 `json:"spamScore"  bun:"spam_score,type:NUMERIC(6,3),nullzero"`

	Classification Classification `json:"classification" bun:"classification,type:VARCHAR(30),nullzero"`
	Confidence     float64        `json:"confidence"     bun:"confidence,type:NUMERIC(4,3),nullzero"`
	Status         Status         `json:"status"         bun:"status,type:VARCHAR(20),notnull"`

	// The records the message turned out to be about, with why. The reason is
	// stored because a match nobody can check is a match nobody will trust: a
	// shipment matched on a pro number in the subject is a different claim
	// from one matched on the sender's domain.
	MatchedCustomerID pulid.ID `json:"matchedCustomerId" bun:"matched_customer_id,type:VARCHAR(100),nullzero"`
	MatchedCarrierID  pulid.ID `json:"matchedCarrierId"  bun:"matched_carrier_id,type:VARCHAR(100),nullzero"`
	MatchedShipmentID pulid.ID `json:"matchedShipmentId" bun:"matched_shipment_id,type:VARCHAR(100),nullzero"`
	MatchReason       string   `json:"matchReason"       bun:"match_reason,type:VARCHAR(500),nullzero"`

	RunID       pulid.ID `json:"runId"       bun:"run_id,type:VARCHAR(100),nullzero"`
	ReviewedBy  pulid.ID `json:"reviewedBy"  bun:"reviewed_by,type:VARCHAR(100),nullzero"`
	ReviewedAt  int64    `json:"reviewedAt"  bun:"reviewed_at,type:BIGINT,nullzero"`
	ReviewNote  string   `json:"reviewNote"  bun:"review_note,type:TEXT,nullzero"`
	FailureCode string   `json:"failureCode" bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureText string   `json:"failureText" bun:"failure_text,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Mailbox      *Mailbox             `json:"mailbox,omitempty"      bun:"rel:belongs-to,join:mailbox_id=id"`
	Attachments  []*Attachment        `json:"attachments,omitempty"  bun:"rel:has-many,join:id=message_id"`
}

func (m *Message) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.MailboxID, validation.Required.Error("Mailbox is required")),
		validation.Field(&m.ProviderMessageID,
			validation.Required.Error("Provider message id is required"),
		),
		validation.Field(&m.FromAddress, validation.Required.Error("Sender is required")),
		validation.Field(&m.Subject, validation.Length(0, maxSubjectLength)),
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
func (m *Message) NeedsReview() bool {
	return m.Status == StatusInReview || m.Status == StatusQuarantined
}

func (m *Message) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("imsg_")
		}
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

func (m *Message) GetID() pulid.ID      { return m.ID }
func (m *Message) GetCreatedAt() int64  { return m.CreatedAt }
func (m *Message) GetTableName() string { return "inbound_messages" }

// Attachment is one file that came with a message, and what became of it.
type Attachment struct {
	bun.BaseModel `bun:"table:inbound_message_attachments,alias:imsga" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	MessageID      pulid.ID `json:"messageId"      bun:"message_id,type:VARCHAR(100),notnull"`

	FileName    string         `json:"fileName"    bun:"file_name,type:VARCHAR(255),notnull"`
	ContentType string         `json:"contentType" bun:"content_type,type:VARCHAR(100),nullzero"`
	ByteSize    int64          `json:"byteSize"    bun:"byte_size,type:BIGINT,nullzero"`
	Kind        AttachmentKind `json:"kind"        bun:"kind,type:VARCHAR(30),notnull"`

	// UploadSessionID and DocumentID trace the file through the document
	// pipeline, so an attachment that failed extraction says where it stopped
	// rather than only that it is not attached to anything.
	UploadSessionID pulid.ID `json:"uploadSessionId" bun:"upload_session_id,type:VARCHAR(100),nullzero"`
	DocumentID      pulid.ID `json:"documentId"      bun:"document_id,type:VARCHAR(100),nullzero"`
	DraftID         pulid.ID `json:"draftId"         bun:"draft_id,type:VARCHAR(100),nullzero"`
	FailureText     string   `json:"failureText"     bun:"failure_text,type:TEXT,nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (a *Attachment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.MessageID, validation.Required.Error("Message is required")),
		validation.Field(&a.FileName, validation.Required.Error("File name is required")),
	))

	if a.Kind != "" && !a.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Kind is not a known one")
	}
}

func (a *Attachment) BeforeAppendModel(_ context.Context, query bun.Query) error {
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

func (a *Attachment) GetID() pulid.ID      { return a.ID }
func (a *Attachment) GetCreatedAt() int64  { return a.CreatedAt }
func (a *Attachment) GetTableName() string { return "inbound_message_attachments" }
