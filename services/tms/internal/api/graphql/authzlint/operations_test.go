package authzlint

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/require"
)

const (
	handlerDir     = "../../handlers"
	permissionDir  = "../../../core/domain/permission"
	resourceSource = "resource_gen.go"
	operationFile  = "operations.go"
)

var (
	resourceConst   = regexp.MustCompile(`(?m)^\s*Resource(\w+)\s+Resource\s*=\s*"([^"]+)"`)
	operationConst  = regexp.MustCompile(`(?m)^\s*Op(\w+)\s+Operation\s*=\s*"([^"]+)"`)
	directCheck     = regexp.MustCompile(`permission\.Resource(\w+)(?:\.String\(\))?,\s*permission\.Op(\w+)\b`)
	resourceVar     = regexp.MustCompile(`\bresource\s*:?=\s*permission\.Resource(\w+)\.String\(\)`)
	resourceVarUsed = regexp.MustCompile(`RequirePermission\(\s*resource,\s*permission\.Op(\w+)\)`)
)

type permissionUse struct {
	file      string
	resource  string
	operation string
}

func constantValues(t *testing.T, file string, pattern *regexp.Regexp) map[string]string {
	t.Helper()

	source, err := os.ReadFile(filepath.Join(permissionDir, file))
	require.NoError(t, err)

	values := make(map[string]string)
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		values[match[1]] = match[2]
	}
	require.NotEmpty(t, values, "no constants read from %s", file)

	return values
}

func permissionUses(t *testing.T, dirs ...string) []permissionUse {
	t.Helper()

	var uses []permissionUse
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}

			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(source)
			for _, match := range directCheck.FindAllStringSubmatch(text, -1) {
				uses = append(uses, permissionUse{file: path, resource: match[1], operation: match[2]})
			}
			if variable := resourceVar.FindStringSubmatch(text); variable != nil {
				for _, match := range resourceVarUsed.FindAllStringSubmatch(text, -1) {
					uses = append(uses, permissionUse{
						file:      path,
						resource:  variable[1],
						operation: match[1],
					})
				}
			}

			return nil
		})
		require.NoError(t, err)
	}
	require.NotEmpty(t, uses)

	return uses
}

/*
A permission check that names an operation its resource never declares can be
passed by no one: the role editor offers only declared operations, and a system
administrator's grants are built from the same list. decideMyProposal asked for
assistant update, which the assistant resource did not have, so every approval
through GraphQL was refused, administrators included.
*/
func TestEveryPermissionCheckNamesAnOperationItsResourceDeclares(t *testing.T) {
	t.Parallel()

	resources := constantValues(t, resourceSource, resourceConst)
	operations := constantValues(t, operationFile, operationConst)
	registry := permission.NewRegistry()

	var problems []string
	for _, use := range permissionUses(t, resolverDir, handlerDir) {
		resource, ok := resources[use.resource]
		if !ok {
			continue
		}
		operation, ok := operations[use.operation]
		if !ok {
			continue
		}
		if !registry.HasResource(resource) {
			problems = append(problems, use.file+": Resource"+use.resource+" is not registered")
			continue
		}
		declared := registry.GetOperationsForResource(resource)
		if slices.Contains(declared, permission.Operation(operation)) ||
			len(registry.ExpandCompositeOperation(resource, operation)) > 0 {
			continue
		}
		problems = append(problems, use.file+": "+resource+" has no "+operation+
			" operation, so no role can pass this check")
	}

	sort.Strings(problems)
	problems = slices.Compact(problems)
	require.Empty(t, problems, "%d permission checks no role can pass:\n%s",
		len(problems), strings.Join(problems, "\n"))
}
