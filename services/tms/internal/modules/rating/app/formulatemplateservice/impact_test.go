package formulatemplateservice

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
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
		func(req *shipmentports.ListRatedByFormulaTemplateRequest) bool {
			return req.TemplateID == template.ID && req.Limit == impactDefaultLimit
		},
	)).Return(shipments, nil)
}

func TestApprovalImpact_UsesVersionRecordedOnEachShipment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "linehaulRate * 2"
	template.CurrentVersionNumber = 2

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	expectRatedShipments(deps, template, shipments)

	// Two shipments stamped with the same version resolve it once.
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.GetVersionRequest) bool {
			return req.TemplateID == template.ID && req.VersionNumber == 1
		},
	)).Return(newApprovedSnapshot(template, 1, "linehaulRate"), nil).Once()

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
	template.Expression = "linehaulRate * 2"

	expectRatedShipments(deps, template, newBacktestShipments())
	deps.versionRepo.On("ListApproved", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.ListScheduledVersionsRequest) bool {
			return req.TemplateID == template.ID
		},
	)).Return([]*formulatemplate.FormulaTemplateVersion{
		newApprovedSnapshot(template, 1, "linehaulRate"),
	}, nil).Once()

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
	template.Expression = "linehaulRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 9)
	expectRatedShipments(deps, template, shipments)

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows).Once()
	deps.versionRepo.On("ListApproved", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{
			newApprovedSnapshot(template, 1, "linehaulRate"),
		}, nil).Once()

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
	deps.versionRepo.On("ListApproved", mock.Anything, mock.Anything).Return(nil, nil).Once()

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
	template.Expression = "linehaulRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.shipmentRepo.On("ListRatedByFormulaTemplate", mock.Anything, mock.Anything).
		Return(shipments, nil)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(newApprovedSnapshot(template, 1, "linehaulRate"), nil).Once()

	result, err := deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Expression: "linehaulRate * 3",
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

func TestApprovalImpact_PricesWithTheRelationsTheFormulaReads(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	store := ratingtest.NewRelationStore()
	expedited := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	store.ServiceTypes[expedited.ID] = expedited
	deps.svc.hydrator = ratingtest.NewHydrator(t, store)
	deps.svc.formulaService = ratingtest.NewFormulaService(t)

	template := newBacktestTemplate()
	template.Expression = `serviceType.code == "EXP" ? 75 : 0`

	shipments := newBacktestShipments()
	for _, entity := range shipments {
		entity.ServiceTypeID = expedited.ID
	}
	expectRatedShipments(deps, template, shipments)

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	for _, row := range result.Results {
		assert.Empty(t, row.CandidateError)
		assert.Empty(t, row.CurrentError)
		assert.True(t, decimal.NewFromInt(75).Equal(row.CandidateAmount), row.CandidateAmount)
		assert.True(t, decimal.NewFromInt(75).Equal(row.CurrentAmount), row.CurrentAmount)
	}
	assert.Len(t, store.Fetches("serviceType"), 1, "the batch loads each relation kind once")
}

func TestApprovalImpact_StopsAtItsDeadlineAndSaysSo(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	expectRatedShipments(deps, template, newBacktestShipments())

	expired, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()

	result, err := deps.svc.ApprovalImpact(expired, &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	assert.True(t, result.Truncated, "a reviewer must know the list is partial")
	assert.Empty(t, result.Results)
}

func TestBacktest_CurrentSideIsTheAmountTheShipmentWasBilled(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "linehaulRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	billed, overridden := shipments[0], shipments[1]
	billed.FreightChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("137.50"))
	billed.RatingDetail.Guardrail = &shipment.RatingGuardrail{Applied: true}
	overridden.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(90))
	overridden.RateOverrideAmount = decimal.NewNullDecimal(decimal.NewFromInt(90))

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.shipmentRepo.On("ListRatedByFormulaTemplate", mock.Anything, mock.Anything).
		Return(shipments, nil)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(newApprovedSnapshot(template, 1, "linehaulRate"), nil).Once()

	result, err := deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Expression: "linehaulRate * 3",
	})
	require.NoError(t, err)
	require.Len(t, result.Results, 2)

	byID := make(map[pulid.ID]*BacktestResult, len(result.Results))
	for _, row := range result.Results {
		byID[row.ShipmentID] = row
	}

	fromInvoice := byID[billed.ID]
	assert.Equal(t, BacktestCurrentBilled, fromInvoice.CurrentSource)
	assert.True(t, decimal.RequireFromString("137.50").Equal(fromInvoice.CurrentAmount),
		fromInvoice.CurrentAmount)
	assert.True(t, decimal.RequireFromString("162.50").Equal(fromInvoice.Delta), fromInvoice.Delta)
	assert.True(t, fromInvoice.GuardrailApplied, "the billed amount was clamped")

	recomputed := byID[overridden.ID]
	assert.Equal(t, BacktestCurrentRecomputed, recomputed.CurrentSource,
		"an override is not the formula's number")
	assert.True(t, decimal.NewFromInt(100).Equal(recomputed.CurrentAmount), recomputed.CurrentAmount)
}

func TestApprovalImpact_RecomputesTheCurrentSideToIsolateTheContentChange(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newBacktestTemplate()
	template.Expression = "linehaulRate * 2"

	shipments := newBacktestShipments()
	stampRatedVersion(shipments, template.ID, 1)
	for _, entity := range shipments {
		entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("137.50"))
	}
	expectRatedShipments(deps, template, shipments)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(newApprovedSnapshot(template, 1, "linehaulRate"), nil).Once()

	result, err := deps.svc.ApprovalImpact(t.Context(), &ApprovalImpactRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	for _, row := range result.Results {
		assert.Equal(t, BacktestCurrentRecomputed, row.CurrentSource)
		assert.True(t, decimal.NewFromInt(100).Equal(row.CurrentAmount), row.CurrentAmount)
	}
}
