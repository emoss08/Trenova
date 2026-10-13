package customertracking

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// Notification is one status email a customer is owed. The row is the ledger:
// its natural key (shipment, event, subject) is unique, so an event that is
// seen twice, out of order or by two workers at once owes exactly one email,
// and the email is sent under the row's id as its idempotency key, so a retry
// never sends a second copy.
type Notification struct {
	bun.BaseModel `bun:"table:customer_status_notifications,alias:csn" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID           `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	CustomerID     pulid.ID           `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull"`
	Event          Event              `json:"event"          bun:"event,type:VARCHAR(20),notnull"`
	SubjectKey     string             `json:"subjectKey"     bun:"subject_key,type:VARCHAR(100),notnull,default:''"`
	StopID         pulid.ID           `json:"stopId"         bun:"stop_id,type:VARCHAR(100),nullzero"`
	OccurredAt     int64              `json:"occurredAt"     bun:"occurred_at,type:BIGINT,notnull"`
	EstimatedAt    *int64             `json:"estimatedAt"    bun:"estimated_at,type:BIGINT,nullzero"`
	Status         NotificationStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	SkipReason     SkipReason         `json:"skipReason"     bun:"skip_reason,type:VARCHAR(30),nullzero"`
	Recipients     []string           `json:"recipients"     bun:"recipients,array,notnull"`
	LinkID         pulid.ID           `json:"linkId"         bun:"link_id,type:VARCHAR(100),nullzero"`
	EmailMessageID pulid.ID           `json:"emailMessageId" bun:"email_message_id,type:VARCHAR(100),nullzero"`
	Attempts       int16              `json:"attempts"       bun:"attempts,type:SMALLINT,notnull,default:0"`
	LastError      string             `json:"lastError"      bun:"last_error,type:VARCHAR(1000),nullzero"`
	NextAttemptAt  int64              `json:"nextAttemptAt"  bun:"next_attempt_at,type:BIGINT,notnull"`
	SentAt         *int64             `json:"sentAt"         bun:"sent_at,type:BIGINT,nullzero"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
}

func NewNotificationID() pulid.ID {
	return pulid.MustNew("csn_")
}

func (n *Notification) IdempotencyKey() string {
	return "customer-status-" + n.ID.String()
}

// RetryDelaySeconds backs a failed delivery off exponentially from a minute to
// an hour, so a mail provider outage is ridden out without hammering it.
func RetryDelaySeconds(attempts int16) int64 {
	delay := int64(60)
	for i := int16(1); i < attempts && delay < 3600; i++ {
		delay *= 2
	}
	return min(delay, 3600)
}
