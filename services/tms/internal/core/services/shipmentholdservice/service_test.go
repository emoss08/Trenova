package shipmentholdservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreate_DerivesFromHoldReasonAuditsAndPublishes(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentHoldRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	holdReasonRepo := mocks.NewMockHoldReasonRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		Realtime:       realtime,
	})

	shipmentID := pulid.MustNew("shp_")
	reasonID := pulid.MustNew("hr_")
	startedAt := int64(200)
	overrideSeverity := holdreason.HoldSeverityBlocking
	overrideVisible := true

	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
		return req.ID == shipmentID && req.TenantInfo.OrgID == testutil.TestOrgID && req.TenantInfo.BuID == testutil.TestBuID
	})).Return(&shipment.Shipment{ID: shipmentID}, nil).Once()

	holdReasonRepo.EXPECT().GetByID(mock.Anything, repositories.GetHoldReasonByIDRequest{ID: reasonID, TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}}).
		Return(&holdreason.HoldReason{ID: reasonID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Code: "APPT_PENDING", Active: true, DefaultSeverity: holdreason.HoldSeverityAdvisory, DefaultBlocksDispatch: true, DefaultBlocksDelivery: false, DefaultBlocksBilling: false, DefaultVisibleToCustomer: false}, nil).
		Once()

	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entity *shipment.ShipmentHold) bool {
		return entity.ShipmentID == shipmentID && entity.Type == holdreason.HoldTypeOperational && entity.ReasonCode == "APPT_PENDING" && entity.Severity == overrideSeverity && entity.VisibleToCustomer == overrideVisible && entity.BlocksDispatch && entity.Source == shipment.HoldSourceUser && entity.StartedAt == startedAt && entity.CreatedByID != nil && *entity.CreatedByID == testutil.TestUserID
	})).RunAndReturn(func(_ context.Context, entity *shipment.ShipmentHold) (*shipment.ShipmentHold, error) {
		entity.ID = pulid.MustNew("shh_")
		return entity, nil
	}).Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *servicesport.LogActionParams, _ ...servicesport.LogOption) {
			assert.Equal(t, permission.ResourceShipmentHold, params.Resource)
		}).
		Return(nil).Once()

	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
		return req.Resource == permission.ResourceShipmentHold.String() && req.Action == "created" && req.RecordID == shipmentID
	})).Return(nil).Once()

	created, err := svc.Create(t.Context(), &repositories.CreateShipmentHoldRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID},
		ShipmentID:        shipmentID,
		HoldReasonID:      reasonID,
		Severity:          &overrideSeverity,
		VisibleToCustomer: &overrideVisible,
		StartedAt:         &startedAt,
		Notes:             "  dock issue  ",
	}, testHoldActor())

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "dock issue", created.Notes)
}

func TestUpdate_RejectsReleasedHold(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentHoldRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	holdReasonRepo := mocks.NewMockHoldReasonRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		Realtime:       realtime,
	})

	holdID := pulid.MustNew("shh_")
	shipmentID := pulid.MustNew("shp_")
	releasedAt := int64(500)

	repo.EXPECT().GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentHoldByIDRequest")).Return(&shipment.ShipmentHold{ID: holdID, ShipmentID: shipmentID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Severity: holdreason.HoldSeverityAdvisory, Source: shipment.HoldSourceUser, StartedAt: 100, ReleasedAt: &releasedAt, Version: 1}, nil).Once()

	updated, err := svc.Update(t.Context(), &repositories.UpdateShipmentHoldRequest{TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}, HoldID: holdID, ShipmentID: shipmentID, Severity: holdreason.HoldSeverityBlocking, BlocksDispatch: true, BlocksDelivery: false, BlocksBilling: false, VisibleToCustomer: false, StartedAt: 100, Version: 1}, testHoldActor())

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestRelease_SetsReleasedFieldsAuditsAndPublishes(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentHoldRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	holdReasonRepo := mocks.NewMockHoldReasonRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		Realtime:       realtime,
	})

	holdID := pulid.MustNew("shh_")
	shipmentID := pulid.MustNew("shp_")

	repo.EXPECT().GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentHoldByIDRequest")).Return(&shipment.ShipmentHold{ID: holdID, ShipmentID: shipmentID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Severity: holdreason.HoldSeverityBlocking, Source: shipment.HoldSourceUser, StartedAt: 100, Version: 1}, nil).Once()

	repo.EXPECT().Release(mock.Anything, mock.MatchedBy(func(entity *shipment.ShipmentHold) bool {
		return entity.ReleasedAt != nil && entity.ReleasedByID != nil && *entity.ReleasedByID == testutil.TestUserID
	})).RunAndReturn(func(_ context.Context, entity *shipment.ShipmentHold) (*shipment.ShipmentHold, error) {
		return entity, nil
	}).Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).
		Run(func(params *servicesport.LogActionParams, _ ...servicesport.LogOption) {
			assert.Equal(t, permission.ResourceShipmentHold, params.Resource)
			assert.Equal(t, holdID.String(), params.ResourceID)
		}).
		Return(nil).Once()

	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *servicesport.PublishResourceInvalidationRequest) bool {
		return req.Resource == permission.ResourceShipmentHold.String() && req.Action == "released" && req.RecordID == shipmentID
	})).Return(nil).Once()

	released, err := svc.Release(t.Context(), &repositories.ReleaseShipmentHoldRequest{TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}, HoldID: holdID, ShipmentID: shipmentID}, testHoldActor())

	require.NoError(t, err)
	require.NotNil(t, released.ReleasedAt)
	require.NotNil(t, released.ReleasedByID)
	assert.Equal(t, testutil.TestUserID, *released.ReleasedByID)
}

func testHoldActor() *servicesport.RequestActor {
	return &servicesport.RequestActor{UserID: testutil.TestUserID, PrincipalID: testutil.TestUserID, PrincipalType: servicesport.PrincipalTypeUser}
}
