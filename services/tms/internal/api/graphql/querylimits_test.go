package graphql

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const limitsTestSchema = `
type Level3 { id: ID! }
type Level2 { id: ID! child: Level3 }
type Level1 { id: ID! child: Level2 }
type Query { root: Level1 }
`

func limitsOperationContext(t *testing.T, query string) *graphql.OperationContext {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "limits", Input: limitsTestSchema})
	require.NoError(t, err)

	doc, errs := gqlparser.LoadQuery(schema, query)
	require.Empty(t, errs)

	return &graphql.OperationContext{Doc: doc, Operation: doc.Operations[0]}
}

func TestOperationDepthLimit_Contract(t *testing.T) {
	t.Parallel()

	limit := operationDepthLimit{max: 3}

	assert.Equal(t, depthLimitExtensionName, limit.ExtensionName())
	assert.NoError(t, limit.Validate(nil))
}

func TestOperationDepthLimit_AllowsWithinLimit(t *testing.T) {
	t.Parallel()

	opCtx := limitsOperationContext(t, `{ root { child { id } } }`)

	assert.Nil(t, operationDepthLimit{max: 3}.MutateOperationContext(context.Background(), opCtx))
}

func TestOperationDepthLimit_RejectsBeyondLimit(t *testing.T) {
	t.Parallel()

	opCtx := limitsOperationContext(t, `{ root { child { child { id } } } }`)

	err := operationDepthLimit{max: 3}.MutateOperationContext(context.Background(), opCtx)

	require.NotNil(t, err)
	assert.Contains(t, err.Message, "depth 4")
	assert.Equal(t, querycost.DepthLimitErrorCode, err.Extensions["code"])
}

func TestOperationDepthLimit_HandlesMissingOperation(t *testing.T) {
	t.Parallel()

	limit := operationDepthLimit{max: 1}

	assert.Nil(t, limit.MutateOperationContext(context.Background(), nil))
	assert.Nil(t, limit.MutateOperationContext(context.Background(), &graphql.OperationContext{}))
}

func TestOperationDefinition_FallsBackToDocument(t *testing.T) {
	t.Parallel()

	opCtx := limitsOperationContext(t, `query Named { root { id } }`)
	opCtx.Operation = nil
	opCtx.OperationName = "Named"

	require.NotNil(t, operationDefinition(opCtx))
	assert.Equal(t, "Named", operationDefinition(opCtx).Name)
}

func TestProtocolErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *gqlerror.Error
		expected string
		ok       bool
	}{
		{name: "nil error"},
		{name: "no extensions", err: &gqlerror.Error{Message: "boom"}},
		{
			name: "domain error code is not a protocol error",
			err: &gqlerror.Error{
				Extensions: map[string]any{"code": "Invalid"},
			},
		},
		{
			name: "non string code",
			err:  &gqlerror.Error{Extensions: map[string]any{"code": 42}},
		},
		{
			name:     "depth limit",
			err:      &gqlerror.Error{Extensions: map[string]any{"code": querycost.DepthLimitErrorCode}},
			expected: querycost.DepthLimitErrorCode,
			ok:       true,
		},
		{
			name: "complexity limit",
			err: &gqlerror.Error{
				Extensions: map[string]any{"code": querycost.ComplexityLimitErrorCode},
			},
			expected: querycost.ComplexityLimitErrorCode,
			ok:       true,
		},
		{
			name:     "gqlparser validation failure",
			err:      &gqlerror.Error{Extensions: map[string]any{"code": errcode.ValidationFailed}},
			expected: errcode.ValidationFailed,
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, ok := protocolErrorCode(tt.err)

			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestCostLimitedSchema_DelegatesAndPreservesCapabilities(t *testing.T) {
	t.Parallel()

	inner := generated.NewExecutableSchema(generated.Config{})
	wrapped := newCostLimitedSchema(inner)

	require.NotNil(t, wrapped.Schema())
	assert.Same(t, inner.Schema(), wrapped.Schema())

	cost, ok := wrapped.Complexity(context.Background(), "Query", "shipments", 4, nil)
	assert.True(t, ok, "connection fields must be priced by the querycost index")
	assert.Positive(t, cost)

	_, isEventSchema := inner.(graphql.ExecutableSchemaWithEventContext)
	assert.False(t, isEventSchema,
		"the generated schema now implements ExecutableSchemaWithEventContext; "+
			"costLimitedSchema must forward ExecWithEventContext before subscriptions ship")
}
