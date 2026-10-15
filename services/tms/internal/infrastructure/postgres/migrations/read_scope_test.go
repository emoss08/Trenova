package migrations

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const readScopesUp = "20261231009570_enforced_read_scopes.tx.up.sql"

var (
	policyKind     = regexp.MustCompile(`trenova_rls\.read_scope\('([a-z_]+)'\)`)
	declaredScopes = regexp.MustCompile(
		`"data_scope" = '([a-z]+)' AND "resource" NOT IN \(([^)]*)\)`,
	)
)

func TestReadScopeMigration_PoliciesCoverEveryDatabaseScopedKind(t *testing.T) {
	t.Parallel()

	up := readMigration(t, readScopesUp)

	kinds := make([]string, 0, 4)
	for _, match := range policyKind.FindAllStringSubmatch(up, -1) {
		if !slices.Contains(kinds, match[1]) {
			kinds = append(kinds, match[1])
		}
	}
	slices.Sort(kinds)

	assert.Equal(
		t,
		permission.NewRegistry().DatabaseScopedResources(),
		kinds,
		"each kind the registry says the database scopes needs a read_scope policy, and no other",
	)
}

func TestReadScopeMigration_KeepsExactlyTheScopesTheRegistryDeclares(t *testing.T) {
	t.Parallel()

	up := readMigration(t, readScopesUp)
	registry := permission.NewRegistry()

	matches := declaredScopes.FindAllStringSubmatch(up, -1)
	require.NotEmpty(t, matches)

	for _, match := range matches {
		listed := make([]string, 0, 8)
		for _, item := range strings.Split(match[2], ",") {
			listed = append(listed, strings.Trim(strings.TrimSpace(item), "'"))
		}
		slices.Sort(listed)

		assert.Equal(
			t,
			registry.ResourcesDeclaring(permission.DataScope(match[1])),
			listed,
			"the %s grants the migration keeps must be the ones the registry declares", match[1],
		)
	}
}
