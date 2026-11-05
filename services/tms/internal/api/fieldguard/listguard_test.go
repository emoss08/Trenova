package fieldguard

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/stretchr/testify/assert"
)

func TestListGuard(t *testing.T) {
	t.Parallel()

	noGroups := NewListGuard(func() protectedfield.GroupSet { return 0 })
	allGroups := NewListGuard(protectedfield.AllGroups)

	assert.True(t, noGroups("workers", "profile.licenseNumber"))
	assert.True(t, noGroups("workers", "profile.dob"))
	assert.True(t, noGroups("rate_quotes", "costAmount"), "a resource-scoped path binds its table")
	assert.False(t, noGroups("shipments", "costAmount"), "and only its table")
	assert.False(t, noGroups("workers", "firstName"))
	assert.False(t, allGroups("workers", "profile.licenseNumber"))
}

func TestListGuard_AsksForTheGroupsOnce(t *testing.T) {
	t.Parallel()

	calls := 0
	guard := NewListGuard(func() protectedfield.GroupSet {
		calls++
		return 0
	})

	guard("workers", "firstName")
	assert.Zero(t, calls, "a field with no group never asks")

	guard("workers", "profile.dob")
	guard("workers", "profile.dob")
	assert.Equal(t, 1, calls)
}
