package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

func autoTransferPolicy(
	requirement tenant.EnforcementLevel,
) services.ShipmentBillingReadinessPolicy {
	return services.ShipmentBillingReadinessPolicy{
		BillingQueueTransferMode:              tenant.BillingQueueTransferModeAutomaticWhenReady,
		ShipmentBillingRequirementEnforcement: requirement,
		RateValidationEnforcement:             requirement,
	}
}

func autoApproveProfile(on bool) *customer.CustomerBillingProfile {
	return &customer.CustomerBillingProfile{AutoApprove: on}
}

func TestCleanFreightAutoApprovesWhenTheCustomerAsksForIt(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldAutoApproveBilling(
		autoTransferPolicy(tenant.EnforcementLevelBlock),
		autoApproveProfile(true),
		false,
		false,
	))
}

// The whole safety argument. canAutoProgress lets a shipment move while
// enforcement is Ignore, issues and all — defensible for getting freight into a
// work queue, not defensible for approving it, because approval is the step that
// puts the freight in front of the customer. Any issue stops auto-approval no
// matter how lenient the organization's enforcement is.
func TestIssuesStopAutoApprovalEvenWhenEnforcementIsIgnore(t *testing.T) {
	t.Parallel()

	lenient := autoTransferPolicy(tenant.EnforcementLevelIgnore)

	assert.False(t, shouldAutoApproveBilling(lenient, autoApproveProfile(true), true, false),
		"a missing document must never auto-approve")
	assert.False(t, shouldAutoApproveBilling(lenient, autoApproveProfile(true), false, true),
		"a rate problem must never auto-approve")
	assert.False(t, shouldAutoApproveBilling(lenient, autoApproveProfile(true), true, true))
}

// The organization gate: a shop that has not enabled automatic queue transfer
// cannot be auto-approving anything, whatever a customer's profile says.
func TestManualTransferOrganizationsNeverAutoApprove(t *testing.T) {
	t.Parallel()

	manual := services.ShipmentBillingReadinessPolicy{
		BillingQueueTransferMode:              tenant.BillingQueueTransferModeManualOnly,
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		RateValidationEnforcement:             tenant.EnforcementLevelBlock,
	}

	assert.False(t, shouldAutoApproveBilling(manual, autoApproveProfile(true), false, false))
}

// Opt-in, not default. Every customer who has not asked for it keeps a human in
// the loop, which is what makes this safe to ship on by default for nobody.
func TestAutoApprovalIsOptIn(t *testing.T) {
	t.Parallel()

	policy := autoTransferPolicy(tenant.EnforcementLevelBlock)

	assert.False(t, shouldAutoApproveBilling(policy, autoApproveProfile(false), false, false))
	assert.False(t, shouldAutoApproveBilling(policy, nil, false, false),
		"a customer with no billing profile has not opted in")
}
