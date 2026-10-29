package settlementloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type returnsByIDsGetter interface {
	GetReturnsByIDs(
		ctx context.Context,
		req *iftaports.GetReturnsByIDsRequest,
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

func newIFTAReturnByIDLoader(f *IFTAReturnByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *ifta.Return] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *IFTAReturnByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*ifta.Return] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*ifta.Return, error) {
			return f.returns.GetReturnsByIDs(ctx, &iftaports.GetReturnsByIDsRequest{
				TenantInfo: tenantInfo,
				IDs:        ids,
			})
		},
		"IFTA return not found within your organization",
	)
}

var iftaReturnByIDKey = loadercore.NewKey(newIFTAReturnByIDLoader)

func IFTAReturnByID(ctx context.Context, key string) (*ifta.Return, error) {
	return loadercore.Load(ctx, iftaReturnByIDKey, key)
}

func (f *IFTAReturnByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
