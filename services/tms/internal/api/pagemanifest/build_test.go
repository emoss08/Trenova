package pagemanifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/api/pagemanifest"
	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const repoRoot = "../../../../.."

// The checked-in manifest is what the generator writes from the screens, the
// client's operations, the schema, the registry and the route table, so a
// change to any of them that moves what a screen needs fails until it is
// regenerated and reviewed.
func TestPageAccessManifestIsGenerated(t *testing.T) {
	t.Parallel()

	inputs, err := pagemanifest.FromRepository(repoRoot, fieldsensitivity.DefaultRegistry())
	require.NoError(t, err)
	manifest, err := pagemanifest.Build(inputs)
	require.NoError(t, err)
	rendered, err := pagemanifest.Render(manifest)
	require.NoError(t, err)

	current, err := os.ReadFile(filepath.Join(repoRoot, pagemanifest.ManifestPath))
	require.NoError(t, err)
	require.Equal(t, string(rendered), string(current),
		"pkg/pageaccess/manifest_gen.json is stale. Run `pnpm --filter @trenova/web "+
			"access:generate`, then `go run ./internal/api/graphql/gqlext/refgen` from "+
			"services/tms, and commit both generated files")
}

const testSchema = `
directive @requiresRead(kind: String!) repeatable on QUERY | MUTATION
type Query {
  shipment(id: ID!): Shipment
  shipments: ShipmentConnection!
}
type Mutation {
  updateShipment(id: ID!): Shipment!
}
type ShipmentConnection { edges: [ShipmentEdge!]! }
type ShipmentEdge { node: Shipment! }
type Shipment {
  id: ID!
  proNumber: String!
  customer: Customer
  moves: [ShipmentMove!]!
}
type ShipmentMove { id: ID! status: String! }
type Customer {
  id: ID!
  name: String!
  code: String!
  creditLimit: Int
}
`

func testInputs(t *testing.T, operations string) *pagemanifest.Inputs {
	t.Helper()

	schema := gqlparser.MustLoadSchema(&ast.Source{Name: "schema", Input: testSchema})
	doc, err := parser.ParseQuery(&ast.Source{Name: "ops", Input: operations})
	require.NoError(t, err)

	in := &pagemanifest.Inputs{
		Schema:     schema,
		Registry:   fieldsensitivity.DefaultRegistry(),
		Operations: doc.Operations,
		Fragments:  doc.Fragments,
	}
	return in
}

func screen(path string, operations []string, rest ...[2]string) pagemanifest.Screens {
	var s pagemanifest.Screens
	entry := pagemanifest.ScreenEntry{Path: path, Operations: operations}
	for _, call := range rest {
		entry.Rest = append(entry.Rest, pagemanifest.RestCall{Method: call[0], Path: call[1]})
	}
	s.Screens = append(s.Screens, entry)
	return s
}

func TestBuild_SortsAScreensKindsByHowItReadsThem(t *testing.T) {
	t.Parallel()

	in := testInputs(t, `
query ShipmentDetail @requiresRead(kind: "customer") {
  shipment(id: "1") { proNumber customer { name creditLimit } }
}
query ShipmentList {
  shipments { edges { node { proNumber customer { name code } } } }
}
mutation ShipmentSave {
  updateShipment(id: "1") { proNumber }
}
`)
	in.Screens = screen("/shipments", []string{"ShipmentDetail", "ShipmentList", "ShipmentSave"})

	manifest, err := pagemanifest.Build(in)
	require.NoError(t, err)
	require.Len(t, manifest.Screens, 1)
	got := manifest.Screens[0]
	assert.Equal(t, []string{"shipment"}, got.Reads)
	assert.Equal(t, []string{"customer"}, got.Declared)
	assert.Equal(t, []string{}, got.References)

	list, ok := manifest.Operation("ShipmentList")
	require.True(t, ok)
	assert.Equal(t, []pageaccess.LinkedKind{{Kind: "customer", Fields: []string{"code", "name"}}},
		list.Linked)
	save, ok := manifest.Operation("ShipmentSave")
	require.True(t, ok)
	assert.Equal(t, "mutation", save.Type)
	assert.Empty(t, save.Reads, "a mutation's payload needs the mutation's own check, not read")
}

func TestBuild_AKindShownOnlyThroughItsViewIsAReference(t *testing.T) {
	t.Parallel()

	in := testInputs(t, `
query ShipmentList {
  shipments { edges { node { proNumber customer { id name } } } }
}
`)
	in.Screens = screen("/shipments", []string{"ShipmentList"})

	manifest, err := pagemanifest.Build(in)
	require.NoError(t, err)
	assert.Equal(t, []string{"customer"}, manifest.Screens[0].References)
	assert.Empty(t, manifest.Screens[0].Declared)
}

func TestBuild_RefusesAnOperationNoClientFileDefines(t *testing.T) {
	t.Parallel()

	in := testInputs(t, `query ShipmentList { shipments { edges { node { id } } } }`)
	in.Screens = screen("/shipments", []string{"Gone"})

	_, err := pagemanifest.Build(in)
	require.ErrorContains(t, err, "/shipments: Gone")
}

func TestBuild_AnEndpointCarriesTheChecksItsRouteRuns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "handler.go")
	in := testInputs(t, `query ShipmentList { shipments { edges { node { id } } } }`)
	in.Screens = screen("/roles", nil,
		[2]string{"GET", "/roles/:param"},
		[2]string{"GET", "/roles/hierarchy"},
		[2]string{"GET", "/nowhere/"},
	)
	in.Routes = []routelint.Route{
		{
			Method:  "GET",
			Path:    "/api/v1/roles/:id/",
			Sources: []routelint.RouteSource{{File: file, From: 10, To: 12}},
		},
		{
			Method: "GET",
			Path:   "/api/v1/roles/hierarchy/",
			Sources: []routelint.RouteSource{
				{File: file, From: 20, To: 20},
				{File: file, From: 4, To: 4},
			},
		},
	}
	in.Checks = []permissionlint.Pair{
		{Position: filepath.ToSlash(file) + ":11", Resource: "role", Operation: "read"},
		{Position: filepath.ToSlash(file) + ":4", Resource: "role", Operation: "update"},
		{Position: filepath.ToSlash(file) + ":30", Resource: "user", Operation: "read"},
	}

	manifest, err := pagemanifest.Build(in)
	require.NoError(t, err)
	assert.Equal(t, []pageaccess.Endpoint{
		{
			Method:   "GET",
			Path:     "/roles/:param",
			Requires: []pageaccess.Requirement{{Resource: "role", Operation: "read"}},
		},
		{
			Method:   "GET",
			Path:     "/roles/hierarchy",
			Requires: []pageaccess.Requirement{{Resource: "role", Operation: "update"}},
		},
		{Method: "GET", Path: "/nowhere/", Requires: []pageaccess.Requirement{}},
	}, manifest.Screens[0].Endpoints)
}
