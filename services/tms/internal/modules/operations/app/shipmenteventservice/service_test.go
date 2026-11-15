package shipmenteventservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
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
		features:  plantest.AllFeatures(),
		observers: []shipmenteventports.ShipmentEventObserver{observer},
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
		features:  plantest.AllFeatures(),
		observers: []shipmenteventports.ShipmentEventObserver{observer},
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
		features: plantest.AllFeatures(),
		observers: []shipmenteventports.ShipmentEventObserver{
			nil,
			failingShipmentEventObserver{},
			&recordingShipmentEventObserver{},
		},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err)
}

func validRecordShipmentEventParams(t *testing.T) *shipmenteventports.RecordShipmentEventParams {
	t.Helper()

	return &shipmenteventports.RecordShipmentEventParams{
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
	shipmenteventports.GetShipmentEventByIDRequest,
) (*shipmentevent.Event, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeShipmentEventRepository) List(
	context.Context,
	*shipmenteventports.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, errors.New("not implemented")
}

type recordingShipmentEventObserver struct {
	feature platformcatalog.FeatureKey
	events  []*shipmentevent.Event
}

func (o *recordingShipmentEventObserver) ObserverFeature() platformcatalog.FeatureKey {
	if o.feature == "" {
		return platformcatalog.FeatureCoreTMS
	}
	return o.feature
}

func (o *recordingShipmentEventObserver) OnShipmentEvent(
	_ context.Context,
	event *shipmentevent.Event,
) error {
	o.events = append(o.events, event)
	return nil
}

type failingShipmentEventObserver struct{}

func (failingShipmentEventObserver) ObserverFeature() platformcatalog.FeatureKey {
	return platformcatalog.FeatureCoreTMS
}

func (failingShipmentEventObserver) OnShipmentEvent(
	context.Context,
	*shipmentevent.Event,
) error {
	return errors.New("observer failed")
}

type noopRealtimeService struct{}

func (noopRealtimeService) PublishResourceInvalidation(
	context.Context,
	*realtimeports.PublishResourceInvalidationRequest,
) error {
	return nil
}

func TestServiceRecordDefersObserversUntilTheTransactionCommits(t *testing.T) {
	t.Parallel()

	observer := &recordingShipmentEventObserver{}
	repo := &fakeShipmentEventRepository{}
	svc := &service{
		l:         testLogger(),
		repo:      repo,
		realtime:  noopRealtimeService{},
		features:  plantest.AllFeatures(),
		observers: []shipmenteventports.ShipmentEventObserver{observer},
	}
	txCtx, hooks := kernelports.WithAfterCommitHooks(t.Context())

	err := svc.Record(txCtx, validRecordShipmentEventParams(t))

	require.NoError(t, err)
	require.Len(t, repo.inserted, 1)
	require.Empty(t, observer.events, "nothing reads the event before it is committed")

	hooks.Run(t.Context())
	require.Len(t, observer.events, 1)
}

func TestServiceRecordTellsOnlyTheObserversThePlanIncludes(t *testing.T) {
	t.Parallel()

	base := &recordingShipmentEventObserver{}
	tracking := &recordingShipmentEventObserver{feature: platformcatalog.FeatureCustomerTracking}
	edi := &recordingShipmentEventObserver{feature: platformcatalog.FeatureEDIIntegration}
	svc := &service{
		l:         testLogger(),
		repo:      &fakeShipmentEventRepository{},
		realtime:  noopRealtimeService{},
		features:  plantest.BaseOnly(),
		observers: []shipmenteventports.ShipmentEventObserver{tracking, base, edi},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err)
	require.Len(t, base.events, 1, "a base observer is told")
	require.Empty(t, tracking.events, "an observer of an add-on that is off is not told")
	require.Empty(t, edi.events, "an observer of an add-on that is off is not told")
}

func TestServiceRecordTellsEveryObserverWithEveryFeatureOn(t *testing.T) {
	t.Parallel()

	base := &recordingShipmentEventObserver{}
	tracking := &recordingShipmentEventObserver{feature: platformcatalog.FeatureCustomerTracking}
	svc := &service{
		l:         testLogger(),
		repo:      &fakeShipmentEventRepository{},
		realtime:  noopRealtimeService{},
		features:  plantest.AllFeatures(),
		observers: []shipmenteventports.ShipmentEventObserver{tracking, base},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err)
	require.Len(t, base.events, 1)
	require.Len(t, tracking.events, 1)
}

type unreachableTrackingGate struct{}

func (unreachableTrackingGate) Enabled(
	_ context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	if feature == platformcatalog.FeatureCustomerTracking {
		return false, errors.New("plan store unreachable")
	}
	return true, nil
}

func TestServiceRecordTreatsAPlanQuestionThatFailsLikeAFailingObserver(t *testing.T) {
	t.Parallel()

	base := &recordingShipmentEventObserver{}
	tracking := &recordingShipmentEventObserver{feature: platformcatalog.FeatureCustomerTracking}
	repo := &fakeShipmentEventRepository{}
	svc := &service{
		l:         testLogger(),
		repo:      repo,
		realtime:  noopRealtimeService{},
		features:  unreachableTrackingGate{},
		observers: []shipmenteventports.ShipmentEventObserver{tracking, base},
	}

	err := svc.Record(t.Context(), validRecordShipmentEventParams(t))

	require.NoError(t, err, "a notification stays best-effort")
	require.Len(t, repo.inserted, 1)
	require.Empty(t, tracking.events, "the observer whose feature could not be asked about is skipped")
	require.Len(t, base.events, 1)
}
