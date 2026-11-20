package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type suspendedOrganization struct {
	organizationID pulid.ID
}

func (s suspendedOrganization) Standing(
	_ context.Context,
	tenant dbscope.Tenant,
) (managedfeaturesports.Standing, error) {
	if tenant.OrganizationID != s.organizationID {
		return managedfeaturesports.Standing{}, nil
	}
	return managedfeaturesports.Standing{
		Suspended: true,
		ManageURL: "https://console.test/instances/1/plan",
	}, nil
}

func TestASuspendedOrganizationsPeopleAreNotSignedIn(t *testing.T) {
	t.Parallel()

	usr := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
	}
	svc := &Service{standing: suspendedOrganization{organizationID: usr.CurrentOrganizationID}}

	resp, err := svc.createLoginResponse(t.Context(), usr, loginSessionContext{})
	require.Error(t, err, "the refusal comes before a session is made")
	assert.Nil(t, resp)
	assert.True(t, errortypes.IsSuspendedError(err))
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok)
	assert.Equal(t, "https://console.test/instances/1/plan", restriction.ManageURL)
}
