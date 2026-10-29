package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/intaketools"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pricingShipments struct {
	rating    *shipmentserviceports.ShipmentRatingOutcome
	duplicate bool
	refusal   error
	saved     *shipment.Shipment
	guard     agenttoolstest.WriteGuard
}

func (f *pricingShipments) prepare(entity *shipment.Shipment) error {
	if f.refusal != nil {
		return f.refusal
	}
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
	_ *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCreatePlan, error) {
	if err := f.prepare(entity); err != nil {
		return nil, err
	}

	return &shipmentserviceports.ShipmentCreatePlan{Shipment: entity, Rating: f.rating}, nil
}

func (f *pricingShipments) Create(
	_ context.Context,
	entity *shipment.Shipment,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.Write(); err != nil {
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

func createShipmentParams() agenttoolports.ToolExecuteParams {
	params := agenttoolstest.ExecuteParams(map[string]any{
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

	shipments := &pricingShipments{rating: &shipmentserviceports.ShipmentRatingOutcome{
		Adopted:       true,
		Amount:        decimal.NewFromInt(1_850),
		Currency:      "USD",
		AgreementName: "Acme 2026 lanes",
		Explanation:   "Priced by the Acme 2026 lanes agreement.",
	}}
	tool := intaketools.NewCreateShipmentTool(intaketools.CreateShipmentDeps{Shipments: shipments}).(*intaketools.CreateShipmentTool)
	params := createShipmentParams()

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&shipments.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "2 stops under BOL BOL-778")
	assert.Contains(t, preview.Summary, "2000.00 in total under Acme 2026 lanes")
	assert.Contains(t, preview.Summary, "Priced by the Acme 2026 lanes agreement.")
	assert.NotContains(t, preview.Summary, "..")
	require.Len(t, preview.Changes, 3)

	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Shipment with BOL BOL-778", created.Label)
	customer := agenttoolstest.FieldByPath(t, created, "customerId")
	require.NotNil(t, customer.AfterRef)
	assert.Equal(t, permission.ResourceCustomer, customer.AfterRef.Resource)
	require.NotNil(t, created.Money)
	assert.Equal(t, "USD", created.Money.Currency)
	require.True(t, created.Money.TotalAfter.Valid)
	assert.True(t, decimal.NewFromInt(2_000).Equal(created.Money.TotalAfter.Decimal))

	delivery := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipmentStop, delivery.Resource)
	assert.Equal(t, "Delivery stop 2", delivery.Label)
	location := agenttoolstest.FieldByPath(t, delivery, "locationId")
	require.NotNil(t, location.AfterRef)
	assert.Equal(t, permission.ResourceLocation, location.AfterRef.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.saved)
	agenttoolstest.RequireCreateParity(
		t,
		created,
		shipments.saved,
		intaketools.EnteredShipmentOptions()...)
	agenttoolstest.RequireCreateParity(
		t,
		delivery,
		shipments.saved.Moves[0].Stops[1],
		intaketools.EnteredStopOptions()...)
}

func TestCreateShipment_PreviewNamesADepartureFromTheContract(t *testing.T) {
	t.Parallel()

	shipments := &pricingShipments{rating: &shipmentserviceports.ShipmentRatingOutcome{
		Amount:   decimal.NewFromInt(1_700),
		Currency: "USD",
	}}
	tool := intaketools.NewCreateShipmentTool(intaketools.CreateShipmentDeps{Shipments: shipments}).(*intaketools.CreateShipmentTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&shipments.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), createShipmentParams())
		},
	)

	assert.Contains(t, preview.Summary, "would charge 1700.00 USD for the linehaul")
}

func TestCreateShipment_PreviewWarnsWhenTheShipmentWouldBeRefused(t *testing.T) {
	t.Parallel()

	shipments := &pricingShipments{duplicate: true}
	tool := intaketools.NewCreateShipmentTool(intaketools.CreateShipmentDeps{Shipments: shipments}).(*intaketools.CreateShipmentTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&shipments.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), createShipmentParams())
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Message, "BOL")
}

// A copied shipment that keeps its BOL is refused over the BOL alone, and
// the person is told so in those words, with the parameter that carries it.
func TestCreateShipment_ARefusedBOLIsAReasonAPersonCanChange(t *testing.T) {
	t.Parallel()

	duplicate := errortypes.NewMultiError()
	duplicate.Add(
		"bol",
		errortypes.ErrInvalid,
		"BOL is already in use by shipment(s) with Pro Number(s): {0}",
		"SEED-DET-009",
	)
	shipments := &pricingShipments{refusal: duplicate}
	tool := intaketools.NewCreateShipmentTool(intaketools.CreateShipmentDeps{Shipments: shipments}).(*intaketools.CreateShipmentTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&shipments.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), createShipmentParams())
		},
	)
	toolpreview.LocateReasons(preview.Warnings, tool.ParamSchema())

	refusal := preview.Refusal()
	require.NotNil(t, refusal)
	assert.Equal(t,
		"This would be refused as it stands: validation failed:\n"+
			"- bol: BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
		refusal.Message,
	)
	require.Len(t, refusal.Reasons, 1)
	assert.Equal(t, agentpreview.PreviewReason{
		Field:   "bol",
		Label:   "BOL",
		Message: "BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
		Param:   "shipment.bol",
	}, refusal.Reasons[0])
}

func TestCreateShipment_SaysABOLMustBeUniqueAndIsOptional(t *testing.T) {
	t.Parallel()

	tool := intaketools.NewCreateShipmentTool(
		intaketools.CreateShipmentDeps{Shipments: &pricingShipments{}},
	)
	properties := tool.ParamSchema()["properties"].(map[string]any)
	shipmentSchema := properties["shipment"].(map[string]any)
	bol := shipmentSchema["properties"].(map[string]any)["bol"].(map[string]any)

	assert.Contains(t, bol["description"], "unique")
	assert.Contains(t, tool.Description(), "BOL must be unique")
	assert.NotContains(t, shipmentSchema["required"], "bol")
}
