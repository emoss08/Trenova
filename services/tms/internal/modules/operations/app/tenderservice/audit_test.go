package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/dbtest"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type auditEntry struct {
	params  *auditports.LogActionParams
	options []auditports.LogOption
}

type recordingAuditService struct {
	auditports.AuditService

	entries []auditEntry
}

func (r *recordingAuditService) LogAction(
	params *auditports.LogActionParams,
	opts ...auditports.LogOption,
) error {
	r.entries = append(r.entries, auditEntry{params: params, options: opts})
	return nil
}

func (r *recordingAuditService) only(t *testing.T) *auditports.LogActionParams {
	t.Helper()
	require.Len(t, r.entries, 1)
	return r.entries[0].params
}

type acceptingTenderRepo struct {
	tenderports.TenderRepository

	offer          *tender.TenderOffer
	revokedOfferID pulid.ID
}

func (f *acceptingTenderRepo) GetOfferByID(
	_ context.Context,
	_ tenderports.GetTenderOfferByIDRequest,
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
	shipmentports.CarrierAssignmentRepository

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
	tenderports.CarrierMoveAssigner

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
		db:                    dbtest.NopConnection{},
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
		db:           dbtest.NopConnection{},
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
	assert.Equal(t, permissionports.PrincipalTypeSystem, entry.PrincipalType)
	assert.Equal(t, permissionports.SystemPrincipalID, entry.PrincipalID)
}

type needsReviewTenderRepo struct {
	tenderports.TenderRepository

	offer *tender.TenderOffer
}

func (f *needsReviewTenderRepo) UpdateTenderStatus(
	_ context.Context,
	_ *tenderports.UpdateTenderStatusRequest,
) (bool, error) {
	return true, nil
}

func (f *needsReviewTenderRepo) BulkUpdateOfferStatus(
	_ context.Context,
	_ *tenderports.BulkOfferStatusRequest,
) (int64, error) {
	return 0, nil
}

func (f *needsReviewTenderRepo) GetByID(
	_ context.Context,
	_ tenderports.GetTenderByIDRequest,
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
	_ tenderports.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return f.offer, nil
}
