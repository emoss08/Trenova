package querycost

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/complexity"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type indexedSchema struct {
	schema *ast.Schema
	index  *Index
}

var _ graphql.ExecutableSchema = indexedSchema{}

func (s indexedSchema) Schema() *ast.Schema { return s.schema }

func (s indexedSchema) Complexity(
	_ context.Context,
	typeName, fieldName string,
	childComplexity int,
	args map[string]any,
) (int, bool) {
	return s.index.Complexity(typeName, fieldName, childComplexity, args)
}

func (indexedSchema) Exec(context.Context) graphql.ResponseHandler { return nil }

func calculate(t *testing.T, query string, vars map[string]any) int {
	t.Helper()

	idx, schema := testIndex(t)
	doc, errs := gqlparser.LoadQuery(schema, query)
	require.Empty(t, errs)

	return complexity.Calculate(
		context.Background(),
		indexedSchema{schema: schema, index: idx},
		doc.Operations[0],
		vars,
	)
}

func enforce(t *testing.T, query string, vars map[string]any, limit int) *gqlerror.Error {
	t.Helper()

	idx, schema := testIndex(t)
	doc, errs := gqlparser.LoadQuery(schema, query)
	require.Empty(t, errs)

	ext := extension.FixedComplexityLimit(limit)
	require.NoError(t, ext.Validate(indexedSchema{schema: schema, index: idx}))

	return ext.MutateOperationContext(context.Background(), &graphql.OperationContext{
		Doc:       doc,
		Operation: doc.Operations[0],
		Variables: vars,
	})
}

func TestGqlgenWalkerAppliesConnectionMultiplier(t *testing.T) {
	t.Parallel()

	query := `query Q($input: ConnectionInput!) {
		shipments(input: $input) { edges { node { id } } }
	}`

	small := calculate(t, query, map[string]any{"input": map[string]any{"first": 1}})
	large := calculate(t, query, map[string]any{"input": map[string]any{"first": 100}})

	assert.Equal(t, 100*small, large,
		"cost must scale linearly with the requested page size")
	assert.Positive(t, small)
}

func TestGqlgenWalkerDoesNotDoubleCountConnectionEdges(t *testing.T) {
	t.Parallel()

	query := `query Q { shipments(input: {first: 10}) { edges { node { id } } } }`

	// edges -> node -> id is 3 nested fields, each contributing 1, so the whole
	// connection selection is 10 * 3. If edges were also treated as an
	// unpaginated list this would be multiplied by UnpaginatedListWeight again.
	assert.Equal(t, 30, calculate(t, query, nil))
}

func TestGqlgenWalkerChargesNestedLists(t *testing.T) {
	t.Parallel()

	flat := calculate(t, `query Q { shipment(id: "1") { origin { id } } }`, nil)
	nested := calculate(t, `query Q { shipment(id: "1") { stops { id } } }`, nil)

	assert.Equal(t, 3, flat, "shipment(1) + origin(1) + id(1)")
	assert.Equal(t, 1+UnpaginatedListWeight, nested,
		"an unpaginated nested list must cost more than a single object")
	assert.Greater(t, nested, flat)
}

func TestComplexityLimitErrorCodeIsStable(t *testing.T) {
	t.Parallel()

	query := `query Q { shipments(input: {first: 100}) { edges { node { id } } } }`
	cost := calculate(t, query, nil)

	err := enforce(t, query, nil, cost-1)
	require.NotNil(t, err, "a query above the limit must be rejected")
	assert.Equal(t, ComplexityLimitErrorCode, err.Extensions["code"],
		"gqlgen changed its complexity error code; update querycost.ComplexityLimitErrorCode")
}

func TestComplexityLimitAcceptsQueriesAtTheLimit(t *testing.T) {
	t.Parallel()

	query := `query Q { shipments(input: {first: 100}) { edges { node { id } } } }`
	cost := calculate(t, query, nil)

	assert.Nil(t, enforce(t, query, nil, cost),
		"a query exactly at the limit must be allowed")
}

func TestUnboundedNestingIsRejectedByTheBudget(t *testing.T) {
	t.Parallel()

	query := `query Q {
		shipments(input: {first: 100}) {
			edges { node { stops { id name } tags } }
		}
	}`

	assert.Greater(t, calculate(t, query, nil), 100,
		"nested lists inside a full page must compound into a meaningful cost")
}
