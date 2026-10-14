package masterdatatools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemoryservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMemories struct {
	agentports.AgentMemoryService
	remembered *agentports.RememberRequest
	status     *agentports.SetAgentMemoryStatusRequest
	created    *agent.Memory
	stored     map[pulid.ID]*agent.Memory
	existing   *agent.Memory
	guard      agenttoolstest.WriteGuard
	mode       agent.MemorySavingMode
	roleIDs    []pulid.ID
	offers     bool
	refuse     error
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
	req *agentports.RememberRequest,
	actor *permissionports.RequestActor,
) (*agent.Memory, error) {
	if err := f.guard.Write(); err != nil {
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
	req *agentports.RememberRequest,
	actor *permissionports.RequestActor,
) (*agentports.RememberPlan, error) {
	planned := agentmemoryservice.NewMemory(req, actor)
	if f.offers {
		planned.Status = agent.MemoryStatusSuggested
	}

	return &agentports.RememberPlan{Memory: planned, Existing: f.existing}, nil
}

func (f *fakeMemories) PreviewSetStatus(
	ctx context.Context,
	req *agentports.SetAgentMemoryStatusRequest,
	_ *permissionports.RequestActor,
) (*agent.Memory, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}

	return f.GetByID(ctx, agentports.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo})
}

func (f *fakeMemories) GetByID(
	_ context.Context,
	req agentports.GetAgentMemoryByIDRequest,
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
	req agentports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
) (*agent.Memory, error) {
	if err := f.guard.Write(); err != nil {
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

func TestRemember_PassesTheRunAndSubjectThrough(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := NewRememberTool(memories)
	customerID := pulid.MustNew("cus_")
	params := agenttoolstest.MemoryParams(map[string]any{
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
	assert.Equal(t, permission.ResourceAssistant, tool.Policy().Resource,
		"remembering is part of using the assistant; who keeps a shared memory is the service's")
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().DefaultTier)
}

func TestRemember_RefusesACorrectionAndHalfASubject(t *testing.T) {
	t.Parallel()

	tool := NewRememberTool(&fakeMemories{})

	err := tool.Execute(
		t.Context(),
		agenttoolstest.MemoryParams(map[string]any{"content": "x", "kind": "Correction"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind must be one of Instruction, Fact, Procedure")

	err = tool.Execute(
		t.Context(),
		agenttoolstest.MemoryParams(map[string]any{"content": "x", "subjectId": "cus_01M3Q2Y3HYKRA6P275AT81N7TF"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subjectId needs subjectType")

	err = tool.Execute(
		t.Context(),
		agenttoolstest.MemoryParams(map[string]any{"content": "x", "expiresOn": "tomorrow"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

// A model that fills every field sends a subject type for a memory about no
// record. Refusing it kept "remember to show me the queue item" from ever
// being saved, so a type with no id is read as no subject.
func TestRemember_ReadsATypeWithNoIDAsNoSubject(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := NewRememberTool(memories)

	err := tool.Execute(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"content":     "Show the billing queue item, not the invoice, unless the invoice is asked for.",
		"kind":        "Instruction",
		"subjectType": "Customer",
		"subjectId":   "",
	}))

	require.NoError(t, err)
	require.NotNil(t, memories.remembered)
	assert.Empty(t, memories.remembered.SubjectType)
	assert.True(t, memories.remembered.SubjectID.IsNil())
}

func TestForgetMemory_RetiresRatherThanDeletes(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	tool := NewForgetMemoryTool(memories)
	memoryID := pulid.MustNew("amem_")

	require.NoError(
		t,
		tool.Execute(t.Context(), agenttoolstest.MemoryParams(map[string]any{"memoryId": memoryID.String()})),
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
	tool := NewRememberTool(memories)

	params := agenttoolstest.MemoryParams(map[string]any{"content": "Group AR by facility."})
	memory, err := tool.(agentports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeUser, memory.Scope)
	require.NotNil(t, memory.OwnerUserID)
	assert.Equal(t, params.Actor.UserID, *memory.OwnerUserID)
	assert.Equal(t, agent.MemoryStatusActive, memory.Status)
	assert.Equal(t, agent.EgressPersonal, tool.Policy().Classified(params).Egress)

	team := agenttoolstest.MemoryParams(map[string]any{"content": "Columbus loads need a lumper receipt.", "visibleTo": "team"})
	memory, err = tool.(agentports.MemoryRecordingTool).Record(t.Context(), team)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeRole, memory.Scope)
	require.NotNil(t, memory.RoleID)
	assert.Equal(t, roleID, *memory.RoleID)
	assert.Equal(t, agent.EgressInternal, tool.Policy().Classified(team).Egress)

	org := agenttoolstest.MemoryParams(map[string]any{"content": "Use the DOE weekly average.", "visibleTo": "organization"})
	memory, err = tool.(agentports.MemoryRecordingTool).Record(t.Context(), org)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, memory.Scope)
	assert.Nil(t, memory.OwnerUserID)
}

// Nobody in the conversation means nobody to keep it for: it is the
// organization's, as it always was.
func TestRemember_WithNobodyInTheConversationKeepsItForTheOrganization(t *testing.T) {
	t.Parallel()

	tool := NewRememberTool(&fakeMemories{})
	params := agenttoolstest.MemoryParams(map[string]any{"content": "Docks close at four."})
	params.Actor = &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agt_"),
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
	}

	memory, err := tool.(agentports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, memory.Scope)

	params.Params["visibleTo"] = "me"
	_, err = tool.(agentports.MemoryRecordingTool).Record(t.Context(), params)
	require.Error(t, err)
}

// A person who asked to be asked first is offered the memory: it is kept as a
// suggestion nothing reads until they accept it. One they already approved as
// a proposal was asked about, and is kept.
func TestRemember_AskFirstOffersInsteadOfKeeping(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{mode: agent.MemorySavingAskFirst}
	tool := NewRememberTool(memories)

	params := agenttoolstest.MemoryParams(map[string]any{"content": "Avery approves invoices after 4 PM."})
	memory, err := tool.(agentports.MemoryRecordingTool).Record(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusSuggested, memory.Status)
	require.NotNil(t, memory.CreatedByUserID)
	assert.Equal(t, params.Actor.UserID, *memory.CreatedByUserID, "offered to the person who asked")

	approved := agenttoolstest.MemoryParams(map[string]any{"content": "Avery approves invoices after 4 PM."})
	approved.ProposalID = pulid.MustNew("aprop_")
	memory, err = tool.(agentports.MemoryRecordingTool).Record(t.Context(), approved)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, memory.Status)
}

// The tools a person uses through the assistant are gated on using the
// assistant; what they may keep, forget or steer for others is the memory and
// profile services' to decide.
func TestMemoryTools_AreGatedOnUsingTheAssistant(t *testing.T) {
	t.Parallel()

	tools := []agentports.AgentTool{
		NewForgetMemoryTool(&fakeMemories{}),
		NewGuideProfileTool(&fakeProfileGuide{}),
	}
	for _, tool := range tools {
		assert.Equal(t, permission.ResourceAssistant, tool.Policy().Resource, tool.Name())
		assert.Equal(t, permission.OpCreate, tool.Policy().Operation, tool.Name())
	}
}
