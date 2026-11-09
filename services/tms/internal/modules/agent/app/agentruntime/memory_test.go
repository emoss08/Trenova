package agentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/accesstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimecontext"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemories struct {
	agentmemoryports.AgentMemoryService

	asked  *agentmemoryports.MemoryContextRequest
	answer *agentmemoryports.MemoryContext
	used   []pulid.ID
}

func (f *fakeMemories) ForContext(
	_ context.Context,
	req *agentmemoryports.MemoryContextRequest,
) (*agentmemoryports.MemoryContext, error) {
	f.asked = req

	return f.answer, nil
}

func (f *fakeMemories) RecordUse(
	_ context.Context,
	req agentmemoryports.RecordMemoryUseRequest,
) error {
	f.used = append(f.used, req.IDs...)

	return nil
}

func TestContextBuilder_ReadsTheMemoriesOfWhatTheTurnIsAbout(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	memories := &fakeMemories{answer: &agentmemoryports.MemoryContext{
		Memories: []*agentmemory.Memory{{ID: pulid.MustNew("amem_"), Content: "Acme pays late."}},
		Subjects: []agentmemory.MemorySubject{{
			Type:     agentmemory.MemorySubjectCustomer,
			ID:       customer,
			Relation: agentmemory.MemoryRelationDirect,
		}},
		Relevance: agentmemory.MemoryRelevance{Judged: true},
	}}
	builder := accesstest.ContextBuilder(runtimecontext.BuilderParams{
		Logger:        zap.NewNop(),
		Organizations: &runtimetest.StubOrganizations{Org: &tenant.Organization{Timezone: "UTC"}},
		Users:         &runtimetest.StubUsers{User: &tenant.User{}},
		Runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{Tools: []agenttoolports.AgentTool{
				actionTool("dispatch_assign_move", agent.TierPropose, nil),
			}},
			nil,
		),
		Memories: memories,
	})

	shipment := pulid.MustNew("shp_").String()
	location := pulid.MustNew("loc_").String()
	parentRecord := pulid.MustNew("wrk_").String()
	definition := runtimetest.Definition("dispatch_assign_move")
	rc, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Trigger:    agent.RunTriggerChat,
		Subject:    &agentturn.RuntimeSubject{Type: agent.SubjectShipment, ID: shipment},
		Page: &agentturn.PageContext{
			EntityType: "customer",
			EntityID:   customer.String(),
		},
		Mentions:         []agentturn.RuntimeMention{{Type: "location", ID: location}},
		DelegatorRecords: []agentpagecontext.EntityRef{{Type: "worker", ID: parentRecord}},
	})
	require.NoError(t, err)

	require.NotNil(t, memories.asked)
	assert.Equal(t, []agentpagecontext.EntityRef{
		{Type: string(agent.SubjectShipment), ID: shipment},
		{Type: "customer", ID: customer.String()},
		{Type: "location", ID: location},
		{Type: "worker", ID: parentRecord},
	}, memories.asked.Records)
	assert.Equal(t, []string{"dispatch_assign_move"}, memories.asked.ToolNames,
		"memories are read for the tools the agent holds")
	assert.Equal(t, memories.answer.Memories, rc.Memories)
	assert.Equal(t, memories.answer.Subjects, rc.MemorySubjects)
	assert.Equal(t, memories.answer.Relevance, rc.MemoryRelevance)
	assert.Empty(t, memories.used, "building the context counts nothing as used")
}

func TestOpenTurn_CarriesWhatFitsTheBudgetAndCountsOnlyThat(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories

	budget := agentdefinition.MinMemoryTokenBudget
	definition := runtimetest.Definition()
	definition.MemoryTokenBudget = &budget

	pool := make([]*agentmemory.Memory, 0, 5)
	for range 5 {
		pool = append(pool, &agentmemory.Memory{
			ID:      pulid.MustNew("amem_"),
			Kind:    agentmemory.MemoryKindFact,
			Content: strings.Repeat("z", 1100),
		})
	}
	tainted := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindFact,
		Content: strings.Repeat("t", 1100),
		Tainted: true,
	}
	pool = append(pool, tainted)

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "What do we know?",
		Context:    agentturn.RuntimeContext{Memories: pool},
	})

	assert.Equal(t, []pulid.ID{pool[0].ID, pool[1].ID, pool[2].ID},
		turn.State().Result.UsedMemoryIDs,
		"three memories of about 290 tokens fill a 1000-token budget")
	assert.Empty(t, memories.used, "use is counted when the turn is saved, not when it opens")
	assert.Equal(t, 3, strings.Count(turn.State().System, "] zzz"))
	assert.False(t, turn.Taint().Tainted(),
		"a tainted memory the prompt had no room for is never read, so it taints nothing")
}

func TestOpenTurn_ReadsNoMemoryForAnAgentThatDoesNotAskForIt(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories

	definition := runtimetest.Definition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextClock}
	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Hello",
		Context: agentturn.RuntimeContext{Memories: []*agentmemory.Memory{{
			ID:      pulid.MustNew("amem_"),
			Content: "From an email.",
			Tainted: true,
		}}},
	})

	assert.Empty(t, memories.used)
	assert.False(t, turn.Taint().Tainted())
	assert.NotContains(t, turn.State().System, "From an email.")
}

type memoryEvents struct {
	TurnEffects

	events []kernelports.StreamEvent
}

func (fx *memoryEvents) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func TestOpenTurn_CountsOnlyTheMemoriesThatBearOnTheTurn(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories

	rule := &agentmemory.Memory{
		ID: pulid.MustNew("amem_"), Kind: agentmemory.MemoryKindInstruction,
		Content: "Assess ELD applicability for each driver.",
	}
	bearing := &agentmemory.Memory{
		ID: pulid.MustNew("amem_"), Kind: agentmemory.MemoryKindFact,
		Content: "Copy dispatch when a customer is told a load delivered.",
	}
	unrelated := &agentmemory.Memory{
		ID: pulid.MustNew("amem_"), Kind: agentmemory.MemoryKindFact,
		Content: "The yard closes at six.",
	}

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Let the customer know the load delivered.",
		Context: agentturn.RuntimeContext{
			Memories: []*agentmemory.Memory{rule, bearing, unrelated},
			MemoryRelevance: agentmemory.MemoryRelevance{
				Judged: true,
				IDs:    map[pulid.ID]struct{}{bearing.ID: {}},
			},
		},
	})

	assert.Empty(t, memories.used)
	assert.Equal(t, []pulid.ID{bearing.ID}, turn.State().Result.UsedMemoryIDs)
	system := turn.State().System
	assert.Contains(t, system, rule.Content, "a standing rule is followed whether or not it bears")
	assert.Contains(t, system, bearing.Content)
	assert.NotContains(t, system, unrelated.Content, "a fact that does not bear is left to recall")
}

func TestNoteToolMemories_AToolsMemoriesAreUsedOnceItIsCalled(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fix := pulid.MustNew("amem_")
	already := pulid.MustNew("amem_")
	turn := rt.RestoreTurn(
		&agentruntimeports.RunRequest{Definition: runtimetest.Definition()},
		&TurnState{
			Result:       agentruntimeports.RunResult{UsedMemoryIDs: []pulid.ID{already}},
			ToolMemories: map[string][]pulid.ID{"dispatch_assign_move": {fix}},
		},
	)
	fx := &memoryEvents{}

	turnmemory.NoteTool(turn, fx, "shipment_list")
	assert.Empty(t, fx.events, "a call to another tool uses nothing")

	turnmemory.NoteTool(turn, fx, "dispatch_assign_move")
	require.Len(t, fx.events, 1)
	assert.Equal(t, assistantstreamports.AssistantEventMemoryUsed, fx.events[0].Event)
	assert.Equal(t, []pulid.ID{already, fix}, turn.State().Result.UsedMemoryIDs)
	assert.Empty(t, turn.State().ToolMemories)

	turnmemory.NoteTool(turn, fx, "dispatch_assign_move")
	assert.Len(t, fx.events, 1, "a second call announces nothing new")
}

func TestNoteToolMemories_ADelegateKeepsWhatItUsedWithoutAnnouncingIt(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fix := pulid.MustNew("amem_")
	turn := rt.RestoreTurn(
		&agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(),
			Delegation: &agentruntimeports.Delegation{},
		},
		&TurnState{ToolMemories: map[string][]pulid.ID{"dispatch_assign_move": {fix}}},
	)
	fx := &memoryEvents{}

	turnmemory.NoteTool(turn, fx, "dispatch_assign_move")

	assert.Empty(t, fx.events, "the person's card shows what the conversation's own agent used")
	assert.Equal(t, []pulid.ID{fix}, turn.State().Result.UsedMemoryIDs)
}

// Use is counted once per turn, when it is saved, from what the turn ended up
// using: the memories that bore on it, those about tools it called, those it
// read whole by id, and those its delegates used.
func TestRecordMemoryUse_CountsTheTurnAndItsDelegatesOnce(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories
	own, shared, delegated := pulid.MustNew("amem_"), pulid.MustNew("amem_"), pulid.MustNew("amem_")

	rt.RecordMemoryUse(t.Context(), runtimetest.Actor().TenantInfo(), &agentruntimeports.RunResult{
		UsedMemoryIDs: []pulid.ID{own, shared},
		Delegations: []agentruntimeports.DelegatedRun{
			{UsedMemoryIDs: []pulid.ID{shared, delegated}},
		},
	})
	rt.RecordMemoryUse(t.Context(), runtimetest.Actor().TenantInfo(), &agentruntimeports.RunResult{})
	rt.RecordMemoryUse(t.Context(), runtimetest.Actor().TenantInfo(), nil)

	assert.Equal(t, []pulid.ID{own, shared, delegated}, memories.used)
}
