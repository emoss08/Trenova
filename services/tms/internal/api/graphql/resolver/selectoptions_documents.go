package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveDocumentTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.documentTypeService.Get(
				ctx,
				repositories.GetDocumentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, documentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.documentTypeService.SelectOptions(
		ctx,
		req.selectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		documentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveFormulaTemplateSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.formulaTemplateService.GetByID(
				ctx,
				repositories.GetFormulaTemplateByIDRequest{
					TemplateID: id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, formulaTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.formulaTemplateService.SelectOptions(
		ctx,
		&repositories.FormulaTemplateSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		formulaTemplateSelectOptionItem,
	)
}

func (r *Resolver) resolveEmailProfileSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.emailService.GetProfile(
				ctx,
				repositories.GetEmailEntityRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, emailProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.emailService.SelectProfileOptions(
		ctx,
		&repositories.EmailProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		emailProfileSelectOptionItem,
	)
}

func documentTypeSelectOptionItem(entity *documenttype.DocumentType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		documentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func documentTypeSelectOption(entity *documenttype.DocumentType) *gqlmodel.SelectOption {
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

func formulaTemplateSelectOptionItem(
	entity *formulatemplate.FormulaTemplate,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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

func emailProfileSelectOptionItem(entity *email.Profile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		emailProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func emailProfileSelectOption(entity *email.Profile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.SenderEmail),
		Meta: map[string]any{
			"senderEmail": entity.SenderEmail,
			"provider":    string(entity.Provider),
		},
	}
}
