package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
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
) *dataloader.Loader[string, *edi.EDIPartner] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *EDIPartnerByCustomerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *edi.EDIPartner] {
	return func(
		ctx context.Context,
		keys []string,
	) []*dataloader.Result[*edi.EDIPartner] {
		results := make([]*dataloader.Result[*edi.EDIPartner], len(keys))

		customerIDs := make([]pulid.ID, 0, len(keys))
		for _, key := range keys {
			parsed, err := pulid.MustParse(key)
			if err != nil {
				continue
			}
			customerIDs = append(customerIDs, parsed)
		}

		partners, err := f.partnerRepo.ListInternalOutboundPartnersByCustomerIDs(
			ctx,
			repositories.ListEDIPartnersByCustomerIDsRequest{
				CustomerIDs: customerIDs,
				TenantInfo:  tenantInfo,
			},
		)
		if err != nil {
			for i := range results {
				results[i] = &dataloader.Result[*edi.EDIPartner]{Error: err}
			}
			return results
		}

		byCustomerID := make(map[pulid.ID]*edi.EDIPartner, len(partners))
		for _, partner := range partners {
			if _, ok := byCustomerID[partner.CustomerID]; !ok {
				byCustomerID[partner.CustomerID] = partner
			}
		}

		for i, key := range keys {
			parsed, parseErr := pulid.MustParse(key)
			if parseErr != nil {
				results[i] = &dataloader.Result[*edi.EDIPartner]{Error: parseErr}
				continue
			}
			results[i] = &dataloader.Result[*edi.EDIPartner]{Data: byCustomerID[parsed]}
		}

		return results
	}
}
