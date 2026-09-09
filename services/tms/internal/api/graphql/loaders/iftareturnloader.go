package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type returnsByIDsGetter interface {
	GetReturnsByIDs(
		ctx context.Context,
		req *repositories.GetReturnsByIDsRequest,
	) ([]*ifta.Return, error)
}

type IFTAReturnByIDLoaderFactoryParams struct {
	fx.In

	IFTAService *iftaservice.Service
}

type IFTAReturnByIDLoaderFactory struct {
	returns returnsByIDsGetter
}

func NewIFTAReturnByIDLoaderFactory(
	p IFTAReturnByIDLoaderFactoryParams,
) *IFTAReturnByIDLoaderFactory {
	return &IFTAReturnByIDLoaderFactory{
		returns: p.IFTAService,
	}
}

func (f *IFTAReturnByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *ifta.Return] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *IFTAReturnByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*ifta.Return] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*ifta.Return, error) {
		return f.returns.GetReturnsByIDs(ctx, &repositories.GetReturnsByIDsRequest{
			TenantInfo: tenantInfo,
			IDs:        ids,
		})
	}, "IFTA return not found within your organization")
}
