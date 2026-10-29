package workerchecklistports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListChecklistTemplatesRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	Cursor       pagination.CursorInfo    `json:"cursor"`
	Status       string                   `json:"status"`
	Kind         string                   `json:"kind"`
	Trigger      string                   `json:"trigger"`
	IncludeItems bool                     `json:"includeItems"`
}

type GetChecklistTemplateByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeItems bool                  `json:"includeItems"`
}

type ChecklistTemplateCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type GetDefaultChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo            `json:"tenantInfo"`
	Trigger    workerchecklist.ChecklistTrigger `json:"trigger"`
}

type ClearDefaultChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo            `json:"tenantInfo"`
	Trigger    workerchecklist.ChecklistTrigger `json:"trigger"`
	ExceptID   pulid.ID                         `json:"exceptId"`
}

type ListWorkerChecklistsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	IncludeClosed bool                  `json:"includeClosed"`
	IncludeItems  bool                  `json:"includeItems"`
}

type GetWorkerChecklistByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeItems bool                  `json:"includeItems"`
}

type GetWorkerChecklistItemByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CountOpenChecklistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	TemplateID pulid.ID              `json:"templateId"`
}

type CountOpenChecklistsByTemplateIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
}

type WorkerChecklistRepository interface {
	ListTemplates(
		ctx context.Context,
		req *ListChecklistTemplatesRequest,
	) (*pagination.CursorListResult[*workerchecklist.WorkerChecklistTemplate], error)
	ListActiveTemplates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*workerchecklist.WorkerChecklistTemplate, error)
	GetTemplateByID(
		ctx context.Context,
		req *GetChecklistTemplateByIDRequest,
	) (*workerchecklist.WorkerChecklistTemplate, error)
	GetDefaultTemplate(
		ctx context.Context,
		req *GetDefaultChecklistTemplateRequest,
	) (*workerchecklist.WorkerChecklistTemplate, error)
	TemplateCodeExists(ctx context.Context, req *ChecklistTemplateCodeExistsRequest) (bool, error)
	CreateTemplate(
		ctx context.Context,
		entity *workerchecklist.WorkerChecklistTemplate,
	) (*workerchecklist.WorkerChecklistTemplate, error)
	UpdateTemplate(
		ctx context.Context,
		entity *workerchecklist.WorkerChecklistTemplate,
	) (*workerchecklist.WorkerChecklistTemplate, error)
	ClearDefaultTemplate(ctx context.Context, req *ClearDefaultChecklistTemplateRequest) error
	CountOpenChecklists(ctx context.Context, req *CountOpenChecklistsRequest) (int, error)
	CountOpenChecklistsByTemplateIDs(
		ctx context.Context,
		req *CountOpenChecklistsByTemplateIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerChecklistsRequest,
	) ([]*workerchecklist.WorkerChecklist, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerChecklistByIDRequest,
	) (*workerchecklist.WorkerChecklist, error)
	GetItemByID(
		ctx context.Context,
		req *GetWorkerChecklistItemByIDRequest,
	) (*workerchecklist.WorkerChecklistItem, error)
	Create(
		ctx context.Context,
		entity *workerchecklist.WorkerChecklist,
	) (*workerchecklist.WorkerChecklist, error)
	Update(
		ctx context.Context,
		entity *workerchecklist.WorkerChecklist,
	) (*workerchecklist.WorkerChecklist, error)
	UpdateItems(ctx context.Context, items []*workerchecklist.WorkerChecklistItem) error
}
