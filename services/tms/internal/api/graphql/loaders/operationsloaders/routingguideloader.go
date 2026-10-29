package operationsloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/operations/app/routingguideservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type routingGuidesByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req tenderports.GetRoutingGuidesByIDsRequest,
	) ([]*tender.RoutingGuide, error)
}

type RoutingGuideWithEntriesByIDLoaderFactoryParams struct {
	fx.In

	RoutingGuideService *routingguideservice.Service
}

type RoutingGuideWithEntriesByIDLoaderFactory struct {
	routingGuides routingGuidesByIDsGetter
}

func NewRoutingGuideWithEntriesByIDLoaderFactory(
	p RoutingGuideWithEntriesByIDLoaderFactoryParams,
) *RoutingGuideWithEntriesByIDLoaderFactory {
	return &RoutingGuideWithEntriesByIDLoaderFactory{
		routingGuides: p.RoutingGuideService,
	}
}

func newRoutingGuideWithEntriesByIDLoader(f *RoutingGuideWithEntriesByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tender.RoutingGuide] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *RoutingGuideWithEntriesByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*tender.RoutingGuide] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*tender.RoutingGuide, error) {
			return f.routingGuides.GetByIDs(ctx, tenderports.GetRoutingGuidesByIDsRequest{
				TenantInfo:      tenantInfo,
				RoutingGuideIDs: ids,
				RoutingGuideFilterOptions: tenderports.RoutingGuideFilterOptions{
					IncludeEntries: true,
				},
			})
		},
		"Routing guide not found within your organization",
	)
}

var routingGuideWithEntriesByIDKey = loadercore.NewKey(newRoutingGuideWithEntriesByIDLoader)

func RoutingGuideWithEntriesByID(ctx context.Context, key string) (*tender.RoutingGuide, error) {
	return loadercore.Load(ctx, routingGuideWithEntriesByIDKey, key)
}

func (f *RoutingGuideWithEntriesByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
