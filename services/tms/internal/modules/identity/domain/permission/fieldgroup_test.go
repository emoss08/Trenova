package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldGroupSetHoldsOnlyWhatWasAdded(t *testing.T) {
	t.Parallel()

	var set FieldGroupSet
	assert.False(t, set.Has(FieldGroupPersonal))
	assert.False(t, set.HasAll())

	set = set.With(FieldGroupMargins)
	assert.True(t, set.Has(FieldGroupMargins))
	assert.False(t, set.Has(FieldGroupPersonal))
	assert.False(t, set.HasAll())

	set = set.With(FieldGroupPersonal)
	assert.True(t, set.HasAll())
	assert.Equal(t, AllFieldGroups(), set)
}

func TestUnknownFieldGroupIsNeverHeld(t *testing.T) {
	t.Parallel()

	unknown := FieldGroup("Pay")
	assert.False(t, unknown.IsValid())
	assert.False(t, AllFieldGroups().Has(unknown))
	assert.Equal(t, AllFieldGroups(), AllFieldGroups().With(unknown))
	assert.Empty(t, unknown.Resource())
}

func TestEveryFieldGroupIsGuardedByARegisteredReadPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, group := range FieldGroups() {
		assert.True(t, group.IsValid())
		assert.Truef(t, registry.Declares(group.Resource().String(), OpRead),
			"field group %s names %q, which the registry does not declare with read",
			group, group.Resource())
	}
}
