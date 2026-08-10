package carriersettlementservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type matchingDeps struct {
	costEvents  *mocks.MockCarrierCostEventRepository
	assignments *mocks.MockCarrierAssignmentRepository
	matches     *mocks.MockCarrierInvoiceMatchRepository
	control     *mocks.MockCarrierSettlementControlRepository
	audit       *mocks.MockAuditService
	svc         *Service
}

func setupMatchingTest(t *testing.T) *matchingDeps {
	t.Helper()
	costEvents := mocks.NewMockCarrierCostEventRepository(t)
	assignments := mocks.NewMockCarrierAssignmentRepository(t)
	matches := mocks.NewMockCarrierInvoiceMatchRepository(t)
	control := mocks.NewMockCarrierSettlementControlRepository(t)
	audit := mocks.NewMockAuditService(t)
	svc := &Service{
		l:                 zap.NewNop(),
		costEventRepo:     costEvents,
		assignmentRepo:    assignments,
		invoiceMatchRepo:  matches,
		settlementControl: control,
		auditService:      audit,
	}
	return &matchingDeps{
		costEvents:  costEvents,
		assignments: assignments,
		matches:     matches,
		control:     control,
		audit:       audit,
		svc:         svc,
	}
}

func matchingActor() *serviceports.RequestActor {
	return &serviceports.RequestActor{UserID: pulid.MustNew("usr_")}
}

func createMatchWithVariance(
	t *testing.T,
	deps *matchingDeps,
	invoiceTotalMinor, toleranceMinor int64,
) *carriersettlement.InvoiceMatch {
	t.Helper()
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	assignment.OrganizationID = tenantInfo.OrgID
	assignment.BusinessUnitID = tenantInfo.BuID
	extractionID := pulid.MustNew("dax_")
	assignmentID := assignment.ID

	deps.matches.On("GetOpenByExtractionID", mock.Anything, tenantInfo, extractionID).
		Return(nil, nil)
	deps.assignments.On("GetByID", mock.Anything, mock.Anything).Return(assignment, nil)
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&tenant.CarrierSettlementControl{
			VarianceToleranceMinor: toleranceMinor,
		}, nil)
	var created *carriersettlement.InvoiceMatch
	deps.matches.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = args.Get(1).(*carriersettlement.InvoiceMatch)
			created.ID = pulid.MustNew("cim_")
		}).
		Return(func(_ context.Context, entity *carriersettlement.InvoiceMatch) *carriersettlement.InvoiceMatch {
			return entity
		}, nil)
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)

	match, err := deps.svc.CreateMatch(t.Context(), &CreateMatchRequest{
		TenantInfo:             tenantInfo,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              assignment.CarrierID,
		CarrierAssignmentID:    &assignmentID,
		InvoiceNumber:          "INV-9001",
		InvoiceTotalMinor:      invoiceTotalMinor,
	}, matchingActor())
	require.NoError(t, err)
	require.NotNil(t, match)
	require.Same(t, created, match)
	return match
}

func TestCreateMatchVarianceMath(t *testing.T) {
	// The flat assignment totals 1500 + 125.50 + 75 + 50.25 = 1750.75 → 175075 minor.
	t.Run("within tolerance lands Matched", func(t *testing.T) {
		deps := setupMatchingTest(t)
		match := createMatchWithVariance(t, deps, 175_575, 500)
		assert.Equal(t, carriersettlement.InvoiceMatchStatusMatched, match.Status)
		assert.Equal(t, int64(175_075), match.ExpectedTotalMinor)
		assert.Equal(t, int64(500), match.VarianceMinor)
	})

	t.Run("exactly at tolerance boundary lands Matched", func(t *testing.T) {
		deps := setupMatchingTest(t)
		match := createMatchWithVariance(t, deps, 174_575, 500)
		assert.Equal(t, carriersettlement.InvoiceMatchStatusMatched, match.Status)
		assert.Equal(t, int64(-500), match.VarianceMinor)
	})

	t.Run("one minor unit past tolerance lands Variance", func(t *testing.T) {
		deps := setupMatchingTest(t)
		match := createMatchWithVariance(t, deps, 175_576, 500)
		assert.Equal(t, carriersettlement.InvoiceMatchStatusVariance, match.Status)
		assert.Equal(t, int64(501), match.VarianceMinor)
	})

	t.Run("exact invoice with zero tolerance lands Matched", func(t *testing.T) {
		deps := setupMatchingTest(t)
		match := createMatchWithVariance(t, deps, 175_075, 0)
		assert.Equal(t, carriersettlement.InvoiceMatchStatusMatched, match.Status)
		assert.Equal(t, int64(0), match.VarianceMinor)
	})
}

func TestCreateMatchRejectsDuplicateExtraction(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	extractionID := pulid.MustNew("dax_")
	carrierID := pulid.MustNew("car_")

	deps.matches.On("GetOpenByExtractionID", mock.Anything, tenantInfo, extractionID).
		Return(&carriersettlement.InvoiceMatch{ID: pulid.MustNew("cim_")}, nil)

	_, err := deps.svc.CreateMatch(t.Context(), &CreateMatchRequest{
		TenantInfo:             tenantInfo,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              carrierID,
		InvoiceNumber:          "INV-9002",
		InvoiceTotalMinor:      100_000,
	}, matchingActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an open match")
	deps.matches.AssertNotCalled(t, "Create")
}

func TestCreateMatchRequiresExactlyOneSource(t *testing.T) {
	deps := setupMatchingTest(t)
	_, err := deps.svc.CreateMatch(t.Context(), &CreateMatchRequest{
		TenantInfo: accrualTenantInfo(),
	}, matchingActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one source")
}

func TestAcceptWithVarianceCreatesAdjustmentEvent(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	matchID := pulid.MustNew("cim_")
	extractionID := pulid.MustNew("dax_")
	match := &carriersettlement.InvoiceMatch{
		ID:                     matchID,
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              assignment.CarrierID,
		CarrierAssignmentID:    assignment.ID,
		CarrierAssignment:      assignment,
		Status:                 carriersettlement.InvoiceMatchStatusVariance,
		InvoiceNumber:          "INV-9001",
		InvoiceTotalMinor:      180_000,
		ExpectedTotalMinor:     175_075,
		VarianceMinor:          4_925,
		CurrencyCode:           "USD",
	}

	deps.matches.On("GetByID", mock.Anything, mock.Anything).Return(match, nil)
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo,
		"carrier-invoice-variance:"+matchID.String(),
	).Return(nil, errors.New("not found"))

	var adjustment *carriersettlement.CostEvent
	deps.costEvents.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			adjustment = args.Get(1).(*carriersettlement.CostEvent)
			adjustment.ID = pulid.MustNew("cacc_")
		}).
		Return(func(_ context.Context, entity *carriersettlement.CostEvent) *carriersettlement.CostEvent {
			return entity
		}, nil)
	deps.matches.On("Update", mock.Anything, match).Return(match, nil)
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.AcceptWithVariance(
		t.Context(),
		tenantInfo,
		matchID,
		"Detention billed on arrival",
		matchingActor(),
	)
	require.NoError(t, err)

	require.NotNil(t, adjustment)
	assert.Equal(t, carriersettlement.CostEventTypeAdjustment, adjustment.EventType)
	assert.Equal(t, int64(4_925), adjustment.AmountMinor)
	assert.Equal(t, carriersettlement.CostEventStatusPending, adjustment.Status)
	assert.Equal(t, match.CarrierID, adjustment.CarrierID)
	assert.Equal(t, "carrier-invoice-variance:"+matchID.String(), adjustment.IdempotencyKey)
	assert.Contains(t, adjustment.Description, "INV-9001")

	assert.Equal(t, carriersettlement.InvoiceMatchStatusResolved, updated.Status)
	require.NotNil(t, updated.AdjustmentCostEventID)
	assert.Equal(t, adjustment.ID, *updated.AdjustmentCostEventID)
	assert.Equal(t, "Detention billed on arrival", updated.ResolutionNote)
}

func TestAcceptWithVarianceRejectsNonVarianceMatch(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	matchID := pulid.MustNew("cim_")
	deps.matches.On("GetByID", mock.Anything, mock.Anything).
		Return(&carriersettlement.InvoiceMatch{
			ID:     matchID,
			Status: carriersettlement.InvoiceMatchStatusMatched,
		}, nil)

	_, err := deps.svc.AcceptWithVariance(t.Context(), tenantInfo, matchID, "", matchingActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variance")
}
