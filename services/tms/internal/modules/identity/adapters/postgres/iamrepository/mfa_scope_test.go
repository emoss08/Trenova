package iamrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSecondFactorIsFoundFromEveryOrganization(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	})
	userID := pulid.MustNew("usr_")

	factor, _ := keeltest.Render(t, userTOTP(ctx, userID))
	assert.NotContains(t, factor, "organization_id")

	codes, _ := keeltest.Render(t, userRecoveryCodes(ctx, userID))
	assert.NotContains(t, codes, "organization_id")
}
