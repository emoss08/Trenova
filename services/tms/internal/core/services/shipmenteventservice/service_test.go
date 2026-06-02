package shipmenteventservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceRecordNotifiesObserversAfterInsert(t *testing.T) {
	t.Parallel()

	observer := &recordingShipmentEventObserver{}
	repo := &fakeShipmentEventRepository{}
	svc := &service{
		l:         testLogger(),
		repo:      repo,
		realtime:  noopRealtimeService{},
		observers: []services.ShipmentEventObserver{observer},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err)
	require.Len(t, repo.inserted, 1)
	require.Len(t, observer.events, 1)
	require.Equal(t, repo.inserted[0].ID, observer.events[0].ID)
	require.True(t, observer.events[0].ID.IsNotNil())
}

func TestServiceRecordSkipsObserversWhenInsertFails(t *testing.T) {
	t.Parallel()

	insertErr := errors.New("insert failed")
	observer := &recordingShipmentEventObserver{}
	svc := &service{
		l:         testLogger(),
		repo:      &fakeShipmentEventRepository{insertErr: insertErr},
		realtime:  noopRealtimeService{},
		observers: []services.ShipmentEventObserver{observer},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.ErrorIs(t, err, insertErr)
	require.Empty(t, observer.events)
}

func TestServiceRecordObserverErrorDoesNotFailRecord(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:        testLogger(),
		repo:     &fakeShipmentEventRepository{},
		realtime: noopRealtimeService{},
		observers: []services.ShipmentEventObserver{
			nil,
			failingShipmentEventObserver{},
			&recordingShipmentEventObserver{},
		},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err)
}

func validRecordShipmentEventParams(t *testing.T) *services.RecordShipmentEventParams {
	t.Helper()

	return &services.RecordShipmentEventParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ShipmentID:     pulid.MustNew("shp_"),
		Type:           shipmentevent.TypeStatusChanged,
		Summary:        "Status updated",
		Metadata: map[string]any{
			"previousStatus": "New",
			"newStatus":      "InTransit",
		},
	}
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}

type fakeShipmentEventRepository struct {
	inserted  []*shipmentevent.Event
	insertErr error
}

func (r *fakeShipmentEventRepository) Insert(
	_ context.Context,
	entity *shipmentevent.Event,
) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("se_")
	}
	r.inserted = append(r.inserted, entity)
	return nil
}

func (r *fakeShipmentEventRepository) GetByID(
	context.Context,
	repositories.GetShipmentEventByIDRequest,
) (*shipmentevent.Event, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeShipmentEventRepository) List(
	context.Context,
	*repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, errors.New("not implemented")
}

type recordingShipmentEventObserver struct {
	events []*shipmentevent.Event
}

func (o *recordingShipmentEventObserver) OnShipmentEvent(
	_ context.Context,
	event *shipmentevent.Event,
) error {
	o.events = append(o.events, event)
	return nil
}

type failingShipmentEventObserver struct{}

func (failingShipmentEventObserver) OnShipmentEvent(
	context.Context,
	*shipmentevent.Event,
) error {
	return errors.New("observer failed")
}

type noopRealtimeService struct{}

func (noopRealtimeService) CreateTokenRequest(
	*services.CreateRealtimeTokenRequest,
) (*services.RealtimeTokenRequest, error) {
	return nil, nil
}

func (noopRealtimeService) PublishResourceInvalidation(
	context.Context,
	*services.PublishResourceInvalidationRequest,
) error {
	return nil
}
