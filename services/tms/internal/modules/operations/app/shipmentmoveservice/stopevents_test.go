package shipmentmoveservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopactual"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentmoveportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingShipmentEvents struct {
	recorded []*shipmenteventports.RecordShipmentEventParams
}

func (r *recordingShipmentEvents) Record(
	_ context.Context,
	params *shipmenteventports.RecordShipmentEventParams,
) error {
	r.recorded = append(r.recorded, params)
	return nil
}

func (r *recordingShipmentEvents) List(
	_ context.Context,
	_ *shipmenteventports.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, nil
}

func (r *recordingShipmentEvents) types() []shipmentevent.Type {
	types := make([]shipmentevent.Type, 0, len(r.recorded))
	for _, params := range r.recorded {
		types = append(types, params.Type)
	}
	return types
}

type quietDashControl struct{}

func (quietDashControl) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*dashcontrol.DashControl, error) {
	return &dashcontrol.DashControl{}, nil
}

func (quietDashControl) Update(
	_ context.Context,
	entity *dashcontrol.DashControl,
) (*dashcontrol.DashControl, error) {
	return entity, nil
}

type stopEventFixture struct {
	tenantInfo pagination.TenantInfo
	move       *shipment.ShipmentMove
	pickup     *shipment.Stop
	drop       *shipment.Stop
	final      *shipment.Stop
}

func newStopEventFixture() *stopEventFixture {
	arrival, departure := int64(1_000), int64(2_000)
	shipmentID := pulid.MustNew("shp_")
	moveID := pulid.MustNew("sm_")
	pickup := &shipment.Stop{
		ID:              pulid.MustNew("stp_"),
		ShipmentMoveID:  moveID,
		Type:            shipmentvocab.StopTypePickup,
		Status:          shipmentvocab.StopStatusCompleted,
		Sequence:        0,
		ActualArrival:   &arrival,
		ActualDeparture: &departure,
		Location:        &location.Location{Name: "Cargill Protein Plant"},
	}
	canceled := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeDelivery,
		Status:         shipmentvocab.StopStatusCanceled,
		Sequence:       1,
	}
	drop := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeSplitDelivery,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       2,
		AddressLine:    "400 Commerce St",
	}
	final := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeDelivery,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       3,
		Location:       &location.Location{Name: "Dallas Terminal"},
	}
	return &stopEventFixture{
		tenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		move: &shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipmentvocab.MoveStatusInTransit,
			Stops:      []*shipment.Stop{final, drop, canceled, pickup},
		},
		pickup: pickup,
		drop:   drop,
		final:  final,
	}
}

func (f *stopEventFixture) service(t *testing.T) (*service, *recordingShipmentEvents) {
	t.Helper()

	moveRepo := shipmentmoveportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentmoveports.GetMoveByIDRequest) bool {
			return req.MoveID == f.move.ID
		})).
		Return(f.move, nil).
		Maybe()
	moveRepo.EXPECT().
		UpdateStopActuals(mock.Anything, f.tenantInfo, mock.AnythingOfType("*shipment.Stop")).
		RunAndReturn(func(
			_ context.Context,
			_ pagination.TenantInfo,
			stop *shipment.Stop,
		) (*shipment.Stop, error) {
			return stop, nil
		}).
		Maybe()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			context.Context,
			*shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return &shipment.Shipment{
				ID:                f.move.ShipmentID,
				OrganizationID:    f.tenantInfo.OrgID,
				BusinessUnitID:    f.tenantInfo.BuID,
				CustomerID:        pulid.MustNew("cus_"),
				FormulaTemplateID: pulid.MustNew("ft_"),
				Status:            shipmentvocab.StatusInTransit,
				Version:           1,
				Moves:             []*shipment.ShipmentMove{f.move},
			}, nil
		}).
		Maybe()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			return entity, nil
		}).
		Maybe()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.Anything).
		Return(&shipmentcontrol.ShipmentControl{}, nil).
		Maybe()

	events := &recordingShipmentEvents{}
	return &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentportsmock.NewMockAssignmentRepository(t),
		shipmentRepo:   shipmentRepo,
		holdRepo:       shipmentholdportsmock.NewMockShipmentHoldRepository(t),
		controlRepo:    controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Features:           plantest.AllFeatures(),
			FuelSurcharge:      ratingtest.NoFuelSurcharge{},
			Hydrator:           ratingtest.NoHydration{},
			BillingControlRepo: ratingtest.NoBillingControl{},
			Logger:             zap.NewNop(),
			RateEngine:         ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo:    accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService:    events,
		dashControlRepo: quietDashControl{},
		coordinator:     shipmentstateCoordinator(),
	}, events
}

func TestRecordStopActual_RecordsTheStopEventOnce(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	svc, events := f.service(t)
	arrived := int64(3_000)
	req := &shipmentmoveports.RecordStopActualRequest{
		TenantInfo: f.tenantInfo,
		MoveID:     f.move.ID,
		StopID:     f.drop.ID,
		Action:     shipmentmoveports.StopActualActionArrive,
		OccurredAt: &arrived,
	}

	_, err := svc.RecordStopActual(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, []shipmentevent.Type{shipmentevent.TypeStopArrived}, events.types())
	assert.Equal(t, f.drop.ID, events.recorded[0].StopID)

	_, err = svc.RecordStopActual(t.Context(), req)
	require.Error(t, err, "the same arrival twice is refused")
	assert.Len(t, events.recorded, 1, "and records nothing")
}

func TestRecordStopActual_ArriveAndDepartRecordsBoth(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	svc, events := f.service(t)
	arrived, departed := int64(3_000), int64(3_600)

	_, err := svc.RecordStopActual(t.Context(), &shipmentmoveports.RecordStopActualRequest{
		TenantInfo: f.tenantInfo,
		MoveID:     f.move.ID,
		StopID:     f.drop.ID,
		Action:     shipmentmoveports.StopActualActionArrive,
		OccurredAt: &arrived,
		DepartedAt: &departed,
	})

	require.NoError(t, err)
	assert.Equal(t, []shipmentevent.Type{
		shipmentevent.TypeStopArrived,
		shipmentevent.TypeStopDeparted,
	}, events.types())
}

func TestReconcileStopActuals_RecordsCorrectionsAndNothingForASettledPlan(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	svc, events := f.service(t)
	corrected := int64(1_500)
	plan := func(_ context.Context, move *shipment.ShipmentMove) ([]stopactual.Change, error) {
		for _, stop := range move.Stops {
			if stop.ID == f.pickup.ID && *stop.ActualArrival != corrected {
				return []stopactual.Change{{StopID: stop.ID, Arrival: &corrected}}, nil
			}
		}
		return nil, nil
	}
	req := &shipmentmoveports.ReconcileStopActualsRequest{
		TenantInfo: f.tenantInfo,
		MoveID:     f.move.ID,
		Plan:       plan,
	}

	_, err := svc.ReconcileStopActuals(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, []shipmentevent.Type{shipmentevent.TypeStopArrivalCorrected}, events.types())
	assert.Equal(t, int64(1_000),
		events.recorded[0].Metadata[shipmentevent.MetadataPreviousActualAt])

	_, err = svc.ReconcileStopActuals(t.Context(), req)
	require.NoError(t, err)
	assert.Len(t, events.recorded, 1, "reconciling to the same times records nothing")
}

func TestMarkFromStopActual_TagsTheMoveEvent(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	params := shipmenteventservice.MarkFromStopActual(shipmenteventservice.BuildMoveStatusChanged(
		tenantRefForMoveTenant(f.tenantInfo),
		f.move,
		shipmentvocab.MoveStatusAssigned,
		actorForMoveTenant(f.tenantInfo),
	))
	event := &shipmentevent.Event{Type: params.Type, Metadata: params.Metadata}

	assert.Equal(t, shipmentevent.TypeMoveDeparted, event.Type)
	assert.True(t, event.FromStopActual())
	assert.Equal(t, string(shipmentvocab.MoveStatusInTransit), params.Metadata["newStatus"])
	assert.False(t, (&shipmentevent.Event{Type: params.Type}).FromStopActual())
}

func TestReconcileStopActuals_TagsTheStopEventsWithTheOrigin(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	svc, events := f.service(t)
	arrived := int64(3_000)
	origin := &shipmentmoveports.StopActualEventOrigin{
		ActorLabel:    "Internal EDI",
		CorrelationID: "corr-1",
		Metadata: map[string]any{
			"ediMirroredFromEventId": "se_source",
			"ediShipmentLinkId":      "edislink_1",
		},
	}
	req := &shipmentmoveports.ReconcileStopActualsRequest{
		TenantInfo: f.tenantInfo,
		MoveID:     f.move.ID,
		Origin:     origin,
		Plan: func(_ context.Context, move *shipment.ShipmentMove) ([]stopactual.Change, error) {
			for _, stop := range move.Stops {
				if stop.ID == f.drop.ID && stop.ActualArrival == nil {
					return []stopactual.Change{{StopID: stop.ID, Arrival: &arrived}}, nil
				}
			}
			return nil, nil
		},
	}

	_, err := svc.ReconcileStopActuals(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, []shipmentevent.Type{shipmentevent.TypeStopArrived}, events.types())
	event := events.recorded[0]
	assert.Equal(t, f.drop.ID, event.StopID)
	assert.Equal(t, "Internal EDI", event.ActorLabel)
	assert.Equal(t, "corr-1", event.CorrelationID)
	assert.Equal(t, "se_source", event.Metadata["ediMirroredFromEventId"])
	assert.Equal(t, "edislink_1", event.Metadata["ediShipmentLinkId"])
	assert.Equal(t, arrived, event.Metadata[shipmentevent.MetadataActualAt])

	_, err = svc.ReconcileStopActuals(t.Context(), req)
	require.NoError(t, err)
	assert.Len(t, events.recorded, 1, "a synced time already in place records nothing")
}

func TestWithEventOrigin_TagsTheMoveEventAndKeepsItsMetadata(t *testing.T) {
	t.Parallel()

	f := newStopEventFixture()
	params := withEventOrigin(
		shipmenteventservice.MarkFromStopActual(shipmenteventservice.BuildMoveStatusChanged(
			tenantRefForMoveTenant(f.tenantInfo),
			f.move,
			shipmentvocab.MoveStatusAssigned,
			actorForMoveTenant(f.tenantInfo),
		)),
		&shipmentmoveports.StopActualEventOrigin{
			ActorLabel: "Internal EDI",
			Metadata:   map[string]any{"ediMirroredFromEventId": "se_source"},
		},
	)

	assert.Equal(t, "se_source", params.Metadata["ediMirroredFromEventId"])
	fromStopActual, _ := params.Metadata[shipmentevent.MetadataFromStopActual].(bool)
	assert.True(t, fromStopActual)
	assert.Equal(t, string(shipmentvocab.MoveStatusInTransit), params.Metadata["newStatus"])
	assert.Equal(t, "Internal EDI", params.ActorLabel)
	assert.Empty(t, params.CorrelationID)
	assert.Same(t, params, withEventOrigin(params, nil))
}
