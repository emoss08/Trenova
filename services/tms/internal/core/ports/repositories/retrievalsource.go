package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RetrievalSourcesRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type RetrievalDocumentsRequest struct {
	TenantInfo   pagination.TenantInfo
	IDs          []pulid.ID
	IncludePages bool
}

type RetrievalDocumentSource struct {
	Document *document.Document
	Content  *documentcontent.Content
	Pages    []*documentcontent.Page
}

type ListRetrievalSourceIDsRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	AfterID    pulid.ID
	Limit      int
}

type RetrievalKeywordSearchRequest struct {
	TenantInfo pagination.TenantInfo
	Query      string
	Limit      int
}

type RetrievalKeywordHit struct {
	SourceID pulid.ID `bun:"source_id"`
	Rank     float64  `bun:"rank"`
}

type ListRetrievalModelKeysRequest struct {
	TenantInfo        pagination.TenantInfo
	IncludeEmbeddings bool
}

type CountRetrievalSourcesRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
}

type AverageRetrievalSourceCharsRequest struct {
	TenantInfo pagination.TenantInfo
	SourceType airetrieval.SourceType
	Sample     int
}

type RetrievalSourceChars struct {
	Sampled      int     `bun:"sampled"`
	AverageChars float64 `bun:"average_chars"`
}

type RetrievalSourceRepository interface {
	CountSources(ctx context.Context, req CountRetrievalSourcesRequest) (int, error)
	AverageSourceChars(
		ctx context.Context,
		req AverageRetrievalSourceCharsRequest,
	) (RetrievalSourceChars, error)
	ListModelKeys(ctx context.Context, req ListRetrievalModelKeysRequest) ([]string, error)
	GetMemories(ctx context.Context, req RetrievalSourcesRequest) ([]*agent.Memory, error)
	GetDocuments(
		ctx context.Context,
		req *RetrievalDocumentsRequest,
	) ([]*RetrievalDocumentSource, error)
	GetInboundMessages(
		ctx context.Context,
		req RetrievalSourcesRequest,
	) ([]*inboundmessage.InboundMessage, error)
	ListSourceIDs(ctx context.Context, req *ListRetrievalSourceIDsRequest) ([]pulid.ID, error)
	SearchDocuments(
		ctx context.Context,
		req RetrievalKeywordSearchRequest,
	) ([]RetrievalKeywordHit, error)
	SearchInboundMessages(
		ctx context.Context,
		req RetrievalKeywordSearchRequest,
	) ([]RetrievalKeywordHit, error)
}
