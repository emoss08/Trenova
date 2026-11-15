package platformcatalogports

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var ErrUnknownFeature = errors.New("unknown platform feature")

type FeatureGate interface {
	Enabled(ctx context.Context, feature platformcatalog.FeatureKey) (bool, error)
}

func RequireFeature(
	ctx context.Context,
	gate FeatureGate,
	feature platformcatalog.FeatureKey,
) error {
	if _, known := platformcatalog.Lookup(feature); !known {
		return fmt.Errorf("%w: %q", ErrUnknownFeature, feature)
	}

	enabled, err := gate.Enabled(ctx, feature)
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}

	addOn, _ := platformcatalog.AddOnFor(feature)

	return errortypes.NewFeatureRestrictionError(string(feature), addOn.Name)
}
