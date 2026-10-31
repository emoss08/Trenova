package watchtowerservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRealtime struct {
	mu        sync.Mutex
	published []*realtimeports.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, req)

	return nil
}

func (r *recordingRealtime) actions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	actions := make([]string, 0, len(r.published))
	for _, req := range r.published {
		actions = append(actions, req.Action)
	}

	return actions
}

func TestReconcile_PublishesOneNoticeForEverythingItChanged(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	tenant := testActor().TenantInfo()
	failures := &stubSource{
		kind: watchtower.SourceServiceFailure,
		items: []watchtowerports.WatchtowerItemInput{
			input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
			input(tenant, watchtower.SourceServiceFailure, "sf_2", watchtower.SeverityWarning, 200),
			input(tenant, watchtower.SourceServiceFailure, "sf_3", watchtower.SeverityWarning, 300),
		},
	}
	svc := newService(repo, &agenttest.StubPermissions{}, failures)
	realtime := &recordingRealtime{}
	svc.projector.realtime = realtime

	result, err := svc.Reconcile(t.Context(), tenant)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Upserted)
	require.Len(t, realtime.published, 1)
	notice := realtime.published[0]
	assert.Equal(t, RealtimeResource, notice.Resource)
	assert.Equal(t, "bulk_swept", notice.Action)
	assert.Equal(t, pulid.Nil, notice.RecordID)
	assert.Equal(t, tenant.OrgID, notice.OrganizationID)
	assert.Equal(t, tenant.BuID, notice.BusinessUnitID)
}

func TestReconcile_PublishesNothingWhenNothingMoved(t *testing.T) {
	t.Parallel()

	tenant := testActor().TenantInfo()
	svc := newService(
		newStubRepo(),
		&agenttest.StubPermissions{},
		&stubSource{kind: watchtower.SourceServiceFailure},
	)
	realtime := &recordingRealtime{}
	svc.projector.realtime = realtime

	_, err := svc.Reconcile(t.Context(), tenant)

	require.NoError(t, err)
	assert.Empty(t, realtime.published)
}

func TestUpsert_PublishesTheItemItWrote(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	tenant := testActor().TenantInfo()
	svc := newService(repo, &agenttest.StubPermissions{})
	realtime := &recordingRealtime{}
	svc.projector.realtime = realtime

	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)

	require.Len(t, realtime.published, 1)
	assert.Equal(t, "created", realtime.published[0].Action)
	assert.Equal(
		t,
		repo.items[repo.key(watchtower.SourceServiceFailure, "sf_1")].ID,
		realtime.published[0].RecordID,
	)
}

func TestSweepAttention_PublishesOneNoticeForEveryItemItMoved(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	realtime := &recordingRealtime{}
	f.exceptions.projector.realtime = realtime
	tenant := f.actor.TenantInfo()
	for _, id := range []string{"sf_2", "sf_3"} {
		in := input(tenant, watchtower.SourceServiceFailure, id, watchtower.SeverityWarning, 100)
		require.NoError(t, f.exceptions.projector.upsert(t.Context(), in))
		item := f.repo.items[f.repo.key(in.SourceKind, in.SourceID)]
		item.Status = watchtower.StatusOpen
		item.Version = 1
	}
	for _, item := range f.repo.items {
		require.NoError(t, item.Snooze(1_700_000_600, 1_700_000_000))
	}
	realtime.published = nil

	result, err := f.exceptions.SweepAttention(t.Context(), tenant, 1_700_000_600)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Woke)
	assert.Equal(t, []string{"bulk_swept"}, realtime.actions())
}
