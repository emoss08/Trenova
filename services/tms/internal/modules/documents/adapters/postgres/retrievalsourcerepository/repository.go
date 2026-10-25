package retrievalsourcerepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval/airetrievaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
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

	anyWordSimple = "replace(replace(websearch_to_tsquery('simple', ?)::text, ' & ', ' | '), " +
		"' | !', ' & !')::tsquery"
	anyWordEnglish = "replace(replace(websearch_to_tsquery('english', ?)::text, ' & ', ' | '), " +
		"' | !', ' & !')::tsquery"
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

		m := agentdb.Memories
		memories, err := m.Where(m.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read memories to index: %w", err)
		}
		if memories == nil {
			memories = []*agent.Memory{}
		}

		return memories, nil
	})
}

func (r *repository) GetDocuments(
	ctx context.Context,
	req *agentports.RetrievalDocumentsRequest,
) ([]*agentports.RetrievalDocumentSource, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentports.RetrievalDocumentSource, error) {
			ids, err := sourceIDs(req.TenantInfo, req.IDs)
			if err != nil || len(ids) == 0 {
				return []*agentports.RetrievalDocumentSource{}, err
			}

			db := r.db.Keel(ctx)
			d := documentdb.Documents
			docs, err := d.Where(d.ID.In(ids...)).Limit(len(ids)).List(ctx, db)
			if err != nil {
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

			dc := documentcontentdb.Contents
			contents, err := dc.Where(dc.DocumentID.In(found...)).Limit(len(found)).List(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("read document contents: %w", err)
			}
			for _, content := range contents {
				if source, ok := sources[content.DocumentID]; ok {
					source.Content = content
				}
			}

			if req.IncludePages {
				if err = attachPages(ctx, db, found, sources); err != nil {
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

func attachPages(
	ctx context.Context,
	db *keel.DB,
	documentIDs []pulid.ID,
	sources map[pulid.ID]*agentports.RetrievalDocumentSource,
) error {
	p := documentcontentdb.Pages
	pages, err := p.Where(p.DocumentID.In(documentIDs...)).
		Order(p.DocumentID.Asc(), p.PageNumber.Asc()).
		All(ctx, db)
	if err != nil {
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

			m := inboundmessagedb.InboundMessages
			messages, err := m.Where(m.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("read inbound messages: %w", err)
			}
			if messages == nil {
				messages = []*inboundmessage.InboundMessage{}
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

		db := r.db.Keel(ctx)
		e := airetrievaldb.IndexEntries
		keys, err := keel.Into[string](ctx, db, e.Pick(e.ModelKey).Group(e.ModelKey))
		if err != nil {
			return nil, fmt.Errorf("list retrieval model keys: %w", err)
		}

		if req.IncludeEmbeddings {
			emb := airetrievaldb.Embeddings
			embedded, keysErr := keel.Into[string](
				ctx,
				db,
				emb.Pick(emb.ModelKey).Group(emb.ModelKey),
			)
			if keysErr != nil {
				return nil, fmt.Errorf("list retrieval model keys: %w", keysErr)
			}
			keys = append(keys, embedded...)
		}

		slices.Sort(keys)

		return slices.Compact(keys), nil
	})
}

func (r *repository) ListSourceIDs(
	ctx context.Context,
	req *agentports.ListRetrievalSourceIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}

		table, err := sourceTableFor(req.SourceType)
		if err != nil {
			return nil, err
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		limit = intutils.Clamp(limit, 1, maxListLimit)

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), table.ids(req.AfterID, limit))
		if err != nil {
			return nil, fmt.Errorf("list %s sources: %w", req.SourceType, err)
		}
		if ids == nil {
			ids = []pulid.ID{}
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

			hits, err := keel.Into[agentports.RetrievalKeywordHit](
				ctx,
				r.db.Keel(ctx),
				searchDocumentsQuery(plan),
			)
			if err != nil {
				return nil, fmt.Errorf("search documents by keyword: %w", err)
			}
			if hits == nil {
				hits = []agentports.RetrievalKeywordHit{}
			}

			return hits, nil
		},
	)
}

func searchDocumentsQuery(plan keywordPlan) *keel.Query[document.Document] {
	d := documentdb.Documents
	dc := documentcontentdb.Contents
	simple := keel.Raw(anyWordSimple, plan.query)
	english := keel.Raw(anyWordEnglish, plan.query)
	rank := keel.Raw(
		"GREATEST(ts_rank_cd(?, ?), COALESCE(ts_rank_cd(?, ?), 0))",
		d.SearchVector, simple, dc.SearchVector, english,
	)

	return d.LeftJoin(dc, dc.DocumentID.EqCol(d.ID)).
		Pick(d.ID, rank).
		Where(
			d.IsCurrentVersion.IsTrue(),
			d.Status.NotIn(document.UnsearchableStatuses()...),
			keel.Or(
				keel.Raw("? @@ ?", d.SearchVector, simple),
				keel.Raw("? @@ ?", dc.SearchVector, english),
			),
		).
		Order(keel.Desc(rank), d.ID.Asc()).
		Limit(plan.limit)
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

			hits, err := keel.Into[agentports.RetrievalKeywordHit](
				ctx,
				r.db.Keel(ctx),
				searchMessagesQuery(plan, req.Messages),
			)
			if err != nil {
				return nil, fmt.Errorf("search inbound messages by keyword: %w", err)
			}
			if hits == nil {
				hits = []agentports.RetrievalKeywordHit{}
			}

			return hits, nil
		},
	)
}

func searchMessagesQuery(
	plan keywordPlan,
	filter agentports.InboundMessageFilter,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	english := keel.Raw(anyWordEnglish, plan.query)
	rank := keel.Raw("ts_rank_cd(?, ?)", m.SearchVector, english)

	return m.Pick(m.ID, rank).
		Where(
			keel.Raw("? @@ ?", m.SearchVector, english),
			keel.When(len(filter.Statuses) > 0, m.Status.In(filter.Statuses...)),
			keel.When(filter.Classification != "", m.Classification.Eq(filter.Classification)),
			keel.When(filter.MailboxID.IsNotNil(), m.MailboxID.Eq(filter.MailboxID)),
		).
		Order(keel.Desc(rank), m.ReceivedAt.Desc(), m.ID.Asc()).
		Limit(plan.limit)
}
