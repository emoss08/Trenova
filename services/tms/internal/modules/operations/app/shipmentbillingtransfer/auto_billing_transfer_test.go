package shipmentbillingtransfer

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/stretchr/testify/assert"
)

func automaticReadiness() *shipmentbillingports.ShipmentBillingReadiness {
	return &shipmentbillingports.ShipmentBillingReadiness{
		Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
			BillingQueueTransferMode:              billingcontrol.BillingQueueTransferModeAutomaticWhenReady,
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		},
	}
}

func TestReadinessAllowsAutoTransfer(t *testing.T) {
	t.Parallel()

	ready := automaticReadiness()
	assert.True(t, readinessAllowsAutoTransfer(shipmentvocab.StatusReadyToInvoice, ready))
	assert.False(t, readinessAllowsAutoTransfer(shipmentvocab.StatusCompleted, ready),
		"a completed shipment goes only when it may be marked ready without a person")

	ready.ShouldAutoMarkReadyToInvoice = true
	assert.True(t, readinessAllowsAutoTransfer(shipmentvocab.StatusCompleted, ready))
	assert.False(t, readinessAllowsAutoTransfer(shipmentvocab.StatusInTransit, ready))

	manual := automaticReadiness()
	manual.Policy.BillingQueueTransferMode = billingcontrol.BillingQueueTransferModeManualOnly
	assert.False(t, readinessAllowsAutoTransfer(shipmentvocab.StatusReadyToInvoice, manual),
		"a payer who opted out of automatic transfer is left for a person")

	blocked := automaticReadiness()
	blocked.MissingRequirements = []shipmentbillingports.ShipmentBillingRequirement{
		{DocumentTypeCode: "POD"},
	}
	assert.False(t, readinessAllowsAutoTransfer(shipmentvocab.StatusReadyToInvoice, blocked),
		"a transfer billing policy would refuse is never attempted")

	assert.False(t, readinessAllowsAutoTransfer(shipmentvocab.StatusReadyToInvoice, nil))
}
