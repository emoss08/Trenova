package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PageThreadKey struct {
	TenantInfo  pagination.TenantInfo
	UserID      pulid.ID
	Origin      conversation.ThreadOrigin
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
}

type ArchiveSubjectThreadsRequest struct {
	TenantInfo  pagination.TenantInfo
	Origin      conversation.ThreadOrigin
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
}

type PageThreadRepository interface {
	GetPageThread(ctx context.Context, key PageThreadKey) (*conversation.Thread, error)
	ClaimPageThread(ctx context.Context, thread *conversation.Thread) (*conversation.Thread, error)
	ArchiveSubjectThreads(ctx context.Context, req ArchiveSubjectThreadsRequest) (int, error)
}
