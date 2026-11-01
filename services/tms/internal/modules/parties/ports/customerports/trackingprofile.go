package customerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// TrackingProfileReader reads a customer's tracking settings on their own,
// for the tracking link and status email paths that need nothing else of the
// customer. A customer whose settings were never saved reads the defaults,
// with no id. Writes go through the customer.
type TrackingProfileReader interface {
	GetByCustomer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*trackingprofile.Profile, error)
}
