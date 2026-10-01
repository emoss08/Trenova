package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ThreadSummaryRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListThreadSummariesRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
}

type ThreadSummaryRepository interface {
	CreateIfNewest(ctx context.Context, summary *conversation.Summary) (bool, error)
	Latest(ctx context.Context, req ThreadSummaryRequest) (*conversation.Summary, error)
	List(ctx context.Context, req ListThreadSummariesRequest) ([]*conversation.Summary, error)
}
