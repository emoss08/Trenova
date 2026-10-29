package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/stretchr/testify/assert"
)

func TestFieldGroupSetHoldsOnlyWhatWasAdded(t *testing.T) {
	t.Parallel()

	var set protectedfield.GroupSet
	assert.False(t, set.Has(protectedfield.Personal))
	assert.False(t, set.HasAll())

	set = set.With(protectedfield.Margins)
	assert.True(t, set.Has(protectedfield.Margins))
	assert.False(t, set.Has(protectedfield.Personal))
	assert.False(t, set.HasAll())

	set = set.With(protectedfield.Personal)
	assert.True(t, set.HasAll())
	assert.Equal(t, protectedfield.AllGroups(), set)
}

func TestUnknownFieldGroupIsNeverHeld(t *testing.T) {
	t.Parallel()

	unknown := protectedfield.Group("Pay")
	assert.False(t, unknown.IsValid())
	assert.False(t, protectedfield.AllGroups().Has(unknown))
	assert.Equal(t, protectedfield.AllGroups(), protectedfield.AllGroups().With(unknown))
	assert.Empty(t, unknown.Resource())
}

func TestEveryFieldGroupIsGuardedByARegisteredReadPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, group := range protectedfield.Groups() {
		assert.True(t, group.IsValid())
		assert.Truef(t, registry.Declares(group.Resource().String(), permission.OpRead),
			"field group %s names %q, which the registry does not declare with read",
			group, group.Resource())
	}
}
