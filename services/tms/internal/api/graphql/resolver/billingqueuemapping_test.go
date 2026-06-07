package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingQueueItemToModel_MapsTypedBillingQueueItem(t *testing.T) {
	t.Parallel()

	assignedBillerID := pulid.MustNew("usr_")
	canceledByID := pulid.MustNew("usr_")
	sourceInvoiceID := pulid.MustNew("inv_")
	exceptionReason := billingqueue.ExceptionServiceFailure
	reviewStartedAt := int64(1780415883)
	canceledAt := int64(1780415999)

	model, err := billingQueueItemToModel(&billingqueue.BillingQueueItem{
		ID:                        pulid.MustNew("bqi_"),
		OrganizationID:            pulid.MustNew("org_"),
		BusinessUnitID:            pulid.MustNew("bu_"),
		ShipmentID:                pulid.MustNew("shp_"),
		AssignedBillerID:          &assignedBillerID,
		Number:                    "BQI-1001",
		Status:                    billingqueue.StatusException,
		BillType:                  billingqueue.BillTypeCreditMemo,
		ExceptionReasonCode:       &exceptionReason,
		ReviewNotes:               "needs review",
		ExceptionNotes:            "unresolved service failure",
		ReviewStartedAt:           &reviewStartedAt,
		CanceledByID:              &canceledByID,
		CanceledAt:                &canceledAt,
		CancelReason:              "duplicate",
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoiceID,
		RequiresReplacementReview: true,
		RerateVariancePercent:     decimal.RequireFromString("2.500000"),
		AdjustmentContext: map[string]any{
			"source": "serviceFailure214",
		},
		Version:   7,
		CreatedAt: 1780415777,
		UpdatedAt: 1780415888,
	})
	require.NoError(t, err)

	require.NotNil(t, model)
	assert.Equal(t, "BQI-1001", model.Number)
	assert.Equal(t, billingqueue.StatusException, model.Status)
	assert.Equal(t, billingqueue.BillTypeCreditMemo, model.BillType)
	assert.Equal(t, billingqueue.ExceptionServiceFailure, *model.ExceptionReasonCode)
	assert.Equal(t, assignedBillerID.String(), *model.AssignedBillerID)
	assert.Equal(t, canceledByID.String(), *model.CanceledByID)
	assert.Equal(t, sourceInvoiceID.String(), *model.SourceInvoiceID)
	assert.Equal(t, "2.5", model.RerateVariancePercent)
	assert.Equal(t, "serviceFailure214", model.AdjustmentContext["source"])
	assert.Equal(t, 7, model.Version)
	assert.Equal(t, 1780415777, model.CreatedAt)
	assert.Equal(t, 1780415888, model.UpdatedAt)
}

func TestBillingQueueItemToModel_DefaultsNilAdjustmentContext(t *testing.T) {
	t.Parallel()

	model, err := billingQueueItemToModel(&billingqueue.BillingQueueItem{
		ID:                    pulid.MustNew("bqi_"),
		OrganizationID:        pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		ShipmentID:            pulid.MustNew("shp_"),
		Status:                billingqueue.StatusReadyForReview,
		BillType:              billingqueue.BillTypeInvoice,
		RerateVariancePercent: decimal.Zero,
	})
	require.NoError(t, err)

	require.NotNil(t, model)
	assert.NotNil(t, model.AdjustmentContext)
	assert.Empty(t, model.AdjustmentContext)
}

func TestRequiredBillingQueueItemToModel_RejectsNilItem(t *testing.T) {
	t.Parallel()

	model, err := requiredBillingQueueItemToModel(nil)

	require.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "Billing queue transfer did not return an item")
}
