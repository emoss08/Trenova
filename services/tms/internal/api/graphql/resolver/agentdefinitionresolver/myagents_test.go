package agentdefinitionresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
