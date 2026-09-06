package querycost

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	schemaGlob        = "../schema/*.graphqls"
	persistedManifest = "../persisted-documents.json"
)

func loadSchema(t *testing.T) *ast.Schema {
	t.Helper()

	paths, err := filepath.Glob(schemaGlob)
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no schema files found")

	sources := make([]*ast.Source, 0, len(paths))
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		sources = append(sources, &ast.Source{Name: path, Input: string(content)})
	}

	schema, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)

	return schema
}

func loadPersistedDocuments(t *testing.T) map[string]string {
	t.Helper()

	raw, err := os.ReadFile(persistedManifest)
	require.NoError(t, err)

	documents := make(map[string]string)
	require.NoError(t, sonic.Unmarshal(raw, &documents))
	require.NotEmpty(t, documents)

	return documents
}

type costWalker struct {
	schema *ast.Schema
	index  *Index
}

func (w costWalker) selectionSetCost(set ast.SelectionSet) int {
	total := 0
	for _, selection := range set {
		switch s := selection.(type) {
		case *ast.Field:
			if s.Definition == nil || s.ObjectDefinition == nil {
				continue
			}
			fieldDefinition := w.schema.Types[s.Definition.Type.Name()]
			if fieldDefinition == nil || fieldDefinition.Name == "__Schema" {
				continue
			}

			childCost := 0
			switch fieldDefinition.Kind {
			case ast.Object, ast.Interface, ast.Union:
				childCost = w.selectionSetCost(s.SelectionSet)
			}

			total += w.fieldCost(s.ObjectDefinition, s.Name, childCost)
		case *ast.FragmentSpread:
			if s.Definition != nil {
				total += w.selectionSetCost(s.Definition.SelectionSet)
			}
		case *ast.InlineFragment:
			total += w.selectionSetCost(s.SelectionSet)
		}
	}

	return total
}

func (w costWalker) fieldCost(def *ast.Definition, field string, childCost int) int {
	if def.Kind == ast.Interface {
		worst := 0
		for _, impl := range w.schema.GetPossibleTypes(def) {
			if cost := w.objectFieldCost(impl.Name, field, childCost); cost > worst {
				worst = cost
			}
		}
		return worst
	}

	return w.objectFieldCost(def.Name, field, childCost)
}

func (w costWalker) objectFieldCost(typeName, field string, childCost int) int {
	worstCaseArgs := map[string]any{connectionPageKey: pagination.MaxLimit}
	if custom, ok := w.index.Complexity(typeName, field, childCost, worstCaseArgs); ok &&
		custom >= 1 {
		return custom
	}

	return 1 + childCost
}

type documentCost struct {
	name  string
	cost  int
	depth int
}

func measurePersistedDocuments(t *testing.T) []documentCost {
	t.Helper()

	schema := loadSchema(t)
	index := NewIndex(schema)
	walker := costWalker{schema: schema, index: index}
	documents := loadPersistedDocuments(t)

	measured := make([]documentCost, 0, len(documents))
	for hash, text := range documents {
		doc, errs := gqlparser.LoadQuery(schema, text)
		require.Empty(t, errs, "persisted document %s failed to validate", hash)

		for _, op := range doc.Operations {
			name := op.Name
			if name == "" {
				name = hash
			}
			measured = append(measured, documentCost{
				name:  name,
				cost:  walker.selectionSetCost(op.SelectionSet),
				depth: Depth(op),
			})
		}
	}

	sort.Slice(measured, func(i, j int) bool { return measured[i].cost > measured[j].cost })

	return measured
}

func TestPersistedDocumentBudget(t *testing.T) {
	t.Parallel()

	measured := measurePersistedDocuments(t)
	require.NotEmpty(t, measured)

	byDepth := make([]documentCost, len(measured))
	copy(byDepth, measured)
	sort.Slice(byDepth, func(i, j int) bool { return byDepth[i].depth > byDepth[j].depth })

	t.Logf("measured %d operations at worst-case page size %d", len(measured), pagination.MaxLimit)
	t.Logf("most expensive:")
	for _, m := range measured[:min(10, len(measured))] {
		t.Logf("  cost=%-9d depth=%-3d %s", m.cost, m.depth, m.name)
	}
	t.Logf("deepest:")
	for _, m := range byDepth[:min(10, len(byDepth))] {
		t.Logf("  depth=%-3d cost=%-9d %s", m.depth, m.cost, m.name)
	}

	require.LessOrEqual(t, measured[0].cost, MaxOperationCost,
		"persisted operation %q costs %d, above the %d limit; it would be rejected in production",
		measured[0].name, measured[0].cost, MaxOperationCost)

	require.LessOrEqual(t, byDepth[0].depth, MaxOperationDepth,
		"persisted operation %q has depth %d, above the %d limit; it would be rejected in production",
		byDepth[0].name, byDepth[0].depth, MaxOperationDepth)
}
