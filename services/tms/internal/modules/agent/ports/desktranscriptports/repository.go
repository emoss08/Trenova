package desktranscriptports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListConversationsRequest struct {
	TenantInfo pagination.TenantInfo
	Filter     desktranscript.ConversationFilter
	Window     desktranscript.Window
	Query      string
	Cursor     pagination.CursorInfo
}

type ConversationPage struct {
	Items       []*conversation.Thread
	EdgeCursors []string
	HasNextPage bool
	TotalCount  *int
	TotalExact  bool
}

type MatchConversationsRequest struct {
	TenantInfo pagination.TenantInfo
	Filter     desktranscript.ConversationFilter
	Window     desktranscript.Window
	Limit      int
}

type ConversationRef struct {
	ID     pulid.ID
	UserID pulid.ID
}

type ConversationsByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type GetConversationRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ThreadMessagesRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadIDs  []pulid.ID
	Limit      int
}

type ConversationRepository interface {
	List(ctx context.Context, req *ListConversationsRequest) (*ConversationPage, error)
	CountUpTo(ctx context.Context, req *MatchConversationsRequest) (int, error)
	Match(ctx context.Context, req *MatchConversationsRequest) ([]ConversationRef, error)
	Get(ctx context.Context, req GetConversationRequest) (*conversation.Thread, error)
	ListByIDs(ctx context.Context, req ConversationsByIDsRequest) ([]*conversation.Thread, error)
	CountMessages(ctx context.Context, req ConversationsByIDsRequest) (map[pulid.ID]int, error)
	ListMessages(ctx context.Context, req *ThreadMessagesRequest) ([]conversation.Message, error)
}

type GetExportRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListExportsRequest struct {
	Filter *pagination.QueryOptions
	Cursor pagination.CursorInfo
}

type ExportRepository interface {
	Create(ctx context.Context, export *desktranscript.Export) (*desktranscript.Export, error)
	Update(ctx context.Context, export *desktranscript.Export) (*desktranscript.Export, error)
	GetByID(ctx context.Context, req GetExportRequest) (*desktranscript.Export, error)
	ListConnection(
		ctx context.Context,
		req *ListExportsRequest,
	) (*pagination.CursorListResult[*desktranscript.Export], error)
	ListExpired(ctx context.Context, before int64, limit int) ([]*desktranscript.Export, error)
	ListStale(ctx context.Context, updatedBefore int64, limit int) ([]*desktranscript.Export, error)
}
