package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	TenantInfo pagination.TenantInfo   `json:"tenantInfo"`
	Trigger    worker.ChecklistTrigger `json:"trigger"`
}

type ClearDefaultChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo   `json:"tenantInfo"`
	Trigger    worker.ChecklistTrigger `json:"trigger"`
	ExceptID   pulid.ID                `json:"exceptId"`
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
	) (*pagination.CursorListResult[*worker.WorkerChecklistTemplate], error)
	ListActiveTemplates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.WorkerChecklistTemplate, error)
	GetTemplateByID(
		ctx context.Context,
		req *GetChecklistTemplateByIDRequest,
	) (*worker.WorkerChecklistTemplate, error)
	GetDefaultTemplate(
		ctx context.Context,
		req *GetDefaultChecklistTemplateRequest,
	) (*worker.WorkerChecklistTemplate, error)
	TemplateCodeExists(ctx context.Context, req *ChecklistTemplateCodeExistsRequest) (bool, error)
	CreateTemplate(
		ctx context.Context,
		entity *worker.WorkerChecklistTemplate,
	) (*worker.WorkerChecklistTemplate, error)
	UpdateTemplate(
		ctx context.Context,
		entity *worker.WorkerChecklistTemplate,
	) (*worker.WorkerChecklistTemplate, error)
	ClearDefaultTemplate(ctx context.Context, req *ClearDefaultChecklistTemplateRequest) error
	CountOpenChecklists(ctx context.Context, req *CountOpenChecklistsRequest) (int, error)
	CountOpenChecklistsByTemplateIDs(
		ctx context.Context,
		req *CountOpenChecklistsByTemplateIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerChecklistsRequest,
	) ([]*worker.WorkerChecklist, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerChecklistByIDRequest,
	) (*worker.WorkerChecklist, error)
	GetItemByID(
		ctx context.Context,
		req *GetWorkerChecklistItemByIDRequest,
	) (*worker.WorkerChecklistItem, error)
	Create(ctx context.Context, entity *worker.WorkerChecklist) (*worker.WorkerChecklist, error)
	Update(ctx context.Context, entity *worker.WorkerChecklist) (*worker.WorkerChecklist, error)
	UpdateItems(ctx context.Context, items []*worker.WorkerChecklistItem) error
}
