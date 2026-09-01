package formulatemplateservice

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newApprovedSnapshot(
	template *formulatemplate.FormulaTemplate,
	number int64,
	expression string,
) *formulatemplate.FormulaTemplateVersion {
	return &formulatemplate.FormulaTemplateVersion{
		ID:                   pulid.MustNew("ftv_"),
		TemplateID:           template.ID,
		OrganizationID:       template.OrganizationID,
		BusinessUnitID:       template.BusinessUnitID,
		VersionNumber:        number,
		Name:                 template.Name,
		Type:                 template.Type,
		Expression:           expression,
		Status:               formulatemplate.StatusActive,
		SchemaID:             template.SchemaID,
		VariableDefinitions:  template.VariableDefinitions,
		BreakdownDefinitions: template.BreakdownDefinitions,
	}
}

func stampRatedVersion(shipments []*shipment.Shipment, templateID pulid.ID, number int64) {
	for _, entity := range shipments {
		entity.RatingDetail = &shipment.RatingDetail{
			FormulaTemplateID: templateID.String(),
			VersionNumber:     number,
			Result:            100,
		}
	}
}

func expectRatedShipments(
	deps *testDeps,
	template *formulatemplate.FormulaTemplate,
	shipments []*shipment.Shipment,
) {
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.shipmentRepo.On("ListRatedByFormulaTemplate", mock.Anything, mock.MatchedBy(
		func(req *repositories.ListRatedByFormulaTemplateRequest) bool {
			return req.TemplateID == template.ID && req.Limit == impactDefaultLimit
		},
	)).Return(shipments, nil)
}

func TestApprovalImpact_UsesVersionRecordedOnEachShipment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "baseRate * 2"
	template.CurrentVersionNumber = 2

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	expectRatedShipments(deps, template, shipments)

	// Two shipments stamped with the same version resolve it once.
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.MatchedBy(
		func(req *repositories.GetVersionRequest) bool {
			return req.TemplateID == template.ID && req.VersionNumber == 1
		},
	)).Return(newApprovedSnapshot(template, 1, "baseRate"), nil).Once()

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	for _, row := range result.Results {
		assert.True(t, decimal.NewFromInt(100).Equal(row.CurrentAmount), row.CurrentAmount)
		assert.True(t, decimal.NewFromInt(200).Equal(row.CandidateAmount), row.CandidateAmount)
		assert.True(t, decimal.NewFromInt(100).Equal(row.Delta), row.Delta)
	}
	assert.Equal(t, 2, result.Summary.ChangedCount)
	assert.True(t, decimal.NewFromInt(200).Equal(result.Summary.TotalDelta))
	deps.versionRepo.AssertNotCalled(t, "GetLatestByStatus", mock.Anything, mock.Anything)
}

func TestApprovalImpact_FallsBackToLastActiveSnapshot(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "baseRate * 2"

	expectRatedShipments(deps, template, newBacktestShipments())
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.MatchedBy(
		func(req *repositories.GetLatestVersionByStatusRequest) bool {
			return req.TemplateID == template.ID && req.Status == formulatemplate.StatusActive
		},
	)).Return(newApprovedSnapshot(template, 1, "baseRate"), nil).Once()

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	for _, row := range result.Results {
		assert.True(t, decimal.NewFromInt(100).Equal(row.CurrentAmount))
		assert.True(t, decimal.NewFromInt(200).Equal(row.CandidateAmount))
	}
	assert.Equal(t, 2, result.Summary.ChangedCount)
}

func TestApprovalImpact_StaleVersionStampDegradesToLastActive(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "baseRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 9)
	expectRatedShipments(deps, template, shipments)

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows).Once()
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(newApprovedSnapshot(template, 1, "baseRate"), nil).Once()

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	for _, row := range result.Results {
		assert.Empty(t, row.CurrentError)
		assert.True(t, decimal.NewFromInt(100).Equal(row.Delta))
	}
}

func TestApprovalImpact_FallsBackToRowWithoutApprovedHistory(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	expectRatedShipments(deps, template, newBacktestShipments())
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil).Once()

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)

	// A template that was never approved has nothing older to compare against,
	// so the row is both sides and the impact is honestly zero.
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

func TestBacktest_CurrentSideUsesRecordedVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "baseRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.shipmentRepo.On("ListRatedByFormulaTemplate", mock.Anything, mock.Anything).
		Return(shipments, nil)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(newApprovedSnapshot(template, 1, "baseRate"), nil).Once()

	result, err := deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Expression: "baseRate * 3",
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	for _, row := range result.Results {
		assert.True(t, decimal.NewFromInt(100).Equal(row.CurrentAmount), row.CurrentAmount)
		assert.True(t, decimal.NewFromInt(300).Equal(row.CandidateAmount), row.CandidateAmount)
	}
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
