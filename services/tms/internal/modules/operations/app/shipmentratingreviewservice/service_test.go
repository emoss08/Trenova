package shipmentratingreviewservice_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentratingreviewservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/ports/ratingreviewports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type store struct {
	mu     sync.Mutex
	row    *shipment.Shipment
	saves  int
	reads  []*ratingreviewports.ApproveRatingReviewRequest
	failOn error
}

func (s *store) GetForRatingReview(
	_ context.Context,
	req *ratingreviewports.ApproveRatingReviewRequest,
) (*shipment.Shipment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = append(s.reads, req)
	if s.row == nil || s.row.ID != req.ShipmentID {
		return nil, errortypes.NewNotFoundError("Shipment not found")
	}
	copied := *s.row
	return &copied, nil
}

func (s *store) SaveRatingReview(
	_ context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOn != nil {
		return nil, s.failOn
	}
	s.saves++
	entity.Version++
	copied := *entity
	s.row = &copied
	return entity, nil
}

type auditLog struct {
	auditports.AuditService

	mu      sync.Mutex
	entries []*auditports.LogActionParams
}

func (a *auditLog) LogAction(params *auditports.LogActionParams, _ ...auditports.LogOption) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, params)
	return nil
}

func heldShipment(tenant pagination.TenantInfo, reviews ...string) *shipment.Shipment {
	bill := &tarifftypes.Bill{Currency: "USD"}
	for _, text := range reviews {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ProNumber:      "PRO-9",
		Version:        3,
		RatingDetail:   &shipment.RatingDetail{Bill: bill, FormulaTemplateName: "Dock rates"},
	}
}

func newService(repo *store, audit *auditLog) *shipmentratingreviewservice.Service {
	return shipmentratingreviewservice.New(shipmentratingreviewservice.Params{
		Logger:       zap.NewNop(),
		DB:           dbtest.NopConnection{},
		Repo:         repo,
		AuditService: audit,
	})
}

func person(tenant pagination.TenantInfo) *permissionports.RequestActor {
	userID := pulid.MustNew("usr_")
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestApprove_RecordsWhoApprovedTheReviewedRatingAndAuditsIt(t *testing.T) {
	t.Parallel()

	tenant := tenantInfo()
	repo := &store{row: heldShipment(tenant, "Holiday pickup", "Liftgate on a dock")}
	audit := &auditLog{}
	actor := person(tenant)

	saved, err := newService(repo, audit).Approve(
		t.Context(),
		&ratingreviewports.ApproveRatingReviewRequest{ShipmentID: repo.row.ID, TenantInfo: tenant},
		actor,
	)
	require.NoError(t, err)

	assert.False(t, shipmentrating.AwaitsReview(saved))
	require.NotNil(t, saved.RatingDetail.ReviewApproval)
	assert.Equal(t, actor.UserID, saved.RatingDetail.ReviewApproval.ApprovedByID)
	assert.Equal(
		t,
		[]string{"Holiday pickup", "Liftgate on a dock"},
		saved.RatingDetail.ReviewApproval.Reviews,
	)
	assert.Equal(t, int64(4), saved.Version)
	assert.Equal(t, 1, repo.saves)
	assert.Equal(t, tenant, repo.reads[0].TenantInfo)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, permission.ResourceShipment, audit.entries[0].Resource)
	assert.Equal(t, permission.OpUpdate, audit.entries[0].Operation)
	assert.Equal(t, saved.ID.String(), audit.entries[0].ResourceID)
	assert.Equal(t, actor.UserID, audit.entries[0].UserID)
}

func TestApprove_IsANoOpOnceTheRatingIsApproved(t *testing.T) {
	t.Parallel()

	tenant := tenantInfo()
	repo := &store{row: heldShipment(tenant, "Holiday pickup")}
	audit := &auditLog{}
	svc := newService(repo, audit)
	req := &ratingreviewports.ApproveRatingReviewRequest{
		ShipmentID: repo.row.ID,
		TenantInfo: tenant,
	}

	_, err := svc.Approve(t.Context(), req, person(tenant))
	require.NoError(t, err)
	again, err := svc.Approve(t.Context(), req, person(tenant))
	require.NoError(t, err)

	assert.False(t, shipmentrating.AwaitsReview(again))
	assert.Equal(t, 1, repo.saves)
	assert.Len(t, audit.entries, 1)
}

func TestApprove_RefusesARatingWithNothingToReview(t *testing.T) {
	t.Parallel()

	tenant := tenantInfo()
	for _, row := range []*shipment.Shipment{
		heldShipment(tenant),
		{ID: pulid.MustNew("shp_"), OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID},
	} {
		repo := &store{row: row}
		_, err := newService(repo, &auditLog{}).Approve(
			t.Context(),
			&ratingreviewports.ApproveRatingReviewRequest{ShipmentID: row.ID, TenantInfo: tenant},
			person(tenant),
		)

		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.Zero(t, repo.saves)
	}
}

func TestApprove_IsAPersonsDecision(t *testing.T) {
	t.Parallel()

	tenant := tenantInfo()
	repo := &store{row: heldShipment(tenant, "Holiday pickup")}
	agentID := pulid.MustNew("agt_")
	req := &ratingreviewports.ApproveRatingReviewRequest{
		ShipmentID: repo.row.ID,
		TenantInfo: tenant,
	}

	for _, actor := range []*permissionports.RequestActor{
		nil,
		{PrincipalType: permissionports.PrincipalTypeAgent, PrincipalID: agentID, UserID: agentID},
		{PrincipalType: permissionports.PrincipalTypeAPIKey, PrincipalID: pulid.MustNew("key_")},
	} {
		_, err := newService(repo, &auditLog{}).Approve(t.Context(), req, actor)

		var denied *errortypes.AuthorizationError
		require.ErrorAs(t, err, &denied)
	}
	assert.Zero(t, repo.saves)
	assert.Empty(t, repo.reads)
}

func TestApprove_ReturnsTheSaveErrorAndAuditsNothing(t *testing.T) {
	t.Parallel()

	tenant := tenantInfo()
	conflict := errors.New("version conflict")
	repo := &store{row: heldShipment(tenant, "Holiday pickup"), failOn: conflict}
	audit := &auditLog{}

	_, err := newService(repo, audit).Approve(
		t.Context(),
		&ratingreviewports.ApproveRatingReviewRequest{ShipmentID: repo.row.ID, TenantInfo: tenant},
		person(tenant),
	)

	require.ErrorIs(t, err, conflict)
	assert.Empty(t, audit.entries)
	assert.True(t, shipmentrating.AwaitsReview(repo.row), "the stored rating is still held")
}
