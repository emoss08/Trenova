package shipmentholdservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/ports/holdreasonports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/holdreasonportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
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

	repo := shipmentholdportsmock.NewMockShipmentHoldRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	holdReasonRepo := holdreasonportsmock.NewMockHoldReasonRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		EventService:   noopShipmentEventService{},
		Realtime:       realtime,
	})

	shipmentID := pulid.MustNew("shp_")
	reasonID := pulid.MustNew("hr_")
	startedAt := int64(200)
	overrideSeverity := holdreason.HoldSeverityBlocking
	overrideVisible := true

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	holdReasonRepo.EXPECT().
		GetByID(mock.Anything, holdreasonports.GetHoldReasonByIDRequest{ID: reasonID, TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID}}).
		Return(&holdreason.HoldReason{ID: reasonID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Code: "APPT_PENDING", Active: true, DefaultSeverity: holdreason.HoldSeverityAdvisory, DefaultBlocksDispatch: true, DefaultBlocksDelivery: false, DefaultBlocksBilling: false, DefaultVisibleToCustomer: false}, nil).
		Once()

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *shipmenthold.ShipmentHold) bool {
			return entity.ShipmentID == shipmentID &&
				entity.Type == holdreason.HoldTypeOperational &&
				entity.ReasonCode == "APPT_PENDING" &&
				entity.Severity == overrideSeverity &&
				entity.VisibleToCustomer == overrideVisible &&
				entity.BlocksDispatch &&
				entity.Source == shipmenthold.HoldSourceUser &&
				entity.StartedAt == startedAt &&
				entity.CreatedByID != nil &&
				*entity.CreatedByID == testutil.TestUserID
		})).
		RunAndReturn(func(_ context.Context, entity *shipmenthold.ShipmentHold) (*shipmenthold.ShipmentHold, error) {
			entity.ID = pulid.MustNew("shh_")
			return entity, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything).
		Run(func(params *auditports.LogActionParams, _ ...auditports.LogOption) {
			assert.Equal(t, permission.ResourceShipmentHold, params.Resource)
		}).
		Return(nil).Once()

	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
			return req.Resource == permission.ResourceShipmentHold.String() &&
				req.Action == "created" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	created, err := svc.Create(t.Context(), &shipmentholdports.CreateShipmentHoldRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		},
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

	repo := shipmentholdportsmock.NewMockShipmentHoldRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	holdReasonRepo := holdreasonportsmock.NewMockHoldReasonRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		EventService:   noopShipmentEventService{},
		Realtime:       realtime,
	})

	holdID := pulid.MustNew("shh_")
	shipmentID := pulid.MustNew("shp_")
	releasedAt := int64(500)

	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentholdports.GetShipmentHoldByIDRequest")).
		Return(&shipmenthold.ShipmentHold{ID: holdID, ShipmentID: shipmentID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Severity: holdreason.HoldSeverityAdvisory, Source: shipmenthold.HoldSourceUser, StartedAt: 100, ReleasedAt: &releasedAt, Version: 1}, nil).
		Once()

	updated, err := svc.Update(
		t.Context(),
		&shipmentholdports.UpdateShipmentHoldRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: testutil.TestOrgID,
				BuID:  testutil.TestBuID,
			},
			HoldID:            holdID,
			ShipmentID:        shipmentID,
			Severity:          holdreason.HoldSeverityBlocking,
			BlocksDispatch:    true,
			BlocksDelivery:    false,
			BlocksBilling:     false,
			VisibleToCustomer: false,
			StartedAt:         100,
			Version:           1,
		},
		testHoldActor(),
	)

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestRelease_SetsReleasedFieldsAuditsAndPublishes(t *testing.T) {
	t.Parallel()

	repo := shipmentholdportsmock.NewMockShipmentHoldRepository(t)
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	holdReasonRepo := holdreasonportsmock.NewMockHoldReasonRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		ShipmentRepo:   shipmentRepo,
		HoldReasonRepo: holdReasonRepo,
		AuditService:   audit,
		EventService:   noopShipmentEventService{},
		Realtime:       realtime,
	})

	holdID := pulid.MustNew("shh_")
	shipmentID := pulid.MustNew("shp_")

	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentholdports.GetShipmentHoldByIDRequest")).
		Return(&shipmenthold.ShipmentHold{ID: holdID, ShipmentID: shipmentID, OrganizationID: testutil.TestOrgID, BusinessUnitID: testutil.TestBuID, Type: holdreason.HoldTypeOperational, Severity: holdreason.HoldSeverityBlocking, Source: shipmenthold.HoldSourceUser, StartedAt: 100, Version: 1}, nil).
		Once()

	repo.EXPECT().
		Release(mock.Anything, mock.MatchedBy(func(entity *shipmenthold.ShipmentHold) bool {
			return entity.ReleasedAt != nil && entity.ReleasedByID != nil &&
				*entity.ReleasedByID == testutil.TestUserID
		})).
		RunAndReturn(func(_ context.Context, entity *shipmenthold.ShipmentHold) (*shipmenthold.ShipmentHold, error) {
			return entity, nil
		}).
		Once()

	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).
		Run(func(params *auditports.LogActionParams, _ ...auditports.LogOption) {
			assert.Equal(t, permission.ResourceShipmentHold, params.Resource)
			assert.Equal(t, holdID.String(), params.ResourceID)
		}).
		Return(nil).Once()

	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
			return req.Resource == permission.ResourceShipmentHold.String() &&
				req.Action == "released" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	released, err := svc.Release(
		t.Context(),
		&shipmentholdports.ReleaseShipmentHoldRequest{
			TenantInfo: pagination.TenantInfo{OrgID: testutil.TestOrgID, BuID: testutil.TestBuID},
			HoldID:     holdID,
			ShipmentID: shipmentID,
		},
		testHoldActor(),
	)

	require.NoError(t, err)
	require.NotNil(t, released.ReleasedAt)
	require.NotNil(t, released.ReleasedByID)
	assert.Equal(t, testutil.TestUserID, *released.ReleasedByID)
}

func testHoldActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		UserID:        testutil.TestUserID,
		PrincipalID:   testutil.TestUserID,
		PrincipalType: permissionports.PrincipalTypeUser,
	}
}
