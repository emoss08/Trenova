package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestApprovalImpact_RatesEachShipmentAgainstPendingContent(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.shipmentRepo.On("ListRatedByFormulaTemplate", mock.Anything, mock.MatchedBy(
		func(req *repositories.ListRatedByFormulaTemplateRequest) bool {
			return req.TemplateID == template.ID && req.Limit == impactDefaultLimit
		},
	)).Return(newBacktestShipments(), nil)

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)

	// With no distinct effective version stored, live and pending content are
	// identical, so the impact of "approving" is zero across the board.
	for _, row := range result.Results {
		assert.True(t, decimal.NewFromInt(100).Equal(row.CurrentAmount))
		assert.True(t, decimal.NewFromInt(100).Equal(row.CandidateAmount))
		assert.True(t, row.Delta.IsZero())
		assert.Empty(t, row.CurrentError)
		assert.Empty(t, row.CandidateError)
	}

	assert.Equal(t, 2, result.Summary.ShipmentCount)
	assert.Equal(t, 0, result.Summary.ChangedCount)
	assert.Equal(t, 0, result.Summary.ErrorCount)
}

func TestApprovalImpact_RejectsExcessiveLimit(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: newTestTemplate().ID,
		Limit:      backtestMaxLimit + 1,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
}

func TestSortResultsByImpact_BiggestMoversFirstErrorsLast(t *testing.T) {
	t.Parallel()

	results := []*BacktestResult{
		{ProNumber: "SMALL", Delta: decimal.NewFromInt(5)},
		{ProNumber: "FAILED", CandidateError: "boom", Delta: decimal.NewFromInt(999)},
		{ProNumber: "BIG-DECREASE", Delta: decimal.NewFromInt(-50)},
		{ProNumber: "MEDIUM", Delta: decimal.NewFromInt(10)},
	}

	sortResultsByImpact(results)

	order := make([]string, 0, len(results))
	for _, result := range results {
		order = append(order, result.ProNumber)
	}

	assert.Equal(t, []string{"BIG-DECREASE", "MEDIUM", "SMALL", "FAILED"}, order)
}
