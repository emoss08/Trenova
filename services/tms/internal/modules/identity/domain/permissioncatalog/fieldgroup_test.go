package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestFieldGroupSetHoldsOnlyWhatWasAdded(t *testing.T) {
	t.Parallel()

	var set permission.FieldGroupSet
	assert.False(t, set.Has(permission.FieldGroupPersonal))
	assert.False(t, set.HasAll())

	set = set.With(permission.FieldGroupMargins)
	assert.True(t, set.Has(permission.FieldGroupMargins))
	assert.False(t, set.Has(permission.FieldGroupPersonal))
	assert.False(t, set.HasAll())

	set = set.With(permission.FieldGroupPersonal)
	assert.True(t, set.HasAll())
	assert.Equal(t, permission.AllFieldGroups(), set)
}

func TestUnknownFieldGroupIsNeverHeld(t *testing.T) {
	t.Parallel()

	unknown := permission.FieldGroup("Pay")
	assert.False(t, unknown.IsValid())
	assert.False(t, permission.AllFieldGroups().Has(unknown))
	assert.Equal(t, permission.AllFieldGroups(), permission.AllFieldGroups().With(unknown))
	assert.Empty(t, unknown.Resource())
}

func TestEveryFieldGroupIsGuardedByARegisteredReadPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, group := range permission.FieldGroups() {
		assert.True(t, group.IsValid())
		assert.Truef(t, registry.Declares(group.Resource().String(), permission.OpRead),
			"field group %s names %q, which the registry does not declare with read",
			group, group.Resource())
	}
}
