package billingtransferjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

const runStarted = int64(1_794_420_696)

func alreadyTransferred(shipmentID pulid.ID) billingtransferports.BillingTransferItemOutcome {
	return billingtransferports.BillingTransferItemOutcome{
		ShipmentID:   shipmentID,
		Status:       billingtransfer.ItemStatusNotTransferred,
		FailureCode:  billingtransfer.FailureAlreadyTransferred,
		ErrorMessage: "Shipment has already been transferred to billing",
	}
}

func TestARetryRecordsAShipmentThisRunQueuedAsTransferred(t *testing.T) {
	t.Parallel()

	outcomes := []billingtransferports.BillingTransferItemOutcome{alreadyTransferred("shp_1")}
	queued := map[pulid.ID][]*billingqueue.BillingQueueItem{
		"shp_1": {{
			ID:        "bqi_1",
			Number:    "INV2611000011",
			Status:    billingqueue.StatusReadyForReview,
			CreatedAt: runStarted + 7,
		}},
	}

	assert.Equal(t, 1, claimQueuedSince(outcomes, queued, runStarted))
	assert.Equal(t, billingtransferports.BillingTransferItemOutcome{
		ShipmentID:         "shp_1",
		Status:             billingtransfer.ItemStatusTransferred,
		BillingQueueItemID: "bqi_1",
		BillingQueueNumber: "INV2611000011",
		BillingQueueStatus: billingqueue.StatusReadyForReview,
	}, outcomes[0])
}

func TestAShipmentQueuedBeforeTheRunStaysAlreadyTransferred(t *testing.T) {
	t.Parallel()

	outcomes := []billingtransferports.BillingTransferItemOutcome{alreadyTransferred("shp_1")}
	queued := map[pulid.ID][]*billingqueue.BillingQueueItem{
		"shp_1": {{ID: "bqi_1", CreatedAt: runStarted - 60}},
	}

	assert.Equal(t, 0, claimQueuedSince(outcomes, queued, runStarted))
	assert.Equal(t, alreadyTransferred("shp_1"), outcomes[0])
}

func TestOnlyAlreadyTransferredOutcomesAreClaimed(t *testing.T) {
	t.Parallel()

	rated := billingtransferports.BillingTransferItemOutcome{
		ShipmentID:  "shp_2",
		Status:      billingtransfer.ItemStatusNotTransferred,
		FailureCode: billingtransfer.FailureRateValidation,
	}
	outcomes := []billingtransferports.BillingTransferItemOutcome{rated}
	queued := map[pulid.ID][]*billingqueue.BillingQueueItem{
		"shp_2": {{ID: "bqi_2", CreatedAt: runStarted + 1}},
	}

	assert.Equal(t, 0, claimQueuedSince(outcomes, queued, runStarted))
	assert.Equal(t, rated, outcomes[0])
}
