package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentPacketRulesRequest struct {
	Filter       *pagination.QueryOptions `json:"filter"`
	ResourceType string                   `json:"resourceType"`
}

type GetDocumentPacketRuleByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListDocumentPacketRulesByResourceRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceType string                `json:"resourceType"`
}

type DocumentPacketRuleRepository interface {
	List(ctx context.Context, req *ListDocumentPacketRulesRequest) (*pagination.ListResult[*documentpacketrule.Rule], error)
	GetByID(ctx context.Context, req GetDocumentPacketRuleByIDRequest) (*documentpacketrule.Rule, error)
	ListByResourceType(ctx context.Context, req *ListDocumentPacketRulesByResourceRequest) ([]*documentpacketrule.Rule, error)
	Create(ctx context.Context, entity *documentpacketrule.Rule) (*documentpacketrule.Rule, error)
	Update(ctx context.Context, entity *documentpacketrule.Rule) (*documentpacketrule.Rule, error)
	Delete(ctx context.Context, req GetDocumentPacketRuleByIDRequest) error
}
