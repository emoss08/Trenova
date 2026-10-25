package email

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Message struct {
	keel.Table `keel:"email_messages"`

	ID                pulid.ID      `json:"id"                keel:"id,pk,autoid:emlmsg_"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID      `json:"organizationId"    keel:"organization_id,pk,tenant"`
	ProfileID         pulid.ID      `json:"profileId"         keel:"profile_id"`
	Purpose           Purpose       `json:"purpose"           keel:"purpose"`
	Provider          Provider      `json:"provider"          keel:"provider"`
	IdempotencyKey    string        `json:"idempotencyKey"    keel:"idempotency_key"`
	ProviderMessageID string        `json:"providerMessageId" keel:"provider_message_id,nullzero"`
	Status            MessageStatus `json:"status"            keel:"status"`
	Attempts          int32         `json:"attempts"          keel:"attempts"`
	FromEmail         string        `json:"fromEmail"         keel:"from_email"`
	FromName          string        `json:"fromName"          keel:"from_name"`
	ReplyToEmail      string        `json:"replyToEmail"      keel:"reply_to_email,nullzero"`
	ToRecipients      []string      `json:"toRecipients"      keel:"to_recipients"`
	CCRecipients      []string      `json:"ccRecipients"      keel:"cc_recipients,nullzero"`
	BCCRecipients     []string      `json:"bccRecipients"     keel:"bcc_recipients,nullzero"`
	Subject           string        `json:"subject"           keel:"subject"`
	BodyTextSize      int64         `json:"bodyTextSize"      keel:"body_text_size"`
	BodyHTMLSize      int64         `json:"bodyHtmlSize"      keel:"body_html_size"`
	LastError         string        `json:"lastError"         keel:"last_error,nullzero"`
	SentAt            int64         `json:"sentAt"            keel:"sent_at,nullzero"`
	DeliveredAt       int64         `json:"deliveredAt"       keel:"delivered_at,nullzero"`
	FailedAt          int64         `json:"failedAt"          keel:"failed_at,nullzero"`
	CreatedAt         int64         `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64         `json:"updatedAt"         keel:"updated_at,autoupdate"`
	SearchVector      string        `json:"-"                 keel:"search_vector,tsvector,readonly,noselect"`

	Profile      *Profile             `json:"profile,omitempty"     keel:"belongs:profile_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	Attachments  []*Attachment        `json:"attachments,omitempty" keel:"many:message_id"`

	Replayed bool `json:"-"`
}

func (m *Message) SameRecipientsAs(other *Message) bool {
	return m.Purpose == other.Purpose &&
		slices.Equal(m.ToRecipients, other.ToRecipients) &&
		slices.Equal(m.CCRecipients, other.CCRecipients) &&
		slices.Equal(m.BCCRecipients, other.BCCRecipients)
}

func (m *Message) GetTableName() string {
	return "email_messages"
}

func (m *Message) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "em",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "subject", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "from_email",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "provider_message_id",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

type Attachment struct {
	keel.Table `keel:"email_message_attachments"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:emlatt_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	MessageID      pulid.ID       `json:"messageId"      keel:"message_id"`
	FileName       string         `json:"fileName"       keel:"file_name"`
	ContentType    string         `json:"contentType"    keel:"content_type"`
	ObjectKey      string         `json:"objectKey"      keel:"object_key"`
	SizeBytes      int64          `json:"sizeBytes"      keel:"size_bytes"`
	Seal           AttachmentSeal `json:"seal"           keel:"seal,nullzero"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
}

type Event struct {
	keel.Table `keel:"email_events"`

	ID              pulid.ID       `json:"id"              keel:"id,pk,autoid:emlevt_"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID       `json:"organizationId"  keel:"organization_id,pk,tenant"`
	MessageID       pulid.ID       `json:"messageId"       keel:"message_id,nullzero"`
	Provider        Provider       `json:"provider"        keel:"provider"`
	ProviderEventID string         `json:"providerEventId" keel:"provider_event_id"`
	Type            EventType      `json:"type"            keel:"type"`
	Recipient       string         `json:"recipient"       keel:"recipient,nullzero"`
	OccurredAt      int64          `json:"occurredAt"      keel:"occurred_at"`
	Raw             map[string]any `json:"raw"             keel:"raw,jsonb,nullzero"`
	CreatedAt       int64          `json:"createdAt"       keel:"created_at,autocreate"`
}

type Suppression struct {
	keel.Table `keel:"email_suppressions"`

	ID             pulid.ID          `json:"id"             keel:"id,pk,autoid:emlsup_"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	EmailAddress   string            `json:"emailAddress"   keel:"email_address"`
	Reason         SuppressionReason `json:"reason"         keel:"reason"`
	Provider       Provider          `json:"provider"       keel:"provider,nullzero"`
	SourceEventID  string            `json:"sourceEventId"  keel:"source_event_id,nullzero"`
	Notes          string            `json:"notes"          keel:"notes,nullzero"`
	CreatedByID    pulid.ID          `json:"createdById"    keel:"created_by_id,nullzero"`
	CreatedAt      int64             `json:"createdAt"      keel:"created_at,autocreate"`
}

func (s *Suppression) BeforeCreate(_ context.Context) error {
	s.EmailAddress = strings.ToLower(strings.TrimSpace(s.EmailAddress))

	return nil
}

func (m *Message) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.ProfileID, validation.Required.Error("Email profile is required")),
		validation.Field(&m.Purpose,
			validation.Required.Error("Purpose is required"),
			domainvalidation.ValidEnum[Purpose]("Purpose is invalid"),
		),
		validation.Field(&m.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[Provider]("Provider is invalid"),
		),
		// The idempotency key is what stops a retried send from mailing the
		// recipient twice.
		validation.Field(&m.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
			validation.Length(1, maxIdempotencyKeyLength).
				Error("Idempotency key cannot be longer than 160 characters"),
		),
		validation.Field(&m.ProviderMessageID,
			validation.Length(0, maxIdempotencyKeyLength).
				Error("Provider message id cannot be longer than 160 characters"),
		),
		validation.Field(&m.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MessageStatus]("Status is invalid"),
		),
		validation.Field(&m.Attempts,
			validation.Min(int32(0)).Error("Attempts cannot be negative"),
		),
		validation.Field(&m.FromEmail,
			validation.Required.Error("From email is required"),
			is.EmailFormat.Error("From email must be a valid email address"),
			validation.Length(1, maxEmailAddressLength).
				Error("From email cannot be longer than 320 characters"),
		),
		validation.Field(&m.FromName,
			validation.Required.Error("From name is required"),
			validation.Length(1, maxFromNameLength).
				Error("From name cannot be longer than 100 characters"),
		),
		validation.Field(&m.ReplyToEmail,
			is.EmailFormat.Error("Reply to email must be a valid email address"),
			validation.Length(0, maxEmailAddressLength).
				Error("Reply to email cannot be longer than 320 characters"),
		),
		// A message with no recipient is queued, retried and reported on while
		// reaching nobody.
		validation.Field(&m.ToRecipients,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&m.CCRecipients,
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&m.BCCRecipients,
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&m.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, maxSubjectLength).
				Error("Subject cannot be longer than 998 characters"),
		),
		validation.Field(&m.BodyTextSize,
			validation.Min(int64(0)).Error("Body text size cannot be negative"),
		),
		validation.Field(&m.BodyHTMLSize,
			validation.Min(int64(0)).Error("Body HTML size cannot be negative"),
		),
		validation.Field(&m.LastError,
			validation.When(
				m.Status == MessageStatusFailed,
				validation.Required.Error("A failed message must record why it failed"),
			),
		),
	))

	for i, attachment := range m.Attachments {
		if attachment == nil {
			continue
		}
		attachment.Validate(multiErr.WithIndex("attachments", i))
	}
}

func (a *Attachment) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning message are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, maxAttachmentNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&a.ContentType,
			validation.Required.Error("Content type is required"),
			validation.Length(1, maxContentTypeLength).
				Error("Content type cannot be longer than 120 characters"),
		),
		// The object key is the only way back to the bytes, so an attachment
		// without one would be listed on the message and never send.
		validation.Field(&a.ObjectKey, validation.Required.Error("Object key is required")),
		validation.Field(&a.SizeBytes,
			validation.Required.Error("Size is required"),
			validation.Min(int64(1)).Error("Size must be greater than zero"),
		),
	))
}

func (e *Event) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[Provider]("Provider is invalid"),
		),
		// The provider event id is what makes a redelivered webhook
		// recognizable as the same event rather than a second one.
		validation.Field(&e.ProviderEventID,
			validation.Required.Error("Provider event id is required"),
			validation.Length(1, maxProviderEventIDLength).
				Error("Provider event id cannot be longer than 200 characters"),
		),
		validation.Field(&e.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[EventType]("Type is invalid"),
		),
		validation.Field(&e.Recipient,
			is.EmailFormat.Error("Recipient must be a valid email address"),
			validation.Length(0, maxEmailAddressLength).
				Error("Recipient cannot be longer than 320 characters"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a valid timestamp"),
		),
	))
}

func (s *Suppression) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// A suppression stops future mail to this address, so an unparseable one
		// would silently suppress nothing.
		validation.Field(&s.EmailAddress,
			validation.Required.Error("Email address is required"),
			is.EmailFormat.Error("Email address must be a valid email address"),
			validation.Length(1, maxEmailAddressLength).
				Error("Email address cannot be longer than 320 characters"),
		),
		validation.Field(&s.Reason,
			validation.Required.Error("Reason is required"),
			domainvalidation.ValidEnum[SuppressionReason]("Reason is invalid"),
		),
		validation.Field(&s.Provider,
			domainvalidation.ValidEnum[Provider]("Provider is invalid"),
		),
		validation.Field(&s.SourceEventID,
			validation.Length(0, maxProviderEventIDLength).
				Error("Source event id cannot be longer than 200 characters"),
		),
	))
}

const (
	maxIdempotencyKeyLength  = 160
	maxEmailAddressLength    = 320
	maxFromNameLength        = 100
	maxSubjectLength         = 998
	maxAttachmentNameLength  = 255
	maxContentTypeLength     = 120
	maxProviderEventIDLength = 200
)
