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
  workerRecognitions(workerId: ID!, filter: CodeFilter, region: String, limit: Int): [Recognition!]!
}
`

func TestAlongsideKeys_AKeyedReadTakesOnlyWhatTheListsCallSends(t *testing.T) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: alongsideSchema})
	boundSchema = schema
	t.Cleanup(func() { boundSchema = nil })
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
		set  []string
	}{
		"keyed by the worker the call names": {
			text: `query DeskWorkerRecognitions($workerId: ID!) {
				workerRecognitions(workerId: $workerId) { id } }`,
			keys: []string{"workerId"},
		},
		"an optional variable the call does not send is the spec's to set": {
			text: `query DeskWorkerRecognitions($workerId: ID!, $region: String, $limit: Int) {
				workerRecognitions(workerId: $workerId, region: $region, limit: $limit) { id } }`,
			keys: []string{"workerId"},
			set:  []string{"region", "limit"},
		},
		"a required variable the call does not send": {
			text: `query DeskWorkerRecognitions($workerId: ID!, $region: String!) {
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
			keys, set, problem := alongsideKeys(list,
				supplyOperation(t, schema, "DeskWorkerRecognitions", tc.text),
				"DeskWorkerRecognitions")
			assert.Equal(t, tc.keys, keys)
			assert.Equal(t, tc.set, set)
			assert.Equal(t, tc.keys == nil, problem != "", problem)
		})
	}
}

const keyedGetSchema = `
type HoursState { workerId: ID! }
type Assignment { id: ID! }
type Query {
  hoursState(workerId: ID!): HoursState
  hoursStates(workerIds: [ID!]): [HoursState!]!
  assignments(workerId: ID!, activeOnly: Boolean): [Assignment!]!
  everyAssignment(activeOnly: Boolean): [Assignment!]!
}
`

func TestGetKey_AGetSendsTheIDOfTheRecordItReads(t *testing.T) {
	t.Parallel()

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: keyedGetSchema})
	cases := map[string]struct {
		text string
		key  string
	}{
		"by its own id": {
			text: `query Hours($id: ID!) { hoursState(workerId: $id) { workerId } }`,
			key:  "id",
		},
		"by the id of the record it belongs to": {
			text: `query Hours($workerId: ID!) { hoursState(workerId: $workerId) { workerId } }`,
			key:  "workerId",
		},
		"by nothing it requires": {
			text: `query Hours($workerIds: [ID!]) { hoursStates(workerIds: $workerIds) { workerId } }`,
		},
		"by two ids, neither of them its own": {
			text: `query Hours($workerId: ID!, $otherId: ID!) {
				a: hoursState(workerId: $workerId) { workerId }
				b: hoursState(workerId: $otherId) { workerId } }`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.key, getKey(supplyOperation(t, schema, "Hours", tc.text)))
		})
	}
}

func TestKeyedWhole_AGetReadsTheListOneRecordHolds(t *testing.T) {
	t.Parallel()

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: keyedGetSchema})
	assert.True(t, keyedWhole(schema.Query.Fields.ForName("assignments")))
	assert.False(t, keyedWhole(schema.Query.Fields.ForName("everyAssignment")),
		"a list keyed by no record is not one record's")
	assert.False(t, keyedWhole(schema.Query.Fields.ForName("hoursState")),
		"one record is not a list")
	assert.False(t, keyedWhole(schema.Query.Fields.ForName("hoursStates")),
		"a list of ids names no one record")
}
