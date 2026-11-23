package documentports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type ResourceDocumentsChange struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   string
}

type ResourceDocumentsWatcher interface {
	ResourceDocumentsChanged(ctx context.Context, change ResourceDocumentsChange)
}
