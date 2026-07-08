package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
	"go.uber.org/fx"
)

type TrailerByIDLoaderFactoryParams struct {
	fx.In

	TrailerService *trailerservice.Service
}

type TrailerByIDLoaderFactory struct {
	trailerService *trailerservice.Service
}

func NewTrailerByIDLoaderFactory(
	p TrailerByIDLoaderFactoryParams,
) *TrailerByIDLoaderFactory {
	return &TrailerByIDLoaderFactory{
		trailerService: p.TrailerService,
	}
}

func (f *TrailerByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloader.Loader[string, *trailer.Trailer] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *TrailerByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *trailer.Trailer] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*trailer.Trailer, error) {
		return f.trailerService.GetByIDs(ctx, repositories.GetTrailersByIDsRequest{
			TenantInfo: tenantInfo,
			TrailerIDs: ids,
		})
	}, "Trailer not found")
}
