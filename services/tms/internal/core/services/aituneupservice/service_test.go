package aituneupservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	now = int64(1_800_000_000)
	day = int64(24 * 60 * 60)
)

var tenantInfo = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}

type tuneUpRepo struct {
	rows    map[pulid.ID]*aituneup.TuneUp
	saved   []*aituneup.Plan
	decided []*repositories.DecideAITuneUpRequest
}

func newTuneUpRepo(rows ...*aituneup.TuneUp) *tuneUpRepo {
	repo := &tuneUpRepo{rows: map[pulid.ID]*aituneup.TuneUp{}}
	for _, row := range rows {
		repo.rows[row.ID] = row
	}
	return repo
}

func (r *tuneUpRepo) List(context.Context, pagination.TenantInfo) ([]*aituneup.TuneUp, error) {
	out := make([]*aituneup.TuneUp, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	return out, nil
}

func (r *tuneUpRepo) GetByID(_ context.Context, req repositories.GetAITuneUpRequest) (*aituneup.TuneUp, error) {
	row, ok := r.rows[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("AITuneUp not found")
	}
	copied := *row
	return &copied, nil
}

func (r *tuneUpRepo) Save(_ context.Context, _ pagination.TenantInfo, plan *aituneup.Plan) error {
	r.saved = append(r.saved, plan)
	for idx, row := range plan.Insert {
		row.ID = pulid.ID("aitu_new_" + string(rune('a'+idx)))
		r.rows[row.ID] = row
	}
	for _, row := range plan.Update {
		r.rows[row.ID] = row
	}
	for _, id := range plan.Delete {
		delete(r.rows, id)
	}
	return nil
}

func (r *tuneUpRepo) Decide(_ context.Context, req *repositories.DecideAITuneUpRequest) (*aituneup.TuneUp, error) {
	r.decided = append(r.decided, req)
	row := r.rows[req.ID]
	if row.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("AITuneUp", req.ID.String())
	}
	row.Status = req.Status
	row.DismissedUntil = req.DismissedUntil
	row.DecidedByID = req.DecidedByID
	row.Version++
	copied := *row
	return &copied, nil
}

func (r *tuneUpRepo) Reopen(_ context.Context, req *repositories.ReopenAITuneUpRequest) (*aituneup.TuneUp, error) {
	row := r.rows[req.ID]
	row.Status = aituneup.StatusOpen
	row.DismissedUntil = nil
	row.Version++
	copied := *row
	return &copied, nil
}

type freshness struct {
	fresh  bool
	marked int
	err    error
}

func (f *freshness) Fresh(context.Context, pagination.TenantInfo) (bool, error) {
	return f.fresh, f.err
}

func (f *freshness) Mark(context.Context, pagination.TenantInfo, int64, time.Duration) error {
	f.marked++
	f.fresh = true
	return nil
}

func (f *freshness) Clear(context.Context, pagination.TenantInfo) error { return nil }

type controlRepo struct {
	repositories.AgentControlRepository
	control *tenant.AgentControl
}

func (c *controlRepo) GetOrCreate(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
	return c.control, nil
}

type definitionRepo struct {
	repositories.AgentDefinitionRepository
	agents []*agentdefinition.Definition
}

func (d *definitionRepo) List(
	_ context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	offset := req.Filter.Pagination.Offset
	if offset >= len(d.agents) {
		return &pagination.ListResult[*agentdefinition.Definition]{}, nil
	}
	return &pagination.ListResult[*agentdefinition.Definition]{Items: d.agents[offset:], Total: len(d.agents)}, nil
}

func (d *definitionRepo) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	out := make([]*agentdefinition.Definition, 0)
	for _, agent := range d.agents {
		for _, id := range req.IDs {
			if agent.ID == id {
				out = append(out, agent)
			}
		}
	}
	return out, nil
}

type providerRepo struct {
	repositories.AIProviderRepository
	providers []*aiprovider.Provider
}

func (p *providerRepo) ListEnabled(context.Context, pagination.TenantInfo) ([]*aiprovider.Provider, error) {
	out := make([]*aiprovider.Provider, 0)
	for _, provider := range p.providers {
		if provider.Enabled {
			out = append(out, provider)
		}
	}
	return out, nil
}

func (p *providerRepo) ListOrdered(context.Context, pagination.TenantInfo) ([]*aiprovider.Provider, error) {
	return p.providers, nil
}

type usageRepo struct {
	repositories.AIUsageRepository
	rows []repositories.AIUsageProviderTaskTotals
}

func (u *usageRepo) ProviderTaskTotals(
	context.Context,
	repositories.AIUsageProviderTaskRequest,
) ([]repositories.AIUsageProviderTaskTotals, error) {
	return u.rows, nil
}

type trust struct {
	services.AgentTrustService
	ready    []services.ToolPromotion
	promoted []*services.PromoteToolRequest
}

func (t *trust) PromotionCandidates(context.Context, *services.PromoteReadyRequest) ([]services.ToolPromotion, error) {
	return t.ready, nil
}

func (t *trust) PromoteTool(_ context.Context, req *services.PromoteToolRequest) (*services.ToolPromotion, error) {
	t.promoted = append(t.promoted, req)
	return &services.ToolPromotion{}, nil
}

type shadow struct {
	reports map[pulid.ID]*agentshadow.Report
	asked   []pulid.ID
}

func (s *shadow) Report(_ context.Context, req *services.AgentShadowReportRequest) (*agentshadow.Report, error) {
	s.asked = append(s.asked, req.AgentID)
	if report, ok := s.reports[req.AgentID]; ok {
		return report, nil
	}
	return &agentshadow.Report{}, nil
}

type agentService struct {
	services.AgentDefinitionService
	agents  map[pulid.ID]*agentdefinition.Definition
	patched []string
}

func (a *agentService) Patch(
	_ context.Context,
	req *services.PatchAgentDefinitionRequest,
	_ *services.RequestActor,
) (*agentdefinition.Definition, error) {
	definition := a.agents[req.ID]
	if err := req.Edit(definition); err != nil {
		return nil, err
	}
	a.patched = append(a.patched, req.Comment)
	return definition, nil
}

type providerService struct {
	services.AIProviderService
	reordered [][]pulid.ID
	assigned  []*services.AssignAIProviderTaskRequest
	fail      error
}

func (p *providerService) Reorder(
	_ context.Context,
	req *services.ReorderAIProvidersRequest,
	_ *services.RequestActor,
) ([]*aiprovider.Provider, error) {
	p.reordered = append(p.reordered, req.ProviderIDs)
	return nil, p.fail
}

func (p *providerService) AssignTask(
	_ context.Context,
	req *services.AssignAIProviderTaskRequest,
	_ *services.RequestActor,
) (*aiprovider.Provider, error) {
	p.assigned = append(p.assigned, req)
	return nil, p.fail
}

type fixture struct {
	svc       *Service
	repo      *tuneUpRepo
	fresh     *freshness
	control   *tenant.AgentControl
	agents    *definitionRepo
	providers *providerRepo
	trust     *trust
	shadow    *shadow
	agentSvc  *agentService
	provSvc   *providerService
	usage     *usageRepo
}

func newFixture(t *testing.T, rows ...*aituneup.TuneUp) *fixture {
	t.Helper()

	f := &fixture{
		repo:      newTuneUpRepo(rows...),
		fresh:     &freshness{fresh: true},
		control:   &tenant.AgentControl{EarnedAutonomy: false, PromotionThreshold: 10},
		agents:    &definitionRepo{},
		providers: &providerRepo{},
		trust:     &trust{},
		shadow:    &shadow{reports: map[pulid.ID]*agentshadow.Report{}},
		agentSvc:  &agentService{agents: map[pulid.ID]*agentdefinition.Definition{}},
		provSvc:   &providerService{},
		usage:     &usageRepo{},
	}
	f.svc = New(Params{
		Logger:           zap.NewNop(),
		Repo:             f.repo,
		Freshness:        f.fresh,
		Control:          &controlRepo{control: f.control},
		Definitions:      f.agents,
		AgentDefinitions: f.agentSvc,
		Providers:        f.providers,
		ProviderService:  f.provSvc,
		Trust:            f.trust,
		Shadow:           f.shadow,
		Usage:            f.usage,
	})
	f.svc.now = func() int64 { return now }
	return f
}

func (f *fixture) addAgent(definition *agentdefinition.Definition) {
	f.agents.agents = append(f.agents.agents, definition)
	f.agentSvc.agents[definition.ID] = definition
}

func agent(id string) *agentdefinition.Definition {
	lastRun := now - day
	return &agentdefinition.Definition{
		ID:          pulid.ID(id),
		Name:        "Agent " + id,
		Icon:        "truck",
		Accent:      "indigo",
		Enabled:     true,
		TriggerMode: agentdefinition.TriggerChat,
		CreatedAt:   now - 90*day,
		LastRunAt:   &lastRun,
	}
}

func open(id string, kind aituneup.Kind) *aituneup.TuneUp {
	return &aituneup.TuneUp{
		ID:          pulid.ID(id),
		Kind:        kind,
		Fingerprint: string(kind) + ":" + id,
		Status:      aituneup.StatusOpen,
		ComputedAt:  now - day,
		Version:     2,
	}
}

var actor = &services.RequestActor{PrincipalType: services.PrincipalTypeUser, UserID: pulid.ID("usr_1")}

func TestListComputesWhenNothingFreshIsStored(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.fresh.fresh = false
	shadowed := agent("agd_shadow")
	shadowed.ShadowMode = true
	f.addAgent(shadowed)
	f.shadow.reports[shadowed.ID] = &agentshadow.Report{Recorded: 30, Matched: 18, WouldReject: 2}

	list, err := f.svc.List(t.Context(), tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, 1, f.fresh.marked)
	require.Len(t, list.Items, 1)
	view := list.Items[0]
	assert.Equal(t, aituneup.KindLeaveShadow, view.TuneUp.Kind)
	require.NotNil(t, view.Agent)
	assert.Equal(t, "Agent agd_shadow", view.Agent.Name)
	assert.Equal(t, "truck", view.Agent.Icon)
	require.NotNil(t, list.ComputedAt)
	assert.Equal(t, now, *list.ComputedAt)
	assert.Equal(t, aituneup.WindowDays, list.WindowDays)
}

func TestListServesWhatIsStoredWhileFresh(t *testing.T) {
	t.Parallel()

	shown := open("aitu_shown", aituneup.KindTurnOffIdleAgent)
	shown.AgentDefinitionID = pulid.ID("agd_idle")
	snoozed := open("aitu_snoozed", aituneup.KindTurnOffIdleAgent)
	snoozed.AgentDefinitionID = pulid.ID("agd_idle")
	snoozed.Status = aituneup.StatusDismissed
	until := now + day
	snoozed.DismissedUntil = &until
	orphan := open("aitu_orphan", aituneup.KindTurnOffIdleAgent)
	orphan.AgentDefinitionID = pulid.ID("agd_deleted")

	f := newFixture(t, shown, snoozed, orphan)
	f.addAgent(agent("agd_idle"))

	list, err := f.svc.List(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Empty(t, f.repo.saved, "nothing is computed while the stored set is fresh")
	require.Len(t, list.Items, 1, "a dismissed one is hidden, and one whose agent is gone is dropped")
	assert.Equal(t, shown.ID, list.Items[0].TuneUp.ID)
}

func TestListServesWhatIsStoredWhenFreshnessCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.fresh.err = errors.New("redis down")

	_, err := f.svc.List(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Empty(t, f.repo.saved)
}

func TestComputeAsksForShadowReportsOnlyOutsideAPause(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	shadowed := agent("agd_shadow")
	shadowed.ShadowMode = true
	f.addAgent(shadowed)
	f.addAgent(agent("agd_live"))

	_, err := f.svc.Compute(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{shadowed.ID}, f.shadow.asked)

	f.control.ShadowMode = true
	f.shadow.asked = nil
	_, err = f.svc.Compute(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Empty(t, f.shadow.asked)
}

func TestComputeReadsTrustOnlyWhileEarnedAutonomyIsOff(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	dispatch := agent("agd_dispatch")
	f.addAgent(dispatch)
	f.trust.ready = []services.ToolPromotion{{
		AgentDefinitionID: dispatch.ID, ToolName: "assign_driver", Streak: 11, Approvals: 22, TrackedSince: now - 14*day,
	}}

	count, err := f.svc.Compute(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	f.control.EarnedAutonomy = true
	count, err = f.svc.Compute(t.Context(), tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyLeaveShadowTakesTheAgentLive(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_live", aituneup.KindLeaveShadow)
	tuneUp.AgentDefinitionID = pulid.ID("agd_cov")
	f := newFixture(t, tuneUp)
	coverage := agent("agd_cov")
	coverage.ShadowMode = true
	f.addAgent(coverage)

	view, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)

	assert.False(t, coverage.ShadowMode)
	assert.Equal(t, aituneup.StatusApplied, view.TuneUp.Status)
	assert.Equal(t, pulid.ID("usr_1"), f.repo.decided[0].DecidedByID)
}

func TestApplyTurnOffIdleTurnsTheAgentOff(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_idle", aituneup.KindTurnOffIdleAgent)
	tuneUp.AgentDefinitionID = pulid.ID("agd_idle")
	f := newFixture(t, tuneUp)
	idle := agent("agd_idle")
	f.addAgent(idle)

	_, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	assert.False(t, idle.Enabled)
}

func TestApplyRaiseToolTierPromotesAtTheOrganizationsThreshold(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_raise", aituneup.KindRaiseToolTier)
	tuneUp.AgentDefinitionID = pulid.ID("agd_dispatch")
	tuneUp.ToolName = "assign_driver"
	f := newFixture(t, tuneUp)
	f.addAgent(agent("agd_dispatch"))
	f.control.PromotionThreshold = 25

	_, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	require.Len(t, f.trust.promoted, 1)
	assert.Equal(t, 25, f.trust.promoted[0].Threshold)
	assert.Equal(t, "assign_driver", f.trust.promoted[0].ToolName)
	assert.Equal(t, pulid.ID("usr_1"), f.trust.promoted[0].DecidedBy)
}

func TestApplyReorderPutsTheRescuerAheadOfTheFailingProvider(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_order", aituneup.KindReorderProviders)
	tuneUp.ProviderID = pulid.ID("aip_ollama")
	tuneUp.OtherProviderID = pulid.ID("aip_vllm")
	f := newFixture(t, tuneUp)
	f.providers.providers = []*aiprovider.Provider{
		{ID: "aip_cloud", Name: "Cloud", Enabled: true},
		{ID: "aip_vllm", Name: "vLLM", Enabled: true},
		{ID: "aip_other", Name: "Other", Enabled: false},
		{ID: "aip_ollama", Name: "Ollama", Enabled: true},
	}

	view, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	require.Len(t, f.provSvc.reordered, 1)
	assert.Equal(t, []pulid.ID{"aip_cloud", "aip_ollama", "aip_vllm", "aip_other"}, f.provSvc.reordered[0])
	require.NotNil(t, view.Provider)
	assert.Equal(t, "Ollama", view.Provider.Name)
	assert.Equal(t, "vLLM", view.OtherProvider.Name)
}

func TestApplyAssignTaskGivesTheProviderTheTask(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_task", aituneup.KindAssignTask)
	tuneUp.ProviderID = pulid.ID("aip_ollama")
	tuneUp.Task = aiprovider.TaskEmbedding
	f := newFixture(t, tuneUp)
	f.providers.providers = []*aiprovider.Provider{{ID: "aip_ollama", Name: "Ollama", Enabled: true}}

	_, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	require.Len(t, f.provSvc.assigned, 1)
	assert.Equal(t, aiprovider.TaskEmbedding, f.provSvc.assigned[0].Task)
}

func TestApplyLeavesTheTuneUpOpenWhenTheChangeFails(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_task", aituneup.KindAssignTask)
	tuneUp.ProviderID = pulid.ID("aip_ollama")
	tuneUp.Task = aiprovider.TaskEmbedding
	f := newFixture(t, tuneUp)
	f.provSvc.fail = errors.New("refused")

	_, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.Error(t, err)
	assert.Empty(t, f.repo.decided)
	assert.Equal(t, aituneup.StatusOpen, f.repo.rows[tuneUp.ID].Status)
}

func TestApplyRefusesAStaleOrDecidedTuneUp(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_idle", aituneup.KindTurnOffIdleAgent)
	tuneUp.AgentDefinitionID = pulid.ID("agd_idle")
	applied := open("aitu_done", aituneup.KindTurnOffIdleAgent)
	applied.Status = aituneup.StatusApplied
	f := newFixture(t, tuneUp, applied)
	idle := agent("agd_idle")
	f.addAgent(idle)

	_, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 1,
	}, actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err))
	assert.True(t, idle.Enabled, "nothing changes on a stale read")

	_, err = f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: applied.ID, Version: 2,
	}, actor)
	require.Error(t, err)
}

func TestApplyRecordsTheDecisionAfterANightlyRefreshMovedTheVersion(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_idle", aituneup.KindTurnOffIdleAgent)
	tuneUp.AgentDefinitionID = pulid.ID("agd_idle")
	f := newFixture(t, tuneUp)
	idle := agent("agd_idle")
	f.addAgent(idle)

	refreshing := &refreshingAgents{agentService: f.agentSvc, repo: f.repo, id: tuneUp.ID}
	f.svc.agentDefinitions = refreshing

	view, err := f.svc.Apply(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, aituneup.StatusApplied, view.TuneUp.Status)
	require.Len(t, f.repo.decided, 2)
	assert.Equal(t, int64(3), f.repo.decided[1].Version)
}

type refreshingAgents struct {
	*agentService
	repo *tuneUpRepo
	id   pulid.ID
}

func (r *refreshingAgents) Patch(
	ctx context.Context,
	req *services.PatchAgentDefinitionRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	r.repo.rows[r.id].Version++
	return r.agentService.Patch(ctx, req, actor)
}

func TestDismissPutsItAwayForTheDaysAskedAndRestoreBringsItBack(t *testing.T) {
	t.Parallel()

	tuneUp := open("aitu_idle", aituneup.KindTurnOffIdleAgent)
	tuneUp.AgentDefinitionID = pulid.ID("agd_idle")
	f := newFixture(t, tuneUp)
	f.addAgent(agent("agd_idle"))

	view, err := f.svc.Dismiss(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: 2,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, aituneup.StatusDismissed, view.TuneUp.Status)
	require.NotNil(t, view.TuneUp.DismissedUntil)
	assert.Equal(t, now+30*day, *view.TuneUp.DismissedUntil)
	assert.True(t, f.agentSvc.agents["agd_idle"].Enabled, "dismissing changes nothing else")

	restored, err := f.svc.Restore(t.Context(), &services.AITuneUpDecisionRequest{
		TenantInfo: tenantInfo, ID: tuneUp.ID, Version: view.TuneUp.Version,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, aituneup.StatusOpen, restored.TuneUp.Status)
	assert.Nil(t, restored.TuneUp.DismissedUntil)
}

func TestMoveAhead(t *testing.T) {
	t.Parallel()

	order := []pulid.ID{"a", "b", "c", "d"}
	got, ok := MoveAhead(order, "d", "b")
	require.True(t, ok)
	assert.Equal(t, []pulid.ID{"a", "d", "b", "c"}, got)
	assert.Equal(t, []pulid.ID{"a", "b", "c", "d"}, order, "the input is not changed")

	got, ok = MoveAhead(order, "a", "c")
	require.True(t, ok)
	assert.Equal(t, order, got, "already ahead")

	_, ok = MoveAhead(order, "z", "a")
	assert.False(t, ok)
}
