package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_InferSchemaTypeFromGqlgenBinding(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Gqlgen: `
models:
  Parent:
    model:
      - example.com/app/internal/core/domain/parent.Parent
`,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    - virtualField
gates:
  Parent:
    child: details
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var ParentSpec TypeSpec")
	require.Contains(t, output, "FieldMap: buncolgen.ParentFieldMap")
	require.Contains(t, output, "AlwaysColumns: []string{\n\t\t\t\"id\",\n\t\t\t\"created_at\",")
	require.Contains(t, output, "Name:        \"statusText\"")
	require.Contains(t, output, "FieldMapKey: \"status\"")
	require.Contains(t, output, "Special: \"virtualField\"")
	require.Contains(t, output, "Target: &ChildSpec")
	require.Contains(t, output, "Gate:   \"details\"")
}

func TestRun_GeneratesGroupedSpecialFields(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Projected {
  id: ID!
  label: String!
  shortLabel: String!
}
`,
		Gqlgen: `
models:
  Projected:
    model:
      - example.com/app/internal/core/domain/projected.Projected
`,
		Manifest: `
specials:
  Projected:
    display:
      - label
      - shortLabel
`,
		DomainFiles: map[string]string{
			"projected/projected.go": `
package projected

type Projected struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
}
`,
		},
		BuncolgenFiles: map[string]string{
			"projected_gen.go": `
package buncolgen

var ProjectedFieldMap = map[string]string{"id": "id"}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "Name:    \"label\"")
	require.Contains(t, output, "Special: \"display\"")
	require.Contains(t, output, "Name:    \"shortLabel\"")
	require.Equal(t, 2, strings.Count(output, "Special: \"display\""))
}

func TestRun_InferNonMatchingGraphQLNameByFieldCoverage(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    - virtualField
`,
		DomainFiles: map[string]string{
			"parent/parent.go":   parentDomainFile,
			"shipment/stop.go":   stopDomainFile,
			"shipment/second.go": lowCoverageDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
			"stop_gen.go":   stopBuncolgenFile,
			"second_gen.go": lowCoverageBuncolgenFile,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var ShipmentStopSpec TypeSpec")
	require.Contains(t, output, "FieldMap: buncolgen.StopFieldMap")
	require.Contains(t, output, "Name:        \"locationId\"")
	require.Contains(t, output, "FieldMapKey: \"locationId\"")
}

func TestRun_InfersBunRelationAndRelationColumnKey(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    - virtualField
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "Name:        \"child\"")
	require.Contains(t, output, "FieldMapKey: \"childId\"")
	require.Contains(t, output, "Relation: &RelationSpec")
	require.Contains(t, output, "Target: &ChildSpec")
}

func TestRun_AutoSkipsWrappersAndDTOs(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    - virtualField
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	output := runFixture(t, fixture)

	require.NotContains(t, output, "QuerySpec")
	require.NotContains(t, output, "PageInfoSpec")
	require.NotContains(t, output, "ParentEdgeSpec")
	require.NotContains(t, output, "ParentConnectionSpec")
	require.NotContains(t, output, "ParentSummarySpec")
}

func TestRun_FailsForAmbiguousModelMatch(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Ambiguous {
  id: ID!
  name: String!
}
`,
		DomainFiles: map[string]string{
			"alpha/alpha.go": `
package alpha

type Alpha struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
	Name string ` + "`json:\"name\" bun:\"name\"`" + `
}
`,
			"bravo/bravo.go": `
package bravo

type Bravo struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
	Name string ` + "`json:\"name\" bun:\"name\"`" + `
}
`,
		},
		BuncolgenFiles: map[string]string{
			"alpha_gen.go": `
package buncolgen

var AlphaFieldMap = map[string]string{"id": "id", "name": "name"}
`,
			"bravo_gen.go": `
package buncolgen

var BravoFieldMap = map[string]string{"id": "id", "name": "name"}
`,
		},
	})

	err := run(fixture.options())

	require.ErrorContains(t, err, "ambiguous Go model matches")
	require.ErrorContains(t, err, "add modelOverrides.Ambiguous")
}

func TestRun_FailsForOldVirtualMapShape(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    virtualField: virtualFlag
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	err := run(fixture.options())

	require.ErrorContains(t, err, "decoding manifest")
	require.ErrorContains(t, err, "cannot unmarshal")
}

func TestRun_FailsForCollidingVirtualAndSpecialFields(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
virtuals:
  Parent:
    - virtualField
specials:
  Parent:
    runtimeFlag:
      - virtualField
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	err := run(fixture.options())

	require.ErrorContains(t, err, `type "Parent" field "virtualField" cannot be declared in both virtuals and specials`)
}

func TestRun_FailsForFieldUnderMultipleSpecialKeys(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: testSchema,
		Manifest: `
aliases:
  Parent:
    statusText: status
specials:
  Parent:
    runtimeFlag:
      - virtualField
    otherRuntimeFlag:
      - virtualField
`,
		DomainFiles: map[string]string{
			"parent/parent.go": parentDomainFile,
		},
		BuncolgenFiles: map[string]string{
			"parent_gen.go": parentBuncolgenFile,
		},
	})

	err := run(fixture.options())

	require.ErrorContains(t, err, `type "Parent" field "virtualField" is listed under multiple special keys`)
}

func TestRun_FailsForUnknownVirtualOrSpecialField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		manifest      string
		expectedError string
	}{
		{
			name: "virtual field",
			manifest: `
virtuals:
  Parent:
    - missingField
`,
			expectedError: `virtuals.Parent field "missingField" does not exist in schema`,
		},
		{
			name: "special field",
			manifest: `
specials:
  Parent:
    runtimeFlag:
      - missingField
`,
			expectedError: `specials.Parent.runtimeFlag field "missingField" does not exist in schema`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newGeneratorFixture(t, generatorFixture{
				Schema:   testSchema,
				Manifest: tt.manifest,
				DomainFiles: map[string]string{
					"parent/parent.go": parentDomainFile,
				},
				BuncolgenFiles: map[string]string{
					"parent_gen.go": parentBuncolgenFile,
				},
			})

			err := run(fixture.options())

			require.ErrorContains(t, err, tt.expectedError)
		})
	}
}

func TestRun_FailsForUnresolvedComputedField(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Projected {
  id: ID!
  displayName: String!
}
`,
		DomainFiles: map[string]string{
			"projected/projected.go": `
package projected

type Projected struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
}
`,
		},
		BuncolgenFiles: map[string]string{
			"projected_gen.go": `
package buncolgen

var ProjectedFieldMap = map[string]string{"id": "id"}
`,
		},
	})

	err := run(fixture.options())

	require.ErrorContains(t, err, `type "Projected" field "displayName" is neither`)
	require.ErrorContains(t, err, "virtuals.Projected list entry")
	require.ErrorContains(t, err, "specials.Projected.<specialKey> list entry")
	require.ErrorContains(t, err, "modelOverrides.Projected")
}

type generatorFixture struct {
	Schema         string
	Gqlgen         string
	Manifest       string
	DomainFiles    map[string]string
	BuncolgenFiles map[string]string
}

type fixturePaths struct {
	root         string
	schemaDir    string
	manifestPath string
	gqlgenPath   string
	domainDir    string
	buncolgenDir string
	goModPath    string
	outputPath   string
}

func (f fixturePaths) options() generatorOptions {
	return generatorOptions{
		ManifestPath: f.manifestPath,
		SchemaDir:    f.schemaDir,
		OutputPath:   f.outputPath,
		GqlgenPath:   f.gqlgenPath,
		DomainDir:    f.domainDir,
		BuncolgenDir: f.buncolgenDir,
		GoModPath:    f.goModPath,
	}
}

func newGeneratorFixture(t *testing.T, fixture generatorFixture) fixturePaths {
	t.Helper()

	root := t.TempDir()
	paths := fixturePaths{
		root:         root,
		schemaDir:    filepath.Join(root, "internal/api/graphql/schema"),
		manifestPath: filepath.Join(root, "internal/api/graphql/projection/projection.yml"),
		gqlgenPath:   filepath.Join(root, "gqlgen.yml"),
		domainDir:    filepath.Join(root, "internal/core/domain"),
		buncolgenDir: filepath.Join(root, "pkg/buncolgen"),
		goModPath:    filepath.Join(root, "go.mod"),
		outputPath:   filepath.Join(root, "internal/api/graphql/projection/specs_gen.go"),
	}

	writeFile(t, paths.goModPath, "module example.com/app\n\ngo 1.25\n")
	writeFile(t, filepath.Join(paths.schemaDir, "schema.graphqls"), strings.TrimSpace(fixture.Schema))
	writeFile(t, paths.gqlgenPath, strings.TrimSpace(fixture.Gqlgen))
	writeFile(t, paths.manifestPath, strings.TrimSpace(fixture.Manifest))
	for name, content := range fixture.DomainFiles {
		writeFile(t, filepath.Join(paths.domainDir, name), strings.TrimSpace(content))
	}
	for name, content := range fixture.BuncolgenFiles {
		writeFile(t, filepath.Join(paths.buncolgenDir, name), strings.TrimSpace(content))
	}

	return paths
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o644))
}

func runFixture(t *testing.T, fixture fixturePaths) string {
	t.Helper()

	require.NoError(t, run(fixture.options()))

	output, err := os.ReadFile(fixture.outputPath)
	require.NoError(t, err)

	return string(output)
}

const testSchema = `
type Parent {
  id: ID!
  createdAt: Int!
  childId: ID
  statusText: String!
  virtualField: String
  child: Child
}

type Child {
  id: ID!
  createdAt: Int!
  name: String!
}

type ShipmentStop {
  id: ID
  businessUnitId: ID!
  organizationId: ID!
  locationId: ID!
  status: String!
  createdAt: Int!
  location: Child
}

type ParentEdge {
  node: Parent!
  cursor: String!
}

type ParentConnection {
  edges: [ParentEdge!]!
  pageInfo: PageInfo!
}

type PageInfo {
  hasNextPage: Boolean!
  endCursor: String
}

type ParentSummary {
  id: ID!
  name: String!
}

type Query {
  parent: Parent
}
`

const parentDomainFile = `
package parent

type Parent struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" bun:\"created_at\"`" + `
	ChildID string ` + "`json:\"childId\" bun:\"child_id\"`" + `
	Status string ` + "`json:\"status\" bun:\"status\"`" + `
	VirtualField string ` + "`json:\"virtualField\" bun:\"-\"`" + `
	Child *Child ` + "`json:\"child,omitempty\" bun:\"rel:belongs-to,join:child_id=id\"`" + `
}

type Child struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" bun:\"created_at\"`" + `
	Name string ` + "`json:\"name\" bun:\"name\"`" + `
}
`

const stopDomainFile = `
package shipment

type Stop struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
	BusinessUnitID string ` + "`json:\"businessUnitId\" bun:\"business_unit_id\"`" + `
	OrganizationID string ` + "`json:\"organizationId\" bun:\"organization_id\"`" + `
	LocationID string ` + "`json:\"locationId\" bun:\"location_id\"`" + `
	Status string ` + "`json:\"status\" bun:\"status\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" bun:\"created_at\"`" + `
	Location any ` + "`json:\"location,omitempty\" bun:\"rel:belongs-to,join:location_id=id\"`" + `
}
`

const lowCoverageDomainFile = `
package shipment

type Second struct {
	ID string ` + "`json:\"id\" bun:\"id\"`" + `
}
`

const parentBuncolgenFile = `
package buncolgen

var ParentFieldMap = map[string]string{
	"id": "id",
	"createdAt": "created_at",
	"childId": "child_id",
	"status": "status",
}

var ParentRelations = struct {
	Child string
}{
	Child: "Child",
}

var ChildFieldMap = map[string]string{
	"id": "id",
	"createdAt": "created_at",
	"name": "name",
}
`

const stopBuncolgenFile = `
package buncolgen

var StopFieldMap = map[string]string{
	"id": "id",
	"businessUnitId": "business_unit_id",
	"organizationId": "organization_id",
	"locationId": "location_id",
	"status": "status",
	"createdAt": "created_at",
}

var StopRelations = struct {
	Location string
}{
	Location: "Location",
}
`

const lowCoverageBuncolgenFile = `
package buncolgen

var SecondFieldMap = map[string]string{
	"id": "id",
}
`
