package retrievalquery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/vectorutils"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	catalogCacheSize    = 16
	catalogLoadTimeout  = 45 * time.Second
	catalogWaitTimeout  = 5 * time.Second
	catalogRetryBackoff = time.Minute
)

var _ serviceports.CatalogVectorIndex = (*CatalogIndex)(nil)

type CatalogParams struct {
	fx.In

	Logger     *zap.Logger
	Repository repositories.AIRetrievalRepository
	Embeddings serviceports.EmbeddingService `optional:"true"`
}

type CatalogIndex struct {
	l           *zap.Logger
	repo        repositories.AIRetrievalRepository
	embeddings  serviceports.EmbeddingService
	entries     *lru.Cache[catalogKey, *catalogVectors]
	flights     singleflight.Group
	now         func() time.Time
	waitTimeout time.Duration

	mu      sync.Mutex
	retryAt map[catalogKey]time.Time
	pruned  map[catalogKey]struct{}
}

type catalogKey struct {
	modelKey string
	corpus   airetrieval.CatalogCorpus
}

func (k catalogKey) String() string { return k.modelKey + "/" + k.corpus.String() }

type catalogVectors struct {
	dimensions int
	items      map[string]catalogVector
}

type catalogVector struct {
	hash string
	unit []float32
}

type catalogLoad struct {
	vectors *catalogVectors
	reason  airetrieval.UnavailableReason
}

func NewCatalogIndex(p CatalogParams) *CatalogIndex {
	entries, err := lru.New[catalogKey, *catalogVectors](catalogCacheSize)
	if err != nil {
		panic(fmt.Sprintf("catalog vector cache: %v", err))
	}

	return &CatalogIndex{
		l:           p.Logger.Named("service.retrievalquery.catalog"),
		repo:        p.Repository,
		embeddings:  p.Embeddings,
		entries:     entries,
		now:         time.Now,
		waitTimeout: catalogWaitTimeout,
		retryAt:     make(map[catalogKey]time.Time, catalogCacheSize),
		pruned:      make(map[catalogKey]struct{}, catalogCacheSize),
	}
}

func (c *CatalogIndex) Similarities(
	ctx context.Context,
	req *serviceports.CatalogSimilarityRequest,
) (serviceports.CatalogSimilarities, error) {
	if err := validateCatalogRequest(req); err != nil {
		return serviceports.CatalogSimilarities{}, err
	}

	query, err := vectorutils.Normalized(req.Query.Vector)
	if err != nil {
		return serviceports.CatalogSimilarities{}, fmt.Errorf(
			"%w: %w", serviceports.ErrCatalogQueryNotUsable, err,
		)
	}

	key := catalogKey{modelKey: req.Query.ModelKey, corpus: req.Corpus}
	vectors, reason := c.vectorsFor(ctx, key, req)
	if vectors == nil {
		return serviceports.CatalogSimilarities{Reason: reason}, nil
	}

	byKey := make(map[string]float64, len(req.Items))
	for _, item := range req.Items {
		stored := vectors.items[item.Key]
		similarity, dotErr := vectorutils.Dot(stored.unit, query)
		if dotErr != nil {
			return serviceports.CatalogSimilarities{}, fmt.Errorf(
				"compare %s with the query: %w", item.Key, dotErr,
			)
		}
		byKey[item.Key] = similarity
	}

	return serviceports.CatalogSimilarities{Available: true, ByKey: byKey}, nil
}

func (c *CatalogIndex) vectorsFor(
	ctx context.Context,
	key catalogKey,
	req *serviceports.CatalogSimilarityRequest,
) (*catalogVectors, airetrieval.UnavailableReason) {
	if cached, ok := c.entries.Get(key); ok && covers(cached, req) {
		return cached, ""
	}
	if c.coolingDown(key) {
		return nil, airetrieval.UnavailableReasonProviderFailed
	}

	snapshot := *req
	flight := c.flights.DoChan(key.String(), func() (any, error) {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), catalogLoadTimeout)
		defer cancel()

		return c.load(loadCtx, key, &snapshot), nil
	})

	wait := time.NewTimer(c.waitTimeout)
	defer wait.Stop()

	select {
	case result := <-flight:
		loaded, _ := result.Val.(catalogLoad)
		if loaded.vectors != nil && covers(loaded.vectors, req) {
			return loaded.vectors, ""
		}
		if loaded.reason != "" {
			return nil, loaded.reason
		}

		return nil, airetrieval.UnavailableReasonProviderFailed
	case <-wait.C:
		return nil, airetrieval.UnavailableReasonQueryTimeout
	case <-ctx.Done():
		return nil, airetrieval.UnavailableReasonQueryTimeout
	}
}

func (c *CatalogIndex) load(
	ctx context.Context,
	key catalogKey,
	req *serviceports.CatalogSimilarityRequest,
) catalogLoad {
	dimensions := req.Query.Dimensions
	merged := &catalogVectors{
		dimensions: dimensions,
		items:      make(map[string]catalogVector, len(req.Items)),
	}
	if cached, ok := c.entries.Get(key); ok && cached.dimensions == dimensions {
		for itemKey, stored := range cached.items {
			merged.items[itemKey] = stored
		}
	}

	absent := missingItems(merged, req.Items)
	if len(absent) > 0 {
		if err := c.readStored(ctx, key, merged, absent); err != nil {
			c.l.Warn("could not read stored catalog embeddings; ranking by keyword",
				zap.String("catalog", key.String()), zap.Error(err))
			c.backOff(key)

			return catalogLoad{reason: airetrieval.UnavailableReasonProviderFailed}
		}
		absent = missingItems(merged, req.Items)
	}

	if len(absent) > 0 {
		if reason := c.embedAbsent(ctx, key, req, merged, absent); reason != "" {
			c.entries.Add(key, merged)
			c.backOff(key)

			return catalogLoad{reason: reason}
		}
	}

	c.entries.Add(key, merged)
	c.clearBackOff(key)
	c.pruneOnce(ctx, key, req.Items)

	return catalogLoad{vectors: merged}
}

func (c *CatalogIndex) readStored(
	ctx context.Context,
	key catalogKey,
	merged *catalogVectors,
	absent []serviceports.EmbeddingCatalogItem,
) error {
	refs := make([]airetrieval.CatalogItemRef, 0, len(absent))
	for _, item := range absent {
		refs = append(refs, airetrieval.CatalogItemRef{
			ItemKey:     item.Key,
			ContentHash: item.ContentHash,
		})
	}

	rows, err := c.repo.GetCatalogEmbeddings(ctx, repositories.GetCatalogEmbeddingsRequest{
		ModelKey: key.modelKey,
		Corpus:   key.corpus,
		Items:    refs,
	})
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row == nil || row.Dimensions != merged.dimensions {
			continue
		}
		unit, normErr := vectorutils.Normalized(row.Vector())
		if normErr != nil {
			continue
		}
		merged.items[row.ItemKey] = catalogVector{hash: row.ContentHash, unit: unit}
	}

	return nil
}

func (c *CatalogIndex) embedAbsent(
	ctx context.Context,
	key catalogKey,
	req *serviceports.CatalogSimilarityRequest,
	merged *catalogVectors,
	absent []serviceports.EmbeddingCatalogItem,
) airetrieval.UnavailableReason {
	if c.embeddings == nil {
		return airetrieval.UnavailableReasonNoProvider
	}

	inputs := make([]string, 0, len(absent))
	for _, item := range absent {
		inputs = append(inputs, item.Text)
	}

	result, err := c.embeddings.Embed(ctx, &serviceports.EmbedRequest{
		TenantInfo: req.TenantInfo,
		Purpose:    serviceports.EmbeddingPurposeDocument,
		Inputs:     inputs,
		ModelKey:   key.modelKey,
		Surface:    aiusage.SurfaceIndexing,
	})
	if err != nil {
		reason := EmbedFailureReason(err)
		c.l.Warn("could not embed the catalog; ranking by keyword until it can be",
			zap.String("catalog", key.String()),
			zap.Int("items", len(absent)),
			zap.String("reason", reason.String()),
			zap.Error(err),
		)

		return reason
	}
	if len(result.Vectors) != len(absent) {
		c.l.Error("the embedding provider returned the wrong number of catalog vectors",
			zap.String("catalog", key.String()),
			zap.Int("want", len(absent)),
			zap.Int("got", len(result.Vectors)),
		)

		return airetrieval.UnavailableReasonProviderFailed
	}

	put := make([]repositories.CatalogEmbeddingInput, 0, len(absent))
	for idx, item := range absent {
		vector := result.Vectors[idx]
		if len(vector) != merged.dimensions {
			c.l.Error("the embedding provider returned a catalog vector of another size",
				zap.String("catalog", key.String()),
				zap.Int("want", merged.dimensions),
				zap.Int("got", len(vector)),
			)

			return airetrieval.UnavailableReasonProviderFailed
		}
		unit, normErr := vectorutils.Normalized(vector)
		if normErr != nil {
			c.l.Error("the embedding provider returned an unusable catalog vector",
				zap.String("catalog", key.String()),
				zap.String("item", item.Key),
				zap.Error(normErr),
			)

			return airetrieval.UnavailableReasonProviderFailed
		}
		merged.items[item.Key] = catalogVector{hash: item.ContentHash, unit: unit}
		put = append(put, repositories.CatalogEmbeddingInput{
			ItemKey:     item.Key,
			ContentHash: item.ContentHash,
			Vector:      vector,
		})
	}

	if _, err = c.repo.PutCatalogEmbeddings(ctx, repositories.PutCatalogEmbeddingsRequest{
		ModelKey:   key.modelKey,
		Corpus:     key.corpus,
		Dimensions: merged.dimensions,
		Items:      put,
	}); err != nil {
		c.l.Warn("could not store catalog embeddings; they are kept in memory only",
			zap.String("catalog", key.String()), zap.Error(err))
	}

	return ""
}

func (c *CatalogIndex) pruneOnce(
	ctx context.Context,
	key catalogKey,
	items []serviceports.EmbeddingCatalogItem,
) {
	c.mu.Lock()
	if _, done := c.pruned[key]; done {
		c.mu.Unlock()
		return
	}
	c.pruned[key] = struct{}{}
	c.mu.Unlock()

	keep := make([]airetrieval.CatalogItemRef, 0, len(items))
	for _, item := range items {
		keep = append(keep, airetrieval.CatalogItemRef{
			ItemKey:     item.Key,
			ContentHash: item.ContentHash,
		})
	}

	removed, err := c.repo.PruneCatalogEmbeddings(ctx, repositories.PruneCatalogEmbeddingsRequest{
		ModelKey: key.modelKey,
		Corpus:   key.corpus,
		Keep:     keep,
	})
	if err != nil {
		c.l.Warn("could not prune retired catalog embeddings",
			zap.String("catalog", key.String()), zap.Error(err))
		return
	}
	if removed > 0 {
		c.l.Info("pruned retired catalog embeddings",
			zap.String("catalog", key.String()), zap.Int("removed", removed))
	}
}

func (c *CatalogIndex) coolingDown(key catalogKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	until, ok := c.retryAt[key]

	return ok && c.now().Before(until)
}

func (c *CatalogIndex) backOff(key catalogKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.retryAt[key] = c.now().Add(catalogRetryBackoff)
}

func (c *CatalogIndex) clearBackOff(key catalogKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.retryAt, key)
}

func covers(vectors *catalogVectors, req *serviceports.CatalogSimilarityRequest) bool {
	if vectors == nil || vectors.dimensions != req.Query.Dimensions {
		return false
	}

	for _, item := range req.Items {
		stored, ok := vectors.items[item.Key]
		if !ok || stored.hash != item.ContentHash {
			return false
		}
	}

	return true
}

func missingItems(
	vectors *catalogVectors,
	items []serviceports.EmbeddingCatalogItem,
) []serviceports.EmbeddingCatalogItem {
	absent := make([]serviceports.EmbeddingCatalogItem, 0, len(items))
	for _, item := range items {
		stored, ok := vectors.items[item.Key]
		if ok && stored.hash == item.ContentHash {
			continue
		}
		absent = append(absent, item)
	}

	return absent
}

func validateCatalogRequest(req *serviceports.CatalogSimilarityRequest) error {
	if req == nil || !req.Corpus.IsValid() || len(req.Items) == 0 {
		return serviceports.ErrCatalogRequestInvalid
	}
	if !req.Query.Usable() {
		return serviceports.ErrCatalogQueryNotUsable
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		if item.Key == "" || item.ContentHash == "" {
			return serviceports.ErrCatalogRequestInvalid
		}
		if len(item.Key) > airetrieval.MaxCatalogItemChars {
			return fmt.Errorf("%w: %s", serviceports.ErrCatalogItemKeyTooLong, item.Key)
		}
		if _, dup := seen[item.Key]; dup {
			return fmt.Errorf("%w: %s", serviceports.ErrCatalogItemKeyRepeated, item.Key)
		}
		seen[item.Key] = struct{}{}
	}

	return nil
}
