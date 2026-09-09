package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
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

func (f *IFTAJurisdictionByIDLoaderFactory) NewForTenant(
	_ pagination.TenantInfo,
) *dataloadgen.Loader[string, *ifta.Jurisdiction] {
	return newLoader(f.batchFunc())
}

func (f *IFTAJurisdictionByIDLoaderFactory) batchFunc() batchFetchFunc[*ifta.Jurisdiction] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*ifta.Jurisdiction, error) {
		return f.jurisdictions.GetJurisdictionsByIDs(ctx, ids)
	}, "IFTA jurisdiction not found")
}
