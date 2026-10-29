package pagemanifest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const (
	schemaDir     = "services/tms/internal/api/graphql/schema"
	operationsDir = "client/packages/graphql/src/operations"
	apiDir        = "services/tms/internal/api"
	handlersDir   = "services/tms/internal/api/handlers"
	permissionDir = "services/tms/internal/modules/identity/domain/permission"
)

// FromRepository reads every input from the repository at root.
func FromRepository(root string, registry *permissioncatalog.Registry) (*Inputs, error) {
	in := &Inputs{Registry: registry}

	raw, err := os.ReadFile(filepath.Join(root, ScreensPath))
	if err != nil {
		return nil, err
	}
	if err = sonic.Unmarshal(raw, &in.Screens); err != nil {
		return nil, fmt.Errorf("read %s: %w", ScreensPath, err)
	}
	if in.Schema, err = schemadiff.LoadDir(filepath.Join(root, schemaDir)); err != nil {
		return nil, err
	}
	if in.Operations, in.Fragments, err = ClientOperations(
		filepath.Join(root, operationsDir),
	); err != nil {
		return nil, err
	}
	if in.Routes, err = routelint.ProtectedRoutes(
		filepath.Join(root, apiDir),
		filepath.Join(root, handlersDir),
	); err != nil {
		return nil, err
	}
	consts, err := permissionlint.LoadConstants(filepath.Join(root, permissionDir))
	if err != nil {
		return nil, err
	}
	result, err := permissionlint.Analyze(consts, filepath.Join(root, handlersDir))
	if err != nil {
		return nil, err
	}
	in.Checks = result.Pairs

	return in, nil
}

// ClientOperations parses every operation and fragment under a directory of
// .graphql files.
func ClientOperations(dir string) (ast.OperationList, ast.FragmentDefinitionList, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	fsys := root.FS()

	var operations ast.OperationList
	var fragments ast.FragmentDefinitionList
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".graphql") {
			return err
		}
		raw, err := fs.ReadFile(fsys, path)
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
	if err != nil {
		return nil, nil, err
	}
	return operations, fragments, nil
}
