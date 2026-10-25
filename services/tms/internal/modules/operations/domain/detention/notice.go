package detention

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
)

const (
	maxThreadKeyLength     = 120
	maxNoticeSubjectLength = 255
)

// DetentionNotice is the customer-facing warning that detention has started or
// is about to. Contracts that pay detention reliably require written notice at
// or before free-time expiry, so this is evidence, not courtesy: its delivery
// receipt is what makes the charge collectable.
type DetentionNotice struct {
	keel.Table `keel:"detention_notices"`

	ID                    pulid.ID             `json:"id"                    keel:"id,pk,autoid:dtn_"`
	BusinessUnitID        pulid.ID             `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	OrganizationID        pulid.ID             `json:"organizationId"        keel:"organization_id,pk,tenant"`
	DetentionOccurrenceID pulid.ID             `json:"detentionOccurrenceId" keel:"detention_occurrence_id"`
	ThreadKey             string               `json:"threadKey"             keel:"thread_key"`
	Kind                  NoticeKind           `json:"kind"                  keel:"kind"`
	Channel               NoticeChannel        `json:"channel"               keel:"channel"`
	DeliveryStatus        NoticeDeliveryStatus `json:"deliveryStatus"        keel:"delivery_status"`
	Recipients            []string             `json:"recipients"            keel:"recipients,jsonb,nullzero"`
	Subject               string               `json:"subject"               keel:"subject"`
	Body                  string               `json:"body"                  keel:"body"`
	// BodyHTML snapshots the rendered HTML alongside the plain text.
	//
	// It exists because the send path used to rebuild HTML by wrapping Body in
	// <pre> with no escaping, which turned any angle bracket in a facility or
	// customer name into markup. Storing what was actually rendered means the
	// sender interpolates nothing.
	BodyHTML string `json:"bodyHtml" keel:"body_html,nullzero"`
	// TemplateVersionID names the template that produced this notice, so a
	// disputed charge can be traced back to the exact wording that was sent.
	TemplateVersionID *pulid.ID `json:"templateVersionId" keel:"template_version_id"`
	// PDFDocumentID is the stored notice PDF, when the policy attaches one.
	PDFDocumentID *pulid.ID `json:"pdfDocumentId" keel:"pdf_document_id"`

	ScheduledFor         int64               `json:"scheduledFor"         keel:"scheduled_for"`
	SentAt               *int64              `json:"sentAt"               keel:"sent_at"`
	DeliveredAt          *int64              `json:"deliveredAt"          keel:"delivered_at"`
	OpenedAt             *int64              `json:"openedAt"             keel:"opened_at"`
	FailedAt             *int64              `json:"failedAt"             keel:"failed_at"`
	FailureReason        string              `json:"failureReason"        keel:"failure_reason,nullzero"`
	ProviderMessageID    string              `json:"providerMessageId"    keel:"provider_message_id,nullzero"`
	SentByID             *pulid.ID           `json:"sentById"             keel:"sent_by_id"`
	WasAutomatic         bool                `json:"wasAutomatic"         keel:"was_automatic"`
	SatisfiesRequirement bool                `json:"satisfiesRequirement" keel:"satisfies_requirement"`
	QuotedFreeMinutes    int32               `json:"quotedFreeMinutes"    keel:"quoted_free_minutes"`
	QuotedRate           decimal.NullDecimal `json:"quotedRate"           keel:"quoted_rate"`
	QuotedAmount         decimal.NullDecimal `json:"quotedAmount"         keel:"quoted_amount"`
	Version              int64               `json:"version"              keel:"version,version"`
	CreatedAt            int64               `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt"            keel:"updated_at,autoupdate"`
}

func (n *DetentionNotice) GetID() pulid.ID { return n.ID }

func (n *DetentionNotice) GetOrganizationID() pulid.ID { return n.OrganizationID }

func (n *DetentionNotice) GetBusinessUnitID() pulid.ID { return n.BusinessUnitID }

func (n *DetentionNotice) GetTableName() string { return "detention_notices" }

// MarkSent records dispatch of the notice and decides whether it landed inside
// the contractual window.
func (n *DetentionNotice) MarkSent(now int64, deadline *int64, providerMessageID string) {
	n.DeliveryStatus = NoticeDeliveryStatusSent
	n.SentAt = &now
	n.ProviderMessageID = providerMessageID
	n.SatisfiesRequirement = deadline == nil || now <= *deadline
}

func (n *DetentionNotice) MarkDelivered(now int64) {
	n.DeliveryStatus = NoticeDeliveryStatusDelivered
	n.DeliveredAt = &now
}

func (n *DetentionNotice) MarkOpened(now int64) {
	n.DeliveryStatus = NoticeDeliveryStatusOpened
	n.OpenedAt = &now
}

// MarkFailed records a bounce or provider rejection. A failed notice never
// satisfies the contractual requirement, regardless of when it was attempted.
func (n *DetentionNotice) MarkFailed(now int64, reason string) {
	n.DeliveryStatus = NoticeDeliveryStatusFailed
	n.FailedAt = &now
	n.FailureReason = reason
	n.SatisfiesRequirement = false
}

func (n *DetentionNotice) MarkBounced(now int64, reason string) {
	n.DeliveryStatus = NoticeDeliveryStatusBounced
	n.FailedAt = &now
	n.FailureReason = reason
	n.SatisfiesRequirement = false
}

func (n *DetentionNotice) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(n,
		validation.Field(&n.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&n.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&n.DetentionOccurrenceID,
			validation.Required.Error("Detention occurrence is required"),
		),
		validation.Field(&n.ThreadKey,
			validation.Length(0, maxThreadKeyLength).
				Error("Thread key cannot be longer than 120 characters"),
		),
		validation.Field(&n.Kind,
			validation.Required.Error("Notice kind is required"),
			domainvalidation.ValidEnumFunc(NoticeKindFromString, "Notice kind is invalid"),
		),
		validation.Field(&n.Channel,
			domainvalidation.ValidEnumFunc(NoticeChannelFromString, "Notice channel is invalid"),
		),
		validation.Field(&n.DeliveryStatus,
			domainvalidation.ValidEnumFunc(
				NoticeDeliveryStatusFromString, "Delivery status is invalid",
			),
		),
		validation.Field(&n.Recipients,
			validation.When(n.Channel == NoticeChannelEmail, validation.Required.Error(
				"At least one recipient is required to send an email notice",
			)),
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&n.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, maxNoticeSubjectLength).
				Error("Subject cannot be longer than 255 characters"),
		),
		validation.Field(&n.Body, validation.Required.Error("Body is required")),
		validation.Field(&n.QuotedFreeMinutes,
			validation.Min(int32(0)).Error("Quoted free minutes cannot be negative"),
		),
		// A failure with no reason is a support ticket nobody can close.
		validation.Field(&n.FailureReason,
			validation.When(
				n.DeliveryStatus == NoticeDeliveryStatusFailed,
				validation.Required.Error("A failed notice must record why it failed"),
			),
		),
	))
}

func (n *DetentionNotice) BeforeCreate(ctx context.Context) error {
	if n.Channel == "" {
		n.Channel = NoticeChannelEmail
	}
	if n.DeliveryStatus == "" {
		n.DeliveryStatus = NoticeDeliveryStatusQueued
	}
	if n.ThreadKey == "" && !n.DetentionOccurrenceID.IsNil() {
		n.ThreadKey = "detention:" + n.DetentionOccurrenceID.String()
	}

	return nil
}

func (n *DetentionNotice) BeforeUpdate(ctx context.Context) error {
	if n.Channel == "" {
		n.Channel = NoticeChannelEmail
	}
	if n.DeliveryStatus == "" {
		n.DeliveryStatus = NoticeDeliveryStatusQueued
	}
	if n.ThreadKey == "" && !n.DetentionOccurrenceID.IsNil() {
		n.ThreadKey = "detention:" + n.DetentionOccurrenceID.String()
	}

	return nil
}

func (DetentionNotice) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_detention_notices_occurrence",
			Message: "Detention occurrence does not exist in your organization",
		},
		{
			Name:    "fk_detention_notices_template_version",
			Message: "Document template version does not exist in your organization",
		},
		{
			Name:    "fk_detention_notices_pdf_document",
			Message: "Document does not exist in your organization",
		},
	}
}
