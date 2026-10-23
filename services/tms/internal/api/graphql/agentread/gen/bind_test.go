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

const accountingSchema = `
scalar Any
enum Sys { Xero, Netsuite }
input DataTableConnectionInput { first: Int, after: String, query: String, fieldFilters: [Any!], sort: [Any!] }
input RecordFilter { statuses: [String!], search: String }
input Nested { inner: RecordFilter! }
type PageInfo { hasNextPage: Boolean!, endCursor: String }
type Record { id: ID! }
type RecordEdge { node: Record! }
type RecordConnection { edges: [RecordEdge!]!, pageInfo: PageInfo! }
type Overview { tolerance: Int! }
type State { objectId: ID! }
type Query {
  records(integrationType: Sys!, input: DataTableConnectionInput!): RecordConnection!
  ledger(integrationType: Sys!, first: Int, after: String, filter: RecordFilter, nested: Nested): RecordConnection!
  overview(integrationType: Sys!): Overview!
  states(objectIds: [ID!]!, integrationType: Sys): [State!]!
  stateFilter(objectIds: [ID!]!, filter: RecordFilter): [State!]!
  mismatched(objectIds: [ID!]!, integrationType: String): [State!]!
  record(id: ID!, region: RecordFilter): Record!
}
`

func accountingBinding(t *testing.T, schema *ast.Schema, verb, root, text string) *binding {
	t.Helper()

	return &binding{
		Tool:      "accounting_list_records",
		Verb:      verb,
		RootField: schema.Query.Fields.ForName(root),
		Operation: supplyOperation(t, schema, "Op", text),
	}
}

// A paged list takes its root's own arguments as parameters, and builds a
// filter input from its fields; a get takes scalar arguments but never an
// input object.
func TestUnsupplied_RootArgumentsAndFilterInputs(t *testing.T) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: accountingSchema})
	boundSchema = schema
	t.Cleanup(func() { boundSchema = nil })

	cases := map[string]struct {
		verb, root, text string
		refused          bool
	}{
		"a required system beside the table input": {
			verb: verbList, root: "records",
			text: `query Op($integrationType: Sys!, $input: DataTableConnectionInput!) {
				records(integrationType: $integrationType, input: $input) { edges { node { id } } pageInfo { hasNextPage } } }`,
		},
		"a filter input on a list paged by its root": {
			verb: verbList, root: "ledger",
			text: `query Op($integrationType: Sys!, $first: Int, $after: String, $filter: RecordFilter!) {
				ledger(integrationType: $integrationType, first: $first, after: $after, filter: $filter) { edges { node { id } } pageInfo { hasNextPage } } }`,
		},
		"a filter input that needs another input": {
			verb: verbList, root: "ledger",
			text: `query Op($integrationType: Sys!, $nested: Nested!) {
				ledger(integrationType: $integrationType, nested: $nested) { edges { node { id } } pageInfo { hasNextPage } } }`,
			refused: true,
		},
		"a get keyed by the system": {
			verb: verbGet, root: "overview",
			text: `query Op($integrationType: Sys!) { overview(integrationType: $integrationType) { tolerance } }`,
		},
		"a get cannot build an input": {
			verb: verbGet, root: "record",
			text:    `query Op($id: ID!, $region: RecordFilter!) { record(id: $id, region: $region) { id } }`,
			refused: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			problem := unsupplied(accountingBinding(t, schema, tc.verb, tc.root, tc.text))
			assert.Equal(t, tc.refused, problem != "", problem)
		})
	}
}

// A read alongside a paged list or a get is keyed by the call, an object or
// an unpaged list; a read instead of a list has a variable of its own, and
// shares the list's others at the same type.
func TestBindAlongsideAndInstead_KeyedByTheCall(t *testing.T) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: accountingSchema})
	boundSchema, boundChecks = schema, map[string]string{
		"records": "accounting_sync", "overview": "accounting_sync", "states": "accounting_sync",
		"statefilter": "accounting_sync", "mismatched": "accounting_sync",
	}
	t.Cleanup(func() { boundSchema, boundChecks = nil, nil })

	op := func(name, text string) *operation {
		doc, errs := gqlparser.LoadQueryWithRules(schema, text, nil)
		require.Empty(t, errs)
		roots := []string{}
		for _, selection := range doc.Operations[0].SelectionSet {
			roots = append(roots, selection.(*ast.Field).Name)
		}
		return &operation{Name: name, Query: doc, Roots: roots}
	}
	catalog := map[string]*operation{
		"Records": op("Records", `query Records($integrationType: Sys!, $input: DataTableConnectionInput!) {
			records(integrationType: $integrationType, input: $input) { edges { node { id } } pageInfo { hasNextPage } } }`),
		"Overview": op("Overview", `query Overview($integrationType: Sys!) {
			overview(integrationType: $integrationType) { tolerance } }`),
		"States": op("States", `query States($objectIds: [ID!]!, $integrationType: Sys) {
			states(objectIds: $objectIds, integrationType: $integrationType) { objectId } }`),
		"StateFilter": op("StateFilter", `query StateFilter($objectIds: [ID!]!, $filter: RecordFilter) {
			stateFilter(objectIds: $objectIds, filter: $filter) { objectId } }`),
		"Mismatched": op("Mismatched", `query Mismatched($objectIds: [ID!]!, $integrationType: String) {
			mismatched(objectIds: $objectIds, integrationType: $integrationType) { objectId } }`),
	}
	list := &binding{
		Tool: "accounting_list_records", Verb: verbList, Operation: catalog["Records"],
		RootField: schema.Query.Fields.ForName("records"),
	}

	beside, problem := bindAlongside(catalog, list, "Overview")
	require.Empty(t, problem)
	assert.True(t, beside.Object)
	assert.Equal(t, []string{"integrationType"}, beside.Keys)
	_, problem = bindAlongside(catalog, list, "Records")
	assert.Contains(t, problem, "only an unpaged list reads one")

	instead, problem := bindInstead(catalog, list, "States")
	require.Empty(t, problem)
	assert.Equal(t, []string{"objectIds"}, instead.Own)
	assert.Equal(t, []string{"integrationType"}, instead.Keys)
	_, problem = bindInstead(catalog, list, "StateFilter")
	assert.Contains(t, problem, "a parameter cannot build it")
	_, problem = bindInstead(catalog, list, "Mismatched")
	assert.Contains(t, problem, "where the list's is")
	_, problem = bindInstead(catalog, list, "Overview")
	assert.Contains(t, problem, "no variable of its own")
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
