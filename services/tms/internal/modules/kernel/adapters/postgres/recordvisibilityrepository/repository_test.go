package recordvisibilityrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedTables_CoverEveryKindTheDatabaseScopes(t *testing.T) {
	t.Parallel()

	kinds := make([]string, 0, len(scopedTables))
	for kind := range scopedTables {
		kinds = append(kinds, kind)
	}
	assert.ElementsMatch(t, permission.NewRegistry().DatabaseScopedResources(), kinds)
}

func TestVisibleIDs_RefusesAKindWithoutAReadScope(t *testing.T) {
	t.Parallel()

	repo := New(Params{})
	assert.False(t, repo.Scoped("customer"))
	_, err := repo.VisibleIDs(t.Context(), &kernelports.ListVisibleRecordsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Kind:       "customer",
		IDs:        []pulid.ID{pulid.MustNew("cus_")},
	})
	require.ErrorIs(t, err, ErrUnscopedKind)
}
