package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCustomFieldValuesByResourceRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   string
}

type GetCustomFieldValuesByResourcesRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceIDs  []string
}

type UpsertCustomFieldValuesRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   string
	Values       map[string]any
}

type GetValuesByDefinitionRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
}

type GetOptionUsageRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
}

type CustomFieldValueRepository interface {
	GetByResource(
		ctx context.Context,
		req *GetCustomFieldValuesByResourceRequest,
	) ([]*customfield.CustomFieldValue, error)

	GetByResources(
		ctx context.Context,
		req *GetCustomFieldValuesByResourcesRequest,
	) (map[string][]*customfield.CustomFieldValue, error)

	Upsert(ctx context.Context, req *UpsertCustomFieldValuesRequest) error

	DeleteByResource(ctx context.Context, req *GetCustomFieldValuesByResourceRequest) error

	CountByDefinition(ctx context.Context, req *GetValuesByDefinitionRequest) (int, error)

	CountResourcesByDefinition(ctx context.Context, req *GetValuesByDefinitionRequest) (int, error)

	GetOptionUsageCounts(ctx context.Context, req *GetOptionUsageRequest) (map[string]int, error)
}
