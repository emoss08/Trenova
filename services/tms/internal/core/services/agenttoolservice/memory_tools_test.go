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
	"github.com/emoss08/trenova/pkg/pagination"
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
	mode       agent.MemorySavingMode
	roleIDs    []pulid.ID
}

func (f *fakeMemories) SavingMode(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (agent.MemorySavingMode, error) {
	if f.mode == "" {
		return agent.MemorySavingAutomatic, nil
	}

	return f.mode, nil
}

func (f *fakeMemories) Reader(
	_ context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) (agent.MemoryReader, error) {
	return agent.MemoryReader{UserID: userID, RoleIDs: f.roleIDs}, nil
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

// With a person in the conversation a memory is theirs alone unless the agent
// says otherwise, and that kind runs without asking them: it is their own
// record. Wider ones go through the tiers like any write.
func TestRemember_KeepsItForThePersonUnlessToldWhoElse(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	memories := &fakeMemories{roleIDs: []pulid.ID{roleID}}
	tool := newRememberTool(memories)

	params := memoryParams(map[string]any{"content": "Group AR by facility."})
	memory, err := tool.(serviceports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeUser, memory.Scope)
	require.NotNil(t, memory.OwnerUserID)
	assert.Equal(t, params.Actor.UserID, *memory.OwnerUserID)
	assert.Equal(t, agent.MemoryStatusActive, memory.Status)
	assert.Equal(t, agent.EgressPersonal, tool.Policy().Classified(params).Egress)

	team := memoryParams(map[string]any{"content": "Columbus loads need a lumper receipt.", "visibleTo": "team"})
	memory, err = tool.(serviceports.MemoryRecordingTool).Record(t.Context(), team)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeRole, memory.Scope)
	require.NotNil(t, memory.RoleID)
	assert.Equal(t, roleID, *memory.RoleID)
	assert.Equal(t, agent.EgressInternal, tool.Policy().Classified(team).Egress)

	org := memoryParams(map[string]any{"content": "Use the DOE weekly average.", "visibleTo": "organization"})
	memory, err = tool.(serviceports.MemoryRecordingTool).Record(t.Context(), org)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, memory.Scope)
	assert.Nil(t, memory.OwnerUserID)
}

// Nobody in the conversation means nobody to keep it for: it is the
// organization's, as it always was.
func TestRemember_WithNobodyInTheConversationKeepsItForTheOrganization(t *testing.T) {
	t.Parallel()

	tool := newRememberTool(&fakeMemories{})
	params := memoryParams(map[string]any{"content": "Docks close at four."})
	params.Actor = &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agt_"),
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
	}

	memory, err := tool.(serviceports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, memory.Scope)

	params.Params["visibleTo"] = "me"
	_, err = tool.(serviceports.MemoryRecordingTool).Record(t.Context(), params)
	require.Error(t, err)
}

// A person who asked to be asked first is offered the memory: it is kept as a
// suggestion nothing reads until they accept it. One they already approved as
// a proposal was asked about, and is kept.
func TestRemember_AskFirstOffersInsteadOfKeeping(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{mode: agent.MemorySavingAskFirst}
	tool := newRememberTool(memories)

	params := memoryParams(map[string]any{"content": "Avery approves invoices after 4 PM."})
	memory, err := tool.(serviceports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusSuggested, memory.Status)
	require.NotNil(t, memory.CreatedByUserID)
	assert.Equal(t, params.Actor.UserID, *memory.CreatedByUserID, "offered to the person who asked")

	approved := memoryParams(map[string]any{"content": "Avery approves invoices after 4 PM."})
	approved.ProposalID = pulid.MustNew("aprop_")
	memory, err = tool.(serviceports.MemoryRecordingTool).Record(t.Context(), approved)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, memory.Status)
}
