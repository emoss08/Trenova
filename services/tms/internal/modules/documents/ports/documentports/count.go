package documentports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type CountDocumentsByResourceRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   string
}

type ResourceDocumentCounter interface {
	CountByResource(ctx context.Context, req *CountDocumentsByResourceRequest) (int, error)
}
