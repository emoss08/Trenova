package shipmenteventports

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestListShipmentEventsRequestValidate(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	require.NoError(t, (&ListShipmentEventsRequest{TenantInfo: tenant}).Validate())

	err := (&ListShipmentEventsRequest{}).Validate()
	var multi *errortypes.MultiError
	require.ErrorAs(t, err, &multi)
	require.NoError(t, multi.Internal(), "a missing tenant is a validation failure, not a failed run")
	require.True(t, multi.HasErrorAt("TenantInfo.orgId"))
	require.True(t, multi.HasErrorAt("TenantInfo.buId"))
}
