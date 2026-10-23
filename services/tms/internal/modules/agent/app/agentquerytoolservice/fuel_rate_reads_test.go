package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateimportports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodedRows(t *testing.T, result any) []map[string]any {
	t.Helper()

	raw, err := sonic.Marshal(result)
	require.NoError(t, err)
	var decoded struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, sonic.Unmarshal(raw, &decoded))

	return decoded.Items
}

type fakeFuelIndexPrices struct {
	fuelsurchargeports.FuelIndexPriceRepository

	captured *fuelsurchargeports.ListFuelIndexPricesRequest
}

func (f *fakeFuelIndexPrices) ListByIndex(
	_ context.Context,
	req *fuelsurchargeports.ListFuelIndexPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	f.captured = req

	return []*fuelsurcharge.FuelIndexPrice{{
		ID:        pulid.MustNew("fip_"),
		PriceDate: "2026-09-21",
		Price:     decimal.RequireFromString("3.799"),
		Currency:  "USD",
		IsManual:  true,
	}}, nil
}

func TestListFuelIndexPrices_BoundsTheWindowAndTheCount(t *testing.T) {
	t.Parallel()

	prices := &fakeFuelIndexPrices{}
	tool := newListFuelIndexPricesTool(prices)
	indexID := pulid.MustNew("fidx_")

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		paramFuelIndexID: indexID.String(),
		"from":           "last month",
	}))
	require.Error(t, err)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramFuelIndexID: indexID.String(),
		"from":           "2026-09-01",
		"limit":          float64(5_000),
	}))
	require.NoError(t, err)
	assert.Equal(t, maxFuelIndexPrices, prices.captured.Limit)
	assert.Equal(t, "2026-09-01", prices.captured.From)
	rows := result.(map[string]any)["prices"].([]fuelIndexPriceRow)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].IsManual)
}

type fakeRateImports struct {
	rateimportports.RateImportRepository

	captured *rateimportports.ListRateImportBatchesRequest
}

func (f *fakeRateImports) List(
	_ context.Context,
	req *rateimportports.ListRateImportBatchesRequest,
) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
	f.captured = req

	return &pagination.ListResult[*rateimport.RateImportBatch]{
		Items: []*rateimport.RateImportBatch{{
			ID:              pulid.MustNew("rib_"),
			RateAgreementID: pulid.MustNew("rag_"),
			FileName:        "acme-2027.csv",
			Status:          rateimport.StatusParsed,
			Summary:         &pkgrateimport.Summary{Added: 2, Changed: 5},
		}},
		Total: 1,
	}, nil
}

func TestListRateImports_NarrowsToOneAgreement(t *testing.T) {
	t.Parallel()

	repo := &fakeRateImports{}
	tool := newListRateImportsTool(repo)
	agreementID := pulid.MustNew("rag_")

	_, err := tool.Query(t.Context(), testParams(map[string]any{"rateAgreementId": "acme"}))
	require.Error(t, err)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"rateAgreementId": agreementID.String(),
	}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured.RateAgreementID)
	assert.Equal(t, agreementID, *repo.captured.RateAgreementID)
	rows := result.(map[string]any)["imports"].([]rateImportRow)
	require.Len(t, rows, 1)
	assert.Equal(t, 5, rows[0].Changed)
}

type fakeReportSchedules struct {
	schedules []*report.ReportSchedule
	captured  *reporting.ListSchedulesRequest
}

func (f *fakeReportSchedules) ListSchedules(
	_ context.Context,
	req *reporting.ListSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	f.captured = req

	return f.schedules, nil
}

func TestListReportSchedules_SaysWhichAreTheCallers(t *testing.T) {
	t.Parallel()

	params := testParams(map[string]any{})
	schedules := &fakeReportSchedules{schedules: []*report.ReportSchedule{
		{ID: pulid.MustNew("rsch_"), RunAsID: params.Actor.UserID, Enabled: true},
		{ID: pulid.MustNew("rsch_"), RunAsID: pulid.MustNew("usr_")},
	}}
	tool := &listReportSchedulesTool{schedules: schedules}

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	rows := result.(map[string]any)["schedules"].([]reportScheduleRow)
	require.Len(t, rows, 2)
	assert.True(t, rows[0].Yours)
	assert.False(t, rows[1].Yours)
	assert.Equal(t, maxReportSchedules, schedules.captured.Limit)
}
