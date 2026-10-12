package customer

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func managedProfile(limit string) *CustomerBillingProfile {
	return &CustomerBillingProfile{
		EnforceCreditLimit: true,
		AutoCreditHold:     true,
		CreditLimit:        decimal.NewNullDecimal(decimal.RequireFromString(limit)),
		CreditStatus:       CreditStatusActive,
		BillingCurrency:    "USD",
	}
}

func TestReviewCredit_HoldsACustomerOverItsLimit(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	outcome := profile.ReviewCredit(1_250_000, 1_700_000_000)

	assert.Equal(t, CreditReviewHeld, outcome)
	assert.Equal(t, CreditStatusHold, profile.CreditStatus)
	assert.True(t, profile.CreditHoldAutomatic)
	assert.Equal(t,
		"Open receivables of 12500.00 USD are over the credit limit of 10000.00 USD",
		profile.CreditHoldReason)
	assert.True(t, profile.CreditBalance.Equal(decimal.RequireFromString("12500")))
	assert.Equal(t, int64(1_700_000_000), *profile.CreditReviewedAt)
	assert.True(t, profile.IsOnCreditHold())
	assert.False(t, profile.BlocksBilling(), "an automatic hold must not stop the invoices that would lift it")
}

func TestReviewCredit_WarnsNearTheLimitAndReleasesOncePaidDown(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	profile.ReviewCredit(1_100_000, 1)

	assert.Equal(t, CreditReviewReleased, profile.ReviewCredit(950_000, 2),
		"falling back under the limit lifts the hold, onto a warning")
	assert.Equal(t, CreditStatusWarning, profile.CreditStatus)
	assert.True(t, profile.CreditHoldAutomatic)

	assert.Equal(t, CreditReviewWarned, profile.ReviewCredit(100_000, 3))
	assert.Equal(t, CreditStatusActive, profile.CreditStatus)
	assert.False(t, profile.CreditHoldAutomatic)
	assert.Empty(t, profile.CreditHoldReason)
}

func TestReviewCredit_ReleasesStraightFromHoldWhenPaidUnderTheWarning(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	profile.ReviewCredit(1_100_000, 1)

	assert.Equal(t, CreditReviewReleased, profile.ReviewCredit(0, 2))
	assert.Equal(t, CreditStatusActive, profile.CreditStatus)
}

func TestReviewCredit_NeverChangesAStatusAPersonSet(t *testing.T) {
	t.Parallel()

	for _, status := range []CreditStatus{
		CreditStatusHold, CreditStatusSuspended, CreditStatusReview, CreditStatusWarning,
	} {
		profile := managedProfile("10000")
		profile.CreditStatus = status
		profile.CreditHoldReason = "Bankruptcy filing"

		assert.Equal(t, CreditReviewUnchanged, profile.ReviewCredit(0, 1), status)
		assert.Equal(t, status, profile.CreditStatus)
		assert.Equal(t, "Bankruptcy filing", profile.CreditHoldReason)
		assert.True(t, profile.CreditBalance.IsZero())
	}

	manual := managedProfile("10000")
	manual.CreditStatus = CreditStatusHold
	assert.True(t, manual.BlocksBilling(), "a hold a person placed still stops billing")
}

func TestReviewCredit_LiftsItsOwnHoldWhenAutomaticHoldsAreTurnedOff(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	profile.ReviewCredit(2_000_000, 1)
	profile.AutoCreditHold = false

	assert.Equal(t, CreditReviewReleased, profile.ReviewCredit(2_000_000, 2))
	assert.Equal(t, CreditStatusActive, profile.CreditStatus)
}

func TestReviewCredit_OnlyRecordsTheBalanceWhenNotManaged(t *testing.T) {
	t.Parallel()

	profile := &CustomerBillingProfile{CreditStatus: CreditStatusActive, AutoCreditHold: true}
	assert.Equal(t, CreditReviewUnchanged, profile.ReviewCredit(5_000_000, 1))
	assert.Equal(t, CreditStatusActive, profile.CreditStatus)
	assert.True(t, profile.CreditBalance.Equal(decimal.RequireFromString("50000")))
}

func TestReviewCredit_AZeroLimitHoldsAnyOpenBalance(t *testing.T) {
	t.Parallel()

	profile := managedProfile("0")
	assert.Equal(t, CreditReviewUnchanged, profile.ReviewCredit(0, 1))
	assert.Equal(t, CreditReviewHeld, profile.ReviewCredit(1, 2))
}

func TestReviewCredit_KeepsAReleaseAPersonMadeUntilTheLimitIsCrossedAgain(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	profile.ReviewCredit(1_200_000, 1)
	require.Equal(t, CreditStatusHold, profile.CreditStatus)

	// Accounts receivable lifts the hold by hand while the customer is still over.
	profile.CreditStatus = CreditStatusActive
	profile.CreditHoldAutomatic = false
	profile.CreditHoldReason = ""

	assert.Equal(t, CreditReviewUnchanged, profile.ReviewCredit(1_300_000, 2),
		"more billing while released does not undo the person's release")
	assert.Equal(t, CreditStatusActive, profile.CreditStatus)

	profile.ReviewCredit(500_000, 3)
	assert.Equal(t, CreditReviewHeld, profile.ReviewCredit(1_050_000, 4),
		"crossing the limit again holds the customer again")
}

func TestReviewCredit_HoldsACustomerAlreadyOverItsLimitOnItsFirstReview(t *testing.T) {
	t.Parallel()

	profile := managedProfile("10000")
	profile.CreditBalance = decimal.RequireFromString("50000")

	assert.Equal(t, CreditReviewHeld, profile.ReviewCredit(5_000_000, 1))
}
