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
  managerFleetCode(managerId: ID!, region: String): FleetCode
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
		"a get keyed by another record sends that record's id": {
			verb: verbGet,
			text: `query ManagerFleetCode($managerId: ID!) { managerFleetCode(managerId: $managerId) { id } }`,
		},
		"a get cannot send a second key": {
			verb: verbGet,
			text: `query ManagerFleetCode($managerId: ID!, $region: String!) {
				managerFleetCode(managerId: $managerId, region: $region) { id } }`,
			refused: true,
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

const keyedSchema = `
input CodeFilter { region: String }
type Checklist { id: ID! }
type Query {
  workerChecklists(workerId: ID!, includeClosed: Boolean = true, filter: CodeFilter): [Checklist!]!
}
`

func TestUnsupplied_AnUnpagedListSendsItsOwnArguments(t *testing.T) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: keyedSchema})
	boundSchema = schema
	t.Cleanup(func() { boundSchema = nil })
	field := schema.Query.Fields.ForName("workerChecklists")

	cases := map[string]struct {
		text    string
		refused bool
	}{
		"the worker and a switch are its parameters": {
			text: `query DeskWorkerChecklists($workerId: ID!, $includeClosed: Boolean) {
				workerChecklists(workerId: $workerId, includeClosed: $includeClosed) { id } }`,
		},
		"an input object is not": {
			text: `query DeskWorkerChecklists($workerId: ID!, $filter: CodeFilter!) {
				workerChecklists(workerId: $workerId, filter: $filter) { id } }`,
			refused: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			problem := unsupplied(&binding{
				Tool:      "worker_list_checklists",
				Verb:      verbList,
				RootField: field,
				Operation: supplyOperation(t, schema, "DeskWorkerChecklists", tc.text),
			})
			assert.Equal(t, tc.refused, problem != "", problem)
		})
	}
}

const alongsideSchema = `
input CodeFilter { region: String }
type Event { id: ID! }
type Recognition { id: ID! }
type Query {
  workerEvents(workerId: ID, openOnly: Boolean): [Event!]!
  workerRecognitions(workerId: ID!, filter: CodeFilter, region: String): [Recognition!]!
}
`

func TestAlongsideKeys_AKeyedReadTakesOnlyWhatTheListsCallSends(t *testing.T) {
	t.Parallel()

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: alongsideSchema})
	list := &binding{
		Tool: "worker_list_events",
		Verb: verbList,
		Operation: supplyOperation(t, schema, "DeskWorkerEvents",
			`query DeskWorkerEvents($workerId: ID, $openOnly: Boolean) {
				workerEvents(workerId: $workerId, openOnly: $openOnly) { id } }`),
	}
	cases := map[string]struct {
		text string
		keys []string
	}{
		"keyed by the worker the call names": {
			text: `query DeskWorkerRecognitions($workerId: ID!) {
				workerRecognitions(workerId: $workerId) { id } }`,
			keys: []string{"workerId"},
		},
		"a variable the call does not send": {
			text: `query DeskWorkerRecognitions($workerId: ID!, $region: String) {
				workerRecognitions(workerId: $workerId, region: $region) { id } }`,
		},
		"an input object": {
			text: `query DeskWorkerRecognitions($workerId: ID!, $filter: CodeFilter) {
				workerRecognitions(workerId: $workerId, filter: $filter) { id } }`,
		},
		"keyed by nothing": {
			text: `query DeskWorkerRecognitions {
				workerRecognitions(workerId: "wrk_1") { id } }`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys, problem := alongsideKeys(list,
				supplyOperation(t, schema, "DeskWorkerRecognitions", tc.text),
				"DeskWorkerRecognitions")
			assert.Equal(t, tc.keys, keys)
			assert.Equal(t, tc.keys == nil, problem != "", problem)
		})
	}
}
