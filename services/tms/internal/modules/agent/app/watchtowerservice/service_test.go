package watchtowerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRepo struct {
	watchtowerports.WatchtowerRepository

	items    map[string]*watchtower.Item
	cursor   *watchtower.Cursor
	lastList watchtowerports.ListWatchtowerItemsQuery
	resolved []watchtowerports.ResolveMissingWatchtowerItemsRequest
}

func newStubRepo() *stubRepo {
	return &stubRepo{items: map[string]*watchtower.Item{}}
}

func (r *stubRepo) key(
	kind watchtower.SourceKind,
	id string,
) string {
	return string(kind) + ":" + id
}

func (r *stubRepo) Upsert(
	_ context.Context,
	item *watchtower.Item,
) (*watchtower.Item, bool, error) {
	key := r.key(item.SourceKind, item.SourceID)
	existing, ok := r.items[key]
	if ok {
		item.ID = existing.ID
	} else {
		item.ID = pulid.MustNew("wt_")
	}
	item.ResolvedAt = nil
	r.items[key] = item

	return item, !ok, nil
}

func (r *stubRepo) Resolve(
	_ context.Context,
	req watchtowerports.ResolveWatchtowerItemRequest,
) (*watchtower.Item, error) {
	item, ok := r.items[r.key(req.SourceKind, req.SourceID)]
	if !ok || item.ResolvedAt != nil {
		return nil, nil
	}
	at := req.ResolvedAt
	item.ResolvedAt = &at

	return item, nil
}

func (r *stubRepo) ResolveByID(
	_ context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
	at int64,
) (*watchtower.Item, error) {
	for _, item := range r.items {
		if item.ID == req.ID {
			item.ResolvedAt = &at
			return item, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Watchtower item not found")
}

func (r *stubRepo) ResolveMissing(
	_ context.Context,
	req watchtowerports.ResolveMissingWatchtowerItemsRequest,
) (int, error) {
	r.resolved = append(r.resolved, req)
	open := map[string]bool{}
	for _, id := range req.OpenSourceIDs {
		open[id] = true
	}
	count := 0
	for _, item := range r.items {
		if item.SourceKind == req.SourceKind && item.ResolvedAt == nil && !open[item.SourceID] {
			at := req.ResolvedAt
			item.ResolvedAt = &at
			count++
		}
	}

	return count, nil
}

func (r *stubRepo) GetByID(
	_ context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
) (*watchtower.Item, error) {
	for _, item := range r.items {
		if item.ID == req.ID {
			return item, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Watchtower item not found")
}

func (r *stubRepo) List(
	_ context.Context,
	req watchtowerports.ListWatchtowerItemsQuery,
) ([]*watchtower.Item, error) {
	r.lastList = req
	allowed := map[watchtower.SourceKind]bool{}
	for _, kind := range req.Kinds {
		allowed[kind] = true
	}
	out := []*watchtower.Item{}
	for _, item := range r.items {
		if allowed[item.SourceKind] && (!req.UnresolvedOnly || item.ResolvedAt == nil) {
			out = append(out, item)
		}
	}
	if len(out) > req.Limit {
		out = out[:req.Limit]
	}

	return out, nil
}

func (r *stubRepo) Counts(
	_ context.Context,
	req watchtowerports.CountWatchtowerItemsRequest,
) (*watchtowerports.WatchtowerCountsRow, error) {
	counts := &watchtowerports.WatchtowerCountsRow{}
	allowed := map[watchtower.SourceKind]bool{}
	for _, kind := range req.Kinds {
		allowed[kind] = true
	}
	for _, item := range r.items {
		if !allowed[item.SourceKind] || item.ResolvedAt != nil {
			continue
		}
		counts.Unresolved++
		if item.Severity == watchtower.SeverityCritical {
			counts.Critical++
			if item.OccurredAt > req.SeenAt {
				counts.UnseenCritical++
			}
		}
	}

	return counts, nil
}

func (r *stubRepo) GetCursor(
	_ context.Context,
	req watchtowerports.GetWatchtowerCursorRequest,
) (*watchtower.Cursor, error) {
	if r.cursor != nil {
		return r.cursor, nil
	}

	return &watchtower.Cursor{UserID: req.UserID}, nil
}

func (r *stubRepo) SetCursor(
	_ context.Context,
	cursor *watchtower.Cursor,
) (*watchtower.Cursor, error) {
	r.cursor = cursor

	return cursor, nil
}

type stubDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	subscribers []*agentdefinition.Definition
	enabled     []*agentdefinition.Definition
	lastTrigger agentdefinitionports.ListAgentDefinitionsByTriggerRequest
}

func (d *stubDefinitions) ListEnabledByTrigger(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByTriggerRequest,
) ([]*agentdefinition.Definition, error) {
	d.lastTrigger = req

	return d.subscribers, nil
}

func (d *stubDefinitions) List(
	_ context.Context,
	_ *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	return &pagination.ListResult[*agentdefinition.Definition]{Items: d.enabled}, nil
}

type stubEvents struct {
	published []agentports.AgentEvent
}

func (e *stubEvents) Publish(_ context.Context, event agentports.AgentEvent) {
	e.published = append(e.published, event)
}

type stubRuns struct {
	agentports.AgentRunService

	last *agentports.StartAgentRunForDefinitionRequest
}

func (r *stubRuns) StartForDefinition(
	_ context.Context,
	req *agentports.StartAgentRunForDefinitionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	r.last = req

	return &agent.AgentRun{ID: pulid.MustNew("ar_")}, nil
}

type stubSource struct {
	kind  watchtower.SourceKind
	items []watchtowerports.WatchtowerItemInput
	err   error
}

func (s *stubSource) Kind() watchtower.SourceKind { return s.kind }

func (s *stubSource) Snapshot(
	context.Context,
	pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	return s.items, s.err
}

func testActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func newService(
	repo *stubRepo,
	perms *agenttest.StubPermissions,
	sources ...watchtowerports.WatchtowerSource,
) *Service {
	projector := NewProjector(ProjectorParams{Logger: zap.NewNop(), Repo: repo})
	projector.now = func() int64 { return 1_700_000_000 }

	svc := New(Params{
		Logger:      zap.NewNop(),
		Repo:        repo,
		Projector:   projector,
		Permissions: perms,
		Definitions: &stubDefinitions{},
		Sources:     sources,
	})
	svc.now = func() int64 { return 1_700_000_000 }

	return svc
}

func input(
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	id string,
	severity watchtower.Severity,
	at int64,
) watchtowerports.WatchtowerItemInput {
	return watchtowerports.WatchtowerItemInput{
		TenantInfo: tenant,
		SourceKind: kind,
		SourceID:   id,
		Severity:   severity,
		Title:      "Something on " + id,
		Path:       "/records/" + id,
		OccurredAt: at,
	}
}

// The feed shows a reader only the kinds whose source record they could
// open. A dispatcher without billing read sees the service failures and
// not the billing exceptions, however the request names them.
func TestList_ShowsOnlyKindsTheReaderMayOpen(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(
		repo,
		&agenttest.StubPermissions{Denied: map[string]bool{"billing_queue:read": true}},
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceBillingException, "bqi_1", watchtower.SeverityWarning, 200),
	)

	page, err := svc.List(
		t.Context(),
		watchtowerports.ListWatchtowerItemsRequest{TenantInfo: tenant, UnresolvedOnly: true},
		actor,
	)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, watchtower.SourceServiceFailure, page.Items[0].SourceKind)
	assert.NotContains(t, repo.lastList.Kinds, watchtower.SourceBillingException)

	narrowed, err := svc.List(t.Context(), watchtowerports.ListWatchtowerItemsRequest{
		TenantInfo: tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceBillingException},
	}, actor)
	require.NoError(t, err)
	assert.Empty(
		t,
		narrowed.Items,
		"asking for a kind you may not see yields nothing, not an error",
	)
}

func TestList_MarksWhatTheReaderHasSeen(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(repo, &agenttest.StubPermissions{})
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceInsight, "ins_old", watchtower.SeverityInfo, 100),
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceInsight, "ins_new", watchtower.SeverityCritical, 300),
	)

	counts, err := svc.MarkSeen(t.Context(), tenant, actor, 200)
	require.NoError(t, err)
	assert.Equal(t, 1, counts.UnseenCritical)
	assert.Equal(t, int64(200), counts.SeenAt)

	page, err := svc.List(
		t.Context(),
		watchtowerports.ListWatchtowerItemsRequest{TenantInfo: tenant},
		actor,
	)
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.SourceID] = item.Seen
	}
	assert.True(t, seen["ins_old"])
	assert.False(t, seen["ins_new"])
}

func TestUpsert_KeepsTheRowAndReopensIt(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(repo, &agenttest.StubPermissions{})

	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)
	first := repo.items[repo.key(watchtower.SourceServiceFailure, "sf_1")]
	svc.projector.Resolve(t.Context(), tenant, watchtower.SourceServiceFailure, "sf_1")
	require.NotNil(t, first.ResolvedAt)

	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityCritical, 150),
	)
	again := repo.items[repo.key(watchtower.SourceServiceFailure, "sf_1")]
	assert.Equal(t, first.ID, again.ID, "the same source keeps its item")
	assert.Nil(t, again.ResolvedAt, "a source reported open again is open again")
	assert.Equal(t, watchtower.SeverityCritical, again.Severity)
}

func TestUpsert_RefusesAPathThatLeavesTheApplication(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(repo, &agenttest.StubPermissions{})

	bad := input(tenant, watchtower.SourceInsight, "ins_1", watchtower.SeverityInfo, 100)
	bad.Path = "//evil.example/x"
	err := svc.projector.upsert(t.Context(), bad)
	require.Error(t, err)
	assert.Empty(t, repo.items)
}

func TestHandOff_PublishesToSubscribersOrNamesWhoCouldTakeIt(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	events := &stubEvents{}
	definitions := &stubDefinitions{}
	svc := newService(repo, &agenttest.StubPermissions{})
	svc.definitions = definitions
	svc.events = events

	subject := pulid.MustNew("shp_")
	in := input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100)
	in.SubjectType = agent.SubjectShipment
	in.SubjectID = subject
	in.EventKind = agent.EventServiceFailureDetected
	require.NoError(t, svc.projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]

	// Nobody subscribes: the person is told who could take it.
	definitions.enabled = []*agentdefinition.Definition{
		{
			ID:          pulid.MustNew("agdef_"),
			Name:        "Load monitor",
			TriggerMode: agentdefinition.TriggerScheduled,
		},
		{
			ID:          pulid.MustNew("agdef_"),
			Name:        "Chat helper",
			TriggerMode: agentdefinition.TriggerChat,
		},
	}
	result, err := svc.HandOff(
		t.Context(),
		watchtowerports.HandOffWatchtowerItemRequest{TenantInfo: tenant, ItemID: item.ID},
		actor,
	)
	require.NoError(t, err)
	assert.Empty(t, events.published)
	require.Len(t, result.Candidates, 1, "a chat agent is not a candidate to run on a record")
	assert.Equal(t, "Load monitor", result.Candidates[0].Name)
	assert.Equal(t, agent.EventServiceFailureDetected, definitions.lastTrigger.EventKind)

	// A subscriber: the event is published to it.
	definitions.subscribers = []*agentdefinition.Definition{{Name: "Service desk"}}
	result, err = svc.HandOff(
		t.Context(),
		watchtowerports.HandOffWatchtowerItemRequest{TenantInfo: tenant, ItemID: item.ID},
		actor,
	)
	require.NoError(t, err)
	require.Len(t, events.published, 1)
	assert.Equal(t, subject, events.published[0].SubjectID)
	assert.Len(t, result.Subscribers, 1)

	// A named agent runs on the subject at once.
	runs := &stubRuns{}
	svc.runs = runs
	chosen := pulid.MustNew("agdef_")
	result, err = svc.HandOff(t.Context(), watchtowerports.HandOffWatchtowerItemRequest{
		TenantInfo: tenant, ItemID: item.ID, AgentDefinitionID: chosen,
	}, actor)
	require.NoError(t, err)
	require.NotNil(t, result.Run)
	assert.Equal(t, chosen, runs.last.DefinitionID)
	assert.Equal(t, subject, runs.last.SubjectID)
	assert.Equal(t, agent.RunTriggerManual, runs.last.Trigger)
}

func TestHandOff_SuggestsTheStarterThatListensForTheEvent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		event    agent.EventKind
		source   watchtower.SourceKind
		template agentdefinition.Template
	}{
		{agent.EventShipmentCreated, watchtower.SourceServiceFailure,
			agentdefinition.TemplateLoadEntryCheck},
		{agent.EventShipmentMoveUnassigned, watchtower.SourceMoveCoverage,
			agentdefinition.TemplateDispatchAssignment},
		{agent.EventServiceFailureDetected, watchtower.SourceServiceFailure,
			agentdefinition.TemplateServiceFailureDesk},
		{agent.EventInsightDetected, watchtower.SourceInsight,
			agentdefinition.TemplateInsightAnalyst},
		{agent.EventEDIFileQuarantined, watchtower.SourceEDIInboundQuarantined,
			agentdefinition.TemplateEDIDesk},
		{agent.EventBillingQueueItemOnHold, watchtower.SourceBillingException,
			agentdefinition.TemplateBillingException},
	}

	for _, tc := range cases {
		t.Run(string(tc.event), func(t *testing.T) {
			t.Parallel()

			repo := newStubRepo()
			actor := testActor()
			tenant := actor.TenantInfo()
			svc := newService(repo, &agenttest.StubPermissions{})
			svc.definitions = &stubDefinitions{}
			svc.events = &stubEvents{}

			in := input(tenant, tc.source, string(tc.event), watchtower.SeverityWarning, 100)
			in.SubjectType = tc.event.SubjectType()
			in.SubjectID = pulid.MustNew("sub_")
			in.EventKind = tc.event
			require.NoError(t, svc.projector.upsert(t.Context(), in))
			item := repo.items[repo.key(in.SourceKind, in.SourceID)]

			result, err := svc.HandOff(
				t.Context(),
				watchtowerports.HandOffWatchtowerItemRequest{TenantInfo: tenant, ItemID: item.ID},
				actor,
			)
			require.NoError(t, err)
			assert.Contains(t, result.Templates, tc.template)
		})
	}
}

func TestHandOff_RefusesAnItemTheReaderMayNotSee(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	svc := newService(
		repo,
		&agenttest.StubPermissions{Denied: map[string]bool{"billing_queue:read": true}},
	)
	in := input(tenant, watchtower.SourceBillingException, "bqi_1", watchtower.SeverityWarning, 100)
	in.SubjectType = agent.SubjectBillingQueueItem
	in.SubjectID = pulid.MustNew("bqi_")
	require.NoError(t, svc.projector.upsert(t.Context(), in))
	item := repo.items[repo.key(in.SourceKind, in.SourceID)]

	_, err := svc.HandOff(
		t.Context(),
		watchtowerports.HandOffWatchtowerItemRequest{TenantInfo: tenant, ItemID: item.ID},
		actor,
	)
	require.Error(t, err)
	_, err = svc.Dismiss(t.Context(), tenant, item.ID, actor)
	require.Error(t, err)
}

// Reconcile trusts the sources: what they report is upserted, and what they
// stopped reporting is resolved. A source that cannot be read leaves its
// items alone rather than resolving them all on a read error.
func TestReconcile_FollowsTheSourcesAndSurvivesOneFailing(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	actor := testActor()
	tenant := actor.TenantInfo()
	failures := &stubSource{
		kind: watchtower.SourceServiceFailure,
		items: []watchtowerports.WatchtowerItemInput{
			input(tenant, watchtower.SourceServiceFailure, "sf_2", watchtower.SeverityWarning, 200),
		},
	}
	broken := &stubSource{kind: watchtower.SourceInsight, err: assert.AnError}
	svc := newService(repo, &agenttest.StubPermissions{}, failures, broken)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceInsight, "ins_1", watchtower.SeverityInfo, 100),
	)

	result, err := svc.Reconcile(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Upserted)
	assert.Equal(t, 1, result.Resolved)
	assert.Equal(t, []string{watchtower.SourceInsight.String()}, result.Failed)
	assert.NotNil(
		t,
		repo.items[repo.key(watchtower.SourceServiceFailure, "sf_1")].ResolvedAt,
		"no longer reported, so resolved",
	)
	assert.Nil(t, repo.items[repo.key(watchtower.SourceServiceFailure, "sf_2")].ResolvedAt)
	assert.Nil(
		t,
		repo.items[repo.key(watchtower.SourceInsight, "ins_1")].ResolvedAt,
		"a source that failed to read resolves nothing",
	)
	assert.Equal(
		t,
		[]watchtower.SourceKind{watchtower.SourceInsight, watchtower.SourceServiceFailure},
		svc.RegisteredKinds(),
	)
}

func TestReconcileKind_TouchesOnlyTheNamedKind(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	tenant := testActor().TenantInfo()
	quiet := &stubSource{kind: watchtower.SourceNoCheckCall}
	failures := &stubSource{kind: watchtower.SourceServiceFailure}
	svc := newService(repo, &agenttest.StubPermissions{}, quiet, failures)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceNoCheckCall, "sm_1", watchtower.SeverityWarning, 100),
	)
	svc.projector.Upsert(
		t.Context(),
		input(tenant, watchtower.SourceServiceFailure, "sf_1", watchtower.SeverityWarning, 100),
	)

	result, err := svc.ReconcileKind(t.Context(), tenant, watchtower.SourceNoCheckCall)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Resolved)
	assert.NotNil(t, repo.items[repo.key(watchtower.SourceNoCheckCall, "sm_1")].ResolvedAt)
	assert.Nil(
		t,
		repo.items[repo.key(watchtower.SourceServiceFailure, "sf_1")].ResolvedAt,
		"another kind is left for the nightly reconcile",
	)
	require.Len(t, repo.resolved, 1)
	assert.Equal(t, watchtower.SourceNoCheckCall, repo.resolved[0].SourceKind)
}

func TestReconcileKind_RefusesAnUnknownKind(t *testing.T) {
	t.Parallel()

	svc := newService(newStubRepo(), &agenttest.StubPermissions{})

	_, err := svc.ReconcileKind(
		t.Context(),
		testActor().TenantInfo(),
		watchtower.SourceKind("Nope"),
	)

	require.Error(t, err)
}
