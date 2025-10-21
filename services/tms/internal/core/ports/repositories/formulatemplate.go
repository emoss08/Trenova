package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type FormulaTemplateOptions struct {
	IncludeInactive bool `form:"includeInactive"`
}

type ListFormulaTemplateRequest struct {
	Filter                 *pagination.QueryOptions `json:"filter"                 form:"filter"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" form:"formulaTemplateOptions"`
}

type GetFormulaTemplateByIDRequest struct {
	ID                     pulid.ID `json:"id"                     form:"id"`
	OrgID                  pulid.ID `json:"orgId"                  form:"orgId"`
	BuID                   pulid.ID `json:"buId"                   form:"buId"`
	UserID                 pulid.ID `json:"userId"                 form:"userId"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" form:"formulaTemplateOptions"`
}

type GetDefaultFormulaTemplateRequest struct {
	Category               formulatemplate.Category `json:"category"               form:"category"`
	OrgID                  pulid.ID                 `json:"orgId"                  form:"orgId"`
	BuID                   pulid.ID                 `json:"buId"                   form:"buId"`
	UserID                 pulid.ID                 `json:"userId"                 form:"userId"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" form:"formulaTemplateOptions"`
}

type SetDefaultFormulaTemplateRequest struct {
	TemplateID pulid.ID                 `json:"templateId" form:"templateId"`
	Category   formulatemplate.Category `json:"category"   form:"category"`
	OrgID      pulid.ID                 `json:"orgId"      form:"orgId"`
	BuID       pulid.ID                 `json:"buId"       form:"buId"`
	UserID     pulid.ID                 `json:"userId"     form:"userId"`
}

type FormulaTemplateRepository interface {
	List(
		ctx context.Context,
		opts *ListFormulaTemplateRequest,
	) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
	GetByID(
		ctx context.Context,
		opts *GetFormulaTemplateByIDRequest,
	) (*formulatemplate.FormulaTemplate, error)
	GetByCategory(
		ctx context.Context,
		category formulatemplate.Category,
		orgID pulid.ID,
		buID pulid.ID,
	) ([]*formulatemplate.FormulaTemplate, error)
	GetDefault(
		ctx context.Context,
		opts *GetDefaultFormulaTemplateRequest,
	) (*formulatemplate.FormulaTemplate, error)
	Create(
		ctx context.Context,
		template *formulatemplate.FormulaTemplate,
	) (*formulatemplate.FormulaTemplate, error)
	Update(
		ctx context.Context,
		template *formulatemplate.FormulaTemplate,
	) (*formulatemplate.FormulaTemplate, error)
	SetDefault(
		ctx context.Context,
		req *SetDefaultFormulaTemplateRequest,
	) error
}
