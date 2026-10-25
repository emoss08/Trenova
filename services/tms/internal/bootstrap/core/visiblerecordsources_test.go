package core

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestVisibleRecordSources_CoverEveryKindTheDatabaseScopes(t *testing.T) {
	t.Parallel()

	sources := newVisibleRecordSources()
	kinds := make([]string, 0, len(sources))
	for _, source := range sources {
		kinds = append(kinds, source.Kind)
	}
	assert.ElementsMatch(t, permission.NewRegistry().DatabaseScopedResources(), kinds)
}
