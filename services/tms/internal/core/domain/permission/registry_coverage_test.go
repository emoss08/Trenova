package permission

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func declaredResources(t *testing.T) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "resource_gen.go", nil, 0)
	require.NoError(t, err)

	var resources []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, specOK := spec.(*ast.ValueSpec)
			if !specOK {
				continue
			}
			ident, identOK := valueSpec.Type.(*ast.Ident)
			if !identOK || ident.Name != "Resource" {
				continue
			}
			for _, value := range valueSpec.Values {
				lit, litOK := value.(*ast.BasicLit)
				if !litOK || lit.Kind != token.STRING {
					continue
				}
				resource, unquoteErr := strconv.Unquote(lit.Value)
				require.NoError(t, unquoteErr)
				resources = append(resources, resource)
			}
		}
	}

	require.NotEmpty(t, resources)
	return resources
}

func TestRegistry_CoversAllDeclaredResources(t *testing.T) {
	reg := NewRegistry()

	for _, resource := range declaredResources(t) {
		def, ok := reg.Get(resource)
		assert.True(t, ok, "resource %q is declared but not registered in the registry", resource)
		if !ok {
			continue
		}
		assert.NotEmpty(t, def.Operations, "resource %q registered without operations", resource)
		assert.NotEmpty(t, def.DisplayName, "resource %q registered without a display name", resource)
		assert.NotEmpty(t, def.Category, "resource %q registered without a category", resource)
	}
}

func TestRegistry_RegisteredResourcesAreDeclared(t *testing.T) {
	declared := make(map[string]bool)
	for _, resource := range declaredResources(t) {
		declared[resource] = true
	}

	for _, def := range NewRegistry().All() {
		assert.True(t, declared[def.Resource],
			"resource %q is registered but has no Resource constant", def.Resource)
	}
}

func TestRegistry_ParentResourcesAreRegistered(t *testing.T) {
	reg := NewRegistry()

	for _, def := range reg.All() {
		if def.ParentResource == "" {
			continue
		}
		assert.True(t, reg.HasResource(def.ParentResource),
			"resource %q declares unregistered parent %q", def.Resource, def.ParentResource)
	}
}

func TestRegistry_OperationsHaveClientBits(t *testing.T) {
	for _, def := range NewRegistry().All() {
		for _, op := range def.Operations {
			_, ok := OperationToBit[op.Operation]
			assert.True(t, ok,
				"resource %q operation %q has no client bitmask mapping", def.Resource, op.Operation)
		}
	}
}
