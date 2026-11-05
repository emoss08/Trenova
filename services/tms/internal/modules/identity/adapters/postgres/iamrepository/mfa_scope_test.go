package iamrepository

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSecondFactorIsReadAcrossOrganizations(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	})

	assert.True(t, dbscope.IsSystem(personScope(ctx)))
}
