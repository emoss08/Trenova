package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type auditEntry struct {
	params  *portservices.LogActionParams
	options []portservices.LogOption
}

type recordingAuditService struct {
	portservices.AuditService

	entries []auditEntry
}

func (r *recordingAuditService) LogAction(
	params *portservices.LogActionParams,
	opts ...portservices.LogOption,
) error {
	r.entries = append(r.entries, auditEntry{params: params, options: opts})
	return nil
}

func (r *recordingAuditService) only(t *testing.T) *portservices.LogActionParams {
	t.Helper()
	require.Len(t, r.entries, 1)
	return r.entries[0].params
}

type acceptingTenderRepo struct {
	repositories.TenderRepository

	offer          *tender.TenderOffer
	revokedOfferID pulid.ID
}

func (f *acceptingTenderRepo) GetOfferByID(
	_ context.Context,
	_ repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return f.offer, nil
}

func (f *acceptingTenderRepo) RevokeTokensForOffer(
	_ context.Context,
	_ pagination.TenantInfo,
	offerID pulid.ID,
	_ int64,
) error {
	f.revokedOfferID = offerID
	return nil
}

type stubCarrierAssignmentRepo struct {
	repositories.CarrierAssignmentRepository

	active *shipment.CarrierAssignment
}

func (f *stubCarrierAssignmentRepo) GetActiveByMoveID(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*shipment.CarrierAssignment, error) {
	return f.active, nil
}

type stubMoveAssigner struct {
	portservices.CarrierMoveAssigner

	confirmedID pulid.ID
}

func (f *stubMoveAssigner) ConfirmAssignment(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
) error {
	f.confirmedID = assignmentID
	return nil
}

func TestCreateSpotLogsTenderAudit(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	healthy := eligibleCarrier("acme")
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy}, nil)

	created, err := deps.svc.CreateSpot(t.Context(), &CreateSpotTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
		Mode:           tender.ModeSpotBroadcast,
		Lines:          []SpotTenderLine{spotLine(healthy.ID)},
	})

	require.NoError(t, err)
	require.NotNil(t, created)

	entry := deps.audit.only(t)
	assert.Equal(t, permission.ResourceTender, entry.Resource)
	assert.Equal(t, permission.OpCreate, entry.Operation)
	assert.Equal(t, created.ID.String(), entry.ResourceID)
	assert.Equal(t, tenantInfo.UserID, entry.UserID)
	assert.Equal(t, tenantInfo.OrgID, entry.OrganizationID)
	assert.Equal(t, tenantInfo.BuID, entry.BusinessUnitID)
	assert.NotEmpty(t, entry.CurrentState)
	assert.Empty(t, entry.PrincipalType)
}

func TestCreateWaterfallLogsTenderAudit(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:      pulid.MustNew("rgd_"),
		Name:    "Chicago to Dallas",
		Status:  "Active",
		Entries: []*tender.RoutingGuideEntry{guideEntry(healthy.ID, 1)},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy}, nil)

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, created)

	assert.Nil(t, created.Screening, "a fully eligible guide reports no screening")

	entry := deps.audit.only(t)
	assert.Equal(t, permission.ResourceTender, entry.Resource)
	assert.Equal(t, permission.OpCreate, entry.Operation)
	assert.Equal(t, created.ID.String(), entry.ResourceID)
	assert.Equal(t, tenantInfo.UserID, entry.UserID)
}

func TestAcceptOfferLogsTenderAudit(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	offer := acceptedOfferFixture()
	offer.Tender.Status = tender.StatusAccepted
	offer.Tender.AcceptedOfferID = &offer.ID
	offer.TenderID = offer.Tender.ID

	assignment := &shipment.CarrierAssignment{
		ID:        pulid.MustNew("cas_"),
		CarrierID: offer.CarrierID,
	}
	audit := &recordingAuditService{}
	assigner := &stubMoveAssigner{}
	svc := &Service{
		l:                     zap.NewNop(),
		db:                    testDBConnection{},
		repo:                  &acceptingTenderRepo{offer: offer},
		carrierAssignmentRepo: &stubCarrierAssignmentRepo{active: assignment},
		assigner:              assigner,
		eventService:          &recordingEventService{},
		auditService:          audit,
	}

	result, err := svc.AcceptOffer(
		t.Context(), tenantInfo, offer.ID, tender.ResponseSourceManual,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.NeedsReview)
	assert.Equal(t, assignment.ID, assigner.confirmedID)

	entry := audit.only(t)
	assert.Equal(t, permission.ResourceTender, entry.Resource)
	assert.Equal(t, permission.OpUpdate, entry.Operation)
	assert.Equal(t, offer.TenderID.String(), entry.ResourceID)
	assert.Equal(t, tenantInfo.UserID, entry.UserID)
	assert.NotEmpty(t, entry.PreviousState)
	assert.NotEmpty(t, entry.CurrentState)
}

func TestMarkNeedsReviewLogsSystemTenderAudit(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	offer := acceptedOfferFixture()
	offer.TenderID = offer.Tender.ID
	offer.Tender.Status = tender.StatusNeedsReview

	audit := &recordingAuditService{}
	svc := &Service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         &needsReviewTenderRepo{offer: offer},
		eventService: &recordingEventService{},
		auditService: audit,
	}

	err := svc.MarkNeedsReview(
		t.Context(), tenantInfo, offer.TenderID, offer.ID, "Move is already covered",
	)

	require.NoError(t, err)

	entry := audit.only(t)
	assert.Equal(t, permission.ResourceTender, entry.Resource)
	assert.Equal(t, permission.OpUpdate, entry.Operation)
	assert.Equal(t, offer.TenderID.String(), entry.ResourceID)
	assert.True(t, entry.UserID.IsNil())
	assert.Equal(t, portservices.PrincipalTypeSystem, entry.PrincipalType)
	assert.Equal(t, portservices.SystemPrincipalID, entry.PrincipalID)
}

type needsReviewTenderRepo struct {
	repositories.TenderRepository

	offer *tender.TenderOffer
}

func (f *needsReviewTenderRepo) UpdateTenderStatus(
	_ context.Context,
	_ *repositories.UpdateTenderStatusRequest,
) (bool, error) {
	return true, nil
}

func (f *needsReviewTenderRepo) BulkUpdateOfferStatus(
	_ context.Context,
	_ *repositories.BulkOfferStatusRequest,
) (int64, error) {
	return 0, nil
}

func (f *needsReviewTenderRepo) GetByID(
	_ context.Context,
	_ repositories.GetTenderByIDRequest,
) (*tender.Tender, error) {
	return f.offer.Tender, nil
}

func (f *needsReviewTenderRepo) RevokeTokensForOffer(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ int64,
) error {
	return nil
}

func (f *needsReviewTenderRepo) GetOfferByID(
	_ context.Context,
	_ repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return f.offer, nil
}
