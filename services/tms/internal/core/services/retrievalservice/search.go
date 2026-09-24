package retrievalservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultSearchLimit = 5
	MaxSearchLimit     = 50
	maxQueryRunes      = 500

	documentTextField   = "contentText"
	documentNameField   = "originalName"
	messageSubjectField = "subject"
	messageSenderField  = "fromAddress"
	messageBodyField    = "textBody"
)

var _ serviceports.RetrievalSearcher = (*Searcher)(nil)

type SearcherParams struct {
	fx.In

	Logger     *zap.Logger
	Repo       repositories.AIRetrievalRepository
	Sources    repositories.RetrievalSourceRepository
	Registry   *permission.Registry
	Vectorizer serviceports.QueryVectorizer `optional:"true"`
}

type Searcher struct {
	l          *zap.Logger
	repo       repositories.AIRetrievalRepository
	sources    repositories.RetrievalSourceRepository
	registry   *permission.Registry
	vectorizer serviceports.QueryVectorizer
	tuning     SearchTuning
}

func NewSearcher(p SearcherParams) *Searcher {
	registry := p.Registry
	if registry == nil {
		registry = permission.NewRegistry()
	}

	return &Searcher{
		l:          p.Logger.Named("service.retrieval-search"),
		repo:       p.Repo,
		sources:    p.Sources,
		registry:   registry,
		vectorizer: p.Vectorizer,
		tuning:     DefaultSearchTuning(),
	}
}

func AsSearcher(s *Searcher) serviceports.RetrievalSearcher { return s }

func (s *Searcher) WithTuning(tuning SearchTuning) *Searcher {
	tuned := *s
	tuned.tuning = tuning.normalized()

	return &tuned
}

type searchPlan struct {
	query      string
	terms      []string
	limit      int
	candidates int
}

func (s *Searcher) plan(req *serviceports.RetrievalSearchRequest) (searchPlan, error) {
	if req == nil || req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return searchPlan{}, errortypes.NewValidationError(
			"tenant", errortypes.ErrRequired, "Organization and business unit are required")
	}
	if req.Access == nil {
		return searchPlan{}, errortypes.NewAuthorizationError(
			"Searching needs to know who is asking")
	}

	query := stringutils.TruncateRunes(strings.TrimSpace(req.Query), maxQueryRunes)
	if query == "" {
		return searchPlan{}, errortypes.NewValidationError(
			"query", errortypes.ErrRequired, "Say what to search for")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	limit = intutils.Clamp(limit, 1, MaxSearchLimit)

	return searchPlan{
		query:      query,
		terms:      QueryTerms(query),
		limit:      limit,
		candidates: s.tuning.candidates(limit),
	}, nil
}

type keywordLeg func(
	context.Context,
	repositories.RetrievalKeywordSearchRequest,
) ([]repositories.RetrievalKeywordHit, error)

func (s *Searcher) legs(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
	plan searchPlan,
	sourceType airetrieval.SourceType,
	keyword keywordLeg,
) ([]FusedHit, serviceports.RetrievalSemantics, error) {
	var (
		keywordHits []repositories.RetrievalKeywordHit
		vectorHits  []repositories.VectorSearchHit
		semantics   serviceports.RetrievalSemantics
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		hits, err := keyword(groupCtx, repositories.RetrievalKeywordSearchRequest{
			TenantInfo: req.TenantInfo,
			Query:      plan.query,
			Limit:      plan.candidates,
		})
		if err != nil {
			return fmt.Errorf("search %s by keyword: %w", sourceType, err)
		}
		keywordHits = hits
		return nil
	})
	group.Go(func() error {
		vectorHits, semantics = s.vectorLeg(groupCtx, req, plan, sourceType)
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, semantics, err
	}

	return Fuse(keywordHits, vectorHits, s.tuning), semantics, nil
}

func (s *Searcher) vectorLeg(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
	plan searchPlan,
	sourceType airetrieval.SourceType,
) ([]repositories.VectorSearchHit, serviceports.RetrievalSemantics) {
	if s.vectorizer == nil {
		return nil, serviceports.RetrievalSemantics{Reason: airetrieval.UnavailableReasonNoProvider}
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		s.l.Warn("retrieval settings could not be read; searching by keyword only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		return nil, serviceports.RetrievalSemantics{
			Reason: airetrieval.UnavailableReasonProviderFailed,
		}
	}
	if !settings.SourceEnabled(sourceType) {
		return nil, serviceports.RetrievalSemantics{Reason: airetrieval.UnavailableReasonDisabled}
	}

	vector, err := s.vectorizer.Vectorize(ctx, &serviceports.QueryVectorRequest{
		TenantInfo:  req.TenantInfo,
		Text:        plan.query,
		Attribution: req.Attribution,
	})
	if err != nil {
		s.l.Warn("query could not be embedded; searching by keyword only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		return nil, serviceports.RetrievalSemantics{
			Reason: airetrieval.UnavailableReasonProviderFailed,
		}
	}
	if !vector.Usable() {
		reason := vector.Reason
		if reason == "" {
			reason = airetrieval.UnavailableReasonNotIndexed
		}
		return nil, serviceports.RetrievalSemantics{Reason: reason}
	}

	hits, err := s.repo.Search(ctx, repositories.VectorSearchRequest{
		TenantInfo:  req.TenantInfo,
		SourceTypes: []airetrieval.SourceType{sourceType},
		ModelKey:    vector.ModelKey,
		Dimensions:  vector.Dimensions,
		Query:       vector.Vector,
		Limit:       plan.candidates,
	})
	if err != nil {
		var unavailable *airetrieval.UnavailableError
		if errors.As(err, &unavailable) {
			return nil, serviceports.RetrievalSemantics{Reason: unavailable.Reason}
		}
		s.l.Warn("vector search failed; searching by keyword only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		return nil, serviceports.RetrievalSemantics{
			Reason: airetrieval.UnavailableReasonProviderFailed,
		}
	}

	return hits, serviceports.RetrievalSemantics{Used: true}
}

func fusedIDs(hits []FusedHit) []pulid.ID {
	ids := make([]pulid.ID, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ID)
	}

	return ids
}

func (s *Searcher) SearchDocuments(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
) (*serviceports.DocumentSearchResult, error) {
	plan, err := s.plan(req)
	if err != nil {
		return nil, err
	}

	result := &serviceports.DocumentSearchResult{Hits: []serviceports.DocumentSearchHit{}}
	if !req.Access.MayReadResource(ctx, permission.ResourceDocument) {
		return result, nil
	}

	fused, semantics, err := s.legs(
		ctx, req, plan, airetrieval.SourceTypeDocument, s.sources.SearchDocuments)
	result.Semantics = semantics
	if err != nil || len(fused) == 0 {
		return result, err
	}

	kept, err := s.readableDocuments(ctx, req, plan, fused)
	if err != nil {
		return nil, err
	}
	if len(kept) == 0 {
		return result, nil
	}

	sources, err := s.sources.GetDocuments(ctx, &repositories.RetrievalDocumentsRequest{
		TenantInfo:   req.TenantInfo,
		IDs:          fusedIDs(kept),
		IncludePages: true,
	})
	if err != nil {
		return nil, fmt.Errorf("read matched documents: %w", err)
	}
	byID := make(map[pulid.ID]*repositories.RetrievalDocumentSource, len(sources))
	for _, source := range sources {
		byID[source.Document.ID] = source
	}

	showsName := req.Access.ShowsField(ctx, permission.ResourceDocument, documentNameField)
	showsText := req.Access.ShowsField(ctx, permission.ResourceDocument, documentTextField)
	for _, hit := range kept {
		source, ok := byID[hit.ID]
		if !ok {
			continue
		}
		result.Hits = append(result.Hits, s.documentHit(ctx, req, plan, hit, source,
			showsName, showsText))
	}

	return result, nil
}

func (s *Searcher) readableDocuments(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
	plan searchPlan,
	fused []FusedHit,
) ([]FusedHit, error) {
	sources, err := s.sources.GetDocuments(ctx, &repositories.RetrievalDocumentsRequest{
		TenantInfo: req.TenantInfo,
		IDs:        fusedIDs(fused),
	})
	if err != nil {
		return nil, fmt.Errorf("read matched documents: %w", err)
	}

	candidates := make([]*document.Document, 0, len(sources))
	for _, source := range sources {
		if source.Document == nil || !source.Document.Searchable() ||
			!s.registry.HasResource(source.Document.OwnerResource().String()) {
			continue
		}
		candidates = append(candidates, source.Document)
	}

	readable, err := req.Access.ReadableDocuments(ctx, candidates)
	if err != nil {
		return nil, fmt.Errorf("check which matched documents may be read: %w", err)
	}

	kept := make([]FusedHit, 0, plan.limit)
	for _, hit := range fused {
		if !readable[hit.ID] {
			continue
		}
		kept = append(kept, hit)
		if len(kept) == plan.limit {
			break
		}
	}

	return kept, nil
}

func (s *Searcher) documentHit(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
	plan searchPlan,
	hit FusedHit,
	source *repositories.RetrievalDocumentSource,
	showsName, showsText bool,
) serviceports.DocumentSearchHit {
	out := serviceports.DocumentSearchHit{
		Document:      source.Document,
		ShowsFileName: showsName,
		Match:         hit.Match(),
		Score:         hit.Score,
		Similarity:    hit.Similarity,
	}
	if !showsText || !req.Access.ShowsRecordText(ctx, source.Document.OwnerResource()) {
		return out
	}

	chunks := ChunkDocument(source)
	preferred := -1
	if hit.VectorRank > 0 {
		preferred = hit.ChunkIndex
	}
	if hit.KeywordRank == 0 && preferred >= 0 && preferred < len(chunks) &&
		chunks[preferred].Body != "" {
		out.Page = chunks[preferred].Page
		out.Snippet = Snippet(chunks[preferred].Body, plan.terms)
		return out
	}

	if idx, ok := BestChunk(chunks, plan.terms, preferred); ok {
		out.Page = chunks[idx].Page
		out.Snippet = Snippet(chunks[idx].Body, plan.terms)
	}

	return out
}

func (s *Searcher) SearchInboundMessages(
	ctx context.Context,
	req *serviceports.RetrievalSearchRequest,
) (*serviceports.InboundMessageSearchResult, error) {
	plan, err := s.plan(req)
	if err != nil {
		return nil, err
	}

	result := &serviceports.InboundMessageSearchResult{
		Hits: []serviceports.InboundMessageSearchHit{},
	}
	if !req.Access.MayReadResource(ctx, permission.ResourceInboundMessage) {
		return result, nil
	}

	fused, semantics, err := s.legs(
		ctx, req, plan, airetrieval.SourceTypeInboundMessage, s.sources.SearchInboundMessages)
	result.Semantics = semantics
	if err != nil || len(fused) == 0 {
		return result, err
	}

	messages, err := s.sources.GetInboundMessages(ctx, repositories.RetrievalSourcesRequest{
		TenantInfo: req.TenantInfo,
		IDs:        fusedIDs(fused),
	})
	if err != nil {
		return nil, fmt.Errorf("read matched inbound messages: %w", err)
	}
	byID := make(map[pulid.ID]*inboundmessage.InboundMessage, len(messages))
	for _, message := range messages {
		byID[message.ID] = message
	}

	visible := messageVisibility{
		subject: req.Access.ShowsField(ctx, permission.ResourceInboundMessage, messageSubjectField),
		sender:  req.Access.ShowsField(ctx, permission.ResourceInboundMessage, messageSenderField),
		body:    req.Access.ShowsField(ctx, permission.ResourceInboundMessage, messageBodyField),
	}
	for _, hit := range fused {
		message, ok := byID[hit.ID]
		if !ok || !req.Access.MayReadRecord(
			ctx, permission.ResourceInboundMessage, message.ID.String()) {
			continue
		}
		result.Hits = append(result.Hits, messageHit(plan, hit, message, visible))
		if len(result.Hits) == plan.limit {
			break
		}
	}

	return result, nil
}

type messageVisibility struct {
	subject bool
	sender  bool
	body    bool
}

func messageHit(
	plan searchPlan,
	hit FusedHit,
	message *inboundmessage.InboundMessage,
	visible messageVisibility,
) serviceports.InboundMessageSearchHit {
	out := serviceports.InboundMessageSearchHit{
		Message:    message,
		Match:      hit.Match(),
		Score:      hit.Score,
		Similarity: hit.Similarity,
	}
	if visible.subject {
		out.Subject = message.Subject
	}
	if visible.sender {
		out.From = emailSender(message)
	}
	if !visible.body {
		return out
	}

	chunks := ChunkEmail(message)
	preferred := -1
	if hit.VectorRank > 0 {
		preferred = hit.ChunkIndex
	}
	if idx, ok := BestChunk(chunks, plan.terms, preferred); ok {
		out.Snippet = Snippet(chunks[idx].Body, plan.terms)
	}

	return out
}
