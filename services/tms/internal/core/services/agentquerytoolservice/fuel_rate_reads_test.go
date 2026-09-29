package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFuelPurchases struct {
	repositories.FuelPurchaseRepository

	purchases []*fuelpurchase.FuelPurchase
	captured  *repositories.ListFuelPurchasesRequest
}

func (f *fakeFuelPurchases) ListPurchases(
	_ context.Context,
	req *repositories.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	f.captured = req

	return pagination.NewCursorListResult(f.purchases, len(f.purchases)+1), nil
}

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

func TestListFuelPurchases_ReadsTheTractorAndJurisdictionItNames(t *testing.T) {
	t.Parallel()

	jurisdiction := &ifta.Jurisdiction{ID: pulid.MustNew("ifj_"), Code: "TX"}
	unit := &tractor.Tractor{ID: pulid.MustNew("trac_"), Code: "TRC-001"}
	repo := &fakeFuelPurchases{purchases: []*fuelpurchase.FuelPurchase{{
		ID:               pulid.MustNew("fpur_"),
		TractorID:        unit.ID,
		Tractor:          unit,
		JurisdictionID:   jurisdiction.ID,
		Jurisdiction:     jurisdiction,
		PurchasedAt:      1_784_000_000,
		FuelType:         domaintypes.IFTAFuelTypeDiesel,
		Quantity:         decimal.RequireFromString("100"),
		QuantityUnit:     fuelpurchase.QuantityUnitGallon,
		Gallons:          decimal.RequireFromString("100"),
		TotalAmountMinor: 38_990,
		CurrencyCode:     "USD",
		TaxPaid:          true,
		Source:           fuelpurchase.PurchaseSourceManual,
	}}}
	tool := newListFuelPurchasesTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)
	assert.True(t, repo.captured.IncludeTractor)
	assert.True(t, repo.captured.IncludeJurisdiction)

	rows := encodedRows(t, result)
	require.Len(t, rows, 1)
	assert.Equal(t, "TRC-001", rows[0]["tractorCode"])
	assert.Equal(t, "TX", rows[0]["jurisdiction"])
	assert.Equal(t, "389.90", rows[0]["totalAmount"])
	assert.Equal(t, permission.ResourceFuelPurchase, tool.Policy().Resource)
}

type fakeFuelIndexPrices struct {
	repositories.FuelIndexPriceRepository

	captured *repositories.ListFuelIndexPricesRequest
}

func (f *fakeFuelIndexPrices) ListByIndex(
	_ context.Context,
	req *repositories.ListFuelIndexPricesRequest,
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

type fakeIFTAJurisdictions struct {
	repositories.IFTARepository

	captured *repositories.ListJurisdictionsRequest
}

func (f *fakeIFTAJurisdictions) ListJurisdictions(
	_ context.Context,
	req *repositories.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	f.captured = req

	return []*ifta.Jurisdiction{{
		ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "TX", Name: "Texas",
		IsIftaMember: true,
	}}, nil
}

func TestListIFTAJurisdictions_OnlyActiveOnes(t *testing.T) {
	t.Parallel()

	repo := &fakeIFTAJurisdictions{}
	tool := newListIFTAJurisdictionsTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"countryCode": " us "}))
	require.NoError(t, err)
	assert.Equal(t, "US", repo.captured.CountryCode)
	assert.Equal(t, []ifta.JurisdictionStatus{ifta.JurisdictionStatusActive},
		repo.captured.Statuses)
	rows := result.(map[string]any)["jurisdictions"].([]iftaJurisdictionRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "TX", rows[0].Code)
}

type fakeRateImports struct {
	repositories.RateImportRepository

	captured *repositories.ListRateImportBatchesRequest
}

func (f *fakeRateImports) List(
	_ context.Context,
	req *repositories.ListRateImportBatchesRequest,
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
