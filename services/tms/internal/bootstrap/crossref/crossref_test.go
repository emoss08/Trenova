package crossref

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/stretchr/testify/assert"
)

func TestVisibleRecordSources_CoverEveryKindTheDatabaseScopes(t *testing.T) {
	t.Parallel()

	sources := VisibleRecordSources()
	kinds := make([]string, 0, len(sources))
	for _, source := range sources {
		kinds = append(kinds, source.Kind)
	}
	assert.ElementsMatch(t, permissioncatalog.NewRegistry().DatabaseScopedResources(), kinds)
}
