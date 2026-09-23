package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
)

type mockScheduleHandle struct {
	id           string
	updateErr    error
	deleteErr    error
	invokeUpdate bool
	updateCalled bool
	deleteCalled bool
	lastUpdate   *client.ScheduleUpdate
}

func (m *mockScheduleHandle) GetID() string { return m.id }
func (m *mockScheduleHandle) Delete(_ context.Context) error {
	m.deleteCalled = true
	return m.deleteErr
}
func (m *mockScheduleHandle) Backfill(_ context.Context, _ client.ScheduleBackfillOptions) error {
	return nil
}
func (m *mockScheduleHandle) Update(_ context.Context, opts client.ScheduleUpdateOptions) error {
	m.updateCalled = true
	if m.invokeUpdate && opts.DoUpdate != nil {
		input := client.ScheduleUpdateInput{
			Description: client.ScheduleDescription{
				Schedule: client.Schedule{},
			},
		}
		m.lastUpdate, _ = opts.DoUpdate(input)
	}
	return m.updateErr
}
func (m *mockScheduleHandle) Describe(_ context.Context) (*client.ScheduleDescription, error) {
	return nil, nil
}
func (m *mockScheduleHandle) Trigger(_ context.Context, _ client.ScheduleTriggerOptions) error {
	return nil
}
func (m *mockScheduleHandle) Pause(_ context.Context, _ client.SchedulePauseOptions) error {
	return nil
}
func (m *mockScheduleHandle) Unpause(_ context.Context, _ client.ScheduleUnpauseOptions) error {
	return nil
}

type mockScheduleListIterator struct {
	entries []*client.ScheduleListEntry
	index   int
	err     error
}

func (m *mockScheduleListIterator) HasNext() bool {
	return m.index < len(m.entries)
}

func (m *mockScheduleListIterator) Next() (*client.ScheduleListEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	entry := m.entries[m.index]
	m.index++
	return entry, nil
}

type mockScheduleClient struct {
	createHandle client.ScheduleHandle
	createErr    error
	listIter     client.ScheduleListIterator
	listErr      error
	handles      map[string]*mockScheduleHandle
}

func (m *mockScheduleClient) Create(
	_ context.Context,
	_ client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createHandle, nil
}

func (m *mockScheduleClient) List(
	_ context.Context,
	_ client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listIter, nil
}

func (m *mockScheduleClient) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	if h, ok := m.handles[id]; ok {
		return h
	}
	return &mockScheduleHandle{id: id}
}

type retryMockScheduleClient struct {
	callCount     int
	failUntilCall int
	listErr       error
	successIter   client.ScheduleListIterator
	createHandle  client.ScheduleHandle
}

func (m *retryMockScheduleClient) Create(
	_ context.Context,
	_ client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	return m.createHandle, nil
}

func (m *retryMockScheduleClient) List(
	_ context.Context,
	_ client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	m.callCount++
	if m.callCount < m.failUntilCall {
		return nil, m.listErr
	}
	return m.successIter, nil
}

func (m *retryMockScheduleClient) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	return &mockScheduleHandle{id: id}
}

type mockTemporalClient struct {
	client.Client
	scheduleClient client.ScheduleClient
}

func (m *mockTemporalClient) ScheduleClient() client.ScheduleClient {
	return m.scheduleClient
}

func TestReconcileResult_HasErrors_NoErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Errors: []error{},
	}

	assert.False(t, result.HasErrors())
}

func TestReconcileResult_HasErrors_WithErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Errors: []error{errors.New("something failed")},
	}

	assert.True(t, result.HasErrors())
}

func TestReconcileResult_Summary_AllZeros(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{},
		Updated: []string{},
		Deleted: []string{},
		Skipped: []string{},
		Errors:  []error{},
	}

	assert.Equal(t, "created=0 updated=0 deleted=0 skipped=0 errors=0", result.Summary())
}

func TestReconcileResult_Summary_MixedValues(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{"sched-1", "sched-2"},
		Updated: []string{"sched-3"},
		Deleted: []string{"sched-4", "sched-5", "sched-6"},
		Skipped: []string{"sched-7"},
		Errors:  []error{},
	}

	assert.Equal(t, "created=2 updated=1 deleted=3 skipped=1 errors=0", result.Summary())
}

func TestReconcileResult_Summary_WithErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{"sched-1"},
		Updated: []string{},
		Deleted: []string{},
		Skipped: []string{"sched-2"},
		Errors:  []error{errors.New("err1"), errors.New("err2")},
	}

	assert.Equal(t, "created=1 updated=0 deleted=0 skipped=1 errors=2", result.Summary())
}

func TestNewReconciler(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	reconciler := NewReconciler(nil, registry, logger)

	require.NotNil(t, reconciler)
	assert.Equal(t, registry, reconciler.registry)
}

func TestReconcile_CreatesNewSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "new-schedule-1",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
			{
				ID:        "new-schedule-2",
				Spec:      Cron("0 9 * * *"),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "new-schedule-1"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 2)
	assert.Contains(t, result.Created, "new-schedule-1")
	assert.Contains(t, result.Created, "new-schedule-2")
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Deleted)
	assert.Empty(t, result.Skipped)
	assert.Empty(t, result.Errors)
}

func TestReconcile_UpdatesExistingSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "existing-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	handle := &mockScheduleHandle{id: "existing-schedule", invokeUpdate: true}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "existing-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte("different-hash")},
							},
						},
					},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"existing-schedule": handle,
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Len(t, result.Updated, 1)
	assert.Contains(t, result.Updated, "existing-schedule")
	assert.Empty(t, result.Deleted)
	assert.Empty(t, result.Skipped)
	assert.Empty(t, result.Errors)
	assert.True(t, handle.updateCalled)
}

func TestReconcile_SkipsUnchangedSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "unchanged-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "unchanged-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte(sched.Hash())},
							},
						},
					},
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Deleted)
	assert.Len(t, result.Skipped, 1)
	assert.Contains(t, result.Skipped, "unchanged-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_DeletesOrphanSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "keep-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "orphan-schedule"},
				},
			},
			createHandle: &mockScheduleHandle{id: "keep-schedule"},
			handles: map[string]*mockScheduleHandle{
				"orphan-schedule": {id: "orphan-schedule"},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Contains(t, result.Created, "keep-schedule")
	assert.Len(t, result.Deleted, 1)
	assert.Contains(t, result.Deleted, "orphan-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_ErrorInCollectSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:   "",
				Spec: Every(30 * time.Minute),
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to collect schedules")
}

func TestReconcile_ErrorInListExistingSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "test-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listErr: errors.New("connection refused"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list existing schedules")
}

func TestReconcile_ErrorInCreateSchedule(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "fail-create",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("permission denied"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "create fail-create")
	assert.Contains(t, result.Errors[0].Error(), "permission denied")
}

func TestReconcile_AlreadyExistsErrorInCreate_FallsBackToUpdate(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "already-exists-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	handle := &mockScheduleHandle{id: "already-exists-schedule", invokeUpdate: true}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: serviceerror.NewAlreadyExists("schedule already exists"),
			handles: map[string]*mockScheduleHandle{
				"already-exists-schedule": handle,
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Contains(t, result.Created, "already-exists-schedule")
	assert.Empty(t, result.Errors)
	assert.True(t, handle.updateCalled)
}

func TestReconcile_ScheduleAlreadyRunningInCreate_FallsBackToUpdate(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "stale-list-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	handle := &mockScheduleHandle{id: "stale-list-schedule", invokeUpdate: true}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: temporal.ErrScheduleAlreadyRunning,
			handles: map[string]*mockScheduleHandle{
				"stale-list-schedule": handle,
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Contains(t, result.Created, "stale-list-schedule")
	assert.Empty(t, result.Errors)
	assert.True(t, handle.updateCalled)
}

func TestReconcile_DeleteSchedule_NotFoundError_Succeeds(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "already-gone-schedule"},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"already-gone-schedule": {
					id:        "already-gone-schedule",
					deleteErr: serviceerror.NewNotFound("schedule not found"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Deleted, 1)
	assert.Contains(t, result.Deleted, "already-gone-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_DeleteSchedule_OtherError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "fail-delete-schedule"},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"fail-delete-schedule": {
					id:        "fail-delete-schedule",
					deleteErr: errors.New("internal error"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Deleted)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "delete fail-delete-schedule")
}

func TestReconcile_ListIteratorError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "test-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "entry-1"},
				},
				err: errors.New("iteration failed"),
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to iterate schedules")
}

func TestReconcile_UpdateScheduleError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "update-fail-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "update-fail-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte("different-hash")},
							},
						},
					},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"update-fail-schedule": {
					id:        "update-fail-schedule",
					updateErr: errors.New("update failed"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Updated)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "update update-fail-schedule")
}

func TestNeedsUpdate_NoMemo(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID:   "test",
		Memo: nil,
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_NoFields(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: nil,
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_NoHashField(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"otherField": {Data: []byte("value")},
			},
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_HashDiffers(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"scheduleHash": {Data: []byte("old-hash-value")},
			},
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_HashMatches(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"scheduleHash": {Data: []byte(desired.Hash())},
			},
		},
	}

	assert.False(t, reconciler.needsUpdate(existing, desired))
}

func TestReconcileWithRetry_SuccessOnFirstTry(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "retry-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "retry-test"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.ReconcileWithRetry(t.Context(), 3)

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Empty(t, result.Errors)
}

func TestReconcileWithRetry_SuccessOnRetry(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "retry-success",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	sc := &retryMockScheduleClient{
		failUntilCall: 2,
		listErr:       errors.New("temporary failure"),
		successIter: &mockScheduleListIterator{
			entries: []*client.ScheduleListEntry{},
		},
		createHandle: &mockScheduleHandle{id: "retry-success"},
	}

	mc := &mockTemporalClient{
		scheduleClient: sc,
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 3)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Created, 1)
}

func TestReconcileWithRetry_ContextCancelled(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "cancel-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("always fails"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 3)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotNil(t, result)
}

func TestReconcileWithRetry_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "exhaust-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listErr: errors.New("persistent failure"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 0)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "persistent failure")
}

func TestReconcileWithRetry_MaxRetriesExhausted_WithResultErrors(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "error-result-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("create always fails"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 0)

	require.Error(t, err)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func ownerMemo(t *testing.T, owner string) *commonpb.Memo {
	t.Helper()

	payload, err := converter.GetDefaultDataConverter().ToPayload(owner)
	require.NoError(t, err)

	return &commonpb.Memo{Fields: map[string]*commonpb.Payload{ManagedByMemoKey: payload}}
}

// Other code keeps schedules in the same namespace, starting with one per agent
// definition. The reconciler must only ever delete its own.
func TestReconcile_LeavesSchedulesItDidNotCreate(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	desk := &mockScheduleHandle{id: "agent-definition/agdef_1"}
	legacy := &mockScheduleHandle{id: "retired-static-schedule"}
	retired := &mockScheduleHandle{id: "retired-registry-schedule"}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: desk.id, Memo: ownerMemo(t, "agent-definition")},
					{ID: legacy.id},
					{ID: retired.id, Memo: ownerMemo(t, ManagedByRegistry)},
				},
			},
			handles: map[string]*mockScheduleHandle{
				desk.id:    desk,
				legacy.id:  legacy,
				retired.id: retired,
			},
		},
	}

	result, err := NewReconciler(mc, registry, logger).Reconcile(t.Context())
	require.NoError(t, err)

	assert.False(t, desk.deleteCalled, "a schedule marked as another owner's must survive")
	assert.True(
		t,
		legacy.deleteCalled,
		"an unmarked schedule predates the marker and is the registry's",
	)
	assert.True(t, retired.deleteCalled)
	assert.ElementsMatch(t, []string{legacy.id, retired.id}, result.Deleted)
}

// The hash lives in the note because the note, unlike the schedule memo, can be
// rewritten by an update. Reading it from there is what stops every schedule
// being rewritten on every start.
func TestNeedsUpdate_ReadsTheHashFromTheNote(t *testing.T) {
	t.Parallel()

	desired := &Schedule{ID: "s", Spec: Every(time.Minute), Workflow: dummyWorkflow, TaskQueue: "q"}
	r := &Reconciler{}

	assert.False(
		t,
		r.needsUpdate(&client.ScheduleListEntry{Note: registryNote(desired.Hash())}, desired),
	)
	assert.True(t, r.needsUpdate(&client.ScheduleListEntry{Note: registryNote("stale")}, desired))
}

// Overlap and paused are hashed, so a change to either is detected. It has to
// be applied as well, not only reported.
func TestReconcile_UpdateAppliesOverlapPausedAndTheNewHash(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	desired := &Schedule{
		ID:            "paused-one",
		Spec:          Every(time.Hour),
		Workflow:      dummyWorkflow,
		TaskQueue:     "test-queue",
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		Paused:        true,
	}
	registry.RegisterProvider(&testProvider{schedules: []*Schedule{desired}})

	handle := &mockScheduleHandle{id: desired.ID, invokeUpdate: true}
	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{{ID: desired.ID, Note: registryNote("stale")}},
			},
			handles: map[string]*mockScheduleHandle{desired.ID: handle},
		},
	}

	_, err := NewReconciler(mc, registry, logger).Reconcile(t.Context())
	require.NoError(t, err)
	require.NotNil(t, handle.lastUpdate)

	got := handle.lastUpdate.Schedule
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE, got.Policy.Overlap)
	assert.True(t, got.State.Paused)
	assert.Equal(t, registryNote(desired.Hash()), got.State.Note)
}

func TestToScheduleOptions_MarksTheScheduleAsTheRegistrys(t *testing.T) {
	t.Parallel()

	sched := &Schedule{ID: "s", Spec: Every(time.Minute), Workflow: dummyWorkflow, TaskQueue: "q"}
	opts := sched.ToScheduleOptions()

	assert.Equal(t, ManagedByRegistry, opts.Memo[ManagedByMemoKey])
	hash, ok := hashFromNote(opts.Note)
	require.True(t, ok)
	assert.Equal(t, sched.Hash(), hash)
}
