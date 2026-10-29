package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/routeaccess"
	"github.com/stretchr/testify/assert"
)

func TestRouteRegistry_EveryRequirementIsDeclared(t *testing.T) {
	t.Parallel()

	routes := routeaccess.NewRouteRegistry()
	registry := NewRegistry()

	var undeclared []string
	for _, route := range routes.All() {
		for _, req := range route.Requirements {
			if !registry.Declares(req.Resource.String(), req.Operation) {
				undeclared = append(
					undeclared,
					route.Path+" needs "+req.Resource.String()+":"+string(req.Operation),
				)
			}
		}
	}

	assert.Empty(
		t,
		undeclared,
		"a page that needs an undeclared operation is hidden from everyone",
	)
}
