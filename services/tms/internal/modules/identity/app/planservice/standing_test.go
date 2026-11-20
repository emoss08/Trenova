package planservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type standing struct {
	suspended map[string]bool
	err       error
}

func (s standing) Standing(
	_ context.Context,
	tenant dbscope.Tenant,
) (managedfeaturesports.Standing, error) {
	if s.err != nil {
		return managedfeaturesports.Standing{}, s.err
	}
	if s.suspended[tenant.OrganizationID.String()] {
		return managedfeaturesports.Standing{
			Suspended: true,
			ManageURL: "https://console.test/plan",
		}, nil
	}
	return managedfeaturesports.Standing{}, nil
}

func TestASuspendedOrganizationIsRefusedEveryRequestWithItsConsoleAddress(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	admission := &planservice.Admission{
		Plans:    planservice.NewUnlimited(),
		Features: allFeatures(),
		Standing: standing{suspended: map[string]bool{tenant.OrgID.String(): true}},
	}

	for _, write := range []bool{false, true} {
		err := admission.Admit(t.Context(), &planservice.AdmissionRequest{
			TenantInfo: tenant,
			Write:      write,
			ExpiredOK:  true,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsSuspendedError(err))
		restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
		require.True(t, ok)
		assert.Equal(t, errortypes.PlanRestrictionReasonSuspended, restriction.Reason)
		assert.Equal(t, "https://console.test/plan", restriction.Params()["manageUrl"])
	}
}

func TestAnActiveOrganizationIsAdmittedAsBefore(t *testing.T) {
	t.Parallel()

	admission := &planservice.Admission{
		Plans:    planservice.NewUnlimited(),
		Features: allFeatures(),
		Standing: standing{suspended: map[string]bool{}},
	}
	require.NoError(t, admission.Admit(t.Context(), &planservice.AdmissionRequest{
		TenantInfo: plantest.Tenant(),
		Write:      true,
	}))
	require.NoError(t, admission.Admit(t.Context(), &planservice.AdmissionRequest{
		TenantInfo: pagination.TenantInfo{},
		Write:      true,
	}), "a request with no tenant is not asked about")
}

func TestStandingThatCannotBeReadRefusesTheRequest(t *testing.T) {
	t.Parallel()

	failure := errors.New("plan unreadable")
	admission := &planservice.Admission{
		Plans:    planservice.NewUnlimited(),
		Features: allFeatures(),
		Standing: standing{err: failure},
	}
	err := admission.Admit(
		t.Context(),
		&planservice.AdmissionRequest{TenantInfo: plantest.Tenant()},
	)
	require.ErrorIs(t, err, failure)
}
