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
