// Package paymentreminder is the email a customer receives once its invoices
// go past due, on the days past due its organization schedules.
package paymentreminder

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// ManualStep marks a reminder a person sent by hand. Scheduled steps count
// from one and are each sent once per invoice; a manual reminder may repeat.
const ManualStep = 0

// Reminder records that one invoice was included in a reminder email: which
// step of the schedule it was, how late and how much was open, and who it
// went to.
type Reminder struct {
	keel.Table `keel:"invoice_payment_reminders"`

	ID              pulid.ID `json:"id"              keel:"id,pk,autoid:ipr_"`
	OrganizationID  pulid.ID `json:"organizationId"  keel:"organization_id,pk,tenant"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	InvoiceID       pulid.ID `json:"invoiceId"       keel:"invoice_id"`
	CustomerID      pulid.ID `json:"customerId"      keel:"customer_id"`
	Step            int16    `json:"step"            keel:"step"`
	DaysPastDue     int      `json:"daysPastDue"     keel:"days_past_due"`
	OpenAmountMinor int64    `json:"openAmountMinor" keel:"open_amount_minor"`
	CurrencyCode    string   `json:"currencyCode"    keel:"currency_code"`
	SentTo          string   `json:"sentTo"          keel:"sent_to"`
	Manual          bool     `json:"manual"          keel:"manual"`
	SentByID        pulid.ID `json:"sentById"        keel:"sent_by_id,nullzero"`
	SentAt          int64    `json:"sentAt"          keel:"sent_at"`
}

func (r *Reminder) BeforeCreate(ctx context.Context) error {
	if r.SentAt == 0 {
		r.SentAt = timeutils.NowUnix()
	}

	return nil
}

// StepReached is the latest step of the schedule an invoice this many days
// past due has reached, counting from one; zero before the first.
func StepReached(daysPastDue int, schedule []int32) int {
	reached := 0
	for i, day := range schedule {
		if daysPastDue >= int(day) {
			reached = i + 1
		}
	}
	return reached
}

// IsFinal reports whether the step is the last one the schedule sends.
func IsFinal(step int, schedule []int32) bool {
	return step > 0 && step == len(schedule)
}

// DaysPastDue counts whole days since the due date; zero on or before it.
func DaysPastDue(dueDate, now int64) int {
	if dueDate <= 0 || now <= dueDate {
		return 0
	}
	return int((now - dueDate) / (24 * 60 * 60))
}
