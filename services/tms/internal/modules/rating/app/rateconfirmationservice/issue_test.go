package rateconfirmationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/dbtest"

	"github.com/emoss08/trenova/internal/testutil/mocks/carrierportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type issueTestDeps struct {
	repo                  *fakeRateConRepo
	carrierAssignmentRepo *shipmentportsmock.MockCarrierAssignmentRepository
	carrierRepo           *carrierportsmock.MockCarrierRepository
	svc                   *Service
}

func setupIssueTest(t *testing.T) *issueTestDeps {
	t.Helper()

	deps := &issueTestDeps{
		repo:                  newFakeRateConRepo(),
		carrierAssignmentRepo: shipmentportsmock.NewMockCarrierAssignmentRepository(t),
		carrierRepo:           carrierportsmock.NewMockCarrierRepository(t),
	}
	deps.svc = &Service{
		l:                     zap.NewNop(),
		db:                    dbtest.NopConnection{},
		repo:                  deps.repo,
		carrierAssignmentRepo: deps.carrierAssignmentRepo,
		carrierRepo:           deps.carrierRepo,
	}
	return deps
}

func issueRequest() *rateconfirmationports.IssueTenderRateConfirmationRequest {
	return &rateconfirmationports.IssueTenderRateConfirmationRequest{
		TenantInfo:     testTenant(),
		ShipmentMoveID: pulid.MustNew("smv_"),
		OfferID:        pulid.MustNew("tof_"),
		CarrierID:      pulid.MustNew("car_"),
		Source:         tender.ResponseSourceEmail,
		AcceptedAt:     1_760_000_000,
	}
}

func pendingAssignment(
	req *rateconfirmationports.IssueTenderRateConfirmationRequest,
) *shipment.CarrierAssignment {
	return &shipment.CarrierAssignment{
		ID:        pulid.MustNew("cass_"),
		CarrierID: req.CarrierID,
		Status:    shipment.CarrierAssignmentStatusPending,
	}
}

func tenderIssuedRateCon(
	req *rateconfirmationports.IssueTenderRateConfirmationRequest,
	assignment *shipment.CarrierAssignment,
	status rateconfirmation.Status,
) *rateconfirmation.RateConfirmation {
	offerID := req.OfferID
	return &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		CarrierAssignmentID: assignment.ID,
		CarrierID:           req.CarrierID,
		ShipmentMoveID:      req.ShipmentMoveID,
		Status:              status,
		GeneratedVia:        rateconfirmation.ViaTenderAcceptance,
		SourceTenderOfferID: &offerID,
	}
}

func TestIssueForTenderAcceptanceNoAssignment(t *testing.T) {
	deps := setupIssueTest(t)
	req := issueRequest()

	deps.carrierAssignmentRepo.On(
		"GetActiveByMoveID", mock.Anything, req.TenantInfo, req.ShipmentMoveID,
	).Return(nil, nil)

	result, err := deps.svc.IssueForTenderAcceptance(t.Context(), req)

	require.NoError(t, err)
	assert.False(t, result.Issued)
	assert.Contains(t, result.Reason, "no active carrier assignment")
	assert.Empty(t, deps.repo.updated)
}

func TestIssueForTenderAcceptanceCarrierMismatch(t *testing.T) {
	deps := setupIssueTest(t)
	req := issueRequest()
	assignment := pendingAssignment(req)
	assignment.CarrierID = pulid.MustNew("car_")

	deps.carrierAssignmentRepo.On(
		"GetActiveByMoveID", mock.Anything, req.TenantInfo, req.ShipmentMoveID,
	).Return(assignment, nil)

	result, err := deps.svc.IssueForTenderAcceptance(t.Context(), req)

	require.NoError(t, err)
	assert.False(t, result.Issued)
	assert.Contains(t, result.Reason, "different carrier")
	assert.Empty(t, deps.repo.updated)
}

func TestIssueForTenderAcceptanceRetryIsNoOp(t *testing.T) {
	deps := setupIssueTest(t)
	req := issueRequest()
	assignment := pendingAssignment(req)
	existing := tenderIssuedRateCon(req, assignment, rateconfirmation.StatusConfirmed)
	deps.repo.add(existing)

	deps.carrierAssignmentRepo.On(
		"GetActiveByMoveID", mock.Anything, req.TenantInfo, req.ShipmentMoveID,
	).Return(assignment, nil)

	result, err := deps.svc.IssueForTenderAcceptance(t.Context(), req)

	require.NoError(t, err)
	assert.True(t, result.Issued)
	assert.Equal(t, existing.ID, result.RateConfirmationID)
	assert.Empty(t, deps.repo.updated)
}

func TestIssueForTenderAcceptanceLeavesManualRevisionAlone(t *testing.T) {
	deps := setupIssueTest(t)
	req := issueRequest()
	assignment := pendingAssignment(req)
	manual := tenderIssuedRateCon(req, assignment, rateconfirmation.StatusSent)
	manual.GeneratedVia = rateconfirmation.ViaDispatcher
	manual.SourceTenderOfferID = nil
	deps.repo.add(manual)

	deps.carrierAssignmentRepo.On(
		"GetActiveByMoveID", mock.Anything, req.TenantInfo, req.ShipmentMoveID,
	).Return(assignment, nil)

	result, err := deps.svc.IssueForTenderAcceptance(t.Context(), req)

	require.NoError(t, err)
	assert.False(t, result.Issued)
	assert.Equal(t, manual.ID, result.RateConfirmationID)
	assert.Contains(t, result.Reason, "dispatcher-managed")
	assert.Empty(t, deps.repo.updated)
	assert.Equal(t, rateconfirmation.StatusSent, manual.Status)
}

func TestIssueForTenderAcceptanceResumesGeneratedRevision(t *testing.T) {
	deps := setupIssueTest(t)
	req := issueRequest()
	assignment := pendingAssignment(req)
	pending := tenderIssuedRateCon(req, assignment, rateconfirmation.StatusGenerated)
	deps.repo.add(pending)

	deps.carrierAssignmentRepo.On(
		"GetActiveByMoveID", mock.Anything, req.TenantInfo, req.ShipmentMoveID,
	).Return(assignment, nil)
	deps.carrierRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&carrier.Carrier{ID: req.CarrierID, Name: "Eastline Transport"}, nil)
	deps.carrierAssignmentRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(assignment, nil)
	deps.carrierAssignmentRepo.On("Update", mock.Anything, assignment).
		Return(assignment, nil)

	result, err := deps.svc.IssueForTenderAcceptance(t.Context(), req)

	require.NoError(t, err)
	assert.True(t, result.Issued)
	assert.Equal(t, pending.ID, result.RateConfirmationID)

	require.NotEmpty(t, deps.repo.updated)
	assert.Equal(t, rateconfirmation.StatusConfirmed, pending.Status)
	assert.Equal(t, rateconfirmation.ViaTenderAcceptance, pending.ConfirmedVia)
	assert.Equal(t, "Eastline Transport", pending.ConfirmedByName)
	assert.Equal(t, "Tender offer acceptance (Email)", pending.ConfirmedByTitle)
	require.NotNil(t, pending.ConfirmedAt)
	assert.Equal(t, req.AcceptedAt, *pending.ConfirmedAt)

	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, assignment.Status)
	require.NotNil(t, assignment.ConfirmedAt)
	assert.Equal(t, req.AcceptedAt, *assignment.ConfirmedAt)
}
