package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/graph-gophers/dataloader/v7"
	"go.uber.org/fx"
)

type contextKey struct{}

type FactoryParams struct {
	fx.In

	TrailerByID      *TrailerByIDLoaderFactory
	OrganizationByID *OrganizationByIDLoaderFactory
	LocationByID     *LocationByIDLoaderFactory
}

type Factory struct {
	trailerByID      *TrailerByIDLoaderFactory
	organizationByID *OrganizationByIDLoaderFactory
	locationByID     *LocationByIDLoaderFactory
}

type Loaders struct {
	TrailerByID      *dataloader.Loader[string, *trailer.Trailer]
	OrganizationByID *dataloader.Loader[string, *tenant.Organization]
	LocationByID     *dataloader.Loader[string, *location.Location]
}

func NewFactory(p FactoryParams) *Factory {
	return &Factory{
		trailerByID:      p.TrailerByID,
		organizationByID: p.OrganizationByID,
		locationByID:     p.LocationByID,
	}
}

func (f *Factory) NewForTenant(tenantInfo pagination.TenantInfo) *Loaders {
	return &Loaders{
		TrailerByID:      f.trailerByID.NewForTenant(tenantInfo),
		OrganizationByID: f.organizationByID.NewForTenant(tenantInfo),
		LocationByID:     f.locationByID.NewForTenant(tenantInfo),
	}
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, contextKey{}, loaders)
}

func FromContext(ctx context.Context) (*Loaders, bool) {
	loaders, ok := ctx.Value(contextKey{}).(*Loaders)
	return loaders, ok
}
