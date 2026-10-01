package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/shared/pulid"
)

func IDValuePtr(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}
	value := id
	return &value
}

func LoadTractor(ctx context.Context, id *pulid.ID) (*tractor.Tractor, error) {
	if id == nil || id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return l.TractorByID.Load(ctx, id.String())
}

func LoadJurisdiction(ctx context.Context, id *pulid.ID) (*ifta.Jurisdiction, error) {
	if id == nil || id.IsNil() {
		return nil, nil //nolint:nilnil // an unset reference resolves to null
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, nil //nolint:nilnil // outside a request with loaders the relation resolves to null
	}

	return l.IFTAJurisdictionByID.Load(ctx, id.String())
}
