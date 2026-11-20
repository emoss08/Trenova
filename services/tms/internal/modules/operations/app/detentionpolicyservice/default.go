package detentionpolicyservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// organizationDefaultID reads the policy shipment control names as the
// organization's default detention policy, the one owner of that fact. It is
// nil when the organization has named none.
func organizationDefaultID(
	ctx context.Context,
	controls tenantports.ShipmentControlRepository,
	tenantInfo pagination.TenantInfo,
) (pulid.ID, error) {
	control, err := controls.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		return pulid.Nil, nil
	}
	if err != nil {
		return pulid.Nil, fmt.Errorf("failed to read the default detention policy: %w", err)
	}
	if control.DefaultDetentionPolicyID == nil {
		return pulid.Nil, nil
	}

	return *control.DefaultDetentionPolicyID, nil
}
