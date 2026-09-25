package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMemories struct {
	serviceports.AgentMemoryService
	remembered *serviceports.RememberRequest
	status     *serviceports.SetAgentMemoryStatusRequest
	created    *agent.Memory
	stored     map[pulid.ID]*agent.Memory
	existing   *agent.Memory
	guard      writeGuard
}

func (f *fakeMemories) Remember(
	_ context.Context,
	req *serviceports.RememberRequest,
	actor *serviceports.RequestActor,
) (*agent.Memory, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.remembered = req
	if f.existing != nil {
		return f.existing, nil
	}
	f.created = agentmemoryservice.NewMemory(req, actor)
	f.created.ID = pulid.MustNew("amem_")

	return f.created, nil
}

func (f *fakeMemories) PreviewRemember(
	_ context.Context,
	req *serviceports.RememberRequest,
	actor *serviceports.RequestActor,
) (*serviceports.RememberPlan, error) {
	return &serviceports.RememberPlan{
		Memory:   agentmemoryservice.NewMemory(req, actor),
		Existing: f.existing,
	}, nil
}

func (f *fakeMemories) GetByID(
	_ context.Context,
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	memory, ok := f.stored[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent memory not found")
	}
	copied := *memory

	return &copied, nil
}

func (f *fakeMemories) SetStatus(
	_ context.Context,
	req serviceports.SetAgentMemoryStatusRequest,
	actor *serviceports.RequestActor,
) (*agent.Memory, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.status = &req
	memory, ok := f.stored[req.ID]
	if !ok {
		return &agent.Memory{ID: req.ID, Status: req.Status}, nil
	}
	if err := agentmemoryservice.PlanStatus(memory, agentmemoryservice.StatusChange{
		Status:   req.Status,
		ByUserID: agentmemoryservice.StatusActor(actor),
		At:       timeutils.NowUnix(),
	}); err != nil {
		return nil, err
	}
	memory.Version++

	return memory, nil
}

func memoryParams(params map[string]any) serviceports.ToolExecuteParams {
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")

	return serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		RunID:  pulid.MustNew("arun_"),
		Params: params,
	}
}

func TestRemember_PassesTheRunAndSubjectThrough(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := newRememberTool(memories)
	customerID := pulid.MustNew("cus_")
	params := memoryParams(map[string]any{
		"content":     "Needs the POD within one day.",
		"kind":        "Instruction",
		"subjectType": "Customer",
		"subjectId":   customerID.String(),
		"expiresOn":   "2026-12-31",
	})

	require.NoError(t, tool.Execute(t.Context(), params))

	assert.Equal(t, params.RunID, memories.remembered.RunID)
	assert.Equal(t, agent.MemoryKindInstruction, memories.remembered.Kind)
	assert.Equal(t, agent.MemorySubjectCustomer, memories.remembered.SubjectType)
	assert.Equal(t, customerID, memories.remembered.SubjectID)
	require.NotNil(t, memories.remembered.ExpiresAt)
	assert.Equal(t, int64(1798761600), *memories.remembered.ExpiresAt, "the end of the day named")
	assert.Equal(t, permission.ResourceAgentMemory, tool.Policy().Resource)
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().DefaultTier)
}

func TestRemember_RefusesACorrectionAndHalfASubject(t *testing.T) {
	t.Parallel()

	tool := newRememberTool(&fakeMemories{})

	err := tool.Execute(
		t.Context(),
		memoryParams(map[string]any{"content": "x", "kind": "Correction"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Instruction or Fact")

	err = tool.Execute(
		t.Context(),
		memoryParams(map[string]any{"content": "x", "subjectType": "Customer"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both or neither")

	err = tool.Execute(
		t.Context(),
		memoryParams(map[string]any{"content": "x", "expiresOn": "tomorrow"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestForgetMemory_RetiresRatherThanDeletes(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := newForgetMemoryTool(memories)
	memoryID := pulid.MustNew("amem_")

	require.NoError(
		t,
		tool.Execute(t.Context(), memoryParams(map[string]any{"memoryId": memoryID.String()})),
	)

	assert.Equal(t, memoryID, memories.status.ID)
	assert.Equal(t, agent.MemoryStatusRetired, memories.status.Status)
	assert.True(t, tool.Policy().Reversible)
}
