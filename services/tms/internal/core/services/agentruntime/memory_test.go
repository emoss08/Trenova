package agentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMemories struct {
	serviceports.AgentMemoryService

	asked  *serviceports.MemoryContextRequest
	answer *serviceports.MemoryContext
	used   []pulid.ID
}

func (f *fakeMemories) ForContext(
	_ context.Context,
	req serviceports.MemoryContextRequest,
) (*serviceports.MemoryContext, error) {
	f.asked = &req

	return f.answer, nil
}

func (f *fakeMemories) RecordUse(
	_ context.Context,
	req serviceports.RecordMemoryUseRequest,
) error {
	f.used = append(f.used, req.IDs...)

	return nil
}

func TestContextBuilder_ReadsTheMemoriesOfWhatTheTurnIsAbout(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	memories := &fakeMemories{answer: &serviceports.MemoryContext{
		Memories: []*agent.Memory{{ID: pulid.MustNew("amem_"), Content: "Acme pays late."}},
		Subjects: []agent.MemorySubject{{
			Type:     agent.MemorySubjectCustomer,
			ID:       customer,
			Relation: agent.MemoryRelationDirect,
		}},
		Relevance: agent.MemoryRelevance{Judged: true},
	}}
	builder := &ContextBuilder{
		logger:        zap.NewNop(),
		organizations: &stubOrganizations{org: &tenant.Organization{Timezone: "UTC"}},
		users:         &stubUsers{user: &tenant.User{}},
		runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
		memories: memories,
	}

	shipment := pulid.MustNew("shp_").String()
	location := pulid.MustNew("loc_").String()
	parentRecord := pulid.MustNew("wrk_").String()
	definition := testDefinition("assign_move")
	rc, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      testActor(),
		Trigger:    agent.RunTriggerChat,
		Subject:    &agentdefinition.RuntimeSubject{Type: agent.SubjectShipment, ID: shipment},
		Page: &agentdefinition.PageContext{
			EntityType: "customer",
			EntityID:   customer.String(),
		},
		Mentions:         []agentdefinition.RuntimeMention{{Type: "location", ID: location}},
		DelegatorRecords: []agent.EntityRef{{Type: "worker", ID: parentRecord}},
	})
	require.NoError(t, err)

	require.NotNil(t, memories.asked)
	assert.Equal(t, []agent.EntityRef{
		{Type: string(agent.SubjectShipment), ID: shipment},
		{Type: "customer", ID: customer.String()},
		{Type: "location", ID: location},
		{Type: "worker", ID: parentRecord},
	}, memories.asked.Records)
	assert.Equal(t, definition.EffectiveToolNames(), memories.asked.ToolNames)
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
	definition := testDefinition()
	definition.MemoryTokenBudget = &budget

	pool := make([]*agent.Memory, 0, 5)
	for range 5 {
		pool = append(pool, &agent.Memory{
			ID:      pulid.MustNew("amem_"),
			Kind:    agent.MemoryKindFact,
			Content: strings.Repeat("z", 1100),
		})
	}
	tainted := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Content: strings.Repeat("t", 1100),
		Tainted: true,
	}
	pool = append(pool, tainted)

	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "What do we know?",
		Context:    agentdefinition.RuntimeContext{Memories: pool},
	})

	assert.Equal(t, []pulid.ID{pool[0].ID, pool[1].ID, pool[2].ID}, memories.used,
		"three memories of about 280 tokens fill a 1000-token budget")
	assert.Equal(t, 3, strings.Count(turn.State().System, "- [Fact] zzz"))
	assert.False(t, turn.Taint().Tainted(),
		"a tainted memory the prompt had no room for is never read, so it taints nothing")
}

func TestOpenTurn_ReadsNoMemoryForAnAgentThatDoesNotAskForIt(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories

	definition := testDefinition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextClock}
	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Hello",
		Context: agentdefinition.RuntimeContext{Memories: []*agent.Memory{{
			ID:      pulid.MustNew("amem_"),
			Content: "From an email.",
			Tainted: true,
		}}},
	})

	assert.Empty(t, memories.used)
	assert.False(t, turn.Taint().Tainted())
	assert.NotContains(t, turn.State().System, "From an email.")
}

// An agent asked to remember what it already knew got a "saved" card with
// an Undo, and undoing it retired the memory the person had kept for months.
// A refreshed memory is now named as used, never offered for undo, and the
// model is told nothing new was recorded.
func TestSavedOrRefreshed_ARefreshedMemoryIsUsedNotSaved(t *testing.T) {
	t.Parallel()

	kept := &agent.Memory{ID: pulid.MustNew("amem_"), Status: agent.MemoryStatusActive}
	saved, refreshed := savedOrRefreshed(kept, "call_1")
	require.NotNil(t, saved)
	assert.Nil(t, refreshed)
	assert.Equal(t, kept.ID, saved.ID)
	assert.Equal(t, "call_1", saved.CallID)

	again := &agent.Memory{ID: pulid.MustNew("amem_"), Refreshed: true}
	saved, refreshed = savedOrRefreshed(again, "call_2")
	assert.Nil(t, saved)
	assert.Same(t, again, refreshed)

	content := refreshedContent("remember", again)
	assert.Contains(t, content, "already remembered as memory "+again.ID.String())
	assert.Contains(t, content, "instead of saving a duplicate")
}

type memoryEvents struct {
	TurnEffects

	events []serviceports.StreamEvent
}

func (fx *memoryEvents) Emit(event serviceports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func TestOpenTurn_CountsOnlyTheMemoriesThatBearOnTheTurn(t *testing.T) {
	t.Parallel()

	memories := &fakeMemories{}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.memories = memories

	rule := &agent.Memory{
		ID: pulid.MustNew("amem_"), Kind: agent.MemoryKindInstruction,
		Content: "Assess ELD applicability for each driver.",
	}
	bearing := &agent.Memory{
		ID: pulid.MustNew("amem_"), Kind: agent.MemoryKindFact,
		Content: "Copy dispatch when a customer is told a load delivered.",
	}
	unrelated := &agent.Memory{
		ID: pulid.MustNew("amem_"), Kind: agent.MemoryKindFact,
		Content: "The yard closes at six.",
	}

	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Let the customer know the load delivered.",
		Context: agentdefinition.RuntimeContext{
			Memories: []*agent.Memory{rule, bearing, unrelated},
			MemoryRelevance: agent.MemoryRelevance{
				Judged: true,
				IDs:    map[pulid.ID]struct{}{bearing.ID: {}},
			},
		},
	})

	assert.Equal(t, []pulid.ID{bearing.ID}, memories.used)
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
	turn := rt.RestoreTurn(&serviceports.RunRequest{Definition: testDefinition()}, TurnState{
		Result:       serviceports.RunResult{UsedMemoryIDs: []pulid.ID{already}},
		ToolMemories: map[string][]pulid.ID{"assign_move": {fix}},
	})
	fx := &memoryEvents{}

	turn.noteToolMemories(fx, "search_shipments")
	assert.Empty(t, fx.events, "a call to another tool uses nothing")

	turn.noteToolMemories(fx, "assign_move")
	require.Len(t, fx.events, 1)
	assert.Equal(t, serviceports.AssistantEventMemoryUsed, fx.events[0].Event)
	assert.Equal(t, []pulid.ID{already, fix}, turn.State().Result.UsedMemoryIDs)
	assert.Empty(t, turn.State().ToolMemories)

	turn.noteToolMemories(fx, "assign_move")
	assert.Len(t, fx.events, 1, "a second call announces nothing new")
}
