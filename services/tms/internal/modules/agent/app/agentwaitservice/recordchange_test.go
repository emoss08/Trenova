package agentwaitservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRecords struct {
	found map[pulid.ID]*agentwaitports.FoundRecord
	names map[pulid.ID]string
	asked []permission.Resource
}

func (f *fakeRecords) Find(
	_ context.Context,
	_ pagination.TenantInfo,
	resource permission.Resource,
	id pulid.ID,
) (*agentwaitports.FoundRecord, error) {
	f.asked = append(f.asked, resource)
	if found, ok := f.found[id]; ok {
		return found, nil
	}

	return &agentwaitports.FoundRecord{Known: true}, nil
}

func (f *fakeRecords) UserNames(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	out := make(map[pulid.ID]string, len(ids))
	for _, id := range ids {
		if name, ok := f.names[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}

type fakeActivity struct {
	read  bool
	asked []*agentwaitports.RecordReadSinceRequest
}

func (f *fakeActivity) RecordReadSince(
	_ context.Context,
	req *agentwaitports.RecordReadSinceRequest,
) (bool, error) {
	f.asked = append(f.asked, req)

	return f.read, nil
}

type fakeFeed struct {
	realtimeports.RecordWatchFeed

	watched   []realtimeports.WatchRecordRequest
	unwatched []realtimeports.WatchRecordRequest
	err       error
}

func (f *fakeFeed) Watch(_ context.Context, req *realtimeports.WatchRecordRequest) error {
	if f.err != nil {
		return f.err
	}
	f.watched = append(f.watched, *req)

	return nil
}

func (f *fakeFeed) Unwatch(_ context.Context, req *realtimeports.WatchRecordRequest) error {
	f.unwatched = append(f.unwatched, *req)

	return nil
}

type fakeReportRuns struct {
	reportports.ReportRunRepository

	run   *report.ReportRun
	asked []*reportports.GetReportRunRequest
}

func (f *fakeReportRuns) GetByID(
	_ context.Context,
	req *reportports.GetReportRunRequest,
) (*report.ReportRun, error) {
	f.asked = append(f.asked, req)
	if f.run == nil || f.run.ID != req.RunID || f.run.RequestedByID != req.RequestedBy {
		return nil, errortypes.NewNotFoundError("Report run not found")
	}

	return f.run, nil
}

type streams struct {
	*harness

	records  *fakeRecords
	activity *fakeActivity
	feed     *fakeFeed
	runs     *fakeReportRuns
}

func newStreams(denied ...string) *streams {
	h := newHarness()
	s := &streams{
		harness:  h,
		records:  &fakeRecords{found: map[pulid.ID]*agentwaitports.FoundRecord{}},
		activity: &fakeActivity{},
		feed:     &fakeFeed{},
		runs:     &fakeReportRuns{},
	}
	h.svc.permissions = &mocks.DenyResourcesPermissionEngine{Denied: denied}
	h.svc.records = s.records
	h.svc.activity = s.activity
	h.svc.feed = s.feed
	h.svc.facts.reportRuns = s.runs

	return s
}

func (s *streams) watchRecord(id pulid.ID) (*agentwait.Wait, error) {
	return s.register(agentwait.KindRecordChanged, agentwait.Condition{RecordID: id})
}

func TestRegister_ARecordChangeReadsTheRecordAndRegistersItWithTheStream(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	s.records.found[shipment] = &agentwaitports.FoundRecord{Known: true, Exists: true, Label: "S-1042"}

	wait, err := s.watchRecord(shipment)

	require.NoError(t, err)
	assert.Equal(t, shipment, wait.WatchID)
	assert.Equal(t, "shipment", wait.Condition.RecordKind)
	assert.Equal(t, "S-1042", wait.Condition.RecordLabel)
	assert.Equal(t, []permission.Resource{permission.ResourceShipment}, s.records.asked)
	require.Len(t, s.feed.watched, 1)
	assert.Equal(t, realtimeports.WatchRecordRequest{
		OrganizationID: s.actor.OrganizationID,
		BusinessUnitID: s.actor.BusinessUnitID,
		RecordID:       shipment,
		Until:          wait.ExpiresAt,
	}, s.feed.watched[0], "registered until the wait runs out")
	assert.Nil(t, wait.DueAt, "no timer: only a change ends it")
}

func TestRegister_ARecordThePersonCannotReadIsRefusedLikeAMissingOne(t *testing.T) {
	t.Parallel()

	s := newStreams(permission.ResourceInvoice.String())
	invoice := pulid.MustNew("inv_")
	s.records.found[invoice] = &agentwaitports.FoundRecord{Known: true, Exists: true, Label: "INV-1"}

	_, err := s.watchRecord(invoice)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "No invoice with that id")
	assert.Empty(t, s.records.asked, "the record is not read for someone who may not read it")
	assert.Empty(t, s.feed.watched)
	assert.Empty(t, s.workflows.started)

	missing := newStreams()
	_, err = missing.watchRecord(pulid.MustNew("shp_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No shipment with that id")
}

func TestRegister_ARecordIDOfNoKnownKindIsRefused(t *testing.T) {
	t.Parallel()

	s := newStreams()
	_, err := s.watchRecord(pulid.MustNew("zzz_"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recordId does not name a kind of record")
}

func TestRegister_AStreamThatCannotBeJoinedFailsTheWait(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	s.records.found[shipment] = &agentwaitports.FoundRecord{Known: true, Exists: true}
	s.feed.err = errors.New("redis down")

	_, err := s.watchRecord(shipment)

	require.Error(t, err)
	require.Len(t, s.waits.items, 1)
	for _, wait := range s.waits.items {
		assert.Equal(t, agentwait.StatusFailed, wait.Status,
			"a wait that cannot hear its record is not left open")
	}
	assert.Len(t, s.workflows.cancelled, 1)
}

func openRecordWait(s *streams, record pulid.ID, createdAt int64) *agentwait.Wait {
	wait := &agentwait.Wait{
		ID:             pulid.MustNew(agentwait.IDPrefix),
		OrganizationID: s.actor.OrganizationID,
		BusinessUnitID: s.actor.BusinessUnitID,
		ThreadID:       pulid.MustNew("athr_"),
		UserID:         s.actor.UserID,
		Kind:           agentwait.KindRecordChanged,
		Condition: &agentwait.Condition{
			RecordID:    record,
			RecordKind:  "shipment",
			RecordLabel: "S-1042",
		},
		WatchID:    record,
		Status:     agentwait.StatusWaiting,
		WorkflowID: "agent-wait:1",
		CreatedAt:  createdAt,
		ExpiresAt:  now + 3600,
	}
	s.waits.items[wait.ID] = wait

	return wait
}

func changeOf(
	s *streams,
	record pulid.ID,
	actorType kernelports.PrincipalType,
	actor pulid.ID,
	at int64,
) realtimeports.TenantRecordChange {
	return realtimeports.TenantRecordChange{
		OrganizationID: s.actor.OrganizationID,
		BusinessUnitID: s.actor.BusinessUnitID,
		Change: realtimeports.WatchedRecordChange{
			RecordID:    record.String(),
			Resource:    "audited:shipment",
			Action:      "updated",
			Fields:      []string{"documents"},
			ActorType:   string(actorType),
			ActorUserID: actor.String(),
			At:          at,
		},
	}
}

func TestNotifyRecordChanges_TellsTheWaitWhoChangedItsRecord(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	colleague := pulid.MustNew("usr_")
	s.records.names = map[pulid.ID]string{colleague: "Dana Ortiz"}
	wait := openRecordWait(s, shipment, now-600)

	s.svc.NotifyRecordChanges(context.Background(), []realtimeports.TenantRecordChange{
		changeOf(s, shipment, permissionports.PrincipalTypeUser, colleague, now-60),
		changeOf(s, shipment, permissionports.PrincipalTypeUser, s.actor.UserID, now-30),
		changeOf(s, pulid.MustNew("shp_"), permissionports.PrincipalTypeUser, colleague, now),
	})

	require.Len(t, s.workflows.signals, 2, "each change to the watched record is passed on")
	first, second := s.workflows.signals[0].met, s.workflows.signals[1].met
	assert.Equal(t, wait.WorkflowID, s.workflows.signals[0].workflowID)
	assert.Equal(t, shipment.String(), first.Key)
	assert.Equal(t, now-60, first.At)
	assert.Contains(t, first.Detail, "shipment S-1042")
	assert.Contains(t, first.Detail, "by Dana Ortiz")
	assert.Contains(t, first.Detail, "documents")
	assert.Contains(t, second.Detail, "by the person you are working for")
}

func TestNotifyRecordChanges_AChangeFromBeforeTheWaitIsWhatTheAgentRead(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	openRecordWait(s, shipment, now)

	s.svc.NotifyRecordChanges(context.Background(), []realtimeports.TenantRecordChange{
		changeOf(s, shipment, permissionports.PrincipalTypeUser, s.actor.UserID, now-5),
	})

	assert.Empty(t, s.workflows.signals)
}

func TestNotifyRecordChanges_AnotherTenantsChangeReachesNoWait(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	openRecordWait(s, shipment, now-600)
	change := changeOf(s, shipment, permissionports.PrincipalTypeUser, s.actor.UserID, now)
	change.OrganizationID = pulid.MustNew("org_")
	s.waits.tenantScoped = true

	s.svc.NotifyRecordChanges(context.Background(), []realtimeports.TenantRecordChange{change})

	assert.Empty(t, s.workflows.signals)
}

func TestCancel_TakesTheRecordOffTheStreamOnlyWhenNoOtherWaitWatchesIt(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	first := openRecordWait(s, shipment, now-600)
	second := openRecordWait(s, shipment, now-300)

	_, err := s.svc.Cancel(context.Background(), &agentwaitports.CancelWaitRequest{
		ID:         first.ID,
		TenantInfo: tenantOf(first),
		ThreadID:   first.ThreadID,
		UserID:     first.UserID,
	})
	require.NoError(t, err)
	assert.Empty(t, s.feed.unwatched, "the other wait still watches the record")

	_, err = s.svc.Cancel(context.Background(), &agentwaitports.CancelWaitRequest{
		ID:         second.ID,
		TenantInfo: tenantOf(second),
		ThreadID:   second.ThreadID,
		UserID:     second.UserID,
	})
	require.NoError(t, err)
	require.Len(t, s.feed.unwatched, 1)
	assert.Equal(t, shipment, s.feed.unwatched[0].RecordID)
	assert.Equal(t, second.ExpiresAt, s.feed.unwatched[0].Until)
}

func basePayload(wait *agentwait.Wait) temporaltype.BasePayload {
	return temporaltype.BasePayload{
		OrganizationID: wait.OrganizationID,
		BusinessUnitID: wait.BusinessUnitID,
	}
}

func finishRecordWait(s *streams, wait *agentwait.Wait, firstMetAt int64) error {
	return s.svc.Finish(context.Background(), &agentwaitcontract.FinishInput{
		Payload: &agentwaitcontract.Payload{
			BasePayload: basePayload(wait),
			WaitID:      wait.ID,
		},
		Status:     agentwait.StatusMet,
		Outcome:    "shipment S-1042 was changed by the person you are working for.",
		FirstMetAt: firstMetAt,
	})
}

func TestFinish_ARecordTheConversationReadSinceTheChangeIsNotAnsweredTwice(t *testing.T) {
	t.Parallel()

	s := newStreams()
	s.activity.read = true
	record := pulid.MustNew("shp_")
	wait := openRecordWait(s, record, now-600)

	require.NoError(t, finishRecordWait(s, wait, now-90))

	assert.Equal(t, agentwait.StatusMet, wait.Status)
	assert.Empty(t, s.turns.started, "the conversation already read the change")
	require.Len(t, s.activity.asked, 1)
	assert.Equal(t, now-90, s.activity.asked[0].Since)
	assert.Equal(t, wait.ThreadID, s.activity.asked[0].ThreadID)
	assert.Equal(t, record, s.activity.asked[0].RecordID)
	require.Len(t, s.feed.unwatched, 1, "a closed wait leaves the stream")
}

/*
A person settled a flagged check, then sent "ok" while the wait settled. The
"ok" turn never looked at the record, so the work is still picked up: closing
the wait because the person spoke left them asking "did you not see it?".
*/
func TestFinish_AChangeNobodyReadPicksTheConversationUpEvenAfterThePersonSpoke(t *testing.T) {
	t.Parallel()

	s := newStreams()
	wait := openRecordWait(s, pulid.MustNew("shp_"), now-600)

	require.NoError(t, finishRecordWait(s, wait, now-90))

	require.Len(t, s.turns.started, 1)
	assert.Equal(t, wait.ThreadID, s.turns.started[0].ThreadID)
}

func openReportWait(s *streams, run *report.ReportRun) *agentwait.Wait {
	wait := &agentwait.Wait{
		ID:             pulid.MustNew(agentwait.IDPrefix),
		OrganizationID: s.actor.OrganizationID,
		BusinessUnitID: s.actor.BusinessUnitID,
		ThreadID:       pulid.MustNew("athr_"),
		UserID:         s.actor.UserID,
		Kind:           agentwait.KindReportRun,
		Condition:      &agentwait.Condition{ReportRunID: run.ID},
		WatchID:        run.ID,
		Status:         agentwait.StatusWaiting,
		WorkflowID:     "agent-wait:2",
		CreatedAt:      now - 60,
		ExpiresAt:      now + 3600,
	}
	s.waits.items[wait.ID] = wait

	return wait
}

func TestWaitForReportRun_ParksTheConversationOnceOnARunStillGoing(t *testing.T) {
	t.Parallel()

	s := newStreams()
	s.runs.run = &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: s.actor.UserID,
		Status:        report.RunStatusRunning,
	}
	thread := pulid.MustNew("athr_")
	req := &agentwaitports.WaitForReportRunRequest{
		Actor:        s.actor,
		DefinitionID: pulid.MustNew("agdef_"),
		ThreadID:     thread,
		ReportRunID:  s.runs.run.ID,
		ReportName:   "AR Aging",
		Then:         "name the customers over 90 days",
	}

	wait, err := s.svc.WaitForReportRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, agentwait.KindReportRun, wait.Kind)
	assert.Equal(t, "AR Aging to finish running", wait.Description)
	assert.Equal(t, "name the customers over 90 days", wait.Then)
	require.NotNil(t, wait.DueAt, "read again on a timer in case the finish is missed")
	assert.Equal(t, now+reportRunRecheckSeconds, *wait.DueAt)
	assert.Equal(t, now+reportRunLifetimeSeconds, wait.ExpiresAt)
	require.Len(t, s.feed.watched, 1)
	assert.Equal(t, s.runs.run.ID, s.feed.watched[0].RecordID)
	assert.Equal(t, s.actor.UserID, s.runs.asked[0].RequestedBy,
		"the run is read as the person who asked for it")

	again, err := s.svc.WaitForReportRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wait.ID, again.ID, "the run is waited on once per conversation")
	assert.Len(t, s.workflows.started, 1)
}

func TestWaitForReportRun_ARunAlreadyFinishedIsActedOnNow(t *testing.T) {
	t.Parallel()

	s := newStreams()
	s.runs.run = &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: s.actor.UserID,
		Status:        report.RunStatusSucceeded,
		RowCount:      42,
	}

	_, err := s.svc.WaitForReportRun(context.Background(), &agentwaitports.WaitForReportRunRequest{
		Actor:        s.actor,
		DefinitionID: pulid.MustNew("agdef_"),
		ThreadID:     pulid.MustNew("athr_"),
		ReportRunID:  s.runs.run.ID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "finished with 42 rows")
	assert.Contains(t, err.Error(), "Act on that now")
}

func TestNotifyRecordChanges_OnlyAFinishedRunEndsItsWait(t *testing.T) {
	t.Parallel()

	s := newStreams()
	run := &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: s.actor.UserID,
		Status:        report.RunStatusRunning,
	}
	s.runs.run = run
	openReportWait(s, run)
	notice := realtimeports.TenantRecordChange{
		OrganizationID: s.actor.OrganizationID,
		BusinessUnitID: s.actor.BusinessUnitID,
		Change: realtimeports.WatchedRecordChange{
			RecordID: run.ID.String(),
			Resource: "report-run",
			Action:   "updated",
			At:       now,
		},
	}

	s.svc.NotifyRecordChanges(context.Background(), []realtimeports.TenantRecordChange{notice})
	assert.Empty(t, s.workflows.signals, "a run still going ends nothing")

	run.Status = report.RunStatusSucceeded
	run.RowCount = 1240
	s.svc.NotifyRecordChanges(context.Background(), []realtimeports.TenantRecordChange{notice})
	require.Len(t, s.workflows.signals, 1)
	assert.Contains(t, s.workflows.signals[0].met.Detail, "finished with 1240 rows")
	assert.Contains(t, s.workflows.signals[0].met.Detail, "report_get_run")
}

func TestCheck_AReportRunWhoseFinishWasMissedIsFoundOnTheTimer(t *testing.T) {
	t.Parallel()

	s := newStreams()
	run := &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: s.actor.UserID,
		Status:        report.RunStatusFailed,
		Error:         &report.RunError{Message: "the query ran past its cost limit"},
	}
	s.runs.run = run
	wait := openReportWait(s, run)

	check, err := s.svc.Check(context.Background(), tenantOf(wait), wait.ID, now)

	require.NoError(t, err)
	assert.True(t, check.Met)
	assert.Contains(t, check.Detail, "did not finish: the query ran past its cost limit")
}
