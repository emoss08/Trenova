package permission_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

// A reference view is what every reader of a linking record sees, so nothing
// in it may rank above Internal.
func TestReferenceViewsAreAtMostInternal(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	var sensitive []string
	for _, def := range registry.All() {
		for _, field := range registry.ReferenceView(def.Resource) {
			level := registry.GetFieldSensitivity(def.Resource, field)
			if !permission.SensitivityInternal.CanAccess(level) {
				sensitive = append(sensitive, def.Resource+"."+field+" is "+string(level))
			}
		}
	}
	slices.Sort(sensitive)
	assert.Empty(t, sensitive,
		"reference view fields ranked above Internal. A view is shown to everyone who can read a "+
			"record linking to the kind, so take the field out of ReferenceView, or, if it is not "+
			"sensitive, lower its FieldSensitivities entry:\n  %s", strings.Join(sensitive, "\n  "))
}

func TestReferenceViewsNameEachFieldOnce(t *testing.T) {
	t.Parallel()

	for _, def := range permission.NewRegistry().All() {
		seen := make(map[string]struct{}, len(def.ReferenceView))
		for _, field := range def.ReferenceView {
			_, again := seen[field]
			assert.False(t, again, "%s names %s twice in its reference view", def.Resource, field)
			seen[field] = struct{}{}
		}
	}
}

func TestReferenceViewIncludesTheBookkeepingFields(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	view := registry.ReferenceView(permission.ResourceCustomer.String())
	for _, field := range []string{"id", "organizationId", "businessUnitId", "version", "createdAt", "updatedAt", "name"} {
		assert.Contains(t, view, field)
	}
	assert.NotContains(t, view, "creditLimit")
	assert.True(t, registry.InReferenceView(permission.ResourceWorker.String(), "wholeName"))
	assert.False(t, registry.InReferenceView(permission.ResourceWorker.String(), "phoneNumber"),
		"a driver's phone needs worker read")
	assert.Equal(t, permission.SensitivityInternal,
		registry.GetFieldSensitivity(permission.ResourceUser.String(), "name"),
		"a view field is Internal unless its resource ranks it otherwise")
}

func TestPartOfFollowsTheParentChain(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	assert.True(t, registry.PartOf(permission.ResourceShipmentStop.String(), permission.ResourceShipment.String()))
	assert.True(t, registry.PartOf(permission.ResourceShipmentStop.String(), permission.ResourceShipmentMove.String()))
	assert.False(t, registry.PartOf(permission.ResourceShipment.String(), permission.ResourceShipmentStop.String()),
		"a family's root is never part of its own child")
	assert.False(t, registry.PartOf(permission.ResourceWorker.String(), permission.ResourceTractor.String()))
}
