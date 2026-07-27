package detention

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DetentionNotice)(nil)

// DetentionNotice is the customer-facing warning that detention has started or
// is about to. Contracts that pay detention reliably require written notice at
// or before free-time expiry, so this is evidence, not courtesy: its delivery
// receipt is what makes the charge collectable.
type DetentionNotice struct {
	bun.BaseModel `bun:"table:detention_notices,alias:dtn" json:"-"`

	ID                    pulid.ID             `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID             `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID             `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DetentionOccurrenceID pulid.ID             `json:"detentionOccurrenceId" bun:"detention_occurrence_id,type:VARCHAR(100),notnull"`
	ThreadKey             string               `json:"threadKey"             bun:"thread_key,type:VARCHAR(120),notnull"`
	Kind                  NoticeKind           `json:"kind"                  bun:"kind,type:detention_notice_kind_enum,notnull"`
	Channel               NoticeChannel        `json:"channel"               bun:"channel,type:detention_notice_channel_enum,notnull,default:'Email'"`
	DeliveryStatus        NoticeDeliveryStatus `json:"deliveryStatus"        bun:"delivery_status,type:detention_notice_delivery_status_enum,notnull,default:'Queued'"`
	Recipients            []string             `json:"recipients"            bun:"recipients,type:JSONB,nullzero"`
	Subject               string               `json:"subject"               bun:"subject,type:VARCHAR(255),notnull"`
	Body                  string               `json:"body"                  bun:"body,type:TEXT,notnull"`
	ScheduledFor          int64                `json:"scheduledFor"          bun:"scheduled_for,type:BIGINT,notnull"`
	SentAt                *int64               `json:"sentAt"                bun:"sent_at,type:BIGINT,nullzero"`
	DeliveredAt           *int64               `json:"deliveredAt"           bun:"delivered_at,type:BIGINT,nullzero"`
	OpenedAt              *int64               `json:"openedAt"              bun:"opened_at,type:BIGINT,nullzero"`
	FailedAt              *int64               `json:"failedAt"              bun:"failed_at,type:BIGINT,nullzero"`
	FailureReason         string               `json:"failureReason"         bun:"failure_reason,type:TEXT,nullzero"`
	ProviderMessageID     string               `json:"providerMessageId"     bun:"provider_message_id,type:VARCHAR(255),nullzero"`
	SentByID              *pulid.ID            `json:"sentById"              bun:"sent_by_id,type:VARCHAR(100),nullzero"`
	WasAutomatic          bool                 `json:"wasAutomatic"          bun:"was_automatic,type:BOOLEAN,notnull,default:false"`
	SatisfiesRequirement  bool                 `json:"satisfiesRequirement"  bun:"satisfies_requirement,type:BOOLEAN,notnull,default:false"`
	QuotedFreeMinutes     int32                `json:"quotedFreeMinutes"     bun:"quoted_free_minutes,type:INTEGER,notnull,default:0"`
	QuotedRate            decimal.NullDecimal  `json:"quotedRate"            bun:"quoted_rate,type:NUMERIC(19,4),nullzero"`
	QuotedAmount          decimal.NullDecimal  `json:"quotedAmount"          bun:"quoted_amount,type:NUMERIC(19,4),nullzero"`
	Version               int64                `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64                `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64                `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
	if n.DetentionOccurrenceID.IsNil() {
		multiErr.Add("detentionOccurrenceId", errortypes.ErrRequired,
			"Detention occurrence is required")
	}
	if n.Subject == "" {
		multiErr.Add("subject", errortypes.ErrRequired, "Subject is required")
	}
	if n.Body == "" {
		multiErr.Add("body", errortypes.ErrRequired, "Body is required")
	}
	if n.Channel == NoticeChannelEmail && len(n.Recipients) == 0 {
		multiErr.Add("recipients", errortypes.ErrRequired,
			"At least one recipient is required to send an email notice")
	}
	if _, err := NoticeKindFromString(string(n.Kind)); err != nil {
		multiErr.Add("kind", errortypes.ErrInvalid, "Notice kind is invalid")
	}
}

func (n *DetentionNotice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if n.Channel == "" {
		n.Channel = NoticeChannelEmail
	}
	if n.DeliveryStatus == "" {
		n.DeliveryStatus = NoticeDeliveryStatusQueued
	}
	if n.ThreadKey == "" && !n.DetentionOccurrenceID.IsNil() {
		n.ThreadKey = "detention:" + n.DetentionOccurrenceID.String()
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if n.ID.IsNil() {
			n.ID = pulid.MustNew("dtn_")
		}
		n.CreatedAt = now
		n.UpdatedAt = now
	case *bun.UpdateQuery:
		n.UpdatedAt = now
	}

	return nil
}
