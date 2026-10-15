package permissionlint_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/require"
)

const (
	permissionDir = "../../modules/identity/domain/permission"
	handlersDir   = "../handlers"
	resolverDir   = "../graphql/resolver"
	middlewareDir = "../middleware"
)

func TestEveryCheckedPermissionIsDeclared(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)
	require.NotEmpty(t, consts.Resources)
	require.NotEmpty(t, consts.Operations)

	pairs, err := permissionlint.Analyze(consts, handlersDir, resolverDir, middlewareDir)
	require.NoError(t, err)
	require.Greater(t, len(pairs), 500, "the analyzer stopped finding permission checks")

	registry := permission.NewRegistry()
	var undeclared []string
	for _, pair := range pairs {
		if !declares(registry, pair.Resource, permission.Operation(pair.Operation)) {
			undeclared = append(undeclared, pair.String())
		}
	}

	require.Empty(
		t,
		undeclared,
		"these checks name an operation the permission registry does not declare for the "+
			"resource, so no role can ever be granted it and the request is always refused; "+
			"declare the operation in permission/resources_*.go or check one that is declared",
	)
}

func TestAnalyzerFollowsBoundWrappers(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)

	pairs, err := permissionlint.Analyze(consts, handlersDir+"/rolehandler")
	require.NoError(t, err)

	found := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		found = append(found, pair.Resource+":"+pair.Operation)
	}

	require.Contains(t, found, "role:read")
	require.Contains(t, found, "user:unassign")
	require.Contains(t, found, "user:assign")
	require.NotContains(t, found, "role:unassign")
	require.NotContains(t, found, "role:assign")
}

func declares(registry *permission.Registry, resource string, operation permission.Operation) bool {
	if declaresOn(registry, resource, operation) {
		return true
	}
	parent := registry.GetEffectiveResource(resource)
	return parent != resource && declaresOn(registry, parent, operation)
}

func declaresOn(
	registry *permission.Registry,
	resource string,
	operation permission.Operation,
) bool {
	if slices.Contains(registry.GetOperationsForResource(resource), operation) {
		return true
	}
	return len(registry.ExpandCompositeOperation(resource, string(operation))) > 0
}
