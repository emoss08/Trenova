package platformcatalogports

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
)

type FeatureObserver interface {
	ObserverFeature() platformcatalog.FeatureKey
}

func Observing[T FeatureObserver](
	ctx context.Context,
	gate FeatureGate,
	tenant dbscope.Tenant,
	observers []T,
) ([]T, error) {
	if len(observers) == 0 {
		return nil, nil
	}
	if !tenant.OrganizationID.IsNil() {
		ctx = dbscope.WithTenant(ctx, tenant)
	}

	answers := make(map[platformcatalog.FeatureKey]bool, len(observers))
	kept := make([]T, 0, len(observers))
	var errs []error
	for _, observer := range observers {
		if any(observer) == nil {
			continue
		}

		feature := observer.ObserverFeature()
		on, asked := answers[feature]
		if !asked {
			var err error
			on, err = observerFeatureOn(ctx, gate, feature)
			if err != nil {
				errs = append(errs, err)
			}
			answers[feature] = on
		}
		if on {
			kept = append(kept, observer)
		}
	}

	return kept, errors.Join(errs...)
}

func observerFeatureOn(
	ctx context.Context,
	gate FeatureGate,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	if _, known := platformcatalog.Lookup(feature); !known {
		return false, fmt.Errorf("observer names %w: %q", ErrUnknownFeature, feature)
	}

	on, err := gate.Enabled(ctx, feature)
	if err != nil {
		return false, fmt.Errorf("ask whether %s is on the plan: %w", feature, err)
	}

	return on, nil
}
