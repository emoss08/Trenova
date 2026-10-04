package tenantbootstraprepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBootstrapRequiresOrganizationAndOwner(t *testing.T) {
	t.Parallel()

	repo := New(Params{Logger: zap.NewNop()})

	_, err := repo.Bootstrap(t.Context(), &repositories.BootstrapTenantRequest{})
	require.ErrorIs(t, err, ErrOrganizationRequired)

	_, err = repo.Bootstrap(t.Context(), &repositories.BootstrapTenantRequest{
		Organization: &tenant.Organization{},
	})
	require.ErrorIs(t, err, ErrOwnerRequired)
}
