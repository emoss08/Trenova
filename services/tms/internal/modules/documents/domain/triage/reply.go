package triage

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

const (
	MaxReplySubjectLength = 500
	MaxReplyBodyLength    = 20000
)

type ReplyStatus string

const (
	ReplyDraft   = ReplyStatus("Draft")
	ReplySending = ReplyStatus("Sending")
	ReplySent    = ReplyStatus("Sent")
	ReplyFailed  = ReplyStatus("Failed")
)

func (s ReplyStatus) IsValid() bool {
	switch s {
	case ReplyDraft, ReplySending, ReplySent, ReplyFailed:
		return true
	default:
		return false
	}
}

type Reply struct {
	bun.BaseModel `bun:"table:inbound_replies,alias:irpl" json:"-"`
	keel.Table    `keel:"inbound_replies"`

	ID                pulid.ID          `json:"id"                bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:irpl_"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID          `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	MessageID         pulid.ID          `json:"messageId"         bun:"message_id,type:VARCHAR(100),notnull" keel:"message_id"`
	MailboxID         pulid.ID          `json:"mailboxId"         bun:"mailbox_id,type:VARCHAR(100),notnull" keel:"mailbox_id"`
	ThreadKey         string            `json:"threadKey"         bun:"thread_key,type:VARCHAR(500),notnull" keel:"thread_key"`
	Status            ReplyStatus       `json:"status"            bun:"status,type:VARCHAR(20),notnull" keel:"status"`
	FromAddress       string            `json:"fromAddress"       bun:"from_address,type:VARCHAR(255),notnull" keel:"from_address"`
	ToAddress         string            `json:"toAddress"         bun:"to_address,type:VARCHAR(255),notnull" keel:"to_address"`
	CcAddresses       []string          `json:"ccAddresses"       bun:"cc_addresses,type:JSONB,nullzero" keel:"cc_addresses,jsonb,nullzero"`
	Subject           string            `json:"subject"           bun:"subject,type:VARCHAR(500),notnull" keel:"subject"`
	Body              string            `json:"body"              bun:"body,type:TEXT,notnull" keel:"body"`
	DraftedByNova     bool              `json:"draftedByNova"     bun:"drafted_by_nova,type:BOOLEAN,notnull,default:false" keel:"drafted_by_nova"`
	TemplateID        pulid.ID          `json:"templateId"        bun:"template_id,type:VARCHAR(100),nullzero" keel:"template_id,nullzero"`
	MessageKey        string            `json:"messageKey"        bun:"message_key,type:VARCHAR(500),nullzero" keel:"message_key,nullzero"`
	ProviderMessageID string            `json:"providerMessageId" bun:"provider_message_id,type:VARCHAR(255),nullzero" keel:"provider_message_id,nullzero"`
	FailureText       string            `json:"failureText"       bun:"failure_text,type:TEXT,nullzero" keel:"failure_text,nullzero"`
	SuggestedAddress  string            `json:"suggestedAddress"  bun:"suggested_address,type:VARCHAR(255),nullzero" keel:"suggested_address,nullzero"`
	Attachments       []ReplyAttachment `json:"attachments"       bun:"attachments,type:JSONB,nullzero" keel:"attachments,jsonb,nullzero"`
	IdempotencyKey    string            `json:"-"                 bun:"idempotency_key,type:VARCHAR(255),nullzero" keel:"idempotency_key,nullzero"`
	CreatedByID       pulid.ID          `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	SentByID          pulid.ID          `json:"sentById"          bun:"sent_by_id,type:VARCHAR(100),nullzero" keel:"sent_by_id,nullzero"`
	SentAt            int64             `json:"sentAt"            bun:"sent_at,type:BIGINT,nullzero" keel:"sent_at,nullzero"`
	Version           int64             `json:"version"           bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt         int64             `json:"createdAt"         bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt"         bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (r *Reply) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.MessageID, validation.Required.Error("Message is required")),
		validation.Field(&r.ToAddress,
			validation.Required.Error("Say who the reply goes to"),
			is.EmailFormat.Error("The reply must go to an email address"),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.RuneLength(1, MaxReplySubjectLength),
		),
		validation.Field(&r.Body,
			validation.Required.Error("Write something before sending"),
			validation.RuneLength(1, MaxReplyBodyLength),
		),
	))
}

type AttachableSource string

const (
	AttachableFromMessage  = AttachableSource("Message")
	AttachableFromShipment = AttachableSource("Shipment")
	AttachableFromCustomer = AttachableSource("Customer")
	AttachableFromCarrier  = AttachableSource("Carrier")
)

func (s AttachableSource) IsValid() bool {
	switch s {
	case AttachableFromMessage, AttachableFromShipment, AttachableFromCustomer,
		AttachableFromCarrier:
		return true
	default:
		return false
	}
}

type ReplyAttachment struct {
	DocumentID  pulid.ID `json:"documentId"`
	FileName    string   `json:"fileName"`
	ContentType string   `json:"contentType"`
	ByteSize    int64    `json:"byteSize"`
}

func (r *Reply) AttachmentDocumentIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.Attachments))
	for i := range r.Attachments {
		ids = append(ids, r.Attachments[i].DocumentID)
	}

	return ids
}

func (r *Reply) AttachmentNames() []string {
	names := make([]string, 0, len(r.Attachments))
	for i := range r.Attachments {
		names = append(names, r.Attachments[i].FileName)
	}

	return names
}

func (r *Reply) Sent() bool {
	return r.Status == ReplySent
}

func (r *Reply) BeforeAppendModel(_ context.Context, query bun.Query) error {
	r.Subject = stringutils.StripNUL(r.Subject)
	r.Body = stringutils.StripNUL(r.Body)
	r.FailureText = stringutils.StripNUL(r.FailureText)
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("irpl_")
		}
		if r.Status == "" {
			r.Status = ReplyDraft
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *Reply) BeforeCreate(ctx context.Context) error {
	return r.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (r *Reply) BeforeUpdate(ctx context.Context) error {
	return r.BeforeAppendModel(ctx, (*bun.UpdateQuery)(nil))
}

func (r *Reply) GetID() pulid.ID      { return r.ID }
func (r *Reply) GetCreatedAt() int64  { return r.CreatedAt }
func (r *Reply) GetTableName() string { return "inbound_replies" }

func LatestReply(replies []*Reply) *Reply {
	var latest *Reply
	for _, reply := range replies {
		if latest == nil || reply.CreatedAt > latest.CreatedAt ||
			(reply.CreatedAt == latest.CreatedAt && reply.ID.String() > latest.ID.String()) {
			latest = reply
		}
	}

	return latest
}
