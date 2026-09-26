package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pricingShipments struct {
	rating    *serviceports.ShipmentRatingOutcome
	duplicate bool
	saved     *shipment.Shipment
	guard     writeGuard
}

func (f *pricingShipments) prepare(entity *shipment.Shipment) error {
	if f.duplicate {
		return errortypes.NewValidationError(
			"bol",
			errortypes.ErrDuplicate,
			"A shipment with this BOL already exists",
		)
	}

	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1_850))
	entity.OtherChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(150))
	entity.TotalChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(2_000))
	entity.FreightTerms = shipment.FreightTermsPrepaid

	return nil
}

func (f *pricingShipments) PreviewCreate(
	_ context.Context,
	entity *shipment.Shipment,
	_ *serviceports.RequestActor,
) (*serviceports.ShipmentCreatePlan, error) {
	if err := f.prepare(entity); err != nil {
		return nil, err
	}

	return &serviceports.ShipmentCreatePlan{Shipment: entity, Rating: f.rating}, nil
}

func (f *pricingShipments) Create(
	_ context.Context,
	entity *shipment.Shipment,
	_ *serviceports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	if err := f.prepare(entity); err != nil {
		return nil, err
	}
	entity.ID = pulid.MustNew("shp_")
	entity.ProNumber = "S-90001"
	f.saved = entity

	return entity, nil
}

func createShipmentParams() serviceports.ToolExecuteParams {
	params := executeParams(map[string]any{
		"shipment": shipmentPayload(
			pulid.MustNew("cust_"),
			pulid.MustNew("st_"),
			pulid.MustNew("loc_"),
			pulid.MustNew("loc_"),
		),
	})
	params.IdempotencyKey = "idem-preview"

	return params
}

func TestCreateShipment_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	shipments := &pricingShipments{rating: &serviceports.ShipmentRatingOutcome{
		Adopted:       true,
		Amount:        decimal.NewFromInt(1_850),
		Currency:      "USD",
		AgreementName: "Acme 2026 lanes",
		Explanation:   "Priced by the Acme 2026 lanes agreement.",
	}}
	tool := newCreateShipmentTool(shipments, nil, nil).(*createShipmentTool)
	params := createShipmentParams()

	preview := previewWithoutWrites(t, &shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "2 stops under BOL BOL-778")
	assert.Contains(t, preview.Summary, "2000.00 in total under Acme 2026 lanes")
	assert.Contains(t, preview.Summary, "Priced by the Acme 2026 lanes agreement.")
	assert.NotContains(t, preview.Summary, "..")
	require.Len(t, preview.Changes, 3)

	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Shipment with BOL BOL-778", created.Label)
	customer := fieldByPath(t, created, "customerId")
	require.NotNil(t, customer.AfterRef)
	assert.Equal(t, permission.ResourceCustomer, customer.AfterRef.Resource)
	require.NotNil(t, created.Money)
	assert.Equal(t, "USD", created.Money.Currency)
	require.True(t, created.Money.TotalAfter.Valid)
	assert.True(t, decimal.NewFromInt(2_000).Equal(created.Money.TotalAfter.Decimal))

	delivery := previewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipmentStop, delivery.Resource)
	assert.Equal(t, "Delivery stop 2", delivery.Label)
	location := fieldByPath(t, delivery, "locationId")
	require.NotNil(t, location.AfterRef)
	assert.Equal(t, permission.ResourceLocation, location.AfterRef.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.saved)
	requireCreateParity(t, created, shipments.saved, enteredShipmentOptions()...)
	requireCreateParity(t, delivery, shipments.saved.Moves[0].Stops[1], enteredStopOptions()...)
}

func TestCreateShipment_PreviewNamesADepartureFromTheContract(t *testing.T) {
	t.Parallel()

	shipments := &pricingShipments{rating: &serviceports.ShipmentRatingOutcome{
		Amount:   decimal.NewFromInt(1_700),
		Currency: "USD",
	}}
	tool := newCreateShipmentTool(shipments, nil, nil).(*createShipmentTool)

	preview := previewWithoutWrites(t, &shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), createShipmentParams())
	})

	assert.Contains(t, preview.Summary, "would charge 1700.00 USD for the linehaul")
}

func TestCreateShipment_PreviewWarnsWhenTheShipmentWouldBeRefused(t *testing.T) {
	t.Parallel()

	shipments := &pricingShipments{duplicate: true}
	tool := newCreateShipmentTool(shipments, nil, nil).(*createShipmentTool)

	preview := previewWithoutWrites(t, &shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), createShipmentParams())
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Message, "BOL")
}
