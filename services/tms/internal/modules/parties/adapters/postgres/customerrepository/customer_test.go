package customerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSaveProfilesSkipWhenProfileMissing(t *testing.T) {
	t.Parallel()

	repo := &repository{l: zap.NewNop()}
	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	require.NoError(t, repo.saveBillingProfile(t.Context(), entity))
	require.NoError(t, repo.saveEmailProfile(t.Context(), entity))
}
