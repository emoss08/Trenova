package agenttoolservice

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errorFields(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}

	return fields
}

func schemaCheck(tool agentports.AgentTool, args map[string]any) error {
	return toolschema.NewValidator().ValidateFor(tool.Name(), tool.ParamSchema(), args)
}

func TestCreateShipment_TheSchemaAsksForWhatTheDomainNeeds(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: &agenttoolstest.FakeShipmentWriter{}})
	payload := shipmentPayload(
		pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
	)
	require.NoError(t, schemaCheck(tool, map[string]any{"shipment": payload}))

	for _, required := range []string{"shipmentTypeId"} {
		without := maps.Clone(payload)
		delete(without, required)

		err := schemaCheck(tool, map[string]any{"shipment": without})
		require.Error(t, err, required)
		assert.Contains(t, errorFields(t, err), "shipment."+required)
	}

	assert.Contains(t, tool.Description(), "shipment_duplicate")
	assert.Contains(t, tool.Description(), "Price it the way the person did")

	withoutRating := maps.Clone(payload)
	delete(withoutRating, "formulaTemplateId")
	require.NoError(t, schemaCheck(tool, map[string]any{"shipment": withoutRating}),
		"the contract supplies the rating method when the call names none")
}

type fakeContractRates struct {
	priced *shipmentports.ContractRateApplication
	err    error
	calls  int
}

func (f *fakeContractRates) PreviewContractRate(
	context.Context,
	*shipment.Shipment,
	*permissionports.RequestActor,
) (*shipmentports.ContractRateApplication, error) {
	f.calls++

	return f.priced, f.err
}

/*
An agent that held shipment_create but not rate_quote asked the person for
a rating method and base rate for Peak Distributing, whose agreement already
prices the lane. Saving the shipment consults the contract either way; leaving
the rating out now seats the contract's.
*/
func TestCreateShipment_UsesTheContractsRatingWhenNoneIsGiven(t *testing.T) {
	t.Parallel()

	template := pulid.MustNew("ft_")
	contracts := &fakeContractRates{priced: &shipmentports.ContractRateApplication{
		Applied:           true,
		Outcome:           ratequote.OutcomeRated,
		FormulaTemplateID: &template,
		BaseRate:          decimal.NewNullDecimal(decimal.RequireFromString("2.10")),
	}}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{
		Shipments: &agenttoolstest.FakeShipmentWriter{},
		Contracts: contracts,
	}).(*operationstools.CreateShipmentTool)
	params := createShipmentParams()
	delete(params.Params["shipment"].(map[string]any), "formulaTemplateId")

	entity, err := tool.Draft(t.Context(), &params)
	require.NoError(t, err)
	assert.Equal(t, template, entity.FormulaTemplateID)
	require.True(t, entity.BaseRate.Valid)
	assert.True(t, entity.BaseRate.Decimal.Equal(decimal.RequireFromString("2.10")))

	named := createShipmentParams()
	_, err = tool.Draft(t.Context(), &named)
	require.NoError(t, err)
	assert.Equal(t, 1, contracts.calls, "a rating the call names is never second-guessed")
}

func TestCreateShipment_AsksForARatingWhenNoAgreementPricesTheLane(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{
		Shipments: &agenttoolstest.FakeShipmentWriter{},
		Contracts: &fakeContractRates{priced: &shipmentports.ContractRateApplication{
			Outcome: ratequote.OutcomeNoRateFound,
		}},
	}).(*operationstools.CreateShipmentTool)
	params := createShipmentParams()
	delete(params.Params["shipment"].(map[string]any), "formulaTemplateId")

	_, err := tool.Draft(t.Context(), &params)
	var refusal *errortypes.Error
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, "formulaTemplateId", refusal.Field)
	assert.Contains(t, err.Error(), "formula_list_templates")
}

func TestCreateShipment_TheSchemaRefusesWhatOnlyTheRaterOrTheSystemSets(t *testing.T) {
	t.Parallel()

	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: &agenttoolstest.FakeShipmentWriter{}})
	for field, value := range map[string]any{
		"rateLocked":          true,
		"rateOverrideAmount":  "0",
		"ratingDetail":        map[string]any{"source": "Rated"},
		"status":              "Completed",
		"freightChargeAmount": "1500.00",
		"rateAgreementId":     pulid.MustNew("ragr_").String(),
	} {
		payload := shipmentPayload(
			pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"),
			pulid.MustNew("loc_"),
		)
		payload[field] = value

		err := schemaCheck(tool, map[string]any{"shipment": payload})
		require.Error(t, err, field)
		assert.Contains(t, errorFields(t, err), "shipment."+field)
	}
}

func TestCreateShipment_ReplaysTheCallThatWasCreatedUnrated(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
	origin, dest := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	sent := map[string]any{"shipment": map[string]any{
		"customerId":     pulid.MustNew("cus_").String(),
		"serviceTypeId":  pulid.MustNew("svc_").String(),
		"shipmentTypeId": pulid.MustNew("sht_").String(),
		"bol":            "SEED-DET-009",
		"moves": []any{map[string]any{
			"type": "Linehaul",
			"stops": []any{
				map[string]any{
					"locationId": origin.String(), "type": "Pickup", "sequence": 1,
					"scheduledWindowStart": 1_790_000_000,
				},
				map[string]any{
					"locationId": dest.String(), "type": "Delivery", "sequence": 2,
					"scheduledWindowStart": 1_790_086_400,
				},
			},
		}},
	}}

	err := schemaCheck(tool, sent)
	require.Error(t, err)
	fields := errorFields(t, err)
	assert.Contains(t, fields, "shipment.moves[0].type", "a move has no type")
	assert.Contains(t, fields, "shipment.moves[0].stops[0].scheduledWindowStart",
		"a window is a local time, not Unix seconds")
	assert.Contains(t, fields, "shipment.moves[0].stops[1].scheduledWindowStart")

	fixed := sent["shipment"].(map[string]any)
	fixed["formulaTemplateId"] = pulid.MustNew("fmt_").String()
	move := fixed["moves"].([]any)[0].(map[string]any)
	delete(move, "type")
	stops := move["stops"].([]any)
	stops[0].(map[string]any)["scheduledWindowStart"] = "2026-10-01T08:00"
	stops[1].(map[string]any)["scheduledWindowStart"] = "2026-10-02T08:00"
	require.NoError(t, schemaCheck(tool, sent), "what is left fits the schema")

	err = tool.(agentports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(sent))
	require.Error(t, err, "stops numbered from 1 are refused before anything is filed")
	assert.Contains(t, err.Error(),
		"moves[0].stops[0].sequence: Sequences count from 0 in travel order, so this one is 0, not 1")
	assert.Contains(t, err.Error(), "moves[0].stops[1].sequence:")
	assert.Zero(t, writer.Previews, "the plan never ran on a misnumbered shipment")
}

func TestCreateShipment_NumbersStopsFromZeroInTravelOrder(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
	payload := shipmentPayload(
		pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
	)
	for _, stop := range payload["moves"].([]any)[0].(map[string]any)["stops"].([]any) {
		delete(stop.(map[string]any), "sequence")
	}
	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})
	params.IdempotencyKey = "idem-seq"

	require.NoError(t, tool.Execute(t.Context(), params))

	stops := writer.Created.Moves[0].Stops
	require.Len(t, stops, 2)
	assert.Equal(t, []int64{0, 1}, []int64{stops[0].Sequence, stops[1].Sequence})
	assert.Equal(t, shipment.StopTypePickup, stops[0].Type)
	assert.True(t, writer.Created.Moves[0].Loaded, "a move is loaded unless it says otherwise")
}

func TestCreateShipment_ReadsEachWindowWhereTheStopIs(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	origin, dest := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{
		Shipments: writer,
		Locations: agenttoolstest.FakeLocationZones{Zones: map[pulid.ID]string{origin: "America/Los_Angeles"}},
	})
	payload := shipmentPayload(pulid.MustNew("cus_"), pulid.MustNew("svc_"), origin, dest)
	stops := payload["moves"].([]any)[0].(map[string]any)["stops"].([]any)
	stops[0].(map[string]any)["scheduledWindowEnd"] = "2026-10-01T10:30"
	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})
	params.IdempotencyKey = "idem-zones"
	params.Timezone = "America/Chicago"

	require.NoError(t, tool.Execute(t.Context(), params))

	created := writer.Created.Moves[0].Stops
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Los_Angeles", 2026, 10, 1, 8, 0),
		created[0].ScheduledWindowStart, "the pickup's own zone")
	require.NotNil(t, created[0].ScheduledWindowEnd)
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Los_Angeles", 2026, 10, 1, 10, 30),
		*created[0].ScheduledWindowEnd)
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Chicago", 2026, 10, 2, 8, 0),
		created[1].ScheduledWindowStart, "a location with no zone is read in the organization's")
}

func TestCreateShipment_RefusesAWindowItCannotRead(t *testing.T) {
	t.Parallel()

	for name, window := range map[string]string{
		"an offset":        "2026-10-01T08:00:00-05:00",
		"unix seconds":     "1790000000",
		"a date alone":     "2026-10-01",
		"a skipped minute": "2026-03-08T02:30",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			writer := &agenttoolstest.FakeShipmentWriter{}
			tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
			payload := shipmentPayload(
				pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"),
				pulid.MustNew("loc_"),
			)
			stops := payload["moves"].([]any)[0].(map[string]any)["stops"].([]any)
			stops[0].(map[string]any)["scheduledWindowStart"] = window
			params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})
			params.IdempotencyKey = "idem-window"
			params.Timezone = "America/Chicago"

			err := tool.Execute(t.Context(), params)

			require.Error(t, err)
			assert.Equal(t, []string{"moves[0].stops[0].scheduledWindowStart"}, errorFields(t, err))
			assert.Nil(t, writer.Created)
		})
	}
}

func TestCreateShipment_EntersTheFreightChargesAndRating(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
	commodityID, accessorialID := pulid.MustNew("com_"), pulid.MustNew("acc_")
	billTo, template := pulid.MustNew("cus_"), pulid.MustNew("fmt_")
	payload := shipmentPayload(
		pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
	)
	payload["formulaTemplateId"] = template.String()
	payload["billToCustomerId"] = billTo.String()
	payload["baseRate"] = "2.45"
	payload["freightTerms"] = "Collect"
	payload["commodities"] = []any{map[string]any{
		"commodityId": commodityID.String(), "pieces": 26, "weight": 42000,
	}}
	payload["additionalCharges"] = []any{map[string]any{
		"accessorialChargeId": accessorialID.String(), "method": "Flat", "unit": 1,
		"amount": "125.00",
	}}
	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})
	params.IdempotencyKey = "idem-freight"

	require.NoError(t, tool.Execute(t.Context(), params))

	created := writer.Created
	assert.Equal(t, template, created.FormulaTemplateID)
	require.NotNil(t, created.BillToCustomerID)
	assert.Equal(t, billTo, *created.BillToCustomerID)
	require.True(t, created.BaseRate.Valid)
	assert.True(t, decimal.RequireFromString("2.45").Equal(created.BaseRate.Decimal))
	assert.Equal(t, shipment.FreightTermsCollect, created.FreightTerms)
	require.Len(t, created.Commodities, 1)
	assert.Equal(t, commodityID, created.Commodities[0].CommodityID)
	assert.EqualValues(t, 26, created.Commodities[0].Pieces)
	assert.EqualValues(t, 42000, created.Commodities[0].Weight)
	require.Len(t, created.AdditionalCharges, 1)
	charge := created.AdditionalCharges[0]
	assert.Equal(t, accessorialID, charge.AccessorialChargeID)
	assert.True(t, decimal.RequireFromString("125").Equal(charge.Amount))
	assert.EqualValues(t, 1, charge.Unit)
	assert.False(t, charge.IsSystemGenerated)
	assert.Equal(t, params.OrganizationID, charge.OrganizationID)
}

func TestCreateShipment_SaysWhatItCreated(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": shipmentPayload(
		pulid.MustNew("cus_"), pulid.MustNew("svc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
	)})
	params.IdempotencyKey = "idem-result"

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	id := writer.Created.ID.String()
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, "shipment", result.Kind)
	assert.Equal(t, writer.Created.ProNumber, result.Name)
	assert.Equal(t, map[string]string{"shipmentId": id}, result.IDs)
	assert.Equal(t, &agent.RecordRef{EntityType: "shipment", ID: id}, result.Record)
	assert.Contains(t, result.Describe(), `created the shipment "`+writer.Created.ProNumber+`"`)
	assert.Contains(t, result.IDNote(), "shipmentId "+id)
}

type ratedShipments struct {
	pricingShipments

	advisories []*errortypes.AdvisoryError
}

func (f *ratedShipments) PreviewCreate(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipmentports.ShipmentCreatePlan, error) {
	plan, err := f.pricingShipments.PreviewCreate(ctx, entity, actor)
	if err != nil {
		return nil, err
	}
	plan.Advisories = f.advisories

	return plan, nil
}

func TestCreateShipment_PreviewWarnsOfAShipmentNothingPrices(t *testing.T) {
	t.Parallel()

	shipments := &ratedShipments{advisories: []*errortypes.AdvisoryError{
		errortypes.NewAdvisory(
			"freightChargeAmount", errortypes.ErrInvalidOperation,
			"No rate agreement covers this lane, so this shipment is priced at zero",
			errortypes.SeverityRequireReview,
		).WithRuleKey(shipment.RateCoverageRuleKey),
		errortypes.NewAdvisory(
			"tractorTypeId", errortypes.ErrInvalidOperation, "Not a rate matter",
			errortypes.SeverityWarn,
		).WithRuleKey("capability_policy"),
	}}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: shipments}).(*operationstools.CreateShipmentTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), createShipmentParams())
	})

	require.Len(t, preview.Warnings, 1)
	warning := preview.Warnings[0]
	assert.Equal(t, agent.PreviewWarningRateCoverage, warning.Code)
	assert.Equal(t,
		[]string{"No rate agreement covers this lane, so this shipment is priced at zero"},
		warning.Args)
	assert.Nil(t, preview.Refusal(), "a flagged shipment can still be approved")
	require.NotNil(t, agenttoolstest.PreviewChange(t, preview, 0).Money, "the rating is shown as money")
}

func TestLocalTimeProperty_NeverAsksForUnixSeconds(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.LocalDateTime("When.")
	assert.Equal(t, toolschema.TypeString, property[toolschema.KeyType])
	assert.Contains(t, property[toolschema.KeyDescription], "2026-10-01T08:00")

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	seconds, err := common.RequireLocalTime(map[string]any{"at": "2026-10-01T08:00"}, "at", chicago)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 8, 0, 0, 0, chicago).Unix(), seconds)

	_, err = common.RequireLocalTime(map[string]any{}, "at", chicago)
	require.Error(t, err)

	absent, err := common.OptionalLocalTime(map[string]any{"at": ""}, "at", chicago)
	require.NoError(t, err)
	assert.Nil(t, absent)
}
