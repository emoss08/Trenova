package querycost

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/complexity"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
)

func TestIntrospectionQueryIsNotBlockedByTheLimits(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	index := NewIndex(schema)

	doc, errs := gqlparser.LoadQuery(schema, introspection.Query)
	require.Empty(t, errs)
	require.NotEmpty(t, doc.Operations)

	for _, op := range doc.Operations {
		depth := Depth(op)
		cost := complexity.Calculate(
			context.Background(),
			indexedSchema{schema: schema, index: index},
			op,
			nil,
		)

		t.Logf("introspection %q depth=%d cost=%d", op.Name, depth, cost)

		assert.LessOrEqual(t, depth, MaxOperationDepth,
			"the playground and admin explorer introspect with this query; "+
				"blocking it breaks schema tooling")
		assert.LessOrEqual(t, cost, MaxOperationCost,
			"introspection must stay under the cost budget")
	}
}

func TestDepthIgnoresIntrospectionMetaFields(t *testing.T) {
	t.Parallel()

	_, schema := testIndex(t)

	assert.Equal(t, 0, Depth(parseOp(t, schema, `{ __typename }`)),
		"introspection meta-fields must not contribute depth")
	assert.Equal(t, 1, Depth(parseOp(t, schema, `{ shipment(id: "1") { __typename } }`)),
		"__typename must not add a level, but its parent still counts")
}
