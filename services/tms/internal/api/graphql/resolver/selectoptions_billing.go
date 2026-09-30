package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveRateMatrixSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.rateMatrixService.GetByID(
				ctx,
				&repositories.GetRateMatrixByIDRequest{
					RateMatrixID: id,
					TenantInfo:   req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateMatrixSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.rateMatrixService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		rateMatrixSelectOptionItem,
	)
}

func (r *Resolver) resolveRateAgreementSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.rateAgreementService.GetByID(
				ctx,
				&repositories.GetRateAgreementByIDRequest{
					RateAgreementID: id,
					TenantInfo:      req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateAgreementSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.rateAgreementService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		rateAgreementSelectOptionItem,
	)
}

func (r *Resolver) resolveAccessorialChargeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.accessorialChargeService.Get(
				ctx,
				repositories.GetAccessorialChargeByIDRequest{
					ID:         id,
					TenantInfo: &req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accessorialChargeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.accessorialChargeService.SelectOptions(
		ctx,
		req.selectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		accessorialChargeSelectOptionItem,
	)
}

func (r *Resolver) resolveAccountTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.accountTypeService.Get(
				ctx,
				repositories.GetAccountTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accountTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.accountTypeService.SelectOptions(
		ctx,
		&repositories.AccountTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		accountTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveDetentionPolicySelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.detentionPolicyService.GetByID(
				ctx,
				&repositories.GetDetentionPolicyByIDRequest{
					DetentionPolicyID: id,
					TenantInfo:        req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, detentionPolicySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.detentionPolicyService.SelectOptions(
		ctx,
		&repositories.DetentionPolicySelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		detentionPolicySelectOptionItem,
	)
}

func rateMatrixSelectOptionItem(entity *ratematrix.RateMatrix) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		rateMatrixSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func rateMatrixSelectOption(entity *ratematrix.RateMatrix) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func rateAgreementSelectOptionItem(entity *rateagreement.RateAgreement) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		rateAgreementSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func rateAgreementSelectOption(entity *rateagreement.RateAgreement) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func accessorialChargeSelectOptionItem(
	entity *accessorialcharge.AccessorialCharge,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		accessorialChargeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func accessorialChargeSelectOption(
	entity *accessorialcharge.AccessorialCharge,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"code":   entity.Code,
			"method": string(entity.Method),
			"amount": entity.Amount.String(),
		},
	}
}

func accountTypeSelectOptionItem(entity *accounttype.AccountType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		accountTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func accountTypeSelectOption(entity *accounttype.AccountType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Name),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
			"name":  entity.Name,
		},
	}
}

func detentionPolicySelectOptionItem(entity *detention.DetentionPolicy) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		detentionPolicySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func detentionPolicySelectOption(entity *detention.DetentionPolicy) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}
