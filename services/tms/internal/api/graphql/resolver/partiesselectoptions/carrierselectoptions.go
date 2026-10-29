package partiesselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	CarrierRepo          carrierports.CarrierRepository
	CustomerRepo         customerports.CustomerRepository
	FactoringCompanyRepo factoringcompanyports.FactoringCompanyRepository
}

func (r *Sources) Carrier(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.CarrierRepo.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
			TenantInfo: req.TenantInfo,
			CarrierIDs: req.IDs,
		})
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*carrier.Carrier).GetID,
			carrierSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.CarrierRepo.SelectOptions(
		ctx,
		&carrierports.CarrierSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		carrierSelectOptionItem,
	)
}

func carrierSelectOptionItem(entity *carrier.Carrier) selectoption.Item {
	return selectoption.ItemFor(
		carrierSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func carrierSelectOption(entity *carrier.Carrier) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			selectoption.MetaKeyCode:   entity.Code,
			selectoption.MetaKeyStatus: string(entity.Status),
			"scac":                     entity.SCAC,
		},
	}
}

func (r *Sources) Customer(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.CustomerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
			TenantInfo:  req.TenantInfo,
			CustomerIDs: req.IDs,
		})
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*customer.Customer).GetID,
			customerSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.CustomerRepo.SelectOptions(
		ctx,
		&customerports.CustomerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		customerSelectOptionItem,
	)
}

func (r *Sources) FactoringCompany(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FactoringCompanyRepo.GetByIDs(
			ctx,
			factoringcompanyports.GetFactoringCompaniesByIDsRequest{
				TenantInfo: req.TenantInfo,
				IDs:        req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*factoringcompany.FactoringCompany).GetID,
			factoringCompanySelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FactoringCompanyRepo.SelectOptions(
		ctx,
		&factoringcompanyports.FactoringCompanySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		factoringCompanySelectOptionItem,
	)
}
