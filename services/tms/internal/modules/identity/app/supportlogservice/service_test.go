package supportlogservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const start = int64(1_800_000_000)

type memoryRepo struct {
	mu   sync.Mutex
	rows map[pulid.ID]*supportlog.LogRequest
}

func (m *memoryRepo) Create(_ context.Context, req *supportlog.LogRequest) (*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.Status.InFlight() {
			return nil, errortypes.NewValidationError("period", errortypes.ErrDuplicate, "in flight")
		}
	}
	row := *req
	row.ID = pulid.MustNew(supportlog.RequestIDPrefix)
	m.rows[row.ID] = &row
	out := row
	return &out, nil
}

func (m *memoryRepo) Recent(_ context.Context, limit int) ([]*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*supportlog.LogRequest, 0, limit)
	for _, row := range m.rows {
		if len(out) < limit {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (m *memoryRepo) Requester(_ context.Context, userID pulid.ID) (*supportlogports.Requester, error) {
	return &supportlogports.Requester{UserID: userID, Name: "Ada Lovelace", Email: "ada@example.com"}, nil
}

func (m *memoryRepo) InstanceLoad(_ context.Context, since int64) (*supportlog.Load, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	load := &supportlog.Load{}
	for _, row := range m.rows {
		if row.Status.InFlight() {
			load.InFlight = true
		}
		if row.CreatedAt > since {
			load.Recent++
		}
	}
	return load, nil
}

func (m *memoryRepo) Waiting(_ context.Context, limit int) ([]*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*supportlog.LogRequest
	for _, row := range m.rows {
		if row.Status.InFlight() && len(out) < limit {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (m *memoryRepo) ForOperator(_ context.Context, id pulid.ID) (*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return nil, supportlog.ErrRequestNotFound
	}
	copied := *row
	return &copied, nil
}

func (m *memoryRepo) SaveReport(_ context.Context, req *supportlog.LogRequest) (*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *req
	m.rows[req.ID] = &copied
	return req, nil
}

func (m *memoryRepo) Overdue(_ context.Context, req *supportlogports.OverdueRequest) ([]*supportlog.LogRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*supportlog.LogRequest
	for _, row := range m.rows {
		waiting := row.Status == supportlog.StatusRequested && row.CreatedAt <= req.RequestedBefore
		stuck := row.Status == supportlog.StatusCollecting && row.CollectingAt != nil &&
			*row.CollectingAt <= req.CollectingBefore
		if (waiting || stuck) && len(out) < req.Limit {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

type notices struct {
	sent []*supportlog.LogRequest
}

func (n *notices) LogRequestFinished(_ context.Context, notice *supportlogports.FinishedNotice) error {
	n.sent = append(n.sent, notice.Request)
	return nil
}

type published struct {
	events []*realtimeports.PublishResourceInvalidationRequest
}

func (p *published) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	p.events = append(p.events, req)
	return nil
}

type rig struct {
	svc      *Service
	repo     *memoryRepo
	audit    *securityaudittest.Recorder
	notices  *notices
	realtime *published
	now      int64
}

func newRig(managed bool) *rig {
	r := &rig{
		repo:     &memoryRepo{rows: map[pulid.ID]*supportlog.LogRequest{}},
		audit:    &securityaudittest.Recorder{},
		notices:  &notices{},
		realtime: &published{},
		now:      start,
	}
	r.svc = New(Params{
		Repo:      r.repo,
		Validator: NewValidator(ValidatorParams{Repo: r.repo}),
		Auditor:   r.audit,
		Notifier:  r.notices,
		Realtime:  r.realtime,
		Managed:   &managedconfig.Config{Enabled: managed},
		Clock:     func() int64 { return r.now },
		Logger:    zap.NewNop(),
	})
	return r
}

func admin() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func (r *rig) actions() []string {
	var out []string
	for _, change := range r.audit.Changes() {
		out = append(out, change.Action)
	}
	return out
}

func TestRequest_StoresWhoAskedAndAnnouncesIt(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	who := admin()
	req, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: who, Period: supportlog.PeriodWeek})
	require.NoError(t, err)

	assert.Equal(t, who.UserID, req.RequestedByID)
	assert.Equal(t, "Ada Lovelace", req.RequesterName)
	assert.Equal(t, "ada@example.com", req.RequesterEmail)
	assert.Equal(t, start-7*86_400, req.Since)
	assert.Equal(t, supportlog.StatusRequested, req.Status)
	assert.Equal(t, []string{ActionRequested}, r.actions())
	require.Len(t, r.realtime.events, 1)
	assert.Equal(t, RealtimeResource, r.realtime.events[0].Resource)
	assert.Nil(t, r.realtime.events[0].Entity, "the event names the record; the page refetches it")
}

func TestRequest_OneInFlightForTheWholeInstance(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	_, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.NoError(t, err)

	_, err = r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodDay})
	require.Error(t, err)
	_, isRefusal := errortypes.RefusalOf(err)
	assert.True(t, isRefusal, "another organization's request in flight refuses this one")
}

func TestRequest_DailyLimit(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	for range supportlog.DailyLimit {
		req, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
		require.NoError(t, err)
		require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{
			Status:   supportlog.StatusSent,
			BundleID: "lb_" + req.ID.String()[4:],
		}))
		r.now += 60
	}

	_, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.Error(t, err)

	r.now += 24 * 3600
	_, err = r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.NoError(t, err, "the limit is over the last 24 hours, not forever")
}

func TestRequest_RefusedOnASelfHostedInstance(t *testing.T) {
	t.Parallel()

	r := newRig(false)
	_, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.Error(t, err)

	overview, err := r.svc.Overview(t.Context())
	require.NoError(t, err)
	assert.False(t, overview.Managed)
	assert.Empty(t, overview.Requests)
}

func TestRequest_RefusedForASupportPrincipal(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	who := admin()
	who.UserID = pulid.MustNew("sup_")
	_, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: who, Period: supportlog.PeriodHour})
	require.Error(t, err)
	assert.Empty(t, r.repo.rows)
}

func TestReport_SentTellsWhoAskedOnceAndIsAudited(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	req, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.NoError(t, err)

	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "lb_1"}))
	assert.Empty(t, r.notices.sent)
	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusSent}))
	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusSent}))

	require.Len(t, r.notices.sent, 1, "a repeated report tells nobody twice")
	assert.Equal(t, supportlog.StatusSent, r.notices.sent[0].Status)
	assert.Equal(t, []string{ActionRequested, ActionSent}, r.actions())

	err = r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusFailed})
	require.ErrorIs(t, err, supportlog.ErrFinished)
}

func TestWaiting_KeepsACollectingRequestForAnOperatorThatRestarted(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	req, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodDay})
	require.NoError(t, err)
	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "lb_1"}))

	waiting, err := r.svc.Waiting(t.Context())
	require.NoError(t, err)
	require.Len(t, waiting, 1)
	assert.Equal(t, supportlog.StatusCollecting, waiting[0].Status)
	assert.Equal(t, "lb_1", waiting[0].BundleID)

	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "lb_1"}))
	assert.Equal(t, []string{ActionRequested}, r.actions(), "picking it up again records nothing new")

	require.NoError(t, r.svc.Report(t.Context(), req.ID, &supportlog.Report{Status: supportlog.StatusSent}))
	waiting, err = r.svc.Waiting(t.Context())
	require.NoError(t, err)
	assert.Empty(t, waiting)
}

func TestFailOverdue_FailsWhatTheOperatorNeverPickedUp(t *testing.T) {
	t.Parallel()

	r := newRig(true)
	req, err := r.svc.Request(t.Context(), &RequestLogsRequest{TenantInfo: admin(), Period: supportlog.PeriodHour})
	require.NoError(t, err)

	result, err := r.svc.FailOverdue(t.Context(), start+29*60)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Failed)

	result, err = r.svc.FailOverdue(t.Context(), start+30*60)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)

	held := r.repo.rows[req.ID]
	assert.Equal(t, supportlog.StatusFailed, held.Status)
	assert.Equal(t, supportlog.MessageNotPickedUp, held.FailureMessage)
	require.Len(t, r.notices.sent, 1)
	assert.Contains(t, r.actions(), ActionFailed)
}
