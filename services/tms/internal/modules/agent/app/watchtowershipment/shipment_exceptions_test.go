package watchtowershipment

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const exceptionNow = int64(1_800_000_000)

type fakeExceptionReader struct {
	stops   []*watchtowerports.MissedStop
	pods    []*watchtowerports.MissingPOD
	tenders []*watchtowerports.ExhaustedTender
	err     error

	stopRequests   []*watchtowerports.ListMissedStopsRequest
	podRequests    []*watchtowerports.ListMissingPODsRequest
	tenderRequests []*watchtowerports.ListExhaustedTendersRequest
	scoped         bool
}

func (f *fakeExceptionReader) ListMissedStops(
	ctx context.Context,
	req *watchtowerports.ListMissedStopsRequest,
) ([]*watchtowerports.MissedStop, error) {
	_, f.scoped = dbscope.TenantFrom(ctx)
	f.stopRequests = append(f.stopRequests, req)
	return f.stops, f.err
}

func (f *fakeExceptionReader) ListMissingPODs(
	_ context.Context,
	req *watchtowerports.ListMissingPODsRequest,
) ([]*watchtowerports.MissingPOD, error) {
	f.podRequests = append(f.podRequests, req)
	return f.pods, f.err
}

func (f *fakeExceptionReader) ListExhaustedTenders(
	_ context.Context,
	req *watchtowerports.ListExhaustedTendersRequest,
) ([]*watchtowerports.ExhaustedTender, error) {
	f.tenderRequests = append(f.tenderRequests, req)
	return f.tenders, f.err
}

type recordingReconciler struct {
	requests []*watchtowerports.ReconcileSubjectRequest
}

func (r *recordingReconciler) ReconcileSubject(
	_ context.Context,
	req *watchtowerports.ReconcileSubjectRequest,
) {
	r.requests = append(r.requests, req)
}

func exceptionParams(
	reader *fakeExceptionReader,
	control *dispatchcontrol.DispatchControl,
) ShipmentExceptionParams {
	return ShipmentExceptionParams{Reader: reader, Controls: &fakeEtaControls{control: control}}
}

func withClock(source watchtowerports.WatchtowerSource) watchtowerports.WatchtowerSource {
	if s, ok := source.(*ShipmentExceptionSource); ok {
		s.exceptions.now = func() int64 { return exceptionNow }
	}
	return source
}

func missedStop(cutoff int64) *watchtowerports.MissedStop {
	return &watchtowerports.MissedStop{
		StopID:       pulid.MustNew("stp_"),
		MoveID:       pulid.MustNew("smv_"),
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "PRO-7",
		OwnerID:      pulid.MustNew("usr_"),
		StopType:     shipment.StopTypePickup,
		Cutoff:       cutoff,
		LocationName: "Acme DC",
		City:         "Reno",
		StateCode:    "NV",
	}
}

func TestMissedPickupSource_AsksForPickupsPastTheGrace(t *testing.T) {
	t.Parallel()

	stop := missedStop(exceptionNow - 45*60)
	reader := &fakeExceptionReader{stops: []*watchtowerports.MissedStop{stop}}
	source := withClock(
		NewMissedPickupSource(exceptionParams(reader, &dispatchcontrol.DispatchControl{
			ExceptionMissedStopGraceMinutes: 15,
		})),
	)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	items, err := source.Snapshot(t.Context(), tenant)
	require.NoError(t, err)
	require.Len(t, reader.stopRequests, 1)
	assert.Equal(t, exceptionNow-15*60, reader.stopRequests[0].ClosedBefore)
	assert.Equal(
		t,
		[]shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup},
		reader.stopRequests[0].StopTypes,
	)

	require.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, watchtower.SourceMissedPickup, item.SourceKind)
	assert.Equal(t, stop.StopID.String(), item.SourceID)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, "Missed pickup on PRO-7", item.Title)
	assert.Contains(t, item.Summary, "Acme DC in Reno, NV")
	assert.Equal(t, agent.SubjectShipment, item.SubjectType)
	assert.Equal(t, stop.ShipmentID, item.SubjectID)
	assert.Equal(t, stop.OwnerID, item.OwnerID)
	assert.Equal(t, stop.Cutoff+15*60, item.OccurredAt)
}

func TestMissedDeliverySource_GoesCriticalLongPastTheGrace(t *testing.T) {
	t.Parallel()

	stop := missedStop(exceptionNow - 5*3600)
	stop.StopType = shipment.StopTypeDelivery
	reader := &fakeExceptionReader{stops: []*watchtowerports.MissedStop{stop}}
	source := withClock(NewMissedDeliverySource(exceptionParams(reader, nil)))

	items, err := source.Snapshot(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_")})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery},
		reader.stopRequests[0].StopTypes,
	)
	assert.Equal(
		t,
		exceptionNow-int64(dispatchcontrol.DefaultExceptionMissedStopGraceMinutes)*60,
		reader.stopRequests[0].ClosedBefore,
	)
	require.Len(t, items, 1)
	assert.Equal(t, watchtower.SeverityCritical, items[0].Severity)
	assert.Equal(t, "Missed delivery on PRO-7", items[0].Title)
}

func TestPodMissingSource_WaitsForTheGraceAndLooksBackNinetyDays(t *testing.T) {
	t.Parallel()

	row := &watchtowerports.MissingPOD{
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "PRO-9",
		CustomerName: "Globex",
		DeliveredAt:  exceptionNow - 30*3600,
	}
	reader := &fakeExceptionReader{pods: []*watchtowerports.MissingPOD{row}}
	source := withClock(
		NewPodMissingSource(exceptionParams(reader, &dispatchcontrol.DispatchControl{
			ExceptionPODGraceHours: 24,
		})),
	)

	items, err := source.Snapshot(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_")})
	require.NoError(t, err)
	require.Len(t, reader.podRequests, 1)
	assert.Equal(t, exceptionNow-24*3600, reader.podRequests[0].DeliveredBefore)
	assert.Equal(t, exceptionNow-90*24*3600, reader.podRequests[0].DeliveredAfter)
	require.Len(t, items, 1)
	assert.Equal(t, watchtower.SourcePodMissing, items[0].SourceKind)
	assert.Equal(t, row.ShipmentID.String(), items[0].SourceID)
	assert.Equal(t, watchtower.SeverityWarning, items[0].Severity)
	assert.Contains(t, items[0].Summary, "Globex")
}

func TestTenderExhaustedSource_IsCritical(t *testing.T) {
	t.Parallel()

	row := &watchtowerports.ExhaustedTender{
		TenderID:    pulid.MustNew("tnd_"),
		MoveID:      pulid.MustNew("smv_"),
		ShipmentID:  pulid.MustNew("shp_"),
		ExhaustedAt: exceptionNow - 60,
	}
	reader := &fakeExceptionReader{tenders: []*watchtowerports.ExhaustedTender{row}}
	source := NewTenderExhaustedSource(exceptionParams(reader, nil))

	items, err := source.Snapshot(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_")})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, watchtower.SeverityCritical, items[0].Severity)
	assert.Equal(t, row.TenderID.String(), items[0].SourceID)
	assert.Equal(t, "Tender exhausted", items[0].Title)
	assert.Equal(t, row.ExhaustedAt, items[0].OccurredAt)
}

func TestShipmentExceptionSource_PassesAReadFailureUp(t *testing.T) {
	t.Parallel()

	reader := &fakeExceptionReader{err: assert.AnError}
	for _, constructor := range []func(ShipmentExceptionParams) watchtowerports.WatchtowerSource{
		NewMissedPickupSource,
		NewMissedDeliverySource,
		NewPodMissingSource,
		NewTenderExhaustedSource,
	} {
		_, err := constructor(exceptionParams(reader, nil)).
			Snapshot(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_")})
		require.ErrorIs(t, err, assert.AnError)
	}
}

func TestShipmentExceptionObserver_SettlesTheShipmentForTheEvent(t *testing.T) {
	t.Parallel()

	stop := missedStop(exceptionNow - 3600)
	reader := &fakeExceptionReader{stops: []*watchtowerports.MissedStop{stop}}
	reconciler := &recordingReconciler{}
	observer := NewShipmentExceptionObserver(ShipmentExceptionObserverParams{
		ShipmentExceptionParams: exceptionParams(reader, nil),
		Reconciler:              reconciler,
		Logger:                  zap.NewNop(),
	})
	event := &shipmentevent.Event{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ShipmentID:     stop.ShipmentID,
		Type:           shipmentevent.TypeMoveArrived,
	}

	require.NoError(t, observer.OnShipmentEvent(t.Context(), event))
	assert.True(t, reader.scoped)
	require.Len(t, reconciler.requests, 3)
	assert.Equal(t, watchtower.SourceMissedPickup, reconciler.requests[0].SourceKind)
	assert.Equal(t, watchtower.SourceMissedDelivery, reconciler.requests[1].SourceKind)
	assert.Equal(t, watchtower.SourcePodMissing, reconciler.requests[2].SourceKind)
	for _, req := range reconciler.requests {
		assert.Equal(t, stop.ShipmentID, req.SubjectID)
	}
	for _, req := range reader.stopRequests {
		assert.Equal(t, []pulid.ID{stop.ShipmentID}, req.ShipmentIDs)
	}

	reconciler.requests = nil
	require.NoError(t, observer.OnShipmentEvent(t.Context(), &shipmentevent.Event{
		ShipmentID: stop.ShipmentID,
		Type:       shipmentevent.TypeCommentPosted,
	}))
	assert.Empty(t, reconciler.requests)
}

func TestShipmentExceptionObserver_SkipsAKindItCannotRead(t *testing.T) {
	t.Parallel()

	reconciler := &recordingReconciler{}
	observer := NewShipmentExceptionObserver(ShipmentExceptionObserverParams{
		ShipmentExceptionParams: exceptionParams(&fakeExceptionReader{err: assert.AnError}, nil),
		Reconciler:              reconciler,
		Logger:                  zap.NewNop(),
	})

	require.NoError(t, observer.OnShipmentEvent(t.Context(), &shipmentevent.Event{
		ShipmentID: pulid.MustNew("shp_"),
		Type:       shipmentevent.TypeRoutingGuideExhausted,
	}))
	assert.Empty(t, reconciler.requests)
}

func TestShipmentExceptionKindsFor(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		ShipmentExceptionKinds(),
		ShipmentExceptionKindsFor(shipmentevent.TypeShipmentCanceled),
	)
	assert.Equal(
		t,
		[]watchtower.SourceKind{watchtower.SourceTenderExhausted},
		ShipmentExceptionKindsFor(shipmentevent.TypeCarrierAssigned),
	)
	assert.Empty(t, ShipmentExceptionKindsFor(shipmentevent.TypeHoldPlaced))
}
