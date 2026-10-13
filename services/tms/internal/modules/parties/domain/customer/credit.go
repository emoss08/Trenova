package customer

import (
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

// CreditWarningRatio is the share of the credit limit at which open
// receivables put a customer on a credit warning before the hold.
var CreditWarningRatio = decimal.NewFromFloat(0.9)

// CreditReviewOutcome is what a credit review did to a customer's standing.
type CreditReviewOutcome string

const (
	CreditReviewUnchanged = CreditReviewOutcome("Unchanged")
	// CreditReviewHeld put the customer on hold.
	CreditReviewHeld = CreditReviewOutcome("Held")
	// CreditReviewReleased lifted a hold the review had placed.
	CreditReviewReleased = CreditReviewOutcome("Released")
	// CreditReviewWarned moved the customer onto or off a credit warning.
	CreditReviewWarned = CreditReviewOutcome("Warned")
)

// ChangesStanding reports whether the review changed the customer's status.
func (o CreditReviewOutcome) ChangesStanding() bool {
	return o != CreditReviewUnchanged
}

// ReviewsCreditAutomatically reports whether the credit review may place and
// lift holds on this customer: the limit is enforced, automatic holds are on,
// and there is a limit to hold against.
func (b *CustomerBillingProfile) ReviewsCreditAutomatically() bool {
	return b.EnforceCreditLimit && b.AutoCreditHold && b.CreditLimit.Valid
}

// IsOnCreditHold reports whether the customer's credit is stopped: held or
// suspended under an enforced limit. No new shipment may be booked for it.
func (b *CustomerBillingProfile) IsOnCreditHold() bool {
	return b.EnforceCreditLimit &&
		(b.CreditStatus == CreditStatusHold || b.CreditStatus == CreditStatusSuspended)
}

// BlocksBilling reports whether the hold also stops invoicing. Only a hold a
// person placed does: a hold the review placed because receivables ran over
// the limit must not stop the invoices whose payment would lift it.
func (b *CustomerBillingProfile) BlocksBilling() bool {
	return b.IsOnCreditHold() && !b.CreditHoldAutomatic
}

// ReviewCredit records the customer's open receivables and, when the review
// manages its credit, holds it once they exceed the limit, warns it as they
// near the limit, and lifts what it placed once they fall back. A status a
// person set is never changed, so a manual hold, suspension or review stays
// until a person lifts it.
//
// A hold is placed when receivables cross the limit, not whenever they are
// over it: once a person lifts a hold on a customer still over its limit,
// that release stands until the balance falls back under and crosses again.
func (b *CustomerBillingProfile) ReviewCredit(exposureMinor, now int64) CreditReviewOutcome {
	previousMinor := money.MinorUnits(b.CreditBalance)
	firstReview := b.CreditReviewedAt == nil
	b.CreditBalance = money.DecimalFromMinor(exposureMinor)
	b.CreditReviewedAt = &now

	if !b.ReviewsCreditAutomatically() {
		if b.CreditHoldAutomatic {
			return b.liftAutomaticStanding()
		}
		return CreditReviewUnchanged
	}
	if !b.CreditHoldAutomatic && b.CreditStatus != CreditStatusActive {
		return CreditReviewUnchanged
	}

	limitMinor := money.MinorUnits(b.CreditLimit.Decimal)
	crossed := firstReview || b.CreditHoldAutomatic || previousMinor <= limitMinor
	switch {
	case exposureMinor > limitMinor && !crossed:
		return CreditReviewUnchanged
	case exposureMinor > limitMinor:
		return b.placeAutomaticStanding(
			CreditStatusHold,
			"Open receivables of "+b.formatMinor(exposureMinor)+
				" are over the credit limit of "+b.formatMinor(limitMinor),
		)
	case limitMinor > 0 && decimal.NewFromInt(exposureMinor).
		GreaterThanOrEqual(decimal.NewFromInt(limitMinor).Mul(CreditWarningRatio)):
		return b.placeAutomaticStanding(
			CreditStatusWarning,
			"Open receivables of "+b.formatMinor(exposureMinor)+
				" are near the credit limit of "+b.formatMinor(limitMinor),
		)
	case b.CreditHoldAutomatic:
		return b.liftAutomaticStanding()
	default:
		return CreditReviewUnchanged
	}
}

func (b *CustomerBillingProfile) placeAutomaticStanding(
	status CreditStatus,
	reason string,
) CreditReviewOutcome {
	previous := b.CreditStatus
	b.CreditStatus = status
	b.CreditHoldAutomatic = true
	b.CreditHoldReason = reason

	switch {
	case previous == status:
		return CreditReviewUnchanged
	case status == CreditStatusHold:
		return CreditReviewHeld
	case previous == CreditStatusHold:
		return CreditReviewReleased
	default:
		return CreditReviewWarned
	}
}

func (b *CustomerBillingProfile) liftAutomaticStanding() CreditReviewOutcome {
	previous := b.CreditStatus
	b.CreditStatus = CreditStatusActive
	b.CreditHoldAutomatic = false
	b.CreditHoldReason = ""

	switch previous { //nolint:exhaustive // an automatic standing other than a hold is a warning
	case CreditStatusActive:
		return CreditReviewUnchanged
	case CreditStatusHold:
		return CreditReviewReleased
	default:
		return CreditReviewWarned
	}
}

func (b *CustomerBillingProfile) formatMinor(minor int64) string {
	return money.FormatMinor(minor, b.BillingCurrency)
}
