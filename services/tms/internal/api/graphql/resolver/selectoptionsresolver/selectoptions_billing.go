package selectoptionsresolver

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

func (r *Deps) resolveRateMatrixSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.RateMatrixService.GetByID(
				ctx,
				&repositories.GetRateMatrixByIDRequest{
					RateMatrixID: id,
					TenantInfo:   req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateMatrixSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.RateMatrixService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		rateMatrixSelectOptionItem,
	)
}

func (r *Deps) resolveRateAgreementSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.RateAgreementService.GetByID(
				ctx,
				&repositories.GetRateAgreementByIDRequest{
					RateAgreementID: id,
					TenantInfo:      req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateAgreementSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.RateAgreementService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		rateAgreementSelectOptionItem,
	)
}

func (r *Deps) resolveAccessorialChargeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.AccessorialChargeService.Get(
				ctx,
				repositories.GetAccessorialChargeByIDRequest{
					ID:         id,
					TenantInfo: &req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accessorialChargeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.AccessorialChargeService.SelectOptions(
		ctx,
		req.SelectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		accessorialChargeSelectOptionItem,
	)
}

func (r *Deps) resolveAccountTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.AccountTypeService.Get(
				ctx,
				repositories.GetAccountTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accountTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.AccountTypeService.SelectOptions(
		ctx,
		&repositories.AccountTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		accountTypeSelectOptionItem,
	)
}

func (r *Deps) resolveDetentionPolicySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DetentionPolicyService.GetByID(
				ctx,
				&repositories.GetDetentionPolicyByIDRequest{
					DetentionPolicyID: id,
					TenantInfo:        req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, detentionPolicySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.DetentionPolicyService.SelectOptions(
		ctx,
		&repositories.DetentionPolicySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
