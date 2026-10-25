package triage

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
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
	keel.Table `keel:"inbound_replies"`

	ID                pulid.ID          `json:"id"                keel:"id,pk,autoid:irpl_"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID          `json:"organizationId"    keel:"organization_id,pk,tenant"`
	MessageID         pulid.ID          `json:"messageId"         keel:"message_id"`
	MailboxID         pulid.ID          `json:"mailboxId"         keel:"mailbox_id"`
	ThreadKey         string            `json:"threadKey"         keel:"thread_key"`
	Status            ReplyStatus       `json:"status"            keel:"status"`
	FromAddress       string            `json:"fromAddress"       keel:"from_address"`
	ToAddress         string            `json:"toAddress"         keel:"to_address"`
	CcAddresses       []string          `json:"ccAddresses"       keel:"cc_addresses,jsonb,nullzero"`
	Subject           string            `json:"subject"           keel:"subject"`
	Body              string            `json:"body"              keel:"body"`
	DraftedByNova     bool              `json:"draftedByNova"     keel:"drafted_by_nova"`
	TemplateID        pulid.ID          `json:"templateId"        keel:"template_id,nullzero"`
	MessageKey        string            `json:"messageKey"        keel:"message_key,nullzero"`
	ProviderMessageID string            `json:"providerMessageId" keel:"provider_message_id,nullzero"`
	FailureText       string            `json:"failureText"       keel:"failure_text,nullzero"`
	SuggestedAddress  string            `json:"suggestedAddress"  keel:"suggested_address,nullzero"`
	Attachments       []ReplyAttachment `json:"attachments"       keel:"attachments,jsonb,nullzero"`
	IdempotencyKey    string            `json:"-"                 keel:"idempotency_key,nullzero"`
	CreatedByID       pulid.ID          `json:"createdById"       keel:"created_by_id,nullzero"`
	SentByID          pulid.ID          `json:"sentById"          keel:"sent_by_id,nullzero"`
	SentAt            int64             `json:"sentAt"            keel:"sent_at,nullzero"`
	Version           int64             `json:"version"           keel:"version,version"`
	CreatedAt         int64             `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt"         keel:"updated_at,autoupdate"`
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

func (r *Reply) BeforeCreate(ctx context.Context) error {
	r.Subject = stringutils.StripNUL(r.Subject)
	r.Body = stringutils.StripNUL(r.Body)
	r.FailureText = stringutils.StripNUL(r.FailureText)
	if r.Status == "" {
		r.Status = ReplyDraft
	}

	return nil
}

func (r *Reply) BeforeUpdate(ctx context.Context) error {
	r.Subject = stringutils.StripNUL(r.Subject)
	r.Body = stringutils.StripNUL(r.Body)
	r.FailureText = stringutils.StripNUL(r.FailureText)

	return nil
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
