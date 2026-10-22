package refview_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPort_DecidesALinkedRecordsFieldsByItsView(t *testing.T) {
	t.Parallel()

	engine := &mocks.DenyResourcesPermissionEngine{Denied: []string{permission.ResourceCustomer.String()}}
	actor := permissionports.UserActor(pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	})
	port := refview.NewPort(refview.New(permission.NewRegistry()), engine)
	linked := port.Person(actor, permissionports.PersonGrants{
		Reads: func(_ context.Context, kind permission.Resource) (bool, error) {
			return kind != permission.ResourceCustomer, nil
		},
	})

	assert.False(t, linked.Reads(t.Context(), permission.ResourceCustomer))
	assert.Equal(t, permissionports.LinkedFieldVisible,
		linked.Decide(t.Context(), permission.ResourceCustomer, "name"))
	assert.Equal(t, permissionports.LinkedFieldWithheldKind,
		linked.Decide(t.Context(), permission.ResourceCustomer, "creditLimit"))
}

func TestPort_ReadsAPersonsCeilingFromTheSurfacesDetail(t *testing.T) {
	t.Parallel()

	actor := permissionports.UserActor(pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	})
	port := refview.NewPort(refview.New(permission.NewRegistry()), nil)
	asked := 0
	linked := port.Person(actor, permissionports.PersonGrants{
		Reads: func(context.Context, permission.Resource) (bool, error) { return false, nil },
		Detail: func(
			_ context.Context,
			kind permission.Resource,
		) (*permissionports.ResourcePermissionDetail, error) {
			asked++
			return &permissionports.ResourcePermissionDetail{
				Resource:       kind.String(),
				MaxSensitivity: permission.SensitivityPublic,
			}, nil
		},
	})

	assert.Equal(t, permissionports.LinkedFieldWithheldSensitivity,
		linked.Decide(t.Context(), permission.ResourceCustomer, "name"),
		"the ceiling comes from the detail the surface handed over, not the engine")
	assert.Equal(t, permissionports.LinkedFieldWithheldSensitivity,
		linked.Decide(t.Context(), permission.ResourceCustomer, "code"))
	assert.Equal(t, 1, asked, "a kind's ceiling is read once per request")
}
