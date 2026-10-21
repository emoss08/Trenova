package gqlext

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const clientOperationsDir = "../../../../../../client/packages/graphql/src/operations"

func clientOperations(t *testing.T) (ast.OperationList, ast.FragmentDefinitionList) {
	t.Helper()

	var operations ast.OperationList
	var fragments ast.FragmentDefinitionList
	err := filepath.WalkDir(clientOperationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".graphql") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, gqlErr := parser.ParseQuery(&ast.Source{Name: path, Input: string(raw)})
		if gqlErr != nil {
			return gqlErr
		}
		operations = append(operations, doc.Operations...)
		fragments = append(fragments, doc.Fragments...)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, operations, "no client operations under %s", clientOperationsDir)

	return operations, fragments
}

func linkedFields(t *testing.T) []LinkedField {
	t.Helper()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	operations, fragments := clientOperations(t)
	found, err := LinkedFields(schema, fieldsensitivity.DefaultRegistry(), operations, fragments)
	require.NoError(t, err)
	sort.Slice(found, func(i, j int) bool { return found[i].String() < found[j].String() })

	return found
}

