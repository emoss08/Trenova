package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const supplySchema = `
scalar Any
input DataTableConnectionInput { first: Int, after: String }
type FleetCode { id: ID! }
type FleetCodeConnection { edges: [FleetCodeEdge!]! }
type FleetCodeEdge { node: FleetCode! }
type Query {
  fleetCodes(input: DataTableConnectionInput!, includeTotalCount: Boolean!, region: String!): FleetCodeConnection!
  fleetCode(id: ID!): FleetCode
}
`

func supplyOperation(t *testing.T, schema *ast.Schema, name, text string) *operation {
	t.Helper()

	doc, errs := gqlparser.LoadQueryWithRules(schema, text, nil)
	require.Empty(t, errs)

	return &operation{Name: name, Query: doc}
}

func TestUnsupplied_RefusesARequiredVariableTheToolCannotSend(t *testing.T) {
	t.Parallel()

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: supplySchema})
	cases := map[string]struct {
		verb    string
		text    string
		refused bool
	}{
		"a list sends its input and the total count": {
			verb: verbList,
			text: `query FleetCodeTable($input: DataTableConnectionInput!, $includeTotalCount: Boolean!) {
				fleetCodes(input: $input, includeTotalCount: $includeTotalCount, region: "west") { edges { node { id } } } }`,
		},
		"a list cannot send a region": {
			verb: verbList,
			text: `query FleetCodeTable($input: DataTableConnectionInput!, $includeTotalCount: Boolean!, $region: String!) {
				fleetCodes(input: $input, includeTotalCount: $includeTotalCount, region: $region) { edges { node { id } } } }`,
			refused: true,
		},
		"a get sends the id": {
			verb: verbGet,
			text: `query FleetCode($id: ID!) { fleetCode(id: $id) { id } }`,
		},
		"a get cannot send a list's input": {
			verb: verbGet,
			text: `query FleetCode($input: DataTableConnectionInput!, $includeTotalCount: Boolean!) {
				fleetCodes(input: $input, includeTotalCount: $includeTotalCount, region: "west") { edges { node { id } } } }`,
			refused: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			problem := unsupplied(&binding{
				Tool:      "masterdata_list_fleet_codes",
				Verb:      tc.verb,
				Operation: supplyOperation(t, schema, "FleetCodeTable", tc.text),
			})
			assert.Equal(t, tc.refused, problem != "", problem)
		})
	}
}
