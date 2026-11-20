package entitlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

type AllowAll struct{}

var _ platformcatalogports.FeatureGate = (*AllowAll)(nil)

func NewAllowAll() *AllowAll {
	return &AllowAll{}
}

func (*AllowAll) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return true, nil
}

func (*AllowAll) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return true, nil
}
