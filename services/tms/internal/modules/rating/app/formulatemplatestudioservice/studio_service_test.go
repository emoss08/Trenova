package formulatemplatestudioservice

import (
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreview_ASourceThatDoesNotCheckIsNotRated(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.BrokenSource,
	})
	require.NoError(t, err)
	assert.False(t, result.Valid)
	require.NotEmpty(t, result.Diagnostics)
	assert.Equal(t, rules.SeverityError, result.Diagnostics[0].Severity)
	assert.Nil(t, result.Bill)
	assert.Empty(t, deps.rater.Requests)
}

func TestPreview_RatesTheSampleRecordAndRunsTheTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.rater.Result = &tariffports.RateResult{
		Bill: &tarifftypes.Bill{Total: decimal.NewFromInt(285)},
		WireBill: &rules.TariffBill{
			Lines:            []rules.TariffLine{},
			FreightTotal:     rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
			AccessorialTotal: rules.TariffQuantity{Value: decimal.Zero, Unit: "$"},
			Total:            rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
		},
		Trace: rules.RawJSON(`[{"kind":"charge"}]`),
	}
	inputs := tarifftypes.Inputs{
		"rate": rules.QuantityValue(decimal.RequireFromString("2.85"), "$/mi"),
	}

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
		Inputs:     inputs,
	})
	require.NoError(t, err)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Bill)
	assert.Equal(t, "285", result.Bill.Total.Value.String())
	assert.Equal(t, "$", result.Bill.Total.Unit)
	require.NotNil(t, result.RatedBill)
	assert.Equal(t, "285", result.RatedBill.Total.String())
	wire, err := sonic.Marshal(result.Bill)
	require.NoError(t, err)
	var sent map[string]any
	require.NoError(t, sonic.Unmarshal(wire, &sent))
	assert.Equal(t, map[string]any{"$q": "285", "u": "$"}, sent["total"],
		"the preview sends the engine's bill, totals as quantities with their unit")
	assert.JSONEq(t, `[{"kind":"charge"}]`, string(result.Trace))
	require.Len(t, result.Tests, 1)
	assert.Equal(t, rules.TariffTestPassed, result.Tests[0].Status)
	require.Len(t, result.Inputs, 2)

	require.Len(t, deps.rater.Requests, 1)
	request := deps.rater.Requests[0]
	assert.Equal(t, formulatemplatetest.PerMileSource, request.Template.Source)
	assert.True(t, request.Trace)
	assert.Equal(t, inputs, request.Inputs)
	assert.Empty(t, deps.records.Requests, "no shipment rates the sample record")
}

func TestPreview_RatesARealShipmentAndReportsARefusal(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.records.Record = rules.RawJSON(`{"shipment":{}}`)
	deps.rater.Result = &tariffports.RateResult{
		Refusal: &rules.TariffRefusal{Text: "This rate needs the trip's miles"},
	}
	shipmentID := pulid.MustNew("shp_")

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
		ShipmentID: &shipmentID,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Refusal)
	assert.Equal(t, "This rate needs the trip's miles", result.Refusal.Text)
	require.Len(t, deps.records.Requests, 1)
	assert.Equal(t, shipmentID, deps.records.Requests[0].ShipmentID)
	assert.JSONEq(t, `{"shipment":{}}`, string(deps.rater.Requests[0].Record))
}

func TestPreview_AModuleErrorIsReportedNotReturned(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.rater.Err = errortypes.NewBusinessError("A weight is needed").WithParam("code", "none")

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Error)
	assert.Equal(t, "none", result.Error.Code)

	deps.rater.Err = errors.New("the database is gone")
	_, err = deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
	})
	require.Error(t, err, "an infrastructure failure is not a preview result")
}

func TestScaffoldTest_NeedsANameAndAShipment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.ScaffoldTest(t.Context(), &ScaffoldTestRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
		ShipmentID: pulid.MustNew("shp_"),
	})
	assert.Contains(t, formulatemplatetest.ValidationFields(t, err), "name")

	_, err = deps.svc.ScaffoldTest(t.Context(), &ScaffoldTestRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
		Name:       "from shipment 1042",
	})
	assert.Contains(t, formulatemplatetest.ValidationFields(t, err), "shipmentId")
	assert.Empty(t, deps.records.Requests)
}

func TestTestSource_RunsTheFilesTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	passing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
	})
	require.NoError(t, err)
	assert.True(t, passing.Check.OK)
	assert.True(t, passing.Tests.Passed())
	assert.Len(t, passing.Inputs, 2)

	failing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileFailingSource,
	})
	require.NoError(t, err)
	assert.False(t, failing.Tests.Passed())
	require.Len(t, failing.Tests.Tests, 1)
	assert.Equal(t, rules.TariffTestFailed, failing.Tests.Tests[0].Status)
}

func TestBacktest_TakesASourceOrAVersionNotBoth(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		TemplateID: pulid.MustNew("ft_"),
	})
	assert.Contains(
		t,
		formulatemplatetest.ValidationFields(t, err),
		formulatemplateservice.FieldSource,
	)

	version := int64(2)
	_, err = deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo:    formulatemplatetest.NewTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		Source:        formulatemplatetest.PerMileSource,
		VersionNumber: &version,
	})
	assert.Contains(
		t,
		formulatemplatetest.ValidationFields(t, err),
		formulatemplateservice.FieldSource,
	)
}
