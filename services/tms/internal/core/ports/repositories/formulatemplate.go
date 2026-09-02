package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetFormulaTemplateByIDRequest struct {
	TemplateID pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListFormulaTemplatesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Type   string                   `json:"type"`
	Status string                   `json:"status"`
}

type ListFormulaTemplateConnectionRequest struct {
	Filter                 *pagination.QueryOptions `json:"filter"`
	Cursor                 pagination.CursorInfo    `json:"-"`
	FormulaTemplateColumns []string                 `json:"-"`
}

type BulkUpdateFormulaTemplateStatusRequest struct {
	TenantInfo  pagination.TenantInfo  `json:"-"`
	TemplateIDs []pulid.ID             `json:"templateIds"`
	Status      formulatemplate.Status `json:"status"`
}

type BulkDuplicateFormulaTemplateRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
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

// TemplateStats is what a list row needs to decide whether a template is
// live and pinned: how many things rate with it and how many scenarios
// guard it.
type TemplateStats struct {
	UsageCount    int `json:"usageCount"`
	ScenarioCount int `json:"scenarioCount"`
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
	BulkDuplicate(
		ctx context.Context,
		req *BulkDuplicateFormulaTemplateRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
	CountUsages(
		ctx context.Context,
		req *GetTemplateUsageRequest,
	) (*GetTemplateUsageResponse, error)
	// CountStatsByIDs returns usage and scenario counts for a page of
	// templates in one statement; ids with no row simply have no entry.
	CountStatsByIDs(
		ctx context.Context,
		req *GetFormulaTemplateStatsRequest,
	) (map[pulid.ID]TemplateStats, error)
	SelectOptions(
		ctx context.Context,
		req *FormulaTemplateSelectOptionsRequest,
	) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
}
