package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCustomFieldDefinitionsRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	ResourceType string                   `json:"resourceType"`
}

type GetCustomFieldDefinitionByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type GetActiveByResourceTypeRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceType string                `json:"resourceType"`
}

type CountByResourceTypeRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceType string                `json:"resourceType"`
}

type CustomFieldDefinitionRepository interface {
	List(
		ctx context.Context,
		req *ListCustomFieldDefinitionsRequest,
	) (*pagination.ListResult[*customfield.CustomFieldDefinition], error)
	GetByID(
		ctx context.Context,
		req GetCustomFieldDefinitionByIDRequest,
	) (*customfield.CustomFieldDefinition, error)
	GetActiveByResourceType(
		ctx context.Context,
		req GetActiveByResourceTypeRequest,
	) ([]*customfield.CustomFieldDefinition, error)
	Create(
		ctx context.Context,
		entity *customfield.CustomFieldDefinition,
	) (*customfield.CustomFieldDefinition, error)
	Update(
		ctx context.Context,
		entity *customfield.CustomFieldDefinition,
	) (*customfield.CustomFieldDefinition, error)
	Delete(
		ctx context.Context,
		req GetCustomFieldDefinitionByIDRequest,
	) error
	CountByResourceType(
		ctx context.Context,
		req CountByResourceTypeRequest,
	) (int, error)
}
