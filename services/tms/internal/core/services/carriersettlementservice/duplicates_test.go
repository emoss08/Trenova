package carriersettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func withoutDuplicateDefaults(matches *mocks.MockCarrierInvoiceMatchRepository) {
	kept := matches.ExpectedCalls[:0]
	for _, call := range matches.ExpectedCalls {
		if call.Method == "GetLiveByCarrierInvoiceNumber" || call.Method == "ListLiveByAssignment" {
			continue
		}
		kept = append(kept, call)
	}
	matches.ExpectedCalls = kept
}

func documentMatchRequest(
	deps *matchingDeps,
	invoiceNumber string,
) (*CreateMatchRequest, pulid.ID) {
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
		Return(&tenant.CarrierSettlementControl{VarianceToleranceMinor: 500}, nil).Maybe()

	return &CreateMatchRequest{
		TenantInfo:             tenantInfo,
		DocumentAIExtractionID: &extractionID,
		CarrierID:              assignment.CarrierID,
		CarrierAssignmentID:    &assignmentID,
		InvoiceNumber:          invoiceNumber,
		InvoiceTotalMinor:      175_075,
	}, assignmentID
}

func TestCreateMatch_RefusesAnInvoiceNumberTheCarrierAlreadyHasMatched(t *testing.T) {
	deps := setupMatchingTest(t)
	withoutDuplicateDefaults(deps.matches)
	req, _ := documentMatchRequest(deps, " inv 9001 ")

	deps.matches.On("GetLiveByCarrierInvoiceNumber", mock.Anything,
		&repositories.GetLiveCarrierInvoiceMatchByNumberRequest{
			TenantInfo:    req.TenantInfo,
			CarrierID:     req.CarrierID,
			InvoiceNumber: " inv 9001 ",
		}).Return(&carriersettlement.InvoiceMatch{ID: pulid.MustNew("cim_")}, nil)

	_, err := deps.svc.CreateMatch(t.Context(), req, matchingActor())

	require.Error(t, err)
	assert.True(t, isDuplicateRefusal(err))
	deps.matches.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	plan, planErr := deps.svc.PlanCreateMatch(t.Context(), req)
	require.NoError(t, planErr)
	assert.True(t, settlementshared.IsRefusal(plan.Refusal), "agent previews refuse it too")
}

func TestCreateMatch_FlagsASecondInvoiceOnTheSameLoad(t *testing.T) {
	deps := setupMatchingTest(t)
	withoutDuplicateDefaults(deps.matches)
	req, assignmentID := documentMatchRequest(deps, "INV-9002")
	earlier := &carriersettlement.InvoiceMatch{ID: pulid.MustNew("cim_"), CarrierAssignmentID: assignmentID}

	deps.matches.On("GetLiveByCarrierInvoiceNumber", mock.Anything, mock.Anything).Return(nil, nil)
	deps.matches.On("ListLiveByAssignment", mock.Anything,
		repositories.ListLiveCarrierInvoiceMatchesByAssignmentRequest{
			TenantInfo:   req.TenantInfo,
			AssignmentID: assignmentID,
		}).Return([]*carriersettlement.InvoiceMatch{earlier}, nil)
	deps.matches.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *carriersettlement.InvoiceMatch) *carriersettlement.InvoiceMatch {
			entity.ID = pulid.MustNew("cim_")
			return entity
		}, nil)
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)

	match, err := deps.svc.CreateMatch(t.Context(), req, matchingActor())

	require.NoError(t, err)
	require.True(t, match.IsPossibleDuplicate())
	assert.Equal(t, earlier.ID, *match.PossibleDuplicateOfID)
}

func TestAcceptWithVariance_RefusesWhenTheLoadWasAlreadyAdjusted(t *testing.T) {
	deps := setupMatchingTest(t)
	withoutDuplicateDefaults(deps.matches)
	tenantInfo := accrualTenantInfo()
	assignmentID := pulid.MustNew("ca_")
	extractionID := pulid.MustNew("dax_")
	adjustmentID := pulid.MustNew("cce_")
	match := &carriersettlement.InvoiceMatch{
		ID:                     pulid.MustNew("cim_"),
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		DocumentAIExtractionID: &extractionID,
		CarrierAssignmentID:    assignmentID,
		Status:                 carriersettlement.InvoiceMatchStatusVariance,
		InvoiceNumber:          "INV-B",
		VarianceMinor:          4_925,
	}
	alreadyPaid := &carriersettlement.InvoiceMatch{
		ID:                    pulid.MustNew("cim_"),
		CarrierAssignmentID:   assignmentID,
		Status:                carriersettlement.InvoiceMatchStatusResolved,
		InvoiceNumber:         "INV-A",
		AdjustmentCostEventID: &adjustmentID,
	}
	deps.matches.On("GetByID", mock.Anything, mock.Anything).Return(match, nil)
	deps.matches.On("ListLiveByAssignment", mock.Anything, mock.Anything).
		Return([]*carriersettlement.InvoiceMatch{alreadyPaid, match}, nil)

	_, err := deps.svc.AcceptWithVariance(t.Context(), tenantInfo, match.ID, "ok", matchingActor())

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, errortypes.ErrInvalidOperation, fieldErr.Code)
	deps.costEvents.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	plan, planErr := deps.svc.PlanMatchDecision(t.Context(), &MatchDecisionRequest{
		TenantInfo: tenantInfo,
		MatchID:    match.ID,
		Decision:   MatchDecisionAcceptWithVariance,
		Note:       "ok",
	})
	require.NoError(t, planErr)
	require.Error(t, plan.Refusal)
	assert.Nil(t, plan.Adjustment)
}

func TestAcceptMatch_RefusesAMatchRepeatingALiveOne(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	originalID := pulid.MustNew("cim_")
	repeat := &carriersettlement.InvoiceMatch{
		ID:                 pulid.MustNew("cim_"),
		Status:             carriersettlement.InvoiceMatchStatusMatched,
		InvoiceNumber:      "INV-7",
		DuplicateOfMatchID: &originalID,
	}
	deps.matches.On("GetByID", mock.Anything,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: repeat.ID, TenantInfo: tenantInfo}).
		Return(repeat, nil)
	deps.matches.On("GetByID", mock.Anything,
		repositories.GetCarrierInvoiceMatchByIDRequest{ID: originalID, TenantInfo: tenantInfo}).
		Return(&carriersettlement.InvoiceMatch{
			ID:     originalID,
			Status: carriersettlement.InvoiceMatchStatusResolved,
		}, nil)

	_, err := deps.svc.AcceptMatch(t.Context(), tenantInfo, repeat.ID, "", matchingActor())

	require.True(t, isDuplicateRefusal(err))
	deps.matches.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestApprovalHold_RefusesASettlementWithUnmatchedLoads(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	settlementID := pulid.MustNew("cs_")
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&tenant.CarrierSettlementControl{HoldUntilInvoiceMatched: true}, nil)
	deps.costEvents.On("CountAwaitingInvoiceMatch", mock.Anything, tenantInfo, settlementID).
		Return(2, nil).Once()

	hold, err := deps.svc.approvalHoldGuard(t.Context(), tenantInfo, settlementID)

	require.NoError(t, err)
	require.True(t, hold.refused())
	assert.Contains(t, hold.refusal.Error(), "waiting on a resolved carrier invoice match")
}

func TestApprovalHold_IsOffByDefault(t *testing.T) {
	deps := setupMatchingTest(t)
	tenantInfo := accrualTenantInfo()
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&tenant.CarrierSettlementControl{}, nil)

	hold, err := deps.svc.approvalHoldGuard(t.Context(), tenantInfo, pulid.MustNew("cs_"))

	require.NoError(t, err)
	require.False(t, hold.refused())
	deps.costEvents.AssertNotCalled(t, "CountAwaitingInvoiceMatch", mock.Anything, mock.Anything, mock.Anything)
}

func TestAutoMatch_LeavesAPossibleDuplicateForReview(t *testing.T) {
	deps := setupAutoMatchTest(t)
	withoutDuplicateDefaults(deps.matches)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1750.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, true, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(assignment, nil)
	deps.matches.On("GetLiveByCarrierInvoiceNumber", mock.Anything, mock.Anything).Return(nil, nil)
	deps.matches.On("ListLiveByAssignment", mock.Anything, mock.Anything).
		Return([]*carriersettlement.InvoiceMatch{{ID: pulid.MustNew("cim_")}}, nil)
	created := deps.expectMatchCreate()
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	deps.invoices.On("UpdateCarrierInvoice", mock.Anything, mock.Anything).Return(invoice, nil).Maybe()

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)

	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.False(t, result.AutoAccepted, "a possible duplicate is never accepted automatically")
	require.Len(t, *created, 1)
	assert.True(t, (*created)[0].IsPossibleDuplicate())
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "possible duplicate")
	deps.matches.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestAutoMatch_SkipsAnInvoiceNumberAlreadyMatched(t *testing.T) {
	deps := setupAutoMatchTest(t)
	withoutDuplicateDefaults(deps.matches)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1750.75)

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, true, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(assignment, nil)
	deps.matches.On("GetLiveByCarrierInvoiceNumber", mock.Anything, mock.Anything).
		Return(&carriersettlement.InvoiceMatch{ID: pulid.MustNew("cim_")}, nil)

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
	assert.Contains(t, result.Warnings[0], "already matched")
	deps.matches.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAutoMatch_SkipsARepeatedPartnerInvoice(t *testing.T) {
	deps := setupAutoMatchTest(t)
	tenantInfo := accrualTenantInfo()
	assignment := newFlatAssignment()
	invoice := autoMatchInvoice(tenantInfo, assignment.CarrierID, 1750.75)
	invoice.DuplicateOfID = pulid.MustNew("edici_")

	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(autoMatchControl(true, true, 500), nil)
	deps.expectInvoiceLoad(tenantInfo, invoice)
	deps.assignments.On("FindForMatching", mock.Anything, mock.Anything).Return(assignment, nil)

	result, err := deps.svc.AutoMatchInboundInvoice(
		t.Context(),
		&serviceports.AutoMatchInboundInvoiceRequest{
			TenantInfo:          tenantInfo,
			EDICarrierInvoiceID: invoice.ID,
		},
	)

	require.NoError(t, err)
	assert.False(t, result.Matched)
	assert.Contains(t, result.Warnings[0], "repeats an invoice")
}
