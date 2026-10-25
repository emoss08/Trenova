package customertracking

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

// Notification is one status email a customer is owed. The row is the ledger:
// its natural key (shipment, event, subject) is unique, so an event that is
// seen twice, out of order or by two workers at once owes exactly one email,
// and the email is sent under the row's id as its idempotency key, so a retry
// never sends a second copy.
type Notification struct {
	keel.Table `keel:"customer_status_notifications"`

	ID             pulid.ID           `json:"id"             keel:"id,pk"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID           `json:"shipmentId"     keel:"shipment_id"`
	CustomerID     pulid.ID           `json:"customerId"     keel:"customer_id"`
	Event          Event              `json:"event"          keel:"event"`
	SubjectKey     string             `json:"subjectKey"     keel:"subject_key"`
	StopID         pulid.ID           `json:"stopId"         keel:"stop_id,nullzero"`
	OccurredAt     int64              `json:"occurredAt"     keel:"occurred_at"`
	EstimatedAt    *int64             `json:"estimatedAt"    keel:"estimated_at"`
	Status         NotificationStatus `json:"status"         keel:"status"`
	SkipReason     SkipReason         `json:"skipReason"     keel:"skip_reason,nullzero"`
	Recipients     []string           `json:"recipients"     keel:"recipients"`
	LinkID         pulid.ID           `json:"linkId"         keel:"link_id,nullzero"`
	EmailMessageID pulid.ID           `json:"emailMessageId" keel:"email_message_id,nullzero"`
	Attempts       int16              `json:"attempts"       keel:"attempts"`
	LastError      string             `json:"lastError"      keel:"last_error,nullzero"`
	NextAttemptAt  int64              `json:"nextAttemptAt"  keel:"next_attempt_at"`
	SentAt         *int64             `json:"sentAt"         keel:"sent_at"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at"`
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
