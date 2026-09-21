package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMemories struct {
	serviceports.AgentMemoryService
	remembered *serviceports.RememberRequest
	status     *serviceports.SetAgentMemoryStatusRequest
}

func (f *fakeMemories) Remember(
	_ context.Context,
	req *serviceports.RememberRequest,
	_ *serviceports.RequestActor,
) (*agent.Memory, error) {
	f.remembered = req

	return &agent.Memory{ID: pulid.MustNew("amem_")}, nil
}

func (f *fakeMemories) SetStatus(
	_ context.Context,
	req serviceports.SetAgentMemoryStatusRequest,
	_ *serviceports.RequestActor,
) (*agent.Memory, error) {
	f.status = &req

	return &agent.Memory{ID: req.ID, Status: req.Status}, nil
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
	assert.Equal(t, permission.ResourceAgentMemory, tool.PermissionResource())
	assert.Equal(t, agent.TierActWithApproval, tool.DefaultAutonomyTier())
}

func TestRemember_RefusesACorrectionAndHalfASubject(t *testing.T) {
	t.Parallel()

	tool := newRememberTool(&fakeMemories{})

	err := tool.Execute(t.Context(), memoryParams(map[string]any{"content": "x", "kind": "Correction"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Instruction or Fact")

	err = tool.Execute(t.Context(), memoryParams(map[string]any{"content": "x", "subjectType": "Customer"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both or neither")

	err = tool.Execute(t.Context(), memoryParams(map[string]any{"content": "x", "expiresOn": "tomorrow"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestForgetMemory_RetiresRatherThanDeletes(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := newForgetMemoryTool(memories)
	memoryID := pulid.MustNew("amem_")

	require.NoError(t, tool.Execute(t.Context(), memoryParams(map[string]any{"memoryId": memoryID.String()})))

	assert.Equal(t, memoryID, memories.status.ID)
	assert.Equal(t, agent.MemoryStatusRetired, memories.status.Status)
	assert.True(t, tool.Reversible())
}
