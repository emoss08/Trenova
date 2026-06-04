package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestGenerate_ValidManifestProducesStableOutput(t *testing.T) {
	t.Parallel()

	schema := mustLoadTestSchema(t)
	output, err := generate(testManifest(), schema, testFieldMaps())
	require.NoError(t, err)

	got := string(output)
	require.Contains(t, got, "var ParentSpec = TypeSpec")
	require.Contains(t, got, "FieldMap: buncolgen.ParentFieldMap")
	require.Contains(t, got, "AlwaysColumns: []string{\n\t\t\"id\",")
	require.Contains(t, got, "Name:        \"statusText\"")
	require.Contains(t, got, "FieldMapKey: \"status\"")
	require.Contains(t, got, "Special: \"virtualFlag\"")
	require.Contains(t, got, "Target: &ChildSpec")
	require.Contains(t, got, "Gate:   \"details\"")
}

func TestGenerate_FailsForMissingSchemaField(t *testing.T) {
	t.Parallel()

	data := testManifest()
	parent := data.Types["Parent"]
	parent.Aliases["missing"] = "status"
	data.Types["Parent"] = parent

	_, err := generate(data, mustLoadTestSchema(t), testFieldMaps())
	require.ErrorContains(t, err, "alias field \"missing\" does not exist")
}

func TestGenerate_FailsForMissingFieldMapKey(t *testing.T) {
	t.Parallel()

	data := testManifest()
	parent := data.Types["Parent"]
	parent.Aliases["statusText"] = "missing"
	data.Types["Parent"] = parent

	_, err := generate(data, mustLoadTestSchema(t), testFieldMaps())
	require.ErrorContains(t, err, "missing field-map key \"missing\"")
}

func TestGenerate_FailsForBadRelationTarget(t *testing.T) {
	t.Parallel()

	data := testManifest()
	parent := data.Types["Parent"]
	parent.Relations["child"] = relationManifest{Target: "Missing"}
	data.Types["Parent"] = parent

	_, err := generate(data, mustLoadTestSchema(t), testFieldMaps())
	require.ErrorContains(t, err, "references unknown target \"Missing\"")
}

func TestGenerate_FailsForUnknownSkipType(t *testing.T) {
	t.Parallel()

	data := testManifest()
	data.Skip = append(data.Skip, "Missing")

	_, err := generate(data, mustLoadTestSchema(t), testFieldMaps())
	require.ErrorContains(t, err, "skip \"Missing\"")
}

func TestRun_WritesGeneratedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	require.NoError(t, os.Mkdir(schemaDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(schemaDir, "schema.graphqls"),
		[]byte(testSchema),
		0o644,
	))
	manifestPath := filepath.Join(dir, "projection.yml")
	require.NoError(t, os.WriteFile(
		manifestPath,
		[]byte(strings.TrimSpace(testManifestYAML)),
		0o644,
	))
	outputPath := filepath.Join(dir, "specs_gen.go")

	err := run(generatorOptions{
		ManifestPath: manifestPath,
		SchemaDir:    schemaDir,
		OutputPath:   outputPath,
		FieldMaps:    testFieldMaps(),
	})
	require.NoError(t, err)

	output, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Contains(t, string(output), "var ParentSpec = TypeSpec")
}

func mustLoadTestSchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{
		Name:  "schema.graphqls",
		Input: testSchema,
	})
	require.NoError(t, err)

	return schema
}

func testManifest() manifest {
	return manifest{
		Types: map[string]typeManifest{
			"Parent": {
				Always:   []string{"id"},
				Aliases:  map[string]string{"statusText": "status"},
				Virtuals: map[string]string{"virtualField": "virtualFlag"},
				Relations: map[string]relationManifest{
					"child": {
						Target: "Child",
						Gate:   "details",
					},
				},
			},
			"Child": {
				Always: []string{"id"},
			},
		},
		Skip: []string{"Query"},
	}
}

func testFieldMaps() map[string]map[string]string {
	return map[string]map[string]string{
		"Parent": {
			"id":     "id",
			"status": "status",
		},
		"Child": {
			"id":   "id",
			"name": "name",
		},
	}
}

const testSchema = `
type Parent {
  id: ID!
  statusText: String!
  virtualField: String
  child: Child
}

type Child {
  id: ID!
  name: String!
}

type Query {
  parent: Parent
}
`

const testManifestYAML = `
types:
  Parent:
    always:
      - id
    aliases:
      statusText: status
    virtuals:
      virtualField: virtualFlag
    relations:
      child:
        target: Child
        gate: details
  Child:
    always:
      - id
skip:
  - Query
`
