package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

func LoadUser(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	if !loaders.Available(ctx) {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return loaders.UserByID(ctx, id.String())
}
