// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

var FormulaTemplateFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"name":       true,
		"category":   true,
		"isActive":   true,
		"isDefault":  true,
		"outputUnit": true,
	},
	SortableFields: map[string]bool{
		"name":      true,
		"category":  true,
		"createdAt": true,
		"updatedAt": true,
	},
	FieldMap: map[string]string{
		"name":       "name",
		"category":   "category",
		"isActive":   "is_active",
		"isDefault":  "is_default",
		"outputUnit": "output_unit",
		"createdAt":  "created_at",
		"updatedAt":  "updated_at",
	},
	EnumMap: map[string]bool{
		"category": true,
	},
}

type FormulaTemplateOptions struct {
	IncludeInactive bool `query:"includeInactive"`
}

type ListFormulaTemplateOptions struct {
	Filter                 *ports.LimitOffsetQueryOptions `json:"filter"                 query:"filter"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" query:"formulaTemplateOptions"`
}

type GetFormulaTemplateByIDOptions struct {
	ID                     pulid.ID `json:"id"                     query:"id"`
	OrgID                  pulid.ID `json:"orgId"                  query:"orgId"`
	BuID                   pulid.ID `json:"buId"                   query:"buId"`
	UserID                 pulid.ID `json:"userId"                 query:"userId"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" query:"formulaTemplateOptions"`
}

type GetDefaultFormulaTemplateOptions struct {
	Category               formulatemplate.Category `json:"category"               query:"category"`
	OrgID                  pulid.ID                 `json:"orgId"                  query:"orgId"`
	BuID                   pulid.ID                 `json:"buId"                   query:"buId"`
	UserID                 pulid.ID                 `json:"userId"                 query:"userId"`
	FormulaTemplateOptions `json:"formulaTemplateOptions" query:"formulaTemplateOptions"`
}

type SetDefaultFormulaTemplateRequest struct {
	TemplateID pulid.ID                 `json:"templateId" query:"templateId"`
	Category   formulatemplate.Category `json:"category"   query:"category"`
	OrgID      pulid.ID                 `json:"orgId"      query:"orgId"`
	BuID       pulid.ID                 `json:"buId"       query:"buId"`
	UserID     pulid.ID                 `json:"userId"     query:"userId"`
}

type FormulaTemplateRepository interface {
	List(
		ctx context.Context,
		opts *ListFormulaTemplateOptions,
	) (*ports.ListResult[*formulatemplate.FormulaTemplate], error)

	GetByID(
		ctx context.Context,
		opts *GetFormulaTemplateByIDOptions,
	) (*formulatemplate.FormulaTemplate, error)

	GetByCategory(
		ctx context.Context,
		category formulatemplate.Category,
		orgID pulid.ID,
		buID pulid.ID,
	) ([]*formulatemplate.FormulaTemplate, error)

	GetDefault(
		ctx context.Context,
		opts *GetDefaultFormulaTemplateOptions,
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

	Delete(
		ctx context.Context,
		id pulid.ID,
		orgID pulid.ID,
		buID pulid.ID,
	) error
}
