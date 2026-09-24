package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
myAgents is read by everyone who may use the assistant, not by the people who
administer agents. MyAgent carries only what a picker shows and what a person
asks with: an agent's instructions, guardrails, tool tiers, budget, provider
and delegates are an administrator's. A field added here is a field every
assistant user can read.
*/
func TestMyAgent_ExposesOnlyChatFacingFields(t *testing.T) {
	t.Parallel()

	schema := generated.NewExecutableSchema(generated.Config{}).Schema()
	myAgent := schema.Types["MyAgent"]
	require.NotNil(t, myAgent, "the schema declares MyAgent")

	fields := make([]string, 0, len(myAgent.Fields))
	for _, field := range myAgent.Fields {
		fields = append(fields, field.Name)
	}

	assert.ElementsMatch(t, []string{
		"id",
		"name",
		"description",
		"template",
		"icon",
		"accent",
		"toolNames",
		"systemKey",
		"starters",
	}, fields)
}

// myAgents takes a closed set of filters, never a free filter over agents: a
// filter on a field the reader is not shown would let them probe it.
func TestMyAgentsInput_TakesNoFreeFilter(t *testing.T) {
	t.Parallel()

	schema := generated.NewExecutableSchema(generated.Config{}).Schema()
	input := schema.Types["MyAgentsInput"]
	require.NotNil(t, input)

	fields := make([]string, 0, len(input.Fields))
	for _, field := range input.Fields {
		fields = append(fields, field.Name)
	}

	assert.ElementsMatch(t,
		[]string{"first", "after", "search", "origin", "excludeIds", "ids"},
		fields,
	)
}

func TestMyAgentFilters(t *testing.T) {
	t.Parallel()

	template := gqlmodel.MyAgentOriginTemplate
	id := pulid.MustNew("agdef_").String()

	filters, none, err := myAgentFilters(&gqlmodel.MyAgentsInput{
		Origin:     &template,
		ExcludeIds: []string{id},
	})
	require.NoError(t, err)
	assert.False(t, none)
	require.Len(t, filters, 2)
	assert.Equal(t, "template", filters[0].Field)
	assert.Equal(t, "isnotnull", filters[0].Operator)
	assert.Equal(t, "notin", filters[1].Operator)
	assert.Equal(t, []any{id}, filters[1].Value)

	_, none, err = myAgentFilters(&gqlmodel.MyAgentsInput{Ids: []string{}})
	require.NoError(t, err)
	assert.True(t, none, "asking for no ids asks for no agents, not every agent")

	tooMany := make([]string, maxMyAgentIDs+1)
	for i := range tooMany {
		tooMany[i] = id
	}
	_, _, err = myAgentFilters(&gqlmodel.MyAgentsInput{Ids: tooMany})
	assert.Error(t, err)
}
