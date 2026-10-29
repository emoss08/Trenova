package ediloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type EDIPartnerByCustomerIDLoaderFactoryParams struct {
	fx.In

	PartnerRepo edipartnerports.EDIPartnerRepository
}

type EDIPartnerByCustomerIDLoaderFactory struct {
	partnerRepo edipartnerports.EDIPartnerRepository
}

func NewEDIPartnerByCustomerIDLoaderFactory(
	p EDIPartnerByCustomerIDLoaderFactoryParams,
) *EDIPartnerByCustomerIDLoaderFactory {
	return &EDIPartnerByCustomerIDLoaderFactory{
		partnerRepo: p.PartnerRepo,
	}
}

func newEDIPartnerByCustomerIDLoader(f *EDIPartnerByCustomerIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *edi.EDIPartner] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *EDIPartnerByCustomerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*edi.EDIPartner] {
	return func(ctx context.Context, keys []string) ([]*edi.EDIPartner, []error) {
		values := make([]*edi.EDIPartner, len(keys))
		errs := make([]error, len(keys))

		customerIDs, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(customerIDs) == 0 {
			return values, errs
		}

		partners, err := f.partnerRepo.ListInternalOutboundPartnersByCustomerIDs(
			ctx,
			edipartnerports.ListEDIPartnersByCustomerIDsRequest{
				CustomerIDs: customerIDs,
				TenantInfo:  tenantInfo,
			},
		)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
			return values, errs
		}

		byCustomerID := make(map[pulid.ID]*edi.EDIPartner, len(partners))
		for _, partner := range partners {
			if _, ok := byCustomerID[partner.CustomerID]; !ok {
				byCustomerID[partner.CustomerID] = partner
			}
		}

		for _, id := range customerIDs {
			for _, idx := range indexesByID[id] {
				values[idx] = byCustomerID[id]
			}
		}

		return values, errs
	}
}

var ediPartnerByCustomerIDKey = loadercore.NewKey(newEDIPartnerByCustomerIDLoader)

func EDIPartnerByCustomerID(ctx context.Context, key string) (*edi.EDIPartner, error) {
	return loadercore.Load(ctx, ediPartnerByCustomerIDKey, key)
}

func (f *EDIPartnerByCustomerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
