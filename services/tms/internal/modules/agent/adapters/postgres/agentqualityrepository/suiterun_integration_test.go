//go:build integration

package agentqualityrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type suiteAgentRow struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func scoredRun(
	tenant pagination.TenantInfo,
	agentID pulid.ID,
	startedAt int64,
	score float64,
) *agentqualitysuite.SuiteRun {
	finished := startedAt + 60

	return &agentqualitysuite.SuiteRun{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: agentID,
		Trigger:           agentqualitysuite.SuiteRunTriggerManual,
		FingerprintHash:   strings.Repeat("f", agentqualitysuite.SuiteRevisionChars),
		SuiteRevision:     strings.Repeat("r", agentqualitysuite.SuiteRevisionChars),
		Status:            agentqualitysuite.SuiteRunStatusCompleted,
		CasesTotal:        12,
		CasesPassed:       12,
		QualityScore:      &score,
		StartedAt:         startedAt,
		FinishedAt:        &finished,
	}
}

func TestSuiteRuns_ATrialNeverScoresTheAgent(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	agents, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowToStructByPos[suiteAgentRow],
		"SELECT id, organization_id, business_unit_id FROM agent_definitions LIMIT 1",
	)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	agent := agents[0]
	tenant := pagination.TenantInfo{OrgID: agent.OrganizationID, BuID: agent.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	repo := NewSuiteRuns(SuiteRunParams{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	const now = int64(1_800_000_000)
	own, err := repo.Create(ctx, scoredRun(tenant, agent.ID, now-7200, 0.9))
	require.NoError(t, err)

	providerID := pulid.MustNew("aip_")
	trial := scoredRun(tenant, agent.ID, now-3600, 0.4)
	trial.TrialProviderID = &providerID
	trial.TrialLabel = "Anthropic fast · claude-haiku-5-5"
	trial.Regression = true
	trial, err = repo.Create(ctx, trial)
	require.NoError(t, err)

	scored := []agentqualitysuite.SuiteRunStatus{agentqualitysuite.SuiteRunStatusCompleted}

	last, err := repo.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: agent.ID,
		TenantInfo:        tenant,
		Statuses:          scored,
	})
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, own.ID, last.ID,
		"the newer trial is not the baseline the agent's next run is compared with")

	any, err := repo.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: agent.ID,
		TenantInfo:        tenant,
		Statuses:          scored,
		IncludeTrials:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, any)
	assert.Equal(t, trial.ID, any.ID, "a stale running trial still has to be found to be closed")

	latest, err := repo.Latest(ctx, agentqualityports.LatestAgentSuiteRunsRequest{
		TenantInfo:         tenant,
		AgentDefinitionIDs: []pulid.ID{agent.ID},
		Statuses:           scored,
	})
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.Equal(t, own.ID, latest[0].ID, "the agent's score is its own last run's")

	history, err := repo.History(ctx, agentqualityports.AgentSuiteRunHistoryRequest{
		TenantInfo:         tenant,
		AgentDefinitionIDs: []pulid.ID{agent.ID},
		Since:              now - 86400,
		PerAgent:           10,
	})
	require.NoError(t, err)
	require.Len(t, history, 1, "a trial's score is not part of the agent's median")
	assert.Equal(t, own.ID, history[0].ID)

	overview, err := repo.Overview(ctx, agentqualityports.AgentSuiteRunOverviewRequest{
		TenantInfo: tenant,
		Since:      now - 86400,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, overview.Runs)
	assert.Zero(t, overview.Regressions)

	stored, err := repo.GetByID(ctx, agentqualityports.GetAgentSuiteRunRequest{
		ID:         trial.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.NotNil(t, stored.TrialProviderID)
	assert.Equal(t, providerID, *stored.TrialProviderID)
	assert.Equal(t, "Anthropic fast · claude-haiku-5-5", stored.TrialLabel)
}
