package invoiceservice

import (
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// guardStatementCadence refuses to quietly take freight off a customer's
// statement.
//
// A customer on a monthly statement agreed to one invoice a month. Invoicing one
// of their shipments on its own is a legitimate thing a biller sometimes has to
// do — a customer closing their books, a load that has to be billed to a
// different party, a dispute that needs isolating — so this warns rather than
// bars. What it will not allow is doing it by accident: the caller has to say
// why, and the reason is stamped on the invoice.
//
// The error is keyed on "offCycleReason" specifically so the client can tell this
// apart from an ordinary validation failure and ask for the reason instead of
// showing a dead end.
func guardStatementCadence(
	cus *customer.Customer,
	reason string,
) error {
	profile := billingProfileOf(cus)
	if profile == nil || !profile.IsStatementBilled() {
		return nil
	}
	if reason != "" {
		return nil
	}

	return errortypes.NewValidationError(
		"offCycleReason",
		errortypes.ErrRequired,
		"{0} is billed on a {1} statement. Invoicing this freight on its own takes it off that statement — say why to continue.",
		cus.Name,
		profile.BillingCycle.Describe(),
	)
}

// offCycleReasonFor is the reason to stamp, which is only meaningful for a
// customer who is actually on a statement. A reason typed for a per-shipment
// customer is noise and is dropped.
func offCycleReasonFor(cus *customer.Customer, reason string) string {
	profile := billingProfileOf(cus)
	if profile == nil || !profile.IsStatementBilled() {
		return ""
	}

	return reason
}
