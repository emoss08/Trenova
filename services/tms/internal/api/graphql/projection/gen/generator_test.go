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
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var ParentSpec TypeSpec")
	require.NotContains(t, output, "import")
	require.Contains(t, output, "AlwaysColumns: []string{\n\t\t\t\"id\",\n\t\t\t\"created_at\",")
	require.Contains(t, output, "Name:   \"statusText\"")
	require.Contains(t, output, "Column: \"status\"")
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
	keel.Table ` + "`keel:\"projecteds\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "Name:    \"label\"")
	require.Contains(t, output, "Special: \"display\"")
	require.Contains(t, output, "Name:    \"shortLabel\"")
	require.Equal(t, 2, strings.Count(output, "Special: \"display\""))
}

func TestRun_UnskipsWrapperSuffixTypeWithModelBinding(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type WidgetConnection {
  id: ID!
  name: String!
}

type Query {
  widget: WidgetConnection
}
`,
		Gqlgen: `
models:
  WidgetConnection:
    model:
      - example.com/app/internal/core/domain/widget.Connection
`,
		DomainFiles: map[string]string{
			"widget/widget.go": `
package widget

import "github.com/emoss08/keel"

type Connection struct {
	keel.Table ` + "`keel:\"widget_connections\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	Name string ` + "`json:\"name\" keel:\"name\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var WidgetConnectionSpec TypeSpec")
	require.Contains(t, output, "Column: \"name\"")
}

func TestRun_ReadsColumnsFromTheKeelTags(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Mailbox {
  id: ID!
  senderName: String!
}

type Query {
  mailbox: Mailbox
}
`,
		Gqlgen: `
models:
  Mailbox:
    model:
      - example.com/app/internal/core/domain/mailbox.Mailbox
`,
		DomainFiles: map[string]string{
			"mailbox/mailbox.go": `
package mailbox

import "github.com/emoss08/keel"

type Mailbox struct {
	keel.Table ` + "`keel:\"mailboxes\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	SenderName string ` + "`json:\"senderName\" keel:\"from_name\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var MailboxSpec TypeSpec")
	require.Contains(t, output, "Column: \"from_name\"")
	require.NotContains(t, output, "FieldMap")
	require.NotContains(t, output, "import")
}

func TestRun_SkipsBoundStructThatIsNotAKeelTable(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Payload {
  id: ID!
}

type Query {
  payload: Payload
}
`,
		Gqlgen: `
models:
  Payload:
    model:
      - example.com/app/internal/core/domain/payload.Payload
`,
		DomainFiles: map[string]string{
			"payload/payload.go": `
package payload

type Payload struct {
	ID string ` + "`json:\"id\" keel:\"id\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.NotContains(t, output, "var PayloadSpec TypeSpec")
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
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var ShipmentStopSpec TypeSpec")
	require.Contains(t, output, "Name:   \"locationId\",\n\t\t\t\tColumn: \"location_id\"")
}

func TestRun_IgnoresAStructThatIsNotAKeelTable(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Item {
  id: ID!
  title: String!
}

type ChecklistRow {
  key: String!
  label: String!
  count: Int!
}

type Query {
  item: Item
}
`,
		DomainFiles: map[string]string{
			"watchtower/item.go": `
package watchtower

import "github.com/emoss08/keel"

type Item struct {
	keel.Table ` + "`keel:\"watchtower_items\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	Title string ` + "`json:\"title\" keel:\"title\"`" + `
}
`,
			"checklist/item.go": `
package checklist

type Item struct {
	Key string ` + "`json:\"key\"`" + `
	Label string ` + "`json:\"label\"`" + `
	Count int ` + "`json:\"count\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "var ItemSpec TypeSpec")
	require.Contains(t, output, "Column: \"title\"")
	require.NotContains(t, output, "ChecklistRowSpec")
}

func TestRun_InfersKeelRelationAndItsColumn(t *testing.T) {
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
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "Name:   \"child\",\n\t\t\t\tColumn: \"child_id\"")
	require.Contains(t, output, "Relation: &RelationSpec")
	require.Contains(t, output, "Target: &ChildSpec")
}

func TestRun_InfersAnUntaggedRelationThroughItsIDColumn(t *testing.T) {
	t.Parallel()

	fixture := newGeneratorFixture(t, generatorFixture{
		Schema: `
type Owner {
  id: ID!
  name: String!
}

type Pet {
  id: ID!
  ownerId: ID!
  owner: Owner
  friends: [Owner!]!
}
`,
		DomainFiles: map[string]string{
			"pet/pet.go": `
package pet

type Owner struct {
	keel.Table ` + "`keel:\"owners\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	Name string ` + "`json:\"name\" keel:\"name\"`" + `
}

type Pet struct {
	keel.Table ` + "`keel:\"pets\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	OwnerID string ` + "`json:\"ownerId\" keel:\"owner_id\"`" + `
	Owner *Owner ` + "`json:\"owner,omitempty\"`" + `
	Friends []*Owner ` + "`json:\"friends\"`" + `
}
`,
		},
	})

	output := runFixture(t, fixture)

	require.Contains(t, output, "Name:   \"owner\",\n\t\t\t\tColumn: \"owner_id\"")
	require.Contains(t, output, "Name: \"friends\",\n\t\t\t\tRelation: &RelationSpec")
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
	keel.Table ` + "`keel:\"alphas\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	Name string ` + "`json:\"name\" keel:\"name\"`" + `
}
`,
			"bravo/bravo.go": `
package bravo

type Bravo struct {
	keel.Table ` + "`keel:\"bravos\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	Name string ` + "`json:\"name\" keel:\"name\"`" + `
}
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
	})

	err := run(fixture.options())

	require.ErrorContains(
		t,
		err,
		`type "Parent" field "virtualField" cannot be declared in both virtuals and specials`,
	)
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
	})

	err := run(fixture.options())

	require.ErrorContains(
		t,
		err,
		`type "Parent" field "virtualField" is listed under multiple special keys`,
	)
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
	keel.Table ` + "`keel:\"projecteds\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
}
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
	Schema      string
	Gqlgen      string
	Manifest    string
	DomainFiles map[string]string
}

type fixturePaths struct {
	root         string
	schemaDir    string
	manifestPath string
	gqlgenPath   string
	domainDir    string
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
		goModPath:    filepath.Join(root, "go.mod"),
		outputPath:   filepath.Join(root, "internal/api/graphql/projection/specs_gen.go"),
	}

	writeFile(t, paths.goModPath, "module example.com/app\n\ngo 1.25\n")
	writeFile(
		t,
		filepath.Join(paths.schemaDir, "schema.graphqls"),
		strings.TrimSpace(fixture.Schema),
	)
	writeFile(t, paths.gqlgenPath, strings.TrimSpace(fixture.Gqlgen))
	writeFile(t, paths.manifestPath, strings.TrimSpace(fixture.Manifest))
	for name, content := range fixture.DomainFiles {
		writeFile(t, filepath.Join(paths.domainDir, name), strings.TrimSpace(content))
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
	keel.Table ` + "`keel:\"parents\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" keel:\"created_at\"`" + `
	ChildID string ` + "`json:\"childId\" keel:\"child_id\"`" + `
	Status string ` + "`json:\"status\" keel:\"status\"`" + `
	VirtualField string ` + "`json:\"virtualField\" keel:\"-\"`" + `
	Child *Child ` + "`json:\"child,omitempty\" keel:\"belongs:child_id\"`" + `
}

type Child struct {
	keel.Table ` + "`keel:\"children\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" keel:\"created_at\"`" + `
	Name string ` + "`json:\"name\" keel:\"name\"`" + `
}
`

const stopDomainFile = `
package shipment

type Stop struct {
	keel.Table ` + "`keel:\"stops\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
	BusinessUnitID string ` + "`json:\"businessUnitId\" keel:\"business_unit_id\"`" + `
	OrganizationID string ` + "`json:\"organizationId\" keel:\"organization_id\"`" + `
	LocationID string ` + "`json:\"locationId\" keel:\"location_id\"`" + `
	Status string ` + "`json:\"status\" keel:\"status\"`" + `
	CreatedAt int64 ` + "`json:\"createdAt\" keel:\"created_at\"`" + `
	Location any ` + "`json:\"location,omitempty\" keel:\"belongs:location_id\"`" + `
}
`

const lowCoverageDomainFile = `
package shipment

type Second struct {
	keel.Table ` + "`keel:\"seconds\"`" + `

	ID string ` + "`json:\"id\" keel:\"id\"`" + `
}
`
