package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func chargedShipment() *shipment.Shipment {
	lumperID := pulid.MustNew("acc_")

	return &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		ProNumber:           "SHP-2001",
		Version:             7,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(2000)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(150)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(2150)),
		AdditionalCharges: []*shipment.AdditionalCharge{{
			ID:                  pulid.MustNew("ac_"),
			AccessorialChargeID: lumperID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(150),
			Unit:                1,
			AccessorialCharge: &accessorialcharge.AccessorialCharge{
				ID:          lumperID,
				Code:        "LUMP",
				Description: "Lumper fee",
			},
		}},
	}
}

func TestCorrectChargeCodePreview_ShowsEachChargeAndTheTotal(t *testing.T) {
	t.Parallel()

	before := chargedShipment()
	after := *before
	edited := *before.AdditionalCharges[0]
	edited.Amount = decimal.NewFromInt(200)
	edited.AccessorialCharge = nil
	after.AdditionalCharges = []*shipment.AdditionalCharge{&edited}
	after.OtherChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(200))
	after.TotalChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(2200))

	billing := mocks.NewMockBillingQueueService(t)
	var previewed, updated *serviceports.UpdateChargesRequest
	billing.EXPECT().
		PreviewUpdateCharges(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *serviceports.UpdateChargesRequest,
			_ *serviceports.RequestActor,
		) (*serviceports.ChargeUpdatePreview, error) {
			previewed = req
			return &serviceports.ChargeUpdatePreview{
				Item:   &billingqueue.BillingQueueItem{Number: "BQ-10"},
				Before: before,
				After:  &after,
			}, nil
		}).
		Once()

	tool := newCorrectChargeCodeTool(billing)
	params := executeParams(map[string]any{
		"billingQueueItemId": pulid.MustNew("bqi_").String(),
		"additionalCharges": []any{map[string]any{
			"id":                  edited.ID.String(),
			"accessorialChargeId": edited.AccessorialChargeID.String(),
			"method":              "Flat",
			"amount":              "200",
			"unit":                1,
		}},
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "BQ-10")
	assert.Contains(t, preview.Summary, "from 2150.00 to 2200.00")

	change := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceShipment, change.Resource)
	assert.Equal(t, "SHP-2001", change.Label)
	findField(t, change, "totalChargeAmount")

	require.NotNil(t, change.Money)
	require.Len(t, change.Money.Lines, 2)
	assert.Equal(t, "Freight", change.Money.Lines[0].Label)
	assert.Equal(t, "Lumper fee", change.Money.Lines[1].Label)
	assert.True(t, change.Money.Lines[1].Before.Decimal.Equal(decimal.NewFromInt(150)))
	assert.True(t, change.Money.Lines[1].After.Decimal.Equal(decimal.NewFromInt(200)))
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.NewFromInt(50)))

	billing.EXPECT().
		UpdateCharges(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *serviceports.UpdateChargesRequest,
			_ *serviceports.RequestActor,
		) (*billingqueue.BillingQueueItem, error) {
			updated = req
			return &billingqueue.BillingQueueItem{}, nil
		}).
		Once()
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, previewed.ItemID, updated.ItemID)
	assert.Equal(t, previewed.TenantInfo, updated.TenantInfo)
	require.Len(t, updated.AdditionalCharges, 1)
	assert.True(t, previewed.AdditionalCharges[0].Amount.Equal(updated.AdditionalCharges[0].Amount))
}
