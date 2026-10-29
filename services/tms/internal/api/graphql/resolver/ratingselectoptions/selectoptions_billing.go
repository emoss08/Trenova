package ratingselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/reference/app/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	RateMatrixService        *ratematrixservice.Service
	RateAgreementService     *rateagreementservice.Service
	AccessorialChargeService *accessorialchargeservice.Service
	DetentionPolicyService   *detentionpolicyservice.Service
	FormulaTemplateService   *formulatemplateservice.Service
	FuelIndexRepo            fuelsurchargeports.FuelIndexRepository
	FuelSurchargeProgramRepo fuelsurchargeports.FuelSurchargeProgramRepository
}

func (r *Sources) RateMatrix(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.RateMatrixService.GetByIDs(
			ctx,
			&ratematrixports.GetRateMatricesByIDsRequest{
				TenantInfo:    req.TenantInfo,
				RateMatrixIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*ratematrix.RateMatrix).GetID,
			rateMatrixSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.RateMatrixService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		rateMatrixSelectOptionItem,
	)
}

func (r *Sources) RateAgreement(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.RateAgreementService.GetByIDs(
			ctx,
			&rateagreementports.GetRateAgreementsByIDsRequest{
				TenantInfo:       req.TenantInfo,
				RateAgreementIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*rateagreement.RateAgreement).GetID,
			rateAgreementSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.RateAgreementService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		rateAgreementSelectOptionItem,
	)
}

func (r *Sources) AccessorialCharge(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.AccessorialChargeService.GetByIDs(
			ctx,
			accessorialchargeports.GetAccessorialChargesByIDsRequest{
				TenantInfo:           req.TenantInfo,
				AccessorialChargeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*accessorialcharge.AccessorialCharge).GetID,
			accessorialChargeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.AccessorialChargeService.SelectOptions(
		ctx,
		req.SelectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		accessorialChargeSelectOptionItem,
	)
}

func (r *Sources) DetentionPolicy(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.DetentionPolicyService.GetByIDs(
			ctx,
			&detentionports.GetDetentionPoliciesByIDsRequest{
				TenantInfo:         req.TenantInfo,
				DetentionPolicyIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*detentionpolicy.DetentionPolicy).GetID,
			detentionPolicySelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DetentionPolicyService.SelectOptions(
		ctx,
		&detentionports.DetentionPolicySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		detentionPolicySelectOptionItem,
	)
}

func rateMatrixSelectOptionItem(entity *ratematrix.RateMatrix) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
		},
	}
}

func rateAgreementSelectOptionItem(entity *rateagreement.RateAgreement) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
		},
	}
}

func accessorialChargeSelectOptionItem(
	entity *accessorialcharge.AccessorialCharge,
) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
			"method":                 string(entity.Method),
			"amount":                 entity.Amount.String(),
		},
	}
}

func detentionPolicySelectOptionItem(
	entity *detentionpolicy.DetentionPolicy,
) selectoption.Item {
	return selectoption.ItemFor(
		detentionPolicySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func detentionPolicySelectOption(entity *detentionpolicy.DetentionPolicy) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
		},
	}
}

func (r *Sources) FormulaTemplate(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FormulaTemplateService.GetByIDs(
			ctx,
			formulatemplateports.GetFormulaTemplatesByIDsRequest{
				TenantInfo:  req.TenantInfo,
				TemplateIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*formulatemplate.FormulaTemplate).GetID,
			formulaTemplateSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FormulaTemplateService.SelectOptions(
		ctx,
		&formulatemplateports.FormulaTemplateSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		formulaTemplateSelectOptionItem,
	)
}

func formulaTemplateSelectOptionItem(
	entity *formulatemplate.FormulaTemplate,
) selectoption.Item {
	return selectoption.ItemFor(
		formulaTemplateSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func formulaTemplateSelectOption(entity *formulatemplate.FormulaTemplate) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
	}
}

func (r *Sources) FuelIndex(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FuelIndexRepo.GetByIDs(
			ctx,
			&fuelsurchargeports.GetFuelIndexesByIDsRequest{
				FuelIndexIDs: req.IDs,
				TenantInfo:   req.TenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*fuelsurcharge.FuelIndex).GetID,
			fuelIndexSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FuelIndexRepo.SelectOptions(
		ctx,
		&fuelsurchargeports.FuelIndexSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		fuelIndexSelectOptionItem,
	)
}

func (r *Sources) FuelSurchargeProgram(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FuelSurchargeProgramRepo.GetByIDs(
			ctx,
			&fuelsurchargeports.GetFuelSurchargeProgramsByIDsRequest{
				ProgramIDs: req.IDs,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*fuelsurcharge.FuelSurchargeProgram).GetID,
			fuelSurchargeProgramSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FuelSurchargeProgramRepo.SelectOptions(
		ctx,
		&fuelsurchargeports.FuelSurchargeProgramSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		fuelSurchargeProgramSelectOptionItem,
	)
}
