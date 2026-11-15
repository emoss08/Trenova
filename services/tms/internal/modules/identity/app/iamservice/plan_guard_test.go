package iamservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSSOAndSCIMWritesRefuseWhenSecurityAndIdentityIsOff(t *testing.T) {
	t.Parallel()

	svc := &service{features: plantest.Without(platformcatalog.FeatureIdentity)}
	tenant := plantest.Tenant()

	provider, err := svc.CreateIdentityProvider(t.Context(), tenant, &iamports.IdentityProviderRequest{})
	require.Nil(t, provider)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	directory, err := svc.CreateSCIMDirectory(t.Context(), tenant, &iam.SCIMDirectory{})
	require.Nil(t, directory)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
