package schemadiff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func load(t *testing.T, sdl string) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "test", Input: sdl})
	require.NoError(t, err)

	return schema
}

func paths(changes []Change) []string {
	out := make([]string, 0, len(changes))
	for _, c := range changes {
		out = append(out, c.Path)
	}

	return out
}

const baseSDL = `
enum Status { ACTIVE INACTIVE }
interface Node { id: ID! }
union SearchResult = Shipment | Customer
input ShipmentFilter { status: Status, limit: Int }
type Customer implements Node { id: ID! name: String! }
type Shipment implements Node {
  id: ID!
  bol: String!
  weight: Float
  customer: Customer
  stops(first: Int, after: String): [String!]!
}
type Query {
  shipment(id: ID!): Shipment
  shipments(filter: ShipmentFilter): [Shipment!]!
  search(term: String!): [SearchResult!]!
}
`

func TestCompare_IdenticalSchemasHaveNoChanges(t *testing.T) {
	t.Parallel()

	report := Compare(load(t, baseSDL), load(t, baseSDL))

	assert.Empty(t, report.Changes)
	assert.False(t, report.HasBreaking())
}

func TestCompare_RemovedTypeAndField(t *testing.T) {
	t.Parallel()

	head := load(t, `
enum Status { ACTIVE INACTIVE }
interface Node { id: ID! }
union SearchResult = Shipment
input ShipmentFilter { status: Status, limit: Int }
type Shipment implements Node {
  id: ID!
  weight: Float
  stops(first: Int, after: String): [String!]!
}
type Query {
  shipment(id: ID!): Shipment
  shipments(filter: ShipmentFilter): [Shipment!]!
  search(term: String!): [SearchResult!]!
}
`)

	report := Compare(load(t, baseSDL), head)

	assert.ElementsMatch(t, []string{
		"Customer",
		"Shipment.bol",
		"Shipment.customer",
		"SearchResult.Customer",
	}, paths(report.Breaking()))
	assert.True(t, report.HasBreaking())
}

func TestCompare_OutputNullability(t *testing.T) {
	t.Parallel()

	loosened := load(t, `
type Shipment { id: ID! bol: String weight: Float! }
type Query { shipment(id: ID!): Shipment }
`)
	base := load(t, `
type Shipment { id: ID! bol: String! weight: Float }
type Query { shipment(id: ID!): Shipment }
`)

	report := Compare(base, loosened)

	assert.Equal(t, []string{"Shipment.bol"}, paths(report.Breaking()),
		"non-null to nullable on output is breaking")
	nonBreaking := paths(report.filter(SeverityNonBreaking))
	assert.Contains(t, nonBreaking, "Shipment.weight",
		"nullable to non-null on output is safe")
}

func TestCompare_InputNullabilityIsInverted(t *testing.T) {
	t.Parallel()

	base := load(t, `
input F { a: Int, b: Int! }
type Query { q(f: F, x: Int, y: Int!): Int }
`)
	head := load(t, `
input F { a: Int!, b: Int }
type Query { q(f: F, x: Int!, y: Int): Int }
`)

	report := Compare(base, head)

	assert.ElementsMatch(t, []string{"F.a", "Query.q(x)"}, paths(report.Breaking()),
		"nullable to non-null on inputs and arguments is breaking")
	assert.ElementsMatch(t, []string{"F.b", "Query.q(y)"},
		paths(report.filter(SeverityNonBreaking)),
		"non-null to nullable on inputs and arguments is safe")
}

func TestCompare_RequiredAdditionsAreBreaking(t *testing.T) {
	t.Parallel()

	base := load(t, `
input F { a: Int }
type Query { q(f: F): Int }
`)
	head := load(t, `
input F { a: Int, required: Int!, withDefault: Int! = 3, optional: Int }
type Query { q(f: F, needed: Boolean!, defaulted: Boolean! = true, extra: Int): Int }
`)

	report := Compare(base, head)

	assert.ElementsMatch(t, []string{"F.required", "Query.q(needed)"},
		paths(report.Breaking()))
	assert.ElementsMatch(t, []string{
		"F.withDefault", "F.optional", "Query.q(defaulted)", "Query.q(extra)",
	}, paths(report.filter(SeverityNonBreaking)))
}

func TestCompare_ArgumentRemovedAndTypeChanged(t *testing.T) {
	t.Parallel()

	base := load(t, `type Query { q(a: Int, b: String, c: [Int!]): Int }`)
	head := load(t, `type Query { q(b: Int, c: Int): Int }`)

	report := Compare(base, head)

	assert.ElementsMatch(t, []string{"Query.q(a)", "Query.q(b)", "Query.q(c)"},
		paths(report.Breaking()))
}

func TestCompare_EnumValues(t *testing.T) {
	t.Parallel()

	base := load(t, `enum S { A B } type Query { s: S }`)
	head := load(t, `enum S { A C } type Query { s: S }`)

	report := Compare(base, head)

	assert.Equal(t, []string{"S.B"}, paths(report.Breaking()))
	assert.Equal(t, []string{"S.C"}, paths(report.Dangerous()),
		"added enum values are dangerous, not breaking")
}

func TestCompare_ListShapeChangeIsBreaking(t *testing.T) {
	t.Parallel()

	base := load(t, `type Query { tags: [String!]! }`)
	head := load(t, `type Query { tags: String! }`)

	report := Compare(base, head)

	assert.Equal(t, []string{"Query.tags"}, paths(report.Breaking()))
}

func TestCompare_InterfaceDropped(t *testing.T) {
	t.Parallel()

	base := load(t, `interface Node { id: ID! } type T implements Node { id: ID! } type Query { t: T }`)
	head := load(t, `interface Node { id: ID! } type T { id: ID! } type Query { t: T }`)

	report := Compare(base, head)

	assert.Equal(t, []string{"T"}, paths(report.Breaking()))
	assert.Contains(t, report.Breaking()[0].Message, "no longer implements Node")
}

func TestCompare_KindChangeIsBreaking(t *testing.T) {
	t.Parallel()

	base := load(t, `type T { id: ID! } type Query { t: T }`)
	head := load(t, `input T { id: ID! } type Query { t(t: T): Int }`)

	report := Compare(base, head)

	assert.Contains(t, paths(report.Breaking()), "T")
}

func TestCompare_AdditionsAreReportedButSafe(t *testing.T) {
	t.Parallel()

	base := load(t, `type Query { a: Int }`)
	head := load(t, `type New { x: Int } type Query { a: Int b: Int @deprecated(reason: "use c") c: New }`)

	report := Compare(base, head)

	assert.False(t, report.HasBreaking())
	assert.ElementsMatch(t, []string{"New", "Query.b", "Query.c"},
		paths(report.filter(SeverityNonBreaking)))
}

func TestCompare_OrdersBreakingFirst(t *testing.T) {
	t.Parallel()

	base := load(t, `enum S { A } type Query { gone: Int s: S }`)
	head := load(t, `enum S { A B } type Query { s: S added: Int }`)

	report := Compare(base, head)

	require.Len(t, report.Changes, 3)
	assert.Equal(t, SeverityBreaking, report.Changes[0].Severity)
	assert.Equal(t, SeverityDangerous, report.Changes[1].Severity)
	assert.Equal(t, SeverityNonBreaking, report.Changes[2].Severity)
}

func TestCompare_WireCompatibleScalarChangeIsDangerous(t *testing.T) {
	t.Parallel()

	base := load(t, `
scalar Timestamp
scalar Decimal
input F { createdAt: Int!, amount: String, ids: [Int!]! }
type Shipment { createdAt: Int!, amount: String, stops: [Int!], rate: Decimal!, ratedAt: Timestamp }
type Query { shipment(since: Int, rate: Decimal): Shipment }
`)
	head := load(t, `
scalar Timestamp
scalar Decimal
input F { createdAt: Timestamp!, amount: Decimal, ids: [Timestamp!]! }
type Shipment { createdAt: Timestamp!, amount: Decimal, stops: [Timestamp!], rate: String!, ratedAt: Int }
type Query { shipment(since: Timestamp, rate: String): Shipment }
`)

	report := Compare(base, head)

	assert.False(t, report.HasBreaking())
	assert.ElementsMatch(t, []string{
		"F.createdAt",
		"F.amount",
		"F.ids",
		"Shipment.createdAt",
		"Shipment.amount",
		"Shipment.stops",
		"Shipment.rate",
		"Shipment.ratedAt",
		"Query.shipment(since)",
		"Query.shipment(rate)",
	}, paths(report.Dangerous()),
		"Int<->Timestamp and String<->Decimal are wire-compatible in both directions")
	for _, change := range report.Dangerous() {
		assert.Equal(t, wireCompatibleMessage, change.Message)
	}
	assert.Empty(t, report.filter(SeverityNonBreaking))
}

func TestCompare_WireCompatibleScalarWithNullabilityChangeIsBreaking(t *testing.T) {
	t.Parallel()

	base := load(t, `
scalar Timestamp
scalar Decimal
input F { createdAt: Int, amount: Decimal! }
type Shipment { createdAt: Int!, amount: String, stops: [Int!]! }
type Query { shipment(since: Int): Shipment }
`)
	head := load(t, `
scalar Timestamp
scalar Decimal
input F { createdAt: Timestamp!, amount: String }
type Shipment { createdAt: Timestamp, amount: Decimal!, stops: [Timestamp]! }
type Query { shipment(since: Timestamp!): Shipment }
`)

	report := Compare(base, head)

	assert.ElementsMatch(t, []string{
		"F.createdAt",
		"F.amount",
		"Shipment.createdAt",
		"Shipment.amount",
		"Shipment.stops",
		"Query.shipment(since)",
	}, paths(report.Breaking()),
		"a nullability or list-shape change is still breaking even between wire-compatible scalars")
	assert.Empty(t, report.Dangerous())
}

func TestLoadDir_RealSchemaIsSelfConsistent(t *testing.T) {
	t.Parallel()

	schema, err := LoadDir("../schema")
	require.NoError(t, err)

	report := Compare(schema, schema)
	assert.Empty(t, report.Changes, "a schema diffed against itself must be clean")
}

func TestLoadDir_MissingDirectory(t *testing.T) {
	t.Parallel()

	_, err := LoadDir("./does-not-exist")
	assert.Error(t, err)
}
