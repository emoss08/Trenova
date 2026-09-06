package querycost

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const testSchema = `
type PageInfo { hasNextPage: Boolean! endCursor: String }

type Stop { id: ID! name: String! }

type Shipment {
  id: ID!
  tags: [String!]!
  stops: [Stop!]!
  origin: Stop
}

type ShipmentEdge { node: Shipment! cursor: String! }

type ShipmentConnection {
  edges: [ShipmentEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

input ConnectionInput { first: Int, after: String }

type Query {
  shipment(id: ID!): Shipment
  shipments(input: ConnectionInput!): ShipmentConnection!
  cursorShipments(first: Int, after: String): ShipmentConnection!
  allStops: [Stop!]!
}
`

func testIndex(t *testing.T) (*Index, *ast.Schema) {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "test", Input: testSchema})
	require.NoError(t, err)

	return NewIndex(schema), schema
}

func TestNewIndex_Classification(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	assert.True(t, idx.IsConnectionType("ShipmentConnection"))
	assert.False(t, idx.IsConnectionType("Shipment"))
	assert.False(t, idx.IsConnectionType("ShipmentEdge"))

	assert.True(t, idx.IsConnectionField("Query", "shipments"))
	assert.True(t, idx.IsConnectionField("Query", "cursorShipments"))
	assert.False(t, idx.IsConnectionField("Query", "shipment"))

	assert.True(t, idx.IsListField("Query", "allStops"))
	assert.True(t, idx.IsListField("Shipment", "stops"))
	assert.True(t, idx.IsListField("Shipment", "tags"))
	assert.False(t, idx.IsListField("Shipment", "origin"))
	assert.False(t, idx.IsListField("Shipment", "id"))
}

func TestNewIndex_NilSchema(t *testing.T) {
	t.Parallel()

	idx := NewIndex(nil)

	assert.False(t, idx.IsConnectionType("Anything"))
	_, ok := idx.Complexity("Query", "shipments", 5, nil)
	assert.False(t, ok)
}

func TestIndexComplexity_ConnectionUsesRequestedPageSize(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Query", "shipments", 10, map[string]any{
		"input": map[string]any{"first": 25},
	})
	require.True(t, ok)
	assert.Equal(t, 250, cost)

	cost, ok = idx.Complexity("Query", "cursorShipments", 10, map[string]any{"first": 7})
	require.True(t, ok)
	assert.Equal(t, 70, cost)
}

func TestIndexComplexity_ConnectionClampsPageSize(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Query", "shipments", 3, map[string]any{
		"input": map[string]any{"first": pagination.MaxLimit * 100},
	})
	require.True(t, ok)
	assert.Equal(t, 3*pagination.MaxLimit, cost)
}

func TestIndexComplexity_ConnectionDefaultsWhenPageSizeAbsent(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Query", "shipments", 4, nil)
	require.True(t, ok)
	assert.Equal(t, 4*pagination.DefaultLimit, cost)
}

func TestIndexComplexity_ReadsJSONNumberPageSize(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Query", "cursorShipments", 2, map[string]any{
		"first": jsonNumber("50"),
	})
	require.True(t, ok)
	assert.Equal(t, 100, cost)
}

func TestIndexComplexity_ConnectionChildrenFallThrough(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	_, ok := idx.Complexity("ShipmentConnection", "edges", 12, nil)
	assert.False(t, ok, "edges must not be multiplied again, the connection already charged for it")
}

func TestIndexComplexity_UnpaginatedList(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Shipment", "stops", 3, nil)
	require.True(t, ok)
	assert.Equal(t, 3*UnpaginatedListWeight, cost)

	cost, ok = idx.Complexity("Shipment", "tags", 0, nil)
	require.True(t, ok)
	assert.Equal(t, UnpaginatedListWeight, cost)
}

func TestIndexComplexity_NonListFieldsFallThrough(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	_, ok := idx.Complexity("Shipment", "origin", 5, nil)
	assert.False(t, ok)
	_, ok = idx.Complexity("Query", "shipment", 5, nil)
	assert.False(t, ok)
}

func TestScale_SaturatesInsteadOfOverflowing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, maxCost, scale(math.MaxInt32, 100))
	assert.Equal(t, maxCost, scale(maxCost/2, 3))
	assert.Positive(t, scale(math.MaxInt32, 100), "overflow must never produce a negative cost")
	assert.Equal(t, 20, scale(0, 20))
	assert.Equal(t, 5, scale(5, 0))
}

func TestPageSize_NegativeAndBogusValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, pagination.DefaultLimit, pageSize(nil))
	assert.Equal(t, pagination.DefaultLimit, pageSize(map[string]any{"first": "abc"}))
	assert.Equal(t, -5, pageSize(map[string]any{"first": -5}))
	assert.Equal(t, pagination.DefaultLimit, pageSize(map[string]any{"other": 3}))
}

func TestIndexComplexity_NegativePageSizeIsClamped(t *testing.T) {
	t.Parallel()

	idx, _ := testIndex(t)

	cost, ok := idx.Complexity("Query", "cursorShipments", 4, map[string]any{"first": -100})
	require.True(t, ok)
	assert.Equal(t, 4*pagination.DefaultLimit, cost,
		"a negative page size must clamp to the default, never to a negative cost")
}

func parseOp(t *testing.T, schema *ast.Schema, query string) *ast.OperationDefinition {
	t.Helper()

	doc, errs := gqlparser.LoadQuery(schema, query)
	require.Empty(t, errs)
	require.NotEmpty(t, doc.Operations)

	return doc.Operations[0]
}

func TestDepth(t *testing.T) {
	t.Parallel()

	_, schema := testIndex(t)

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "single scalar field",
			query:    `{ shipment(id: "1") { id } }`,
			expected: 2,
		},
		{
			name:     "nested object",
			query:    `{ shipment(id: "1") { origin { id } } }`,
			expected: 3,
		},
		{
			name:     "connection traversal",
			query:    `{ shipments(input: {}) { edges { node { stops { id } } } } }`,
			expected: 5,
		},
		{
			name: "fragment spread does not add a level",
			query: `
				fragment ShipmentFields on Shipment { origin { id } }
				{ shipment(id: "1") { ...ShipmentFields } }`,
			expected: 3,
		},
		{
			name:     "inline fragment does not add a level",
			query:    `{ shipment(id: "1") { ... on Shipment { origin { id } } } }`,
			expected: 3,
		},
		{
			name:     "deepest branch wins",
			query:    `{ shipment(id: "1") { id origin { id } stops { id } } }`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, Depth(parseOp(t, schema, tt.query)))
		})
	}
}

func TestDepth_NilOperation(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, Depth(nil))
}

type jsonNumber string

func (n jsonNumber) Int64() (int64, error) {
	var value int64
	for _, r := range string(n) {
		if r < '0' || r > '9' {
			return 0, errNotANumber
		}
		value = value*10 + int64(r-'0')
	}

	return value, nil
}

var errNotANumber = assertError("not a number")

type assertError string

func (e assertError) Error() string { return string(e) }
