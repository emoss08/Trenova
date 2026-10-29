package loaderfields

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/equipmentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/identityloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/settlementloaders"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func LoadTractor(ctx context.Context, id *pulid.ID) (*tractor.Tractor, error) {
	if id == nil || id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	if !loadercore.Available(ctx) {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return equipmentloaders.TractorByID(ctx, id.String())
}

func LoadJurisdiction(ctx context.Context, id *pulid.ID) (*ifta.Jurisdiction, error) {
	if id == nil || id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	if !loadercore.Available(ctx) {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return settlementloaders.IFTAJurisdictionByID(ctx, id.String())
}

func LoadUser(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	if !loadercore.Available(ctx) {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return identityloaders.UserByID(ctx, id.String())
}

// optionalMatch reads a matched record for display. A record that is gone or
// out of the reader's reach is no match, not a failed inbox; anything else is
// a real failure and stays one.
func OptionalMatch[T any](value T, err error) (T, error) {
	if err != nil {
		var zero T
		if errortypes.IsNotFoundError(err) {
			return zero, nil
		}

		return zero, err
	}

	return value, nil
}

func RequireLoaders(ctx context.Context) error {
	if !loadercore.Available(ctx) {
		return errortypes.NewDatabaseError("Request loaders are not configured")
	}

	return nil
}
