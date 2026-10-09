package agenttoolruleservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shipmentTool = "update_shipment"

type fakeRepo struct {
	mu    sync.Mutex
	rows  map[string]*agent.ToolRuleOverride
	lists int
	err   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: map[string]*agent.ToolRuleOverride{}}
}

func (f *fakeRepo) List(context.Context, pagination.TenantInfo) ([]*agent.ToolRuleOverride, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]*agent.ToolRuleOverride, 0, len(f.rows))
	for _, row := range f.rows {
		copied := *row
		out = append(out, &copied)
	}

	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, req repositories.GetToolRuleOverrideRequest) (*agent.ToolRuleOverride, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ToolName]
	if !ok {
		return nil, nil
	}
	copied := *row

	return &copied, nil
}

func (f *fakeRepo) Create(_ context.Context, override *agent.ToolRuleOverride) (*agent.ToolRuleOverride, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	saved := *override
	saved.ID = pulid.MustNew(agent.ToolRuleOverrideIDPrefix)
	f.rows[saved.ToolName] = &saved
	copied := saved

	return &copied, nil
}

func (f *fakeRepo) Update(_ context.Context, override *agent.ToolRuleOverride) (*agent.ToolRuleOverride, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	saved := *override
	saved.Version++
	f.rows[saved.ToolName] = &saved
	copied := saved

	return &copied, nil
}

type fakePolicies map[string]services.ToolPolicy

func (f fakePolicies) Get(name string) (services.ToolPolicy, bool) {
	policy, ok := f[name]

	return policy, ok
}

type fakeImpact struct {
	requests []services.RuleImpactRequest
}

func (f *fakeImpact) RuleImpact(
	_ context.Context,
	req *services.RuleImpactRequest,
) ([]services.AgentToolRuleImpact, error) {
	f.requests = append(f.requests, *req)
	answer := func(policy services.ToolPolicy) agent.AutonomyAnswer {
		if policy.MaxTier == agent.TierAutoExecute {
			return agent.AutonomyRunsOnItsOwn
		}

		return agent.AutonomyNeedsApproval
	}

	return []services.AgentToolRuleImpact{
		{AgentID: "agd_moved", AgentName: "Dispatcher", Before: answer(req.Before), After: answer(req.After)},
		{AgentID: "agd_same", AgentName: "Clerk", Before: agent.AutonomyNeedsApproval, After: agent.AutonomyNeedsApproval},
	}, nil
}

type fakeAuditor struct {
	changes []*services.SecurityChange
}

func (f *fakeAuditor) RecordChange(_ context.Context, change *services.SecurityChange) {
	f.changes = append(f.changes, change)
}

type fixture struct {
	service *Service
	repo    *fakeRepo
	impact  *fakeImpact
	auditor *fakeAuditor
	tenant  pagination.TenantInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	repo := newFakeRepo()
	impact := &fakeImpact{}
	auditor := &fakeAuditor{}

	return &fixture{
		service: &Service{
			repo:   repo,
			loader: newLoader(repo, time.Now),
			policies: fakePolicies{shipmentTool: {
				Name:          shipmentTool,
				Kind:          agent.ToolKindAction,
				DefaultTier:   agent.TierActWithApproval,
				MaxTier:       agent.TierAutoExecute,
				ReadsExternal: agent.ExternalReadNever,
			}},
			impact:  impact,
			auditor: auditor,
		},
		repo:    repo,
		impact:  impact,
		auditor: auditor,
		tenant:  pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
	}
}

func (f *fixture) save(t *testing.T, req *services.SaveToolRuleRequest) (*services.SaveToolRuleResult, error) {
	t.Helper()
	req.TenantInfo = f.tenant
	if req.ToolName == "" {
		req.ToolName = shipmentTool
	}

	return f.service.Save(t.Context(), req, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    "usr_1",
		UserID:         "usr_1",
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
	})
}

func TestSave_HoldsAToolLowerAndRecordsWhoItMoves(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	result, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierActWithApproval},
		Reason: "  Carrier changes need a person this quarter  ",
	})
	require.NoError(t, err)

	assert.Equal(t, agent.TierActWithApproval, result.Override.MaxTier)
	assert.Equal(t, "Carrier changes need a person this quarter", result.Override.Reason)
	assert.Equal(t, pulid.ID("usr_1"), result.Override.UpdatedByID)
	require.Len(t, result.Affected, 1, "only agents whose answer moved are returned")
	assert.Equal(t, pulid.ID("agd_moved"), result.Affected[0].AgentID)
	assert.Equal(t, agent.AutonomyRunsOnItsOwn, result.Affected[0].Before)
	assert.Equal(t, agent.AutonomyNeedsApproval, result.Affected[0].After)

	require.Len(t, f.auditor.changes, 1)
	change := f.auditor.changes[0]
	assert.Equal(t, shipmentTool, change.ResourceID)
	assert.Equal(t, "Tool rule changed: Carrier changes need a person this quarter", change.Comment)
	assert.Equal(t, "AutoExecute", change.Before.(map[string]any)["maxTier"])
	assert.Equal(t, "ActWithApproval", change.After.(map[string]any)["maxTier"])
	agents := change.Metadata["affectedAgents"].([]map[string]string)
	require.Len(t, agents, 1)
	assert.Equal(t, "Dispatcher", agents[0]["name"])
}

func TestSave_RequiresAReasonWhenTheMostFreedomChanges(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierPropose},
		Reason: "   ",
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "reason", multiErr.Errors[0].Field)
	assert.Empty(t, f.repo.rows, "nothing is written")
	assert.Empty(t, f.auditor.changes)
}

func TestSave_TakesOutsideTextWithoutAReason(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	result, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{ReadsExternal: agent.ExternalReadAlways},
	})
	require.NoError(t, err)
	assert.Equal(t, agent.ExternalReadAlways, result.Override.ReadsExternal)
	assert.Empty(t, result.Override.MaxTier, "the declared most freedom is kept")
}

func TestSave_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierPropose},
		Reason: "first",
	})
	require.NoError(t, err)

	_, err = f.save(t, &services.SaveToolRuleRequest{
		Change:  services.ToolRuleChange{MaxTier: agent.TierActWithApproval},
		Reason:  "second",
		Version: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Version mismatch")
	assert.Equal(t, agent.TierPropose, f.repo.rows[shipmentTool].MaxTier)

	_, err = f.save(t, &services.SaveToolRuleRequest{
		Change:  services.ToolRuleChange{MaxTier: agent.TierActWithApproval},
		Reason:  "second",
		Version: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.TierActWithApproval, f.repo.rows[shipmentTool].MaxTier)
	assert.Equal(t, int64(1), f.repo.rows[shipmentTool].Version)
}

func TestSave_ReturningToTheDeclaredRuleKeepsTheRow(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierPropose},
		Reason: "hold",
	})
	require.NoError(t, err)

	result, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierAutoExecute},
		Reason: "release",
	})
	require.NoError(t, err)
	assert.True(t, result.Override.Empty(), "choosing the declared tier stores no override")
	assert.Equal(t, int64(1), result.Override.Version)
}

func TestSave_RefusesLooseningAndUnknownTools(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{ReadsExternal: agent.ExternalReadNever, MaxTier: "Anything"},
		Reason: "loosen",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "maxTier", multiErr.Errors[0].Field)

	_, err = f.save(t, &services.SaveToolRuleRequest{ToolName: "no_such_tool", Reason: "x"})
	require.Error(t, err)
	assert.Empty(t, f.repo.rows)
}

func TestSave_ForgetsTheCachedRules(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	rules, err := f.service.For(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Empty(t, rules)

	_, err = f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierPropose},
		Reason: "hold",
	})
	require.NoError(t, err)

	rules, err = f.service.For(t.Context(), f.tenant)
	require.NoError(t, err)
	require.Contains(t, rules, shipmentTool)
	assert.Equal(t, agent.TierPropose, rules[shipmentTool].MaxTier)
}

func TestImpact_ComparesTheCurrentRuleWithTheProposedOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.save(t, &services.SaveToolRuleRequest{
		Change: services.ToolRuleChange{MaxTier: agent.TierActWithApproval},
		Reason: "hold",
	})
	require.NoError(t, err)
	f.impact.requests = nil

	impacts, err := f.service.Impact(t.Context(), &services.ToolRuleImpactRequest{
		TenantInfo: f.tenant,
		ToolName:   shipmentTool,
		Change:     services.ToolRuleChange{MaxTier: agent.TierPropose},
	})
	require.NoError(t, err)
	assert.Len(t, impacts, 2, "the preview lists every holder, moved or not")
	require.Len(t, f.impact.requests, 1)
	assert.Equal(t, agent.TierActWithApproval, f.impact.requests[0].Before.MaxTier)
	assert.Equal(t, agent.TierPropose, f.impact.requests[0].After.MaxTier)
}

func TestLoader_CachesPerTenantUntilItExpires(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Unix(1_700_000_000, 0)
	loader := newLoader(repo, func() time.Time { return now })
	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

	for range 3 {
		_, err := loader.For(t.Context(), tenant)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.lists)

	now = now.Add(loaderTTL)
	_, err := loader.For(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.lists)

	_, err = loader.For(t.Context(), pagination.TenantInfo{OrgID: "org_2", BuID: "bu_2"})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.lists, "each tenant reads its own rules")

	loader.Forget(tenant)
	_, err = loader.For(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, 4, repo.lists)
}

func TestLoader_ReturnsTheReadError(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	repo.err = errors.New("db down")
	loader := newLoader(repo, time.Now)

	_, err := loader.For(t.Context(), pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"})
	require.ErrorIs(t, err, repo.err)
}
