package agentquerytoolservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRateAgreements struct {
	agreement *rateagreement.RateAgreement
	listed    *repositories.ListRateAgreementRequest
}

func (f *fakeRateAgreements) List(
	_ context.Context,
	req *repositories.ListRateAgreementRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	f.listed = req

	return &pagination.ListResult[*rateagreement.RateAgreement]{
		Items: []*rateagreement.RateAgreement{f.agreement},
	}, nil
}

func (f *fakeRateAgreements) GetByID(
	context.Context,
	*repositories.GetRateAgreementByIDRequest,
) (*rateagreement.RateAgreement, error) {
	return f.agreement, nil
}

func (f *fakeRateAgreements) ListVersions(
	context.Context,
	*repositories.ListRateAgreementVersionsRequest,
) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
	return &pagination.ListResult[*rateagreement.RateAgreementVersion]{
		Items: []*rateagreement.RateAgreementVersion{{
			VersionNumber: 3,
			ChangeMessage: "Raised Dallas rates 4 percent",
		}},
	}, nil
}

func rateAgreementWithRules(count int) *rateagreement.RateAgreement {
	rules := make([]*rateagreement.RateAgreementRule, 0, count)
	for idx := range count {
		rules = append(rules, &rateagreement.RateAgreementRule{
			ID:               pulid.MustNew("ragr_"),
			OriginScopeType:  "State",
			OriginScopeValue: "TX",
			Rate:             decimal.NewNullDecimal(decimal.NewFromInt(int64(idx + 1))),
			MinCharge:        decimal.NewNullDecimal(decimal.NewFromInt(150)),
		})
	}

	return &rateagreement.RateAgreement{
		ID:                 pulid.MustNew("rag_"),
		Code:               "ACME-2026",
		PartyType:          rateagreement.PartyTypeCustomer,
		Status:             rateagreement.StatusActive,
		MarginFloorPercent: decimal.NewNullDecimal(decimal.NewFromInt(12)),
		Rules:              rules,
	}
}

func TestGetRateAgreement_BoundsTheRulesAndWithholdsRates(t *testing.T) {
	t.Parallel()

	agreement := rateAgreementWithRules(maxAgreementRules + 7)
	tool := newGetRateAgreementTool(&fakeRateAgreements{agreement: agreement}, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"rateAgreementId": agreement.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(rateAgreementView)
	assert.Len(t, view.Rules, maxAgreementRules)
	assert.Equal(t, 7, view.RulesOmitted)
	assert.Equal(t, "State TX", view.Rules[0].Origin)
	assert.Empty(t, view.Rules[0].Rate)
	assert.Empty(t, view.Rules[0].MinCharge)
	require.NotNil(t, view.CurrentVersion)
	assert.Equal(t, int64(3), view.CurrentVersion.VersionNumber)
	assert.Empty(t, view.CurrentVersion.ChangeMessage)
	assert.Subset(t, view.Withheld, []string{"rules.rate", "rules.minCharge", "changeMessage"})
	assert.NotContains(t, view.Withheld, "marginFloorPercent",
		"a confidential term is never sent and never named")
}

func TestGetRateAgreement_ShowsRatesAtRestricted(t *testing.T) {
	t.Parallel()

	agreement := rateAgreementWithRules(1)
	tool := newGetRateAgreementTool(&fakeRateAgreements{agreement: agreement}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"rateAgreementId": agreement.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(rateAgreementView)
	assert.Equal(t, "1", view.Rules[0].Rate)
	assert.Equal(t, "150", view.Rules[0].MinCharge)
	assert.Equal(t, "Raised Dallas rates 4 percent", view.CurrentVersion.ChangeMessage)
}

func TestListRateAgreements_NarrowsToTheCustomer(t *testing.T) {
	t.Parallel()

	fake := &fakeRateAgreements{agreement: rateAgreementWithRules(0)}
	tool := newListRateAgreementsTool(fake, &fakePermissions{})
	customer := pulid.MustNew("cus_")

	_, err := tool.Query(t.Context(), agentParams(map[string]any{
		"customerId": customer.String(),
		"status":     "Active",
	}, ""))
	require.NoError(t, err)

	require.NotNil(t, fake.listed.CustomerID)
	assert.Equal(t, customer, *fake.listed.CustomerID)
	assert.Equal(t, rateagreement.StatusActive, fake.listed.Status)
}

type fakeRateMatrices struct {
	matrix *ratematrix.RateMatrix
	cells  []*ratematrix.RateMatrixCell
	pages  int
}

func (f *fakeRateMatrices) GetByID(
	context.Context,
	*repositories.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return f.matrix, nil
}

func (f *fakeRateMatrices) ListCells(
	_ context.Context,
	req *repositories.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	f.pages++
	offset := min(req.Filter.Pagination.Offset, len(f.cells))
	end := min(offset+req.Filter.Pagination.Limit, len(f.cells))

	return &pagination.ListResult[*ratematrix.RateMatrixCell]{
		Items: f.cells[offset:end],
		Total: len(f.cells),
	}, nil
}

func TestGetRateMatrix_ReturnsAtMostTwoHundredCells(t *testing.T) {
	t.Parallel()

	cells := make([]*ratematrix.RateMatrixCell, 0, 450)
	for idx := range 450 {
		cells = append(cells, &ratematrix.RateMatrixCell{
			D0Key: fmt.Sprintf("Z%d", idx),
			D1Min: decimal.NewNullDecimal(decimal.NewFromInt(0)),
			D1Max: decimal.NewNullDecimal(decimal.NewFromInt(500)),
			Value: decimal.RequireFromString("2.15"),
		})
	}
	fake := &fakeRateMatrices{
		matrix: &ratematrix.RateMatrix{
			ID:   pulid.MustNew("rmx_"),
			Code: "LTL-70",
			Dimensions: []*ratematrix.RateMatrixDimension{
				{Position: 0, Kind: ratematrix.DimensionKindZone},
				{Position: 1, Kind: ratematrix.DimensionKindWeightBreak},
			},
		},
		cells: cells,
	}
	tool := newGetRateMatrixTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"rateMatrixId": fake.matrix.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(rateMatrixView)
	assert.Len(t, view.Cells, maxMatrixCells)
	assert.Equal(t, 450, view.CellCount)
	assert.True(t, view.CellsHasMore)
	require.NotNil(t, view.NextCellOffset)
	assert.Equal(t, maxMatrixCells, *view.NextCellOffset)
	assert.Equal(t, []string{"Z0", ""}, view.Cells[0].Keys)
	assert.Equal(t, []string{"", "0 to 500"}, view.Cells[0].Ranges)
	assert.Equal(t, "2.15", view.Cells[0].Value)

	result, err = tool.Query(t.Context(), agentParams(map[string]any{
		"rateMatrixId": fake.matrix.ID.String(),
		"cellOffset":   float64(400),
	}, ""))
	require.NoError(t, err)
	last := result.(rateMatrixView)
	assert.Len(t, last.Cells, 50)
	assert.False(t, last.CellsHasMore)
	assert.Empty(t, last.Cells[0].Value)
	assert.Contains(t, last.Withheld, "cells.value")
}

type fakeFuelRates struct {
	rates []*fuelsurchargeservice.ProgramCurrentRate
}

func (f *fakeFuelRates) ProgramCurrentRates(
	context.Context,
	pagination.TenantInfo,
) ([]*fuelsurchargeservice.ProgramCurrentRate, error) {
	return f.rates, nil
}

func TestGetFuelSurchargeRates_SaysWhenNoPriceIsOnFile(t *testing.T) {
	t.Parallel()

	perMile := decimal.RequireFromString("0.42")
	tool := newGetFuelSurchargeRatesTool(&fakeFuelRates{
		rates: []*fuelsurchargeservice.ProgramCurrentRate{
			{
				Program: &fuelsurcharge.FuelSurchargeProgram{
					ID: pulid.MustNew("fsp_"), Code: "DOE", Name: "DOE weekly",
				},
				Price: &fuelsurcharge.FuelIndexPrice{
					PriceDate: "2026-09-21",
					Price:     decimal.RequireFromString("3.899"),
				},
				RatePerMile: &perMile,
			},
			{
				Program: &fuelsurcharge.FuelSurchargeProgram{
					ID: pulid.MustNew("fsp_"), Code: "CA", Name: "California",
				},
			},
		},
	})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)

	rows := result.(searchOutcome).Items.([]fuelRateRow)
	require.Len(t, rows, 2)
	assert.Equal(t, "0.42", rows[0].RatePerMile)
	assert.Equal(t, "3.899", rows[0].IndexPrice)
	assert.NotEmpty(t, rows[1].Note)

	result, err = tool.Query(t.Context(), agentParams(map[string]any{"query": "calif"}, ""))
	require.NoError(t, err)
	assert.Equal(t, 1, result.(searchOutcome).Count)
}
