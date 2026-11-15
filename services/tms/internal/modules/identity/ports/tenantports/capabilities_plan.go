package tenantports

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

func CapabilitiesOnPlan(
	ctx context.Context,
	gate platformcatalogports.FeatureGate,
	stored *OrganizationCapabilities,
) (*OrganizationCapabilities, error) {
	if stored == nil || !stored.BrokerageEnabled {
		return stored, nil
	}

	brokerage, err := gate.Enabled(ctx, platformcatalog.FeatureCarrierBrokerage)
	if err != nil {
		return nil, fmt.Errorf("ask whether carrier brokerage is on the plan: %w", err)
	}

	effective := *stored
	effective.BrokerageEnabled = brokerage
	return &effective, nil
}
