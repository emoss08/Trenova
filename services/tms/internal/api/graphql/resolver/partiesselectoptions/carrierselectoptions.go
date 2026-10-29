package partiesselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
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
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.CarrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, carrierSelectOptionItem(entity))
		}

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
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.CustomerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, customerSelectOptionItem(entity))
		}

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
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FactoringCompanyRepo.GetByID(
				ctx,
				factoringcompanyports.GetFactoringCompanyByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, factoringCompanySelectOptionItem(entity))
		}

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
