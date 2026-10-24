package customertracking

import (
	"github.com/emoss08/keel"
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
	keel.Table    `keel:"customer_status_notifications"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID           `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	CustomerID     pulid.ID           `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	Event          Event              `json:"event"          bun:"event,type:VARCHAR(20),notnull" keel:"event"`
	SubjectKey     string             `json:"subjectKey"     bun:"subject_key,type:VARCHAR(100),notnull,default:''" keel:"subject_key"`
	StopID         pulid.ID           `json:"stopId"         bun:"stop_id,type:VARCHAR(100),nullzero" keel:"stop_id,nullzero"`
	OccurredAt     int64              `json:"occurredAt"     bun:"occurred_at,type:BIGINT,notnull" keel:"occurred_at"`
	EstimatedAt    *int64             `json:"estimatedAt"    bun:"estimated_at,type:BIGINT,nullzero" keel:"estimated_at"`
	Status         NotificationStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull" keel:"status"`
	SkipReason     SkipReason         `json:"skipReason"     bun:"skip_reason,type:VARCHAR(30),nullzero" keel:"skip_reason,nullzero"`
	Recipients     []string           `json:"recipients"     bun:"recipients,array,notnull" keel:"recipients"`
	LinkID         pulid.ID           `json:"linkId"         bun:"link_id,type:VARCHAR(100),nullzero" keel:"link_id,nullzero"`
	EmailMessageID pulid.ID           `json:"emailMessageId" bun:"email_message_id,type:VARCHAR(100),nullzero" keel:"email_message_id,nullzero"`
	Attempts       int16              `json:"attempts"       bun:"attempts,type:SMALLINT,notnull,default:0" keel:"attempts"`
	LastError      string             `json:"lastError"      bun:"last_error,type:VARCHAR(1000),nullzero" keel:"last_error,nullzero"`
	NextAttemptAt  int64              `json:"nextAttemptAt"  bun:"next_attempt_at,type:BIGINT,notnull" keel:"next_attempt_at"`
	SentAt         *int64             `json:"sentAt"         bun:"sent_at,type:BIGINT,nullzero" keel:"sent_at"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull" keel:"updated_at"`
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
