package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type EDIPartnerByCustomerIDLoaderFactoryParams struct {
	fx.In

	PartnerRepo repositories.EDIPartnerRepository
}

type EDIPartnerByCustomerIDLoaderFactory struct {
	partnerRepo repositories.EDIPartnerRepository
}

func NewEDIPartnerByCustomerIDLoaderFactory(
	p EDIPartnerByCustomerIDLoaderFactoryParams,
) *EDIPartnerByCustomerIDLoaderFactory {
	return &EDIPartnerByCustomerIDLoaderFactory{
		partnerRepo: p.PartnerRepo,
	}
}

func (f *EDIPartnerByCustomerIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *edi.EDIPartner] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *EDIPartnerByCustomerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*edi.EDIPartner] {
	return func(ctx context.Context, keys []string) ([]*edi.EDIPartner, []error) {
		values := make([]*edi.EDIPartner, len(keys))
		errs := make([]error, len(keys))

		customerIDs, indexesByID := parseBatchKeys(keys, errs)
		if len(customerIDs) == 0 {
			return values, errs
		}

		partners, err := f.partnerRepo.ListInternalOutboundPartnersByCustomerIDs(
			ctx,
			repositories.ListEDIPartnersByCustomerIDsRequest{
				CustomerIDs: customerIDs,
				TenantInfo:  tenantInfo,
			},
		)
		if err != nil {
			fillMissingErrors(errs, err)
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
