package billingqueuereadiness

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func heldItem() *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		ID:     pulid.MustNew("bqi_"),
		Status: billingqueue.StatusReadyForReview,
		BillToCustomer: &customer.Customer{
			Name: "Acme Foods",
			BillingProfile: &customer.CustomerBillingProfile{
				PaymentTerm:        customer.PaymentTermNet30,
				EnforceCreditLimit: true,
				CreditStatus:       customer.CreditStatusHold,
				CreditHoldReason:   "Two invoices past 90 days",
			},
		},
	}
}

func reviewerOn(gate platformcatalogports.FeatureGate) *Reviewer {
	return &Reviewer{l: zap.NewNop(), features: gate}
}

func TestReviewFacts_WithholdsTheCreditHoldWithoutAccounting(t *testing.T) {
	t.Parallel()

	reviewer := reviewerOn(plantest.BaseOnly())
	facts := reviewer.reviewFacts(t.Context(), heldItem(), plantest.Tenant())

	require.NotNil(t, facts.terms)
	assert.Equal(t, "Net30", facts.terms.PaymentTerm)
	assert.Nil(t, facts.terms.CreditHold, "the terms say nothing about a hold")
	assert.False(t, facts.findings.CreditHold)
	assert.Empty(t, facts.findings.CreditHoldReason)
}

func TestReviewFacts_CarriesTheCreditHoldWithAccounting(t *testing.T) {
	t.Parallel()

	reviewer := reviewerOn(plantest.AllFeatures())
	facts := reviewer.reviewFacts(t.Context(), heldItem(), plantest.Tenant())

	require.NotNil(t, facts.terms)
	require.NotNil(t, facts.terms.CreditHold)
	assert.True(t, *facts.terms.CreditHold)
	assert.True(t, facts.findings.CreditHold)
	assert.Equal(t, "Two invoices past 90 days", facts.findings.CreditHoldReason)
}
