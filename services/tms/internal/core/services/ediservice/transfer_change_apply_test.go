package ediservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/internaledilifecycle"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestService_ApplyTransferChange_CancelsLinkedShipment(t *testing.T) {
	t.Parallel()

	fixture := newTransferChangeApplyFixture(t, edi.TransferChangeDirectionSourceToTarget)
	change := fixture.pendingChange(edi.TransferChangeTypeShipmentCancel214, shipment.StatusCanceled)
	change.Payload["cancellationReason"] = "Customer canceled"
	fixture.target.Status = shipment.StatusInTransit

	fixture.expectLoadPendingChange(change)
	fixture.expectLoadLink(2)
	fixture.expectLoadShipments()
	fixture.shipmentRepo.EXPECT().
		Cancel(mock.Anything, mock.MatchedBy(func(req *repositories.CancelShipmentRequest) bool {
			return req.TenantInfo.OrgID == fixture.link.TargetOrganizationID &&
				req.TenantInfo.BuID == fixture.link.BusinessUnitID &&
				req.ShipmentID == fixture.link.TargetShipmentID &&
				req.CanceledByID == fixture.actor.UserID &&
				req.CancelReason == "Customer canceled"
		})).
		Return(fixture.canceledTarget(), nil).
		Once()
	fixture.expectInsertMirroredEvent(shipmentevent.TypeShipmentCanceled)
	fixture.expectUpdateReviewedChange(edi.TransferChangeStatusApplied)

	updated, err := fixture.service.ApplyTransferChange(
		t.Context(),
		&TransferChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
		},
		fixture.actor,
	)

	require.NoError(t, err)
	require.Equal(t, edi.TransferChangeStatusApplied, updated.Status)
	require.Equal(t, fixture.actor.UserID, updated.AppliedByID)
}

func TestService_RejectTransferChange_DoesNotCancelLinkedShipment(t *testing.T) {
	t.Parallel()

	fixture := newTransferChangeApplyFixture(t, edi.TransferChangeDirectionSourceToTarget)
	change := fixture.pendingChange(edi.TransferChangeTypeShipmentCancel214, shipment.StatusCanceled)

	fixture.expectLoadPendingChange(change)
	fixture.expectLoadLink(1)
	fixture.expectUpdateReviewedChange(edi.TransferChangeStatusRejected)

	updated, err := fixture.service.RejectTransferChange(
		t.Context(),
		&TransferChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
			Reason:     "Rejected by reviewer",
		},
		fixture.actor,
	)

	require.NoError(t, err)
	require.Equal(t, edi.TransferChangeStatusRejected, updated.Status)
	require.Equal(t, "Rejected by reviewer", updated.FailureReason)
}

func TestService_ApplyTransferChange_StatusRegressionAppliesLinkedShipmentStatus(t *testing.T) {
	t.Parallel()

	fixture := newTransferChangeApplyFixture(t, edi.TransferChangeDirectionTargetToSource)
	change := fixture.pendingChange(edi.TransferChangeTypeShipmentStatus214, shipment.StatusInTransit)
	fixture.source.Status = shipment.StatusNew
	fixture.target.Status = shipment.StatusInTransit

	fixture.expectLoadPendingChange(change)
	fixture.expectLoadLink(2)
	fixture.expectLoadShipments()
	fixture.shipmentRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *repositories.UpdateShipmentStatusRequest) bool {
			return req.TenantInfo.OrgID == fixture.link.SourceOrganizationID &&
				req.TenantInfo.BuID == fixture.link.BusinessUnitID &&
				req.ShipmentID == fixture.link.SourceShipmentID &&
				req.Status == shipment.StatusInTransit &&
				req.Version == fixture.source.Version
		})).
		Return(fixture.statusUpdatedSource(shipment.StatusInTransit), nil).
		Once()
	fixture.expectInsertMirroredEvent(shipmentevent.TypeStatusChanged)
	fixture.expectUpdateReviewedChange(edi.TransferChangeStatusApplied)

	updated, err := fixture.service.ApplyTransferChange(
		t.Context(),
		&TransferChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
		},
		fixture.actor,
	)

	require.NoError(t, err)
	require.Equal(t, edi.TransferChangeStatusApplied, updated.Status)
}

func TestService_ApplyTransferChange_LifecycleAppliesActualsThroughCoordinator(t *testing.T) {
	t.Parallel()

	fixture := newTransferChangeApplyFixture(t, edi.TransferChangeDirectionTargetToSource)
	fixture.installLifecycleStops()
	fixture.source.Status = shipment.StatusNew
	fixture.target.Status = shipment.StatusInTransit
	arrival := int64(1_000)
	departure := int64(1_100)
	fixture.target.Moves[0].Stops[0].ActualArrival = &arrival
	fixture.target.Moves[0].Stops[0].ActualDeparture = &departure
	change := fixture.pendingChange(edi.TransferChangeTypeShipmentLifecycle214, shipment.StatusInTransit)

	fixture.expectLoadPendingChange(change)
	fixture.expectLoadLink(2)
	fixture.expectLoadShipments()
	fixture.shipmentRepo.EXPECT().
		UpdateOperationalLifecycle(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity.ID == fixture.link.SourceShipmentID &&
				entity.Status == shipment.StatusInTransit &&
				entity.Moves[0].Status == shipment.MoveStatusInTransit &&
				entity.Moves[0].Stops[0].Status == shipment.StopStatusCompleted &&
				entity.Moves[0].Stops[0].ActualArrival != nil &&
				*entity.Moves[0].Stops[0].ActualArrival == arrival &&
				entity.ActualShipDate != nil &&
				*entity.ActualShipDate == departure
		})).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			return entity, nil
		}).
		Once()
	fixture.expectInsertMirroredEvent(shipmentevent.TypeStatusChanged)
	fixture.expectUpdateReviewedChange(edi.TransferChangeStatusApplied)

	updated, err := fixture.service.ApplyTransferChange(
		t.Context(),
		&TransferChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
		},
		fixture.actor,
	)

	require.NoError(t, err)
	require.Equal(t, edi.TransferChangeStatusApplied, updated.Status)
}

type transferChangeApplyFixture struct {
	service            *Service
	link               *edi.ShipmentLink
	source             *shipment.Shipment
	target             *shipment.Shipment
	actor              *services.RequestActor
	direction          edi.TransferChangeDirection
	transferChangeRepo *mocks.MockEDITransferChangeRepository
	shipmentLinkRepo   *mocks.MockEDIShipmentLinkRepository
	shipmentRepo       *mocks.MockShipmentRepository
	eventRepo          *mocks.MockShipmentEventRepository
}

func newTransferChangeApplyFixture(
	t *testing.T,
	direction edi.TransferChangeDirection,
) *transferChangeApplyFixture {
	t.Helper()

	sourceOrgID := pulid.MustNew("org_")
	targetOrgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	sourceID := pulid.MustNew("shp_")
	targetID := pulid.MustNew("shp_")

	fixture := &transferChangeApplyFixture{
		link: &edi.ShipmentLink{
			ID:                   pulid.MustNew("edislink_"),
			BusinessUnitID:       buID,
			SourceOrganizationID: sourceOrgID,
			TargetOrganizationID: targetOrgID,
			SourceShipmentID:     sourceID,
			TargetShipmentID:     targetID,
			SyncPolicy:           edi.ShipmentSyncPolicyManualReview,
			Status:               edi.ShipmentLinkStatusActive,
		},
		source: &shipment.Shipment{
			ID:             sourceID,
			OrganizationID: sourceOrgID,
			BusinessUnitID: buID,
			ProNumber:      "SOURCE-1",
			Status:         shipment.StatusCanceled,
			Version:        3,
		},
		target: &shipment.Shipment{
			ID:             targetID,
			OrganizationID: targetOrgID,
			BusinessUnitID: buID,
			ProNumber:      "TARGET-1",
			Status:         shipment.StatusNew,
			Version:        5,
		},
		actor: &services.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			PrincipalType:  services.PrincipalTypeUser,
			OrganizationID: sourceOrgID,
			BusinessUnitID: buID,
		},
		direction:          direction,
		transferChangeRepo: mocks.NewMockEDITransferChangeRepository(t),
		shipmentLinkRepo:   mocks.NewMockEDIShipmentLinkRepository(t),
		shipmentRepo:       mocks.NewMockShipmentRepository(t),
		eventRepo:          mocks.NewMockShipmentEventRepository(t),
	}
	fixture.service = &Service{
		db:                 transferChangeApplyDB{},
		transferChangeRepo: fixture.transferChangeRepo,
		shipmentLinkRepo:   fixture.shipmentLinkRepo,
		shipmentRepo:       fixture.shipmentRepo,
		shipmentEventRepo:  fixture.eventRepo,
		coordinator:        shipmentstate.NewCoordinatorWithClock(func() int64 { return 2_000 }),
	}
	fixture.service.lifecycleApplier = internaledilifecycle.New(internaledilifecycle.Params{
		ShipmentRepo: fixture.shipmentRepo,
		Coordinator:  fixture.service.coordinator,
	})
	return fixture
}

func (f *transferChangeApplyFixture) pendingChange(
	changeType string,
	nextStatus shipment.Status,
) *edi.TransferChange {
	return &edi.TransferChange{
		ID:             pulid.MustNew("editc_"),
		BusinessUnitID: f.link.BusinessUnitID,
		ShipmentLinkID: f.link.ID,
		Direction:      f.direction,
		ChangeType:     changeType,
		Status:         edi.TransferChangeStatusPendingReview,
		ConflictStatus: edi.TransferChangeConflictNone,
		IdempotencyKey: pulid.MustNew("idk_").String(),
		Payload: map[string]any{
			"sourceEventId": pulid.MustNew("se_").String(),
			"newStatus":     string(nextStatus),
		},
	}
}

func (f *transferChangeApplyFixture) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: f.link.SourceOrganizationID,
		BuID:  f.link.BusinessUnitID,
	}
}

func (f *transferChangeApplyFixture) expectLoadPendingChange(change *edi.TransferChange) {
	f.transferChangeRepo.EXPECT().
		GetTransferChangeByID(
			mock.Anything,
			repositories.GetEDITransferChangeByIDRequest{
				ID:         change.ID,
				TenantInfo: f.tenantInfo(),
			},
		).
		Return(change, nil).
		Once()
}

func (f *transferChangeApplyFixture) expectLoadLink(times int) {
	f.shipmentLinkRepo.EXPECT().
		GetShipmentLinkByID(
			mock.Anything,
			repositories.GetEDIShipmentLinkByIDRequest{
				ID:         f.link.ID,
				TenantInfo: f.tenantInfo(),
			},
		).
		Return(f.link, nil).
		Times(times)
}

func (f *transferChangeApplyFixture) expectLoadShipments() {
	f.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == f.link.SourceShipmentID &&
				req.TenantInfo.OrgID == f.link.SourceOrganizationID
		})).
		Return(f.source, nil).
		Once()
	f.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == f.link.TargetShipmentID &&
				req.TenantInfo.OrgID == f.link.TargetOrganizationID
		})).
		Return(f.target, nil).
		Once()
}

func (f *transferChangeApplyFixture) expectInsertMirroredEvent(eventType shipmentevent.Type) {
	f.eventRepo.EXPECT().
		Insert(mock.Anything, mock.MatchedBy(func(event *shipmentevent.Event) bool {
			return event.Type == eventType &&
				event.ActorType == shipmentevent.ActorEDI &&
				event.Metadata[edi.InternalEDIMirroredFromEventIDKey] != "" &&
				event.Metadata[edi.InternalEDIShipmentLinkIDKey] == f.link.ID.String()
		})).
		Return(nil).
		Once()
}

func (f *transferChangeApplyFixture) expectUpdateReviewedChange(status edi.TransferChangeStatus) {
	f.transferChangeRepo.EXPECT().
		UpdateTransferChange(mock.Anything, mock.MatchedBy(func(change *edi.TransferChange) bool {
			return change.Status == status &&
				change.ReviewedByID == f.actor.UserID &&
				change.ReviewedAt != nil
		})).
		RunAndReturn(func(_ context.Context, change *edi.TransferChange) (*edi.TransferChange, error) {
			return change, nil
		}).
		Once()
}

func (f *transferChangeApplyFixture) canceledTarget() *shipment.Shipment {
	updated := *f.target
	updated.Status = shipment.StatusCanceled
	updated.CancelReason = "Customer canceled"
	return &updated
}

func (f *transferChangeApplyFixture) statusUpdatedSource(status shipment.Status) *shipment.Shipment {
	updated := *f.source
	updated.Status = status
	return &updated
}

func (f *transferChangeApplyFixture) installLifecycleStops() {
	f.source.Moves = []*shipment.ShipmentMove{
		transferLifecycleMove(f.source),
	}
	f.target.Moves = []*shipment.ShipmentMove{
		transferLifecycleMove(f.target),
	}
}

func transferLifecycleMove(entity *shipment.Shipment) *shipment.ShipmentMove {
	moveID := pulid.MustNew("sm_")
	return &shipment.ShipmentMove{
		ID:             moveID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ID,
		Status:         shipment.MoveStatusNew,
		Sequence:       0,
		Stops: []*shipment.Stop{
			transferLifecycleStop(entity, moveID, shipment.StopTypePickup, 0),
			transferLifecycleStop(entity, moveID, shipment.StopTypeDelivery, 1),
		},
	}
}

func transferLifecycleStop(
	entity *shipment.Shipment,
	moveID pulid.ID,
	stopType shipment.StopType,
	sequence int64,
) *shipment.Stop {
	return &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		ShipmentMoveID:       moveID,
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipment.StopStatusNew,
		Type:                 stopType,
		Sequence:             sequence,
		ScheduledWindowStart: 500 + sequence,
	}
}

type transferChangeApplyDB struct{}

func (transferChangeApplyDB) DB() *bun.DB                          { return nil }
func (transferChangeApplyDB) DBForContext(context.Context) bun.IDB { return nil }
func (transferChangeApplyDB) HealthCheck(context.Context) error    { return nil }
func (transferChangeApplyDB) IsHealthy(context.Context) bool       { return true }
func (transferChangeApplyDB) Close() error                         { return nil }
func (transferChangeApplyDB) WithTx(
	ctx context.Context,
	_ coreports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
