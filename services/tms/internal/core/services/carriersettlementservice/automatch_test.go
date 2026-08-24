package carriersettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type autoMatchDeps struct {
	assignments *mocks.MockCarrierAssignmentRepository
	matches     *mocks.MockCarrierInvoiceMatchRepository
	control     *mocks.MockCarrierSettlementControlRepository
	invoices    *mocks.MockEDICarrierInvoiceRepository
	audit       *mocks.MockAuditService
	svc         *Service
}

func setupAutoMatchTest(t *testing.T) *autoMatchDeps {
	t.Helper()
	assignments := mocks.NewMockCarrierAssignmentRepository(t)
	matches := mocks.NewMockCarrierInvoiceMatchRepository(t)
	control := mocks.NewMockCarrierSettlementControlRepository(t)
	invoices := mocks.NewMockEDICarrierInvoiceRepository(t)
	audit := mocks.NewMockAuditService(t)
	svc := &Service{
		l:                 zap.NewNop(),
		assignmentRepo:    assignments,
		invoiceMatchRepo:  matches,
		settlementControl: control,
		ediInvoiceRepo:    invoices,
		auditService:      audit,
	}
	return &autoMatchDeps{
		assignments: assignments,
		matches:     matches,
		control:     control,
		invoices:    invoices,
		audit:       audit,
		svc:         svc,
	}
}

func autoMatchControl(
	autoMatch, autoAccept bool,
	toleranceMinor int64,
) *tenant.CarrierSettlementControl {
	return &tenant.CarrierSettlementControl{
		AutoMatchInboundInvoices:  autoMatch,
		AutoAcceptWithinTolerance: autoAccept,
		VarianceToleranceMinor:    toleranceMinor,
	}
}

func autoMatchInvoice(
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
	total float64,
) *edi.CarrierInvoice {
	return &edi.CarrierInvoice{
		ID:             pulid.MustNew("edici_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CarrierID:      carrierID,
		ShipmentID:     pulid.MustNew("shp_"),
		ProNumber:      "PRO-77",
		InvoiceNumber:  "INV-210",
		TotalAmount:    decimal.NullDecimal{Decimal: decimal.NewFromFloat(total), Valid: true},
	}
}

func (deps *autoMatchDeps) expectInvoiceLoad(
	tenantInfo pagination.TenantInfo,
	invoice *edi.CarrierInvoice,
) {
	deps.matches.On("GetOpenByEDIInvoiceID", mock.Anything, tenantInfo, invoice.ID).
		Return(nil, nil)
	deps.invoices.On(
		"GetCarrierInvoiceByID",
		mock.Anything,
		repositories.GetEDICarrierInvoiceByIDRequest{ID: invoice.ID, TenantInfo: tenantInfo},
	).Return(invoice, nil)
}

func (deps *autoMatchDeps) expectMatchCreate() *[]*carriersettlement.InvoiceMatch {
	created := &[]*carriersettlement.InvoiceMatch{}
	deps.matches.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			match := args.Get(1).(*carriersettlement.InvoiceMatch)
			match.ID = pulid.MustNew("cim_")
			*created = append(*created, match)
		}).
		Return(func(_ context.Context, entity *carriersettlement.InvoiceMatch) *carriersettlement.InvoiceMatch {
			return entity
		}, nil)
	return created
}

// The flat assignment totals 1500 + 125.50 + 75 + 50.25 = 1750.75 → 175075 minor.
func TestAutoMatchDisabledNoOps(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoiceID := pulid.MustNew("edici_")

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(false, false, 500), nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoiceID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	assert.False(t, result.AutoAccepted)
	assert.Empty(t, result.Status)
	assert.Empty(t, result.Warnings)
	deps.matches.AssertNotCalled(t, "GetOpenByEDIInvoiceID")
	deps.matches.AssertNotCalled(t, "Create")
}

func TestAutoMatchIdempotentOnReprocess(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoiceID := pulid.MustNew("edici_")

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.matches.On("GetOpenByEDIInvoiceID", mock.Anything, tenantInfo, invoiceID).
		Return(&carriersettlement.InvoiceMatch{
			ID:     pulid.MustNew("cim_"),
			Status: carriersettlement.InvoiceMatchStatusVariance,
		}, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoiceID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	assert.Equal(t, string(carriersettlement.InvoiceMatchStatusVariance), result.Status)
	assert.Empty(t, result.Warnings)
	deps.matches.AssertNotCalled(t, "Create")
	deps.invoices.AssertNotCalled(t, "GetCarrierInvoiceByID")
}

func TestAutoMatchWithinToleranceCreatesMatched(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1755.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On(
		"FindForMatching",
		mock.Anything,
		&repositories.FindCarrierAssignmentForMatchingRequest{
			TenantInfo: tenantInfo,
			CarrierID:  invoice.CarrierID,
			ProNumber:  invoice.ProNumber,
			ShipmentID: invoice.ShipmentID,
		},
	).Return(assignment, nil)
	created := deps.expectMatchCreate()

	var auditParams []*serviceports.LogActionParams
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *serviceports.LogActionParams, _ ...serviceports.LogOption) {
			auditParams = append(auditParams, params)
		}).
		Return(nil)
	var syncedInvoice *edi.CarrierInvoice
	deps.invoices.On("UpdateCarrierInvoice", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			syncedInvoice = args.Get(1).(*edi.CarrierInvoice)
		}).
		Return(invoice, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.False(t, result.AutoAccepted)
	assert.Equal(t, string(carriersettlement.InvoiceMatchStatusMatched), result.Status)
	assert.Empty(t, result.Warnings)

	require.Len(t, *created, 1)
	match := (*created)[0]
	assert.Equal(t, carriersettlement.InvoiceMatchStatusMatched, match.Status)
	assert.Equal(t, int64(175_575), match.InvoiceTotalMinor)
	assert.Equal(t, int64(175_075), match.ExpectedTotalMinor)
	assert.Equal(t, int64(500), match.VarianceMinor)
	require.NotNil(t, match.EDICarrierInvoiceID)
	assert.Equal(t, invoice.ID, *match.EDICarrierInvoiceID)
	assert.Equal(t, assignment.ID, match.CarrierAssignmentID)
	assert.Equal(t, carriersettlement.MatchViaAuto, match.MatchedVia)
	assert.True(t, match.ResolvedByID.IsNil())

	require.Len(t, auditParams, 1)
	assert.Equal(t, serviceports.PrincipalTypeSystem, auditParams[0].PrincipalType)
	assert.Equal(t, serviceports.SystemPrincipalID, auditParams[0].PrincipalID)
	assert.True(t, auditParams[0].UserID.IsNil())

	require.NotNil(t, syncedInvoice)
	assert.Equal(t, edi.CarrierInvoiceReconciliationStatusMatched, syncedInvoice.ReconciliationStatus)
	require.True(t, syncedInvoice.ExpectedAmount.Valid)
	assert.True(t, syncedInvoice.ExpectedAmount.Decimal.Equal(decimal.NewFromFloat(1750.75)))
	deps.matches.AssertNotCalled(t, "Update")
}

func TestAutoMatchWithinToleranceAutoAccepts(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1750.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, true, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(assignment, nil)
	deps.expectMatchCreate()

	var resolved *carriersettlement.InvoiceMatch
	deps.matches.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			resolved = args.Get(1).(*carriersettlement.InvoiceMatch)
		}).
		Return(func(_ context.Context, entity *carriersettlement.InvoiceMatch) *carriersettlement.InvoiceMatch {
			return entity
		}, nil)
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Times(2)
	deps.invoices.On("UpdateCarrierInvoice", mock.Anything, mock.Anything).Return(invoice, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.True(t, result.AutoAccepted)
	assert.Equal(t, string(carriersettlement.InvoiceMatchStatusResolved), result.Status)
	assert.Empty(t, result.Warnings)

	require.NotNil(t, resolved)
	assert.Equal(t, carriersettlement.InvoiceMatchStatusResolved, resolved.Status)
	assert.Equal(t, "Auto-accepted: within variance tolerance", resolved.ResolutionNote)
	assert.True(t, resolved.ResolvedByID.IsNil())
	require.NotNil(t, resolved.ResolvedAt)
	assert.Positive(t, *resolved.ResolvedAt)
}

func TestAutoMatchOutsideToleranceCreatesVariance(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1760.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, true, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(assignment, nil)
	created := deps.expectMatchCreate()
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)
	var syncedInvoice *edi.CarrierInvoice
	deps.invoices.On("UpdateCarrierInvoice", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			syncedInvoice = args.Get(1).(*edi.CarrierInvoice)
		}).
		Return(invoice, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.False(t, result.AutoAccepted)
	assert.Equal(t, string(carriersettlement.InvoiceMatchStatusVariance), result.Status)

	require.Len(t, *created, 1)
	assert.Equal(t, carriersettlement.InvoiceMatchStatusVariance, (*created)[0].Status)
	assert.Equal(t, int64(1_000), (*created)[0].VarianceMinor)
	deps.matches.AssertNotCalled(t, "Update")

	require.NotNil(t, syncedInvoice)
	assert.Equal(
		t,
		edi.CarrierInvoiceReconciliationStatusVariance,
		syncedInvoice.ReconciliationStatus,
	)
}

func TestAutoMatchMissingCarrierSkipsWithWarning(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoice := autoMatchInvoice(tenantInfo, pulid.Nil, 1750.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "not linked to a carrier")
	deps.matches.AssertNotCalled(t, "Create")
}

func TestAutoMatchMissingTotalSkipsWithWarning(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoice := autoMatchInvoice(tenantInfo, pulid.MustNew("car_"), 0)
	invoice.TotalAmount = decimal.NullDecimal{}

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "does not carry an invoice total")
	deps.assignments.AssertNotCalled(t, "FindForMatching")
}

func TestAutoMatchMissingAssignmentSkipsWithWarning(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoice := autoMatchInvoice(tenantInfo, pulid.MustNew("car_"), 1750.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(nil, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no carrier assignment matches")
	deps.matches.AssertNotCalled(t, "Create")
}

func TestAutoMatchMissingReferencesSkipsWithWarning(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	invoice := autoMatchInvoice(tenantInfo, pulid.MustNew("car_"), 1750.75)
	invoice.ShipmentID = pulid.Nil
	invoice.ProNumber = ""

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, false, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	require.NoError(t, err)
	assert.False(t, result.Matched)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no pro number or shipment reference")
	deps.assignments.AssertNotCalled(t, "FindForMatching")
}
