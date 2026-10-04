package iamservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSSOAndSCIMWritesRefuseWhenThePlanRestrictsSSO(t *testing.T) {
	t.Parallel()

	svc := &service{plans: plantest.Restricting(t, platformplan.CapabilitySSO)}
	tenant := plantest.Tenant()

	provider, err := svc.CreateIdentityProvider(t.Context(), tenant, &services.IdentityProviderRequest{})
	require.Nil(t, provider)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	directory, err := svc.CreateSCIMDirectory(t.Context(), tenant, &iam.SCIMDirectory{})
	require.Nil(t, directory)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
