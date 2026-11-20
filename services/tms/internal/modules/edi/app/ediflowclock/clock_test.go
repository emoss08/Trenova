package ediflowclock

import (
	"context"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var clockNow = time.Unix(1_800_000_000, 0)

type fakeTimers struct {
	tenant   pagination.TenantInfo
	scope    dbscope.Tenant
	called   pulid.ID
	instance *ladingproject.FlowInstance
}

func (f *fakeTimers) DispatchFlowTimer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instanceID pulid.ID,
) (*edioutbound.FlowTimer, error) {
	f.tenant = tenantInfo
	f.called = instanceID
	f.scope, _ = dbscope.TenantFrom(ctx)
	return &edioutbound.FlowTimer{Instance: f.instance}, nil
}

func ref() ladingports.FlowInstanceRef {
	return ladingports.FlowInstanceRef{
		ID:             pulid.MustNew("edfi_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func openInstance(r ladingports.FlowInstanceRef, wakeIn int64) *ladingproject.FlowInstance {
	wake := clockNow.Unix() + wakeIn
	sent := clockNow.Unix() - 60
	return &ladingproject.FlowInstance{
		ID:             r.ID,
		OrganizationID: r.OrganizationID,
		BusinessUnitID: r.BusinessUnitID,
		Flow:           "tender",
		Key:            "PO-1",
		Status:         ladingproject.FlowInstanceStatusOpen,
		WakeAt:         &wake,
		State: jsonutils.RawJSON(`{"key":"PO-1","awaitingResponse":` +
			itoa(sent) + `,"responseDue":` + itoa(wake) + `,"pending":[],"batches":[]}`),
	}
}

func itoa(value int64) string {
	encoded, _ := sonic.Marshal(value)
	return string(encoded)
}

func newService(repo ladingports.FlowRepository, timers timerDispatcher) *Service {
	return &Service{flows: repo, timers: timers, now: func() time.Time { return clockNow }}
}

func TestAdvanceMovesTheInstancesDeadlinesBackAndWakesItInItsTenant(t *testing.T) {
	t.Parallel()

	r := ref()
	instance := openInstance(r, 7200)
	wasDue := *instance.WakeAt

	repo := ladingportsmock.NewMockFlowRepository(t)
	repo.EXPECT().LocateInstance(mock.Anything, r.ID).Return(r, nil).Once()
	repo.EXPECT().
		GetInstance(mock.Anything, ladingports.GetFlowInstanceRequest{
			TenantInfo: r.TenantInfo(),
			ID:         r.ID,
		}).
		Return(instance, nil).Once()
	var saved *ladingproject.FlowInstance
	repo.EXPECT().
		SaveInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, i *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			saved = i
			return i, nil
		}).
		Once()
	repo.EXPECT().
		AppendEvent(mock.Anything, mock.MatchedBy(func(e *ladingproject.FlowEvent) bool {
			return e.Kind == advancedEventKind && e.InstanceID == r.ID &&
				string(e.Detail) == `{"by":3600}`
		})).
		Return(nil).Once()

	timers := &fakeTimers{instance: instance}
	advance, err := newService(repo, timers).Advance(t.Context(), &AdvanceRequest{
		InstanceID: r.ID,
		By:         time.Hour + 500*time.Millisecond,
	})

	require.NoError(t, err)
	assert.Equal(t, time.Hour, advance.By)
	assert.Equal(t, wasDue, *advance.PreviousDue)
	require.NotNil(t, saved)
	assert.Equal(t, wasDue-3600, *saved.WakeAt)
	var state lading.FlowSummary
	require.NoError(t, sonic.Unmarshal(saved.State, &state))
	assert.Equal(t, wasDue-3600, *state.ResponseDue)
	assert.Equal(t, clockNow.Unix()-60-3600, *state.AwaitingResponse)
	assert.Equal(t, r.ID, timers.called)
	assert.Equal(t, r.TenantInfo(), timers.tenant)
	assert.Equal(t, r.TenantInfo().DBTenant(), timers.scope,
		"the wake runs under the instance's tenant scope")
}

func TestAdvanceToDueMovesTheClockExactlyToTheNextDueTime(t *testing.T) {
	t.Parallel()

	r := ref()
	instance := openInstance(r, 5400)

	repo := ladingportsmock.NewMockFlowRepository(t)
	repo.EXPECT().LocateInstance(mock.Anything, r.ID).Return(r, nil).Once()
	repo.EXPECT().GetInstance(mock.Anything, mock.Anything).Return(instance, nil).Once()
	repo.EXPECT().
		SaveInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, i *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			return i, nil
		}).
		Once()
	repo.EXPECT().AppendEvent(mock.Anything, mock.Anything).Return(nil).Once()

	advance, err := newService(repo, &fakeTimers{instance: instance}).
		Advance(t.Context(), &AdvanceRequest{InstanceID: r.ID, ToDue: true})

	require.NoError(t, err)
	assert.Equal(t, 90*time.Minute, advance.By)
	assert.Equal(t, clockNow.Unix(), *instance.WakeAt)
}

func TestAdvanceRefuses(t *testing.T) {
	t.Parallel()

	t.Run("a negative duration", func(t *testing.T) {
		t.Parallel()
		repo := ladingportsmock.NewMockFlowRepository(t)
		_, err := newService(repo, &fakeTimers{}).Advance(t.Context(), &AdvanceRequest{
			InstanceID: pulid.MustNew("edfi_"),
			By:         -time.Hour,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
	})

	t.Run("an unknown instance", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("edfi_")
		repo := ladingportsmock.NewMockFlowRepository(t)
		missing := errortypes.NewNotFoundError("FlowInstance not found")
		repo.EXPECT().LocateInstance(mock.Anything, id).
			Return(ladingports.FlowInstanceRef{}, missing).Once()
		_, err := newService(repo, &fakeTimers{}).Advance(t.Context(), &AdvanceRequest{
			InstanceID: id,
			By:         time.Hour,
		})
		require.ErrorIs(t, err, missing)
	})

	t.Run("a closed instance", func(t *testing.T) {
		t.Parallel()
		r := ref()
		instance := openInstance(r, 60)
		instance.Status = ladingproject.FlowInstanceStatusClosed
		repo := ladingportsmock.NewMockFlowRepository(t)
		repo.EXPECT().LocateInstance(mock.Anything, r.ID).Return(r, nil).Once()
		repo.EXPECT().GetInstance(mock.Anything, mock.Anything).Return(instance, nil).Once()
		_, err := newService(repo, &fakeTimers{}).Advance(t.Context(), &AdvanceRequest{
			InstanceID: r.ID,
			By:         time.Hour,
		})
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
	})

	t.Run("the next due time of an instance that waits on nothing", func(t *testing.T) {
		t.Parallel()
		r := ref()
		instance := openInstance(r, 60)
		instance.WakeAt = nil
		repo := ladingportsmock.NewMockFlowRepository(t)
		repo.EXPECT().LocateInstance(mock.Anything, r.ID).Return(r, nil).Once()
		repo.EXPECT().GetInstance(mock.Anything, mock.Anything).Return(instance, nil).Once()
		_, err := newService(repo, &fakeTimers{}).Advance(t.Context(), &AdvanceRequest{
			InstanceID: r.ID,
			ToDue:      true,
		})
		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
	})

	t.Run("no instance", func(t *testing.T) {
		t.Parallel()
		repo := ladingportsmock.NewMockFlowRepository(t)
		_, err := newService(repo, &fakeTimers{}).Advance(t.Context(), &AdvanceRequest{
			By: time.Hour,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
	})
}
