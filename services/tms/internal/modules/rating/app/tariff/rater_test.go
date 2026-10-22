package tariff_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const laneSource = `tariff "Lane rate"

let rate = tables.LANES[lane.origin.state] ?? fail "No rate from {lane.origin.state}"
charge "Linehaul" = rate
`

type lanesContext struct {
	cells      map[string]decimal.Decimal
	loadCells  int
	loadTables int
	ignore     bool
}

func (c *lanesContext) table() rules.TariffTable {
	return rules.TariffTable{
		Code:      "LANES",
		Axes:      []rules.TariffAxis{{Match: "Exact", Kind: "State"}},
		ValueType: "Money",
		Cells:     []rules.TariffCell{},
		Loaded:    [][]string{},
	}
}

func (c *lanesContext) Context(
	_ context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Tables:    []rules.TariffTable{c.table()},
	}, nil
}

func (c *lanesContext) LoadCells(
	_ context.Context,
	_ pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	needs []rules.TariffNeed,
) error {
	c.loadCells++
	if c.ignore {
		return nil
	}
	table := &tariffCtx.Tables[0]
	for _, need := range needs {
		table.Loaded = append(table.Loaded, need.Keys)
		if value, ok := c.cells[need.Keys[0]]; ok {
			table.Cells = append(table.Cells, rules.TariffCell{
				At:    []rules.TariffCellPosition{{Key: need.Keys[0]}},
				Value: value,
			})
		}
	}
	return nil
}

func (c *lanesContext) LoadTables(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]string,
) error {
	c.loadTables++
	return nil
}

type counter struct{ calls int }

func (c *counter) evaluate(
	ctx context.Context,
	req *rules.TariffEvaluateRequest,
) (*rules.TariffEvaluation, error) {
	c.calls++
	return rules.EvaluateTariffs(ctx, req)
}

func recordFrom(t *testing.T, state string) rules.RawJSON {
	t.Helper()

	entity := &shipment.Shipment{
		Status: shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{{
			Loaded:   true,
			Distance: new(100.0),
			Stops: []*shipment.Stop{
				{Type: shipment.StopTypePickup, Status: shipment.StopStatusNew},
				{Type: shipment.StopTypeDelivery, Status: shipment.StopStatusNew},
			},
		}},
	}
	raw, err := record.Build(&record.Input{
		Shipment:     entity,
		Origin:       record.Place{State: state, Country: "USA"},
		DistanceUnit: record.DistanceMiles,
		Rating: record.Rating{
			At:       time.Unix(1_772_000_000, 0).UTC(),
			Mode:     record.ModeSell,
			Currency: "USD",
		},
	})
	require.NoError(t, err)
	return raw
}

func template(source string, inputs ...tarifftypes.InputDecl) *tariffports.Template {
	return &tariffports.Template{
		TemplateID: pulid.MustNew("ft_"),
		Name:       "Lane rate",
		Source:     source,
		Inputs:     inputs,
	}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestRateLoadsTheCellsTheTariffNeedsAndEvaluatesAgain(t *testing.T) {
	t.Parallel()

	contexts := &lanesContext{cells: map[string]decimal.Decimal{"TX": decimal.NewFromInt(1450)}}
	calls := &counter{}
	rater := tariff.NewRaterWith(zap.NewNop(), contexts, calls.evaluate)

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   template(laneSource),
		Record:     recordFrom(t, "TX"),
		Currency:   "USD",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Bill)
	assert.True(t, result.Bill.Total.Equal(decimal.NewFromInt(1450)), "total %s", result.Bill.Total)
	assert.Equal(t, 1, contexts.loadCells)
	assert.Equal(t, 2, calls.calls, "one call that asks for the cell, one that rates")
}

func TestRateReturnsTheTariffsRefusal(t *testing.T) {
	t.Parallel()

	contexts := &lanesContext{cells: map[string]decimal.Decimal{}}
	rater := tariff.NewRaterWith(zap.NewNop(), contexts, rules.EvaluateTariffs)

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   template(laneSource),
		Record:     recordFrom(t, "CA"),
		Currency:   "USD",
	})

	require.NoError(t, err)
	assert.Nil(t, result.Bill)
	require.NotNil(t, result.Refusal)
	assert.Equal(t, "No rate from CA", result.Refusal.Text)
}

func TestRateGivesUpWhenCellsNeverArrive(t *testing.T) {
	t.Parallel()

	contexts := &lanesContext{ignore: true}
	calls := &counter{}
	rater := tariff.NewRaterWith(zap.NewNop(), contexts, calls.evaluate)

	_, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   template(laneSource),
		Record:     recordFrom(t, "TX"),
		Currency:   "USD",
	})

	require.ErrorIs(t, err, tariff.ErrUnresolvedNeeds)
	assert.Equal(t, 3, contexts.loadCells)
	assert.Equal(t, 1, contexts.loadTables, "the last pass sends the tables whole")
	assert.Equal(t, 5, calls.calls)
}

func TestRateEvaluatesConditionsWithTheTariffInOneCall(t *testing.T) {
	t.Parallel()

	calls := &counter{}
	rater := tariff.NewRaterWith(zap.NewNop(), &lanesContext{}, calls.evaluate)

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   template("tariff \"Flat\"\n\ncharge \"Freight\" = $100\n"),
		Conditions: []tariffports.Condition{
			{ID: "texas", Source: "condition \"Texas\"\n\nlane.origin.state == \"TX\"\n"},
			{ID: "ohio", Source: "condition \"Ohio\"\n\nlane.origin.state == \"OH\"\n"},
			{ID: "broken", Source: "condition \"Broken\"\n\nnot a condition\n"},
		},
		Record:   recordFrom(t, "TX"),
		Currency: "USD",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls.calls)
	assert.True(t, result.Conditions["texas"])
	assert.False(t, result.Conditions["ohio"])
	assert.False(
		t,
		result.Conditions["broken"],
		"a condition that does not evaluate withholds its charge",
	)
	assert.True(t, result.Bill.Total.Equal(decimal.NewFromInt(100)))
}

func TestRateOnlyPassesTheInputsTheTemplateDeclares(t *testing.T) {
	t.Parallel()

	var seen map[string]rules.RawJSON
	rater := tariff.NewRaterWith(zap.NewNop(), &lanesContext{}, func(
		ctx context.Context,
		req *rules.TariffEvaluateRequest,
	) (*rules.TariffEvaluation, error) {
		seen = req.Programs[0].Inputs
		return rules.EvaluateTariffs(ctx, req)
	})

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template: template(
			"tariff \"Flat\"\n\ninput rate: Money\n\ncharge \"Freight\" = rate\n",
			tarifftypes.InputDecl{
				Name:       "rate",
				Descriptor: rules.RawJSON(`{"kind":"money"}`),
				Required:   true,
			},
		),
		Inputs: tarifftypes.Inputs{
			"rate":  rules.MoneyValue(decimal.NewFromInt(640)),
			"stray": rules.NumberValue(decimal.NewFromInt(1)),
		},
		Record:   recordFrom(t, "TX"),
		Currency: "USD",
	})

	require.NoError(t, err)
	assert.Contains(t, seen, "rate")
	assert.NotContains(t, seen, "stray")
	assert.True(t, result.Bill.Total.Equal(decimal.NewFromInt(640)))
}

func flatWithRate() *tariffports.Template {
	return template(
		"tariff \"Flat\"\n\ninput rate: Money\n\ncharge \"Freight\" = rate\n",
		tarifftypes.InputDecl{
			Name:       "rate",
			Descriptor: rules.RawJSON(`{"kind":"money"}`),
			Required:   true,
		},
	)
}

func TestRateRefusesWhenARequiredInputIsMissing(t *testing.T) {
	t.Parallel()

	rater := tariff.NewRaterWith(zap.NewNop(), &lanesContext{}, rules.EvaluateTariffs)

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   flatWithRate(),
		Record:     recordFrom(t, "TX"),
		Currency:   "USD",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Refusal)
	assert.Contains(t, result.Refusal.Localize("en"), "rate")
}

func TestRateReportsAMistypedInputAgainstTheInputs(t *testing.T) {
	t.Parallel()

	rater := tariff.NewRaterWith(zap.NewNop(), &lanesContext{}, rules.EvaluateTariffs)

	_, err := rater.Rate(t.Context(), &tariffports.RateRequest{
		TenantInfo: tenant(),
		Template:   flatWithRate(),
		Inputs:     tarifftypes.Inputs{"rate": rules.QuantityValue(decimal.NewFromInt(2), "$/mi")},
		Record:     recordFrom(t, "TX"),
		Currency:   "USD",
	})

	require.Error(t, err)
	var validation *errortypes.Error
	if errortypes.IsError(err) {
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "inputs", validation.Field)
		return
	}
	assert.True(t, errortypes.IsBusinessError(err), "%v", err)
}

func TestRateWithNothingToEvaluateMakesNoCall(t *testing.T) {
	t.Parallel()

	calls := &counter{}
	rater := tariff.NewRaterWith(zap.NewNop(), &lanesContext{}, calls.evaluate)

	result, err := rater.Rate(t.Context(), &tariffports.RateRequest{TenantInfo: tenant()})

	require.NoError(t, err)
	assert.Zero(t, calls.calls)
	assert.Empty(t, result.Conditions)
}
