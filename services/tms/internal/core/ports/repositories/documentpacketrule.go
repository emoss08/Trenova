package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentPacketRulesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
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
	List(
		ctx context.Context,
		req *ListDocumentPacketRulesRequest,
	) (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error)
	GetByID(
		ctx context.Context,
		req GetDocumentPacketRuleByIDRequest,
	) (*documentpacketrule.DocumentPacketRule, error)
	ListByResourceType(
		ctx context.Context,
		req *ListDocumentPacketRulesByResourceRequest,
	) ([]*documentpacketrule.DocumentPacketRule, error)
	Create(
		ctx context.Context,
		entity *documentpacketrule.DocumentPacketRule,
	) (*documentpacketrule.DocumentPacketRule, error)
	Update(
		ctx context.Context,
		entity *documentpacketrule.DocumentPacketRule,
	) (*documentpacketrule.DocumentPacketRule, error)
	Delete(ctx context.Context, req GetDocumentPacketRuleByIDRequest) error
}
