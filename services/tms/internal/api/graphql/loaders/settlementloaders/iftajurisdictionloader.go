package settlementloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type jurisdictionsByIDsGetter interface {
	GetJurisdictionsByIDs(ctx context.Context, ids []pulid.ID) ([]*ifta.Jurisdiction, error)
}

type IFTAJurisdictionByIDLoaderFactoryParams struct {
	fx.In

	IFTAService *iftaservice.Service
}

type IFTAJurisdictionByIDLoaderFactory struct {
	jurisdictions jurisdictionsByIDsGetter
}

func NewIFTAJurisdictionByIDLoaderFactory(
	p IFTAJurisdictionByIDLoaderFactoryParams,
) *IFTAJurisdictionByIDLoaderFactory {
	return &IFTAJurisdictionByIDLoaderFactory{
		jurisdictions: p.IFTAService,
	}
}

func newIFTAJurisdictionByIDLoader(f *IFTAJurisdictionByIDLoaderFactory,
	_ pagination.TenantInfo,
) *dataloadgen.Loader[string, *ifta.Jurisdiction] {
	return loadercore.NewLoader(f.batchFunc())
}

func (f *IFTAJurisdictionByIDLoaderFactory) batchFunc() loadercore.BatchFetchFunc[*ifta.Jurisdiction] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*ifta.Jurisdiction, error) {
			return f.jurisdictions.GetJurisdictionsByIDs(ctx, ids)
		},
		"IFTA jurisdiction not found",
	)
}

var iftaJurisdictionByIDKey = loadercore.NewKey(newIFTAJurisdictionByIDLoader)

func IFTAJurisdictionByID(ctx context.Context, key string) (*ifta.Jurisdiction, error) {
	return loadercore.Load(ctx, iftaJurisdictionByIDKey, key)
}

func (f *IFTAJurisdictionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
