package formulatemplateports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetFormulaTemplateByIDRequest struct {
	TemplateID pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListFormulaTemplatesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Status string                   `json:"status"`
}

type ListFormulaTemplateConnectionRequest struct {
	Filter                 *pagination.QueryOptions `json:"filter"`
	Cursor                 pagination.CursorInfo    `json:"-"`
	FormulaTemplateColumns []string                 `json:"-"`
}

type BulkUpdateFormulaTemplateStatusRequest struct {
	TenantInfo      pagination.TenantInfo  `json:"-"`
	TemplateIDs     []pulid.ID             `json:"templateIds"`
	Status          formulatemplate.Status `json:"status"`
	ConfirmInUse    bool                   `json:"confirmInUse"`
	MayArchiveInUse bool                   `json:"-"`
}

type BulkDuplicateFormulaTemplateRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
}

type ListTakenNamesRequest struct {
	TenantInfo pagination.TenantInfo
	Bases      []string
}

type GetFormulaTemplatesByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
}

type GetFormulaTemplatesByNamesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Names      []string              `json:"names"`
}

type TemplateUsageCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type GetTemplateUsageRequest struct {
	TemplateID pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetTemplateUsageResponse struct {
	InUse  bool                 `json:"inUse"`
	Usages []TemplateUsageCount `json:"usages"`
}

type TemplateStats struct {
	UsageCount int `json:"usageCount"`
}

type GetFormulaTemplateStatsRequest struct {
	TenantInfo  pagination.TenantInfo
	TemplateIDs []pulid.ID
}

type FormulaTemplateSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type FormulaTemplateRepository interface {
	Create(
		ctx context.Context,
		entity *formulatemplate.FormulaTemplate,
	) (*formulatemplate.FormulaTemplate, error)
	Update(
		ctx context.Context,
		entity *formulatemplate.FormulaTemplate,
	) (*formulatemplate.FormulaTemplate, error)
	GetByID(
		ctx context.Context,
		req GetFormulaTemplateByIDRequest,
	) (*formulatemplate.FormulaTemplate, error)
	GetByIDs(
		ctx context.Context,
		req GetFormulaTemplatesByIDsRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
	FindByNames(
		ctx context.Context,
		req GetFormulaTemplatesByNamesRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
	List(
		ctx context.Context,
		req *ListFormulaTemplatesRequest,
	) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
	ListConnection(
		ctx context.Context,
		req *ListFormulaTemplateConnectionRequest,
	) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateFormulaTemplateStatusRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
	BulkCreate(
		ctx context.Context,
		entities []*formulatemplate.FormulaTemplate,
	) ([]*formulatemplate.FormulaTemplate, error)
	TakenNames(
		ctx context.Context,
		req *ListTakenNamesRequest,
	) (map[string]struct{}, error)
	CountUsages(
		ctx context.Context,
		req *GetTemplateUsageRequest,
	) (*GetTemplateUsageResponse, error)
	CountStatsByIDs(
		ctx context.Context,
		req *GetFormulaTemplateStatsRequest,
	) (map[pulid.ID]TemplateStats, error)
	SelectOptions(
		ctx context.Context,
		req *FormulaTemplateSelectOptionsRequest,
	) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
}
