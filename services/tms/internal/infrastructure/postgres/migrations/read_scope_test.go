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

const (
	readScopesUp       = "20261231009720_enforced_read_scopes.tx.up.sql"
	unreadableScopesUp = "20261231010140_read_scope_unreadable_kinds.tx.up.sql"
	linkedRowsUp       = "20261231010190_read_scope_linked_rows.tx.up.sql"
)

var (
	policyKind     = regexp.MustCompile(`trenova_rls\.read_scope\('([a-z_]+)'\)`)
	declaredScopes = regexp.MustCompile(
		`"data_scope" = '([a-z]+)' AND "resource" NOT IN \(([^)]*)\)`,
	)
)

func policyKinds(sql string) []string {
	kinds := make([]string, 0, 4)
	for _, match := range policyKind.FindAllStringSubmatch(sql, -1) {
		if !slices.Contains(kinds, match[1]) {
			kinds = append(kinds, match[1])
		}
	}
	slices.Sort(kinds)
	return kinds
}

func TestReadScopeMigration_PoliciesCoverEveryDatabaseScopedKind(t *testing.T) {
	t.Parallel()

	for _, name := range []string{readScopesUp, unreadableScopesUp, linkedRowsUp} {
		assert.Equal(
			t,
			permission.NewRegistry().DatabaseScopedResources(),
			policyKinds(readMigration(t, name)),
			"%s: each kind the registry says the database scopes needs a read_scope policy, and no other",
			name,
		)
	}
}

// The current policies narrow only by "own" and "team". A kind the principal
// cannot read is left to the application, which shows a linked row as its
// kind's reference view and refuses a root read, so no policy hides a kind
// by itself.
func TestReadScopeMigration_NoPolicyHidesAKindThePrincipalCannotRead(t *testing.T) {
	t.Parallel()

	up := readMigration(t, linkedRowsUp)
	policies := strings.Split(up, "CREATE POLICY read_scope ON ")
	require.Len(t, policies, len(permission.NewRegistry().DatabaseScopedResources())+1)

	for _, policy := range policies[1:] {
		kinds := policyKinds(policy)
		require.Len(t, kinds, 1, policy)
		assert.NotContains(t, policy, "'none'", kinds[0])
		assert.NotContains(t, policy, "NOT IN (", kinds[0])
	}

	previous := readMigration(t, unreadableScopesUp)
	assert.Equal(t,
		previous[strings.Index(previous, "DROP POLICY"):],
		readMigration(t, strings.Replace(linkedRowsUp, ".up.", ".down.", 1)),
		"the down migration restores the policies it replaced",
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
