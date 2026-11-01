package storedmileageresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDeactivateStoredMileage_ChecksTheStoredMileageDeletePermission(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	deactivated, err := (&MutationResolver{deps}).DeactivateStoredMileage(
		ctx,
		pulid.MustNew("sm_").String(),
	)

	assert.False(t, deactivated)
	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceStoredMileage,
		permission.OpDelete,
	)
}
