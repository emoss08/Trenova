package retrievalsourcerepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/agentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/airetrievalcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcontentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSourcesPerCall  = 1000
	defaultListLimit   = 500
	maxListLimit       = 5000
	defaultSearchLimit = 20
	maxSearchLimit     = 200
	maxQueryChars      = 500
	rankColumn         = "rank"
	sourceIDColumn     = "source_id"
)

var _ agentports.RetrievalSourceRepository = (*repository)(nil)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.RetrievalSourceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.retrieval-source-repository"),
	}
}

func validateTenant(tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return fmt.Errorf(
			"%w: organization and business unit are required",
			airetrieval.ErrInvalidStorageRequest,
		)
	}

	return nil
}

func sourceIDs(tenantInfo pagination.TenantInfo, ids []pulid.ID) ([]pulid.ID, error) {
	if err := validateTenant(tenantInfo); err != nil {
		return nil, err
	}
	if len(ids) > maxSourcesPerCall {
		return nil, fmt.Errorf(
			"%w: at most %d sources per call",
			airetrieval.ErrInvalidStorageRequest,
			maxSourcesPerCall,
		)
	}

	unique := sliceutils.Dedupe(ids)
	kept := make([]pulid.ID, 0, len(unique))
	for _, id := range unique {
		if id.IsNotNil() {
			kept = append(kept, id)
		}
	}

	return kept, nil
}

func (r *repository) GetMemories(
	ctx context.Context,
	req agentports.RetrievalSourcesRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		ids, err := sourceIDs(req.TenantInfo, req.IDs)
		if err != nil || len(ids) == 0 {
			return []*agent.Memory{}, err
		}

		memories := make([]*agent.Memory, 0, len(ids))
		if err = r.db.DBForContext(ctx).
			NewSelect().
			Model(&memories).
			Apply(agentcol.MemoryApplyTenant(req.TenantInfo)).
			Where(agentcol.MemoryColumns.ID.In(), bun.List(ids)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("read memories to index: %w", err)
		}

		return memories, nil
	})
}

func (r *repository) GetDocuments(
	ctx context.Context,
	req *agentports.RetrievalDocumentsRequest,
) ([]*agentports.RetrievalDocumentSource, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentports.RetrievalDocumentSource, error) {
			ids, err := sourceIDs(req.TenantInfo, req.IDs)
			if err != nil || len(ids) == 0 {
				return []*agentports.RetrievalDocumentSource{}, err
			}

			dba := r.db.DBForContext(ctx)
			docs := make([]*document.Document, 0, len(ids))
			if err = dba.NewSelect().
				Model(&docs).
				Apply(documentcol.DocumentApplyTenant(req.TenantInfo)).
				Where(documentcol.DocumentColumns.ID.In(), bun.List(ids)).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("read documents: %w", err)
			}
			if len(docs) == 0 {
				return []*agentports.RetrievalDocumentSource{}, nil
			}

			found := make([]pulid.ID, 0, len(docs))
			sources := make(map[pulid.ID]*agentports.RetrievalDocumentSource, len(docs))
			for _, doc := range docs {
				found = append(found, doc.ID)
				sources[doc.ID] = &agentports.RetrievalDocumentSource{Document: doc}
			}

			contents := make([]*documentcontent.Content, 0, len(found))
			if err = dba.NewSelect().
				Model(&contents).
				Apply(documentcontentcol.ContentApplyTenant(req.TenantInfo)).
				Where(documentcontentcol.ContentColumns.DocumentID.In(), bun.List(found)).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("read document contents: %w", err)
			}
			for _, content := range contents {
				if source, ok := sources[content.DocumentID]; ok {
					source.Content = content
				}
			}

			if req.IncludePages {
				if err = r.attachPages(ctx, dba, req.TenantInfo, found, sources); err != nil {
					return nil, err
				}
			}

			ordered := make([]*agentports.RetrievalDocumentSource, 0, len(docs))
			for _, doc := range docs {
				ordered = append(ordered, sources[doc.ID])
			}

			return ordered, nil
		},
	)
}

func (r *repository) attachPages(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	documentIDs []pulid.ID,
	sources map[pulid.ID]*agentports.RetrievalDocumentSource,
) error {
	cols := documentcontentcol.PageColumns
	pages := make([]*documentcontent.Page, 0, len(documentIDs))
	if err := dba.NewSelect().
		Model(&pages).
		Apply(documentcontentcol.PageApplyTenant(tenantInfo)).
		Where(cols.DocumentID.In(), bun.List(documentIDs)).
		OrderExpr(cols.DocumentID.OrderAsc()).
		OrderExpr(cols.PageNumber.OrderAsc()).
		Scan(ctx); err != nil {
		return fmt.Errorf("read document pages: %w", err)
	}

	for _, page := range pages {
		source, ok := sources[page.DocumentID]
		if !ok {
			continue
		}
		if source.Content != nil && page.DocumentContentID != source.Content.ID {
			continue
		}
		source.Pages = append(source.Pages, page)
	}

	return nil
}

func (r *repository) GetInboundMessages(
	ctx context.Context,
	req agentports.RetrievalSourcesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			ids, err := sourceIDs(req.TenantInfo, req.IDs)
			if err != nil || len(ids) == 0 {
				return []*inboundmessage.InboundMessage{}, err
			}

			messages := make([]*inboundmessage.InboundMessage, 0, len(ids))
			if err = r.db.DBForContext(ctx).
				NewSelect().
				Model(&messages).
				Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
				Where(inboundmessagecol.InboundMessageColumns.ID.In(), bun.List(ids)).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("read inbound messages: %w", err)
			}

			return messages, nil
		},
	)
}

func (r *repository) ListModelKeys(
	ctx context.Context,
	req agentports.ListRetrievalModelKeysRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}

		dba := r.db.DBForContext(ctx)
		entries := airetrievalcol.IndexEntryColumns
		q := dba.NewSelect().
			Model((*airetrieval.IndexEntry)(nil)).
			ColumnExpr(entries.ModelKey.Qualified()).
			Apply(airetrievalcol.IndexEntryApplyTenant(req.TenantInfo)).
			GroupExpr(entries.ModelKey.Qualified())

		if req.IncludeEmbeddings {
			embeddings := airetrievalcol.EmbeddingColumns
			q = q.UnionAll(dba.NewSelect().
				Model((*airetrieval.Embedding)(nil)).
				ColumnExpr(embeddings.ModelKey.Qualified()).
				Apply(airetrievalcol.EmbeddingApplyTenant(req.TenantInfo)).
				GroupExpr(embeddings.ModelKey.Qualified()))
		}

		keys := make([]string, 0, 4)
		if err := q.Scan(ctx, &keys); err != nil {
			return nil, fmt.Errorf("list retrieval model keys: %w", err)
		}
		slices.Sort(keys)

		return slices.Compact(keys), nil
	})
}

type sourceTable struct {
	model       any
	id          buncolgen.Column
	applyTenant func(pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery
}

var sourceTables = map[airetrieval.SourceType]sourceTable{
	airetrieval.SourceTypeMemory: {
		model:       (*agent.Memory)(nil),
		id:          agentcol.MemoryColumns.ID,
		applyTenant: agentcol.MemoryApplyTenant,
	},
	airetrieval.SourceTypeDocument: {
		model:       (*document.Document)(nil),
		id:          documentcol.DocumentColumns.ID,
		applyTenant: documentcol.DocumentApplyTenant,
	},
	airetrieval.SourceTypeInboundMessage: {
		model:       (*inboundmessage.InboundMessage)(nil),
		id:          inboundmessagecol.InboundMessageColumns.ID,
		applyTenant: inboundmessagecol.InboundMessageApplyTenant,
	},
}

func (r *repository) ListSourceIDs(
	ctx context.Context,
	req *agentports.ListRetrievalSourceIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}

		table, ok := sourceTables[req.SourceType]
		if !ok {
			return nil, fmt.Errorf(
				"%w: source type %q is not one retrieval indexes",
				airetrieval.ErrInvalidStorageRequest,
				req.SourceType,
			)
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		limit = intutils.Clamp(limit, 1, maxListLimit)

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(table.model).
			ColumnExpr(table.id.Qualified()).
			Apply(table.applyTenant(req.TenantInfo)).
			OrderExpr(table.id.OrderAsc()).
			Limit(int64(limit))
		if req.AfterID.IsNotNil() {
			q = q.Where(table.id.Gt(), req.AfterID)
		}

		ids := make([]pulid.ID, 0, limit)
		if err := q.Scan(ctx, &ids); err != nil {
			return nil, fmt.Errorf("list %s sources: %w", req.SourceType, err)
		}

		return ids, nil
	})
}

type keywordPlan struct {
	query string
	limit int
}

func planKeywordSearch(req agentports.RetrievalKeywordSearchRequest) (keywordPlan, bool, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return keywordPlan{}, false, err
	}

	query := stringutils.TruncateRunes(strings.TrimSpace(req.Query), maxQueryChars)
	if query == "" {
		return keywordPlan{}, false, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	return keywordPlan{query: query, limit: intutils.Clamp(limit, 1, maxSearchLimit)}, true, nil
}

func anyWordQuery(config string) string {
	return "replace(replace(websearch_to_tsquery('" + config + "', ?)::text, ' & ', ' | '), " +
		"' | !', ' & !')::tsquery"
}

func matchesAnyWord(vector *buncolgen.Column, config string) string {
	return vector.Expr("{} @@ " + anyWordQuery(config))
}

func rankAnyWord(vector *buncolgen.Column, config string) string {
	return vector.Expr("ts_rank_cd({}, " + anyWordQuery(config) + ")")
}

func (r *repository) SearchDocuments(
	ctx context.Context,
	req agentports.RetrievalKeywordSearchRequest,
) ([]agentports.RetrievalKeywordHit, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentports.RetrievalKeywordHit, error) {
			plan, ok, err := planKeywordSearch(req)
			if err != nil || !ok {
				return []agentports.RetrievalKeywordHit{}, err
			}

			docs := documentcol.DocumentColumns
			content := documentcontentcol.ContentColumns
			q := r.db.DBForContext(ctx).NewSelect().
				Model((*document.Document)(nil)).
				ColumnExpr(docs.ID.As(sourceIDColumn)).
				Join("LEFT JOIN "+documentcontentcol.ContentTable.As(documentcontentcol.ContentTable.Alias)).
				JoinOn(content.DocumentID.EqColumn(docs.ID)).
				JoinOn(content.OrganizationID.EqColumn(docs.OrganizationID)).
				JoinOn(content.BusinessUnitID.EqColumn(docs.BusinessUnitID)).
				Apply(documentcol.DocumentApplyTenant(req.TenantInfo)).
				Where(docs.IsCurrentVersion.IsTrue()).
				Where(docs.Status.NotIn(), bun.List(document.UnsearchableStatuses())).
				ColumnExpr(
					"GREATEST("+rankAnyWord(&docs.SearchVector, "simple")+", COALESCE("+
						rankAnyWord(&content.SearchVector, "english")+", 0)) AS "+rankColumn,
					plan.query, plan.query,
				).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.
						Where(matchesAnyWord(&docs.SearchVector, "simple"), plan.query).
						WhereOr(matchesAnyWord(&content.SearchVector, "english"), plan.query)
				}).
				OrderExpr(rankColumn + " DESC")

			hits := make([]agentports.RetrievalKeywordHit, 0, plan.limit)
			if err = q.OrderExpr(docs.ID.OrderAsc()).
				Limit(int64(plan.limit)).
				Scan(ctx, &hits); err != nil {
				return nil, fmt.Errorf("search documents by keyword: %w", err)
			}

			return hits, nil
		},
	)
}

func (r *repository) SearchInboundMessages(
	ctx context.Context,
	req agentports.RetrievalKeywordSearchRequest,
) ([]agentports.RetrievalKeywordHit, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentports.RetrievalKeywordHit, error) {
			plan, ok, err := planKeywordSearch(req)
			if err != nil || !ok {
				return []agentports.RetrievalKeywordHit{}, err
			}

			cols := inboundmessagecol.InboundMessageColumns
			q := r.db.DBForContext(ctx).NewSelect().
				Model((*inboundmessage.InboundMessage)(nil)).
				ColumnExpr(cols.ID.As(sourceIDColumn)).
				Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
				ColumnExpr(
					rankAnyWord(&cols.SearchVector, "english")+" AS "+rankColumn,
					plan.query,
				).
				Where(matchesAnyWord(&cols.SearchVector, "english"), plan.query).
				OrderExpr(rankColumn + " DESC").
				OrderExpr(cols.ReceivedAt.OrderDesc())
			if len(req.Messages.Statuses) > 0 {
				q = q.Where(cols.Status.In(), bun.List(req.Messages.Statuses))
			}
			if req.Messages.Classification != "" {
				q = q.Where(cols.Classification.Eq(), req.Messages.Classification)
			}
			if req.Messages.MailboxID.IsNotNil() {
				q = q.Where(cols.MailboxID.Eq(), req.Messages.MailboxID)
			}

			hits := make([]agentports.RetrievalKeywordHit, 0, plan.limit)
			if err = q.OrderExpr(cols.ID.OrderAsc()).
				Limit(int64(plan.limit)).
				Scan(ctx, &hits); err != nil {
				return nil, fmt.Errorf("search inbound messages by keyword: %w", err)
			}

			return hits, nil
		},
	)
}
