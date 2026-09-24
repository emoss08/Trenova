package retrievalsourcerepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbdialect"
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
	maxFallbackWords   = 8
	rankColumn         = "rank"
	sourceIDColumn     = "source_id"
)

var _ repositories.RetrievalSourceRepository = (*repository)(nil)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RetrievalSourceRepository {
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
	req repositories.RetrievalSourcesRequest,
) ([]*agent.Memory, error) {
	ids, err := sourceIDs(req.TenantInfo, req.IDs)
	if err != nil || len(ids) == 0 {
		return []*agent.Memory{}, err
	}

	memories := make([]*agent.Memory, 0, len(ids))
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&memories).
		Apply(buncolgen.MemoryApplyTenant(req.TenantInfo)).
		Where(buncolgen.MemoryColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read memories to index: %w", err)
	}

	return memories, nil
}

func (r *repository) GetDocuments(
	ctx context.Context,
	req repositories.RetrievalDocumentsRequest,
) ([]*repositories.RetrievalDocumentSource, error) {
	ids, err := sourceIDs(req.TenantInfo, req.IDs)
	if err != nil || len(ids) == 0 {
		return []*repositories.RetrievalDocumentSource{}, err
	}

	dba := r.db.DBForContext(ctx)
	docs := make([]*document.Document, 0, len(ids))
	if err = dba.NewSelect().
		Model(&docs).
		Apply(buncolgen.DocumentApplyTenant(req.TenantInfo)).
		Where(buncolgen.DocumentColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read documents: %w", err)
	}
	if len(docs) == 0 {
		return []*repositories.RetrievalDocumentSource{}, nil
	}

	found := make([]pulid.ID, 0, len(docs))
	sources := make(map[pulid.ID]*repositories.RetrievalDocumentSource, len(docs))
	for _, doc := range docs {
		found = append(found, doc.ID)
		sources[doc.ID] = &repositories.RetrievalDocumentSource{Document: doc}
	}

	contents := make([]*documentcontent.Content, 0, len(found))
	if err = dba.NewSelect().
		Model(&contents).
		Apply(buncolgen.ContentApplyTenant(req.TenantInfo)).
		Where(buncolgen.ContentColumns.DocumentID.In(), bun.List(found)).
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

	ordered := make([]*repositories.RetrievalDocumentSource, 0, len(docs))
	for _, doc := range docs {
		ordered = append(ordered, sources[doc.ID])
	}

	return ordered, nil
}

func (r *repository) attachPages(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	documentIDs []pulid.ID,
	sources map[pulid.ID]*repositories.RetrievalDocumentSource,
) error {
	cols := buncolgen.PageColumns
	pages := make([]*documentcontent.Page, 0, len(documentIDs))
	if err := dba.NewSelect().
		Model(&pages).
		Apply(buncolgen.PageApplyTenant(tenantInfo)).
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
	req repositories.RetrievalSourcesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	ids, err := sourceIDs(req.TenantInfo, req.IDs)
	if err != nil || len(ids) == 0 {
		return []*inboundmessage.InboundMessage{}, err
	}

	messages := make([]*inboundmessage.InboundMessage, 0, len(ids))
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&messages).
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo)).
		Where(buncolgen.InboundMessageColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read inbound messages: %w", err)
	}

	return messages, nil
}

func (r *repository) ListModelKeys(
	ctx context.Context,
	req repositories.ListRetrievalModelKeysRequest,
) ([]string, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}

	dba := r.db.DBForContext(ctx)
	entries := buncolgen.IndexEntryColumns
	q := dba.NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		ColumnExpr(entries.ModelKey.Qualified()).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		GroupExpr(entries.ModelKey.Qualified())

	if req.IncludeEmbeddings {
		embeddings := buncolgen.EmbeddingColumns
		q = q.UnionAll(dba.NewSelect().
			Model((*airetrieval.Embedding)(nil)).
			ColumnExpr(embeddings.ModelKey.Qualified()).
			Apply(buncolgen.EmbeddingApplyTenant(req.TenantInfo)).
			GroupExpr(embeddings.ModelKey.Qualified()))
	}

	keys := make([]string, 0, 4)
	if err := q.Scan(ctx, &keys); err != nil {
		return nil, fmt.Errorf("list retrieval model keys: %w", err)
	}
	slices.Sort(keys)

	return slices.Compact(keys), nil
}

type sourceTable struct {
	model       any
	id          buncolgen.Column
	applyTenant func(pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery
}

var sourceTables = map[airetrieval.SourceType]sourceTable{
	airetrieval.SourceTypeMemory: {
		model:       (*agent.Memory)(nil),
		id:          buncolgen.MemoryColumns.ID,
		applyTenant: buncolgen.MemoryApplyTenant,
	},
	airetrieval.SourceTypeDocument: {
		model:       (*document.Document)(nil),
		id:          buncolgen.DocumentColumns.ID,
		applyTenant: buncolgen.DocumentApplyTenant,
	},
	airetrieval.SourceTypeInboundMessage: {
		model:       (*inboundmessage.InboundMessage)(nil),
		id:          buncolgen.InboundMessageColumns.ID,
		applyTenant: buncolgen.InboundMessageApplyTenant,
	},
}

func (r *repository) ListSourceIDs(
	ctx context.Context,
	req repositories.ListRetrievalSourceIDsRequest,
) ([]pulid.ID, error) {
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
		Limit(limit)
	if req.AfterID.IsNotNil() {
		q = q.Where(table.id.Gt(), req.AfterID)
	}

	ids := make([]pulid.ID, 0, limit)
	if err := q.Scan(ctx, &ids); err != nil {
		return nil, fmt.Errorf("list %s sources: %w", req.SourceType, err)
	}

	return ids, nil
}

type keywordPlan struct {
	query string
	limit int
}

func planKeywordSearch(req repositories.RetrievalKeywordSearchRequest) (keywordPlan, bool, error) {
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

func matchesAnyWord(vector buncolgen.Column, config string) string {
	return vector.Expr("{} @@ " + anyWordQuery(config))
}

func rankAnyWord(vector buncolgen.Column, config string) string {
	return vector.Expr("ts_rank_cd({}, " + anyWordQuery(config) + ")")
}

func likeAnyWord(query string, columns ...buncolgen.Column) (string, []any) {
	words := strings.Fields(strings.ToLower(query))
	if len(words) > maxFallbackWords {
		words = words[:maxFallbackWords]
	}

	clauses := make([]string, 0, len(words)*len(columns))
	args := make([]any, 0, len(words)*len(columns))
	for _, word := range words {
		pattern := "%" + stringutils.EscapeLikePattern(word) + "%"
		for _, column := range columns {
			clauses = append(clauses, column.Expr("LOWER({}) LIKE ? ESCAPE '\\'"))
			args = append(args, pattern)
		}
	}

	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func (r *repository) SearchDocuments(
	ctx context.Context,
	req repositories.RetrievalKeywordSearchRequest,
) ([]repositories.RetrievalKeywordHit, error) {
	plan, ok, err := planKeywordSearch(req)
	if err != nil || !ok {
		return []repositories.RetrievalKeywordHit{}, err
	}

	docs := buncolgen.DocumentColumns
	content := buncolgen.ContentColumns
	dba := r.db.DBForContext(ctx)

	q := dba.NewSelect().
		Model((*document.Document)(nil)).
		ColumnExpr(docs.ID.As(sourceIDColumn)).
		Join("LEFT JOIN "+buncolgen.ContentTable.As(buncolgen.ContentTable.Alias)).
		JoinOn(content.DocumentID.EqColumn(docs.ID)).
		JoinOn(content.OrganizationID.EqColumn(docs.OrganizationID)).
		JoinOn(content.BusinessUnitID.EqColumn(docs.BusinessUnitID)).
		Apply(buncolgen.DocumentApplyTenant(req.TenantInfo)).
		Where(docs.IsCurrentVersion.IsTrue()).
		Where(docs.Status.NotIn(), bun.List(document.UnsearchableStatuses()))

	if dbdialect.FromBun(dba).Supports(dbdialect.CapFullTextSearch) {
		q = q.
			ColumnExpr(
				"GREATEST("+rankAnyWord(docs.SearchVector, "simple")+", COALESCE("+
					rankAnyWord(content.SearchVector, "english")+", 0)) AS "+rankColumn,
				plan.query, plan.query,
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where(matchesAnyWord(docs.SearchVector, "simple"), plan.query).
					WhereOr(matchesAnyWord(content.SearchVector, "english"), plan.query)
			}).
			OrderExpr(rankColumn + " DESC")
	} else {
		clause, args := likeAnyWord(plan.query,
			docs.OriginalName, docs.Description, content.ContentText)
		q = q.
			ColumnExpr("1.0 AS "+rankColumn).
			Where(clause, args...).
			OrderExpr(docs.UpdatedAt.OrderDesc())
	}

	hits := make([]repositories.RetrievalKeywordHit, 0, plan.limit)
	if err = q.OrderExpr(docs.ID.OrderAsc()).Limit(plan.limit).Scan(ctx, &hits); err != nil {
		return nil, fmt.Errorf("search documents by keyword: %w", err)
	}

	return hits, nil
}

func (r *repository) SearchInboundMessages(
	ctx context.Context,
	req repositories.RetrievalKeywordSearchRequest,
) ([]repositories.RetrievalKeywordHit, error) {
	plan, ok, err := planKeywordSearch(req)
	if err != nil || !ok {
		return []repositories.RetrievalKeywordHit{}, err
	}

	cols := buncolgen.InboundMessageColumns
	dba := r.db.DBForContext(ctx)

	q := dba.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(cols.ID.As(sourceIDColumn)).
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo))

	if dbdialect.FromBun(dba).Supports(dbdialect.CapFullTextSearch) {
		q = q.
			ColumnExpr(rankAnyWord(cols.SearchVector, "english")+" AS "+rankColumn, plan.query).
			Where(matchesAnyWord(cols.SearchVector, "english"), plan.query).
			OrderExpr(rankColumn + " DESC").
			OrderExpr(cols.ReceivedAt.OrderDesc())
	} else {
		clause, args := likeAnyWord(plan.query,
			cols.Subject, cols.FromName, cols.FromAddress, cols.TextBody)
		q = q.
			ColumnExpr("1.0 AS "+rankColumn).
			Where(clause, args...).
			OrderExpr(cols.ReceivedAt.OrderDesc())
	}

	hits := make([]repositories.RetrievalKeywordHit, 0, plan.limit)
	if err = q.OrderExpr(cols.ID.OrderAsc()).Limit(plan.limit).Scan(ctx, &hits); err != nil {
		return nil, fmt.Errorf("search inbound messages by keyword: %w", err)
	}

	return hits, nil
}
