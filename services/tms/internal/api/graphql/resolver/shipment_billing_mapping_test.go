package resolver

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkTransferToBillingToModel(t *testing.T) {
	t.Parallel()

	transferredID := pulid.MustNew("shp_")
	blockedID := pulid.MustNew("shp_")
	brokenID := pulid.MustNew("shp_")
	item := &billingqueue.BillingQueueItem{
		ID:         pulid.MustNew("bqi_"),
		ShipmentID: transferredID,
		Number:     "INV-100",
		Status:     billingqueue.StatusApproved,
		BillType:   billingqueue.BillTypeInvoice,
	}
	policyErr := errortypes.NewValidationError(
		"billingReadiness",
		errortypes.ErrInvalidOperation,
		"Shipment billing requirements must be resolved before transfer to billing",
	)
	internalErr := errors.New(
		`pq: duplicate key value violates unique constraint "billing_queue_items_pkey"`,
	)

	model, err := bulkTransferToBillingToModel(t.Context(), &services.BulkTransferToBillingResponse{
		TotalCount:   3,
		SuccessCount: 1,
		ErrorCount:   2,
		Results: []services.BulkTransferToBillingResult{
			{
				ShipmentID:           transferredID,
				ProNumber:            "PRO-100",
				Success:              true,
				MarkedReadyToInvoice: true,
				Item:                 item,
				MissingRequirements:  []services.ShipmentBillingRequirement{},
				ValidationFailures:   []services.ShipmentBillingValidation{},
			},
			{
				ShipmentID:  blockedID,
				ProNumber:   "PRO-200",
				FailureCode: services.BillingTransferFailureRequirementsUnmet,
				Error:       policyErr.Error(),
				Err:         policyErr,
				MissingRequirements: []services.ShipmentBillingRequirement{
					{
						DocumentTypeID:   "dt_1",
						DocumentTypeCode: "POD",
						DocumentTypeName: "Proof of Delivery",
					},
				},
				ValidationFailures: []services.ShipmentBillingValidation{
					{
						Field:   "bol",
						Code:    "missing_bol",
						Message: "BOL is required before the shipment can be invoiced",
					},
				},
			},
			{
				ShipmentID:          brokenID,
				FailureCode:         services.BillingTransferFailureUnexpected,
				Error:               "The shipment could not be transferred to billing because of an unexpected error",
				Err:                 internalErr,
				MissingRequirements: []services.ShipmentBillingRequirement{},
				ValidationFailures:  []services.ShipmentBillingValidation{},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 3, model.TotalCount)
	assert.Equal(t, 1, model.SuccessCount)
	assert.Equal(t, 2, model.ErrorCount)
	require.Len(t, model.Results, 3)

	transferred := model.Results[0]
	assert.Equal(t, transferredID.String(), transferred.ShipmentID)
	require.NotNil(t, transferred.ProNumber)
	assert.Equal(t, "PRO-100", *transferred.ProNumber)
	assert.True(t, transferred.Success)
	assert.True(t, transferred.MarkedReadyToInvoice)
	assert.Nil(t, transferred.FailureCode)
	assert.Nil(t, transferred.Error)
	require.NotNil(t, transferred.BillingQueueItem)
	assert.Equal(t, item.ID.String(), transferred.BillingQueueItem.ID)
	assert.Equal(t, "INV-100", transferred.BillingQueueItem.Number)
	assert.NotNil(t, transferred.MissingRequirements)
	assert.Empty(t, transferred.MissingRequirements)
	assert.NotNil(t, transferred.ValidationFailures)

	blocked := model.Results[1]
	assert.False(t, blocked.Success)
	assert.Nil(t, blocked.BillingQueueItem)
	require.NotNil(t, blocked.FailureCode)
	assert.Equal(t, services.BillingTransferFailureRequirementsUnmet, *blocked.FailureCode)
	require.NotNil(t, blocked.Error)
	assert.Equal(
		t,
		"Shipment billing requirements must be resolved before transfer to billing",
		*blocked.Error,
	)
	require.Len(t, blocked.MissingRequirements, 1)
	assert.Equal(t, "Proof of Delivery", blocked.MissingRequirements[0].DocumentTypeName)
	require.Len(t, blocked.ValidationFailures, 1)
	assert.Equal(t, "missing_bol", blocked.ValidationFailures[0].Code)

	broken := model.Results[2]
	assert.Nil(t, broken.ProNumber)
	require.NotNil(t, broken.FailureCode)
	assert.Equal(t, services.BillingTransferFailureUnexpected, *broken.FailureCode)
	require.NotNil(t, broken.Error)
	assert.NotContains(t, *broken.Error, "pq:")
	assert.NotContains(t, *broken.Error, "billing_queue_items_pkey")
}
