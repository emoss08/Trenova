package email

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Message struct {
	bun.BaseModel `bun:"table:email_messages,alias:em" json:"-"`

	ID                pulid.ID      `json:"id"                bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID      `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ProfileID         pulid.ID      `json:"profileId"         bun:"profile_id,type:VARCHAR(100),notnull"`
	Purpose           Purpose       `json:"purpose"           bun:"purpose,type:email_purpose_enum,notnull"`
	Provider          Provider      `json:"provider"          bun:"provider,type:email_provider_type_enum,notnull"`
	IdempotencyKey    string        `json:"idempotencyKey"    bun:"idempotency_key,type:VARCHAR(160),notnull"`
	ProviderMessageID string        `json:"providerMessageId" bun:"provider_message_id,type:VARCHAR(160),nullzero"`
	Status            MessageStatus `json:"status"            bun:"status,type:email_message_status_enum,notnull"`
	Attempts          int32         `json:"attempts"          bun:"attempts,type:INTEGER,notnull,default:0"`
	FromEmail         string        `json:"fromEmail"         bun:"from_email,type:VARCHAR(320),notnull"`
	FromName          string        `json:"fromName"          bun:"from_name,type:VARCHAR(100),notnull"`
	ReplyToEmail      string        `json:"replyToEmail"      bun:"reply_to_email,type:VARCHAR(320),nullzero"`
	ToRecipients      []string      `json:"toRecipients"      bun:"to_recipients,array,type:text[],notnull"`
	CCRecipients      []string      `json:"ccRecipients"      bun:"cc_recipients,array,type:text[],nullzero"`
	BCCRecipients     []string      `json:"bccRecipients"     bun:"bcc_recipients,array,type:text[],nullzero"`
	Subject           string        `json:"subject"           bun:"subject,type:VARCHAR(998),notnull"`
	BodyTextSize      int64         `json:"bodyTextSize"      bun:"body_text_size,type:BIGINT,notnull,default:0"`
	BodyHTMLSize      int64         `json:"bodyHtmlSize"      bun:"body_html_size,type:BIGINT,notnull,default:0"`
	LastError         string        `json:"lastError"         bun:"last_error,type:TEXT,nullzero"`
	SentAt            int64         `json:"sentAt"            bun:"sent_at,type:BIGINT,nullzero"`
	DeliveredAt       int64         `json:"deliveredAt"       bun:"delivered_at,type:BIGINT,nullzero"`
	FailedAt          int64         `json:"failedAt"          bun:"failed_at,type:BIGINT,nullzero"`
	CreatedAt         int64         `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64         `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector      string        `json:"-"                 bun:"search_vector,type:TSVECTOR,scanonly"`

	Profile      *Profile             `json:"profile,omitempty" bun:"rel:belongs-to,join:profile_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"                 bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                 bun:"rel:belongs-to,join:organization_id=id"`
	Attachments  []*Attachment        `json:"attachments,omitempty" bun:"rel:has-many,join:id=message_id"`
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
			{Name: "from_email", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "provider_message_id", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (m *Message) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("emlmsg_")
		}
		m.CreatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}
	return nil
}

type Attachment struct {
	bun.BaseModel `bun:"table:email_message_attachments,alias:ema" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	MessageID      pulid.ID `json:"messageId"      bun:"message_id,type:VARCHAR(100),notnull"`
	FileName       string   `json:"fileName"       bun:"file_name,type:VARCHAR(255),notnull"`
	ContentType    string   `json:"contentType"    bun:"content_type,type:VARCHAR(120),notnull"`
	ObjectKey      string   `json:"objectKey"      bun:"object_key,type:TEXT,notnull"`
	SizeBytes      int64    `json:"sizeBytes"      bun:"size_bytes,type:BIGINT,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (a *Attachment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("emlatt_")
		}
		a.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

type Event struct {
	bun.BaseModel `bun:"table:email_events,alias:ee" json:"-"`

	ID              pulid.ID       `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID       `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	MessageID       pulid.ID       `json:"messageId"       bun:"message_id,type:VARCHAR(100),nullzero"`
	Provider        Provider       `json:"provider"        bun:"provider,type:email_provider_type_enum,notnull"`
	ProviderEventID string         `json:"providerEventId" bun:"provider_event_id,type:VARCHAR(200),notnull"`
	Type            EventType      `json:"type"            bun:"type,type:email_event_type_enum,notnull"`
	Recipient       string         `json:"recipient"       bun:"recipient,type:VARCHAR(320),nullzero"`
	OccurredAt      int64          `json:"occurredAt"      bun:"occurred_at,type:BIGINT,notnull"`
	Raw             map[string]any `json:"raw"             bun:"raw,type:JSONB,nullzero"`
	CreatedAt       int64          `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *Event) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("emlevt_")
		}
		e.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

type Suppression struct {
	bun.BaseModel `bun:"table:email_suppressions,alias:es" json:"-"`

	ID             pulid.ID          `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EmailAddress   string            `json:"emailAddress"   bun:"email_address,type:VARCHAR(320),notnull"`
	Reason         SuppressionReason `json:"reason"         bun:"reason,type:email_suppression_reason_enum,notnull"`
	Provider       Provider          `json:"provider"       bun:"provider,type:email_provider_type_enum,nullzero"`
	SourceEventID  string            `json:"sourceEventId"  bun:"source_event_id,type:VARCHAR(200),nullzero"`
	Notes          string            `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	CreatedByID    pulid.ID          `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *Suppression) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("emlsup_")
		}
		s.EmailAddress = strings.ToLower(strings.TrimSpace(s.EmailAddress))
		s.CreatedAt = timeutils.NowUnix()
	}
	return nil
}
