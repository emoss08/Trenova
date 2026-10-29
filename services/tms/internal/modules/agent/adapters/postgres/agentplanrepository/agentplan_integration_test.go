//go:build integration

package agentplanrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan/agentplandb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type planHarness struct {
	ctx    context.Context
	admin  *keel.DB
	repo   *repository
	tenant pagination.TenantInfo
}

func newPlanHarness(t *testing.T) *planHarness {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	return &planHarness{
		ctx:    dbscope.WithTenant(ctx, tenant.DBTenant()),
		admin:  admin,
		repo:   &repository{db: dbtest.NewRLSConnection(t, admin, dsn), l: zap.NewNop()},
		tenant: tenant,
	}
}

func (h *planHarness) run(t *testing.T, definitionID pulid.ID) *agent.AgentRun {
	t.Helper()

	run := &agent.AgentRun{
		OrganizationID:    h.tenant.OrgID,
		BusinessUnitID:    h.tenant.BuID,
		AgentDefinitionID: definitionID,
		AgentType:         agent.TypeGeneral,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         pulid.MustNew("athr_"),
		Status:            agent.RunStatusCompleted,
		PromptVersion:     "v1",
		InputContextHash:  "hash",
		StartedAt:         timeutils.NowUnix(),
	}
	require.NoError(t, agentdb.AgentRuns.Create(h.ctx, h.admin, run))

	return run
}

func (h *planHarness) plan(t *testing.T, run *agent.AgentRun) *agentplan.AgentPlan {
	t.Helper()

	plan := &agentplan.AgentPlan{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		RunID:          run.ID,
		Title:          "Move the loads",
		Status:         agentplan.PlanStatusPending,
		StepCount:      2,
	}
	require.NoError(t, agentplandb.AgentPlans.Create(h.ctx, h.admin, plan))

	return plan
}

func (h *planHarness) read(t *testing.T, id pulid.ID) *agentplan.AgentPlan {
	t.Helper()

	plan, err := agentplandb.AgentPlans.Get(h.ctx, h.admin, id)
	require.NoError(t, err)

	return plan
}

func TestUpdateStatus_KeepsTheApprovalWithTheWindowAndClearsItOnReopen(t *testing.T) {
	h := newPlanHarness(t)
	plan := h.plan(t, h.run(t, ""))

	approving, err := h.repo.UpdateStatus(h.ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:         plan.ID,
		TenantInfo: h.tenant,
		Status:     agentplan.PlanStatusApproving,
		CommitsAt:  5,
		Approval:   agentplanports.PlanApproval{ReasonCode: "looks right", PreviewDigest: "abc"},
	})
	require.NoError(t, err)
	assert.Equal(t, agentplan.PlanStatusApproving, approving.Status)
	require.NotNil(t, approving.CommitsAt)
	assert.Equal(t, int64(5), *approving.CommitsAt)
	assert.Equal(t, "looks right", approving.ApprovalReasonCode)
	assert.Equal(t, "abc", approving.ApprovalPreviewDigest)
	assert.Empty(t, approving.ApprovalNote)
	assert.Nil(t, approving.UndoneAt)
	assert.Greater(t, approving.Version, plan.Version)

	reopened, err := h.repo.UpdateStatus(h.ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:         plan.ID,
		TenantInfo: h.tenant,
		Status:     agentplan.PlanStatusPending,
		Reopen:     true,
	})
	require.NoError(t, err)
	stored := h.read(t, reopened.ID)
	assert.Equal(t, agentplan.PlanStatusPending, stored.Status)
	assert.Nil(t, stored.CommitsAt)
	assert.Empty(t, stored.ApprovalReasonCode)
	assert.Empty(t, stored.ApprovalPreviewDigest)
	assert.Nil(t, stored.DecidedByUserID)
	assert.Nil(t, stored.DecidedAt)
}

func TestUpdateStatus_ADecisionWithoutAWindowKeepsItsNoteAndDigest(t *testing.T) {
	h := newPlanHarness(t)
	plan := h.plan(t, h.run(t, ""))

	rejected, err := h.repo.UpdateStatus(h.ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:              plan.ID,
		TenantInfo:      h.tenant,
		Status:          agentplan.PlanStatusRejected,
		DecidedByUserID: h.tenant.UserID,
		DecidedAt:       9,
		Approval:        agentplanports.PlanApproval{Note: "Not today"},
	})
	require.NoError(t, err)
	assert.Equal(t, agentplan.PlanStatusRejected, rejected.Status)
	require.NotNil(t, rejected.DecidedByUserID)
	assert.Equal(t, h.tenant.UserID, *rejected.DecidedByUserID)
	require.NotNil(t, rejected.DecidedAt)
	assert.Equal(t, int64(9), *rejected.DecidedAt)
	assert.Equal(t, "Not today", rejected.ApprovalNote)
	assert.Empty(t, rejected.ApprovalReasonCode)
	assert.Nil(t, rejected.CommitsAt)
}

func TestUpdateStatus_AStaleFromStatusIsNotFound(t *testing.T) {
	h := newPlanHarness(t)
	plan := h.plan(t, h.run(t, ""))

	_, err := h.repo.UpdateStatus(h.ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:         plan.ID,
		TenantInfo: h.tenant,
		Status:     agentplan.PlanStatusApproved,
		FromStatus: agentplan.PlanStatusApproving,
	})
	require.Error(t, err)
	assert.Equal(t, agentplan.PlanStatusPending, h.read(t, plan.ID).Status)
}

func TestExpirePendingByDefinitionExpiresOnlyThePendingPlansOfTheAgentsRuns(t *testing.T) {
	h := newPlanHarness(t)
	definition := &agentdefinition.Definition{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Name:           "Dispatch",
		Enabled:        true,
	}
	require.NoError(t, agentdefinitiondb.Definitions.Create(h.ctx, h.admin, definition))

	pending := h.plan(t, h.run(t, definition.ID))
	decided := h.plan(t, h.run(t, definition.ID))
	_, err := h.repo.UpdateStatus(h.ctx, agentplanports.UpdateAgentPlanStatusRequest{
		ID:              decided.ID,
		TenantInfo:      h.tenant,
		Status:          agentplan.PlanStatusRejected,
		DecidedByUserID: h.tenant.UserID,
		DecidedAt:       9,
	})
	require.NoError(t, err)
	unrelated := h.plan(t, h.run(t, ""))

	expired, err := h.repo.ExpirePendingByDefinition(
		h.ctx,
		agentplanports.ExpireAgentPlansByDefinitionRequest{
			AgentDefinitionID: definition.ID,
			TenantInfo:        h.tenant,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)
	assert.Equal(t, agentplan.PlanStatusExpired, h.read(t, pending.ID).Status)
	assert.Equal(t, agentplan.PlanStatusRejected, h.read(t, decided.ID).Status)
	assert.Equal(t, agentplan.PlanStatusPending, h.read(t, unrelated.ID).Status)
}
