package airetrievalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval/airetrievaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pgvector/pgvector-go"
)

const maxCatalogItemsPerCall = 2000

func validateCatalogScope(modelKey string, corpus airetrieval.CatalogCorpus) error {
	if err := validateModelKey(modelKey); err != nil {
		return err
	}
	if !corpus.IsValid() {
		return invalid("corpus %q is not one retrieval indexes", corpus)
	}

	return nil
}

func validateCatalogRef(itemKey, contentHash string) error {
	if itemKey == "" || len(itemKey) > airetrieval.MaxCatalogItemChars {
		return invalid("an item key is 1 to %d characters", airetrieval.MaxCatalogItemChars)
	}

	return validateContentHash(contentHash)
}

func catalogRefsMatch(refs []airetrieval.CatalogItemRef) keel.Pred {
	c := airetrievaldb.CatalogEmbeddings
	keys := make([]string, 0, len(refs))
	hashes := make([]string, 0, len(refs))
	for _, ref := range refs {
		keys = append(keys, ref.ItemKey)
		hashes = append(hashes, ref.ContentHash)
	}

	return keel.Tuple(c.ItemKey, c.ContentHash).
		InUnnest(keel.TextArray(keys), keel.TextArray(hashes))
}

func catalogScope(
	modelKey string,
	corpus airetrieval.CatalogCorpus,
) *keel.Query[*airetrieval.CatalogEmbedding] {
	c := airetrievaldb.CatalogEmbeddings

	return c.Where(c.ModelKey.Eq(modelKey), c.Corpus.Eq(corpus))
}

func (r *repository) GetCatalogEmbeddings(
	ctx context.Context,
	req agentports.GetCatalogEmbeddingsRequest,
) ([]*airetrieval.CatalogEmbedding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*airetrieval.CatalogEmbedding, error) {
			if err := validateCatalogScope(req.ModelKey, req.Corpus); err != nil {
				return nil, err
			}
			if len(req.Items) > maxCatalogItemsPerCall {
				return nil, invalid("at most %d catalog items per call", maxCatalogItemsPerCall)
			}
			for _, item := range req.Items {
				if err := validateCatalogRef(item.ItemKey, item.ContentHash); err != nil {
					return nil, err
				}
			}
			if err := r.requireVector(ctx); err != nil {
				return nil, err
			}
			if len(req.Items) == 0 {
				return []*airetrieval.CatalogEmbedding{}, nil
			}

			c := airetrievaldb.CatalogEmbeddings
			entities, err := catalogScope(req.ModelKey, req.Corpus).
				Where(catalogRefsMatch(req.Items)).
				Order(c.ItemKey.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("read catalog embeddings: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) PutCatalogEmbeddings(
	ctx context.Context,
	req agentports.PutCatalogEmbeddingsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := validateCatalogScope(req.ModelKey, req.Corpus); err != nil {
			return 0, err
		}
		if err := validateDimensions(req.Dimensions); err != nil {
			return 0, err
		}
		if len(req.Items) > maxCatalogItemsPerCall {
			return 0, invalid("at most %d catalog items per call", maxCatalogItemsPerCall)
		}

		now := timeutils.NowUnix()
		seen := make(map[airetrieval.CatalogItemRef]struct{}, len(req.Items))
		rows := make([]*airetrieval.CatalogEmbedding, 0, len(req.Items))
		for _, item := range req.Items {
			if err := validateCatalogRef(item.ItemKey, item.ContentHash); err != nil {
				return 0, err
			}
			if err := validateVector(item.Vector, req.Dimensions); err != nil {
				return 0, err
			}

			ref := airetrieval.CatalogItemRef{ItemKey: item.ItemKey, ContentHash: item.ContentHash}
			if _, duplicate := seen[ref]; duplicate {
				continue
			}
			seen[ref] = struct{}{}

			rows = append(rows, &airetrieval.CatalogEmbedding{
				ModelKey:    req.ModelKey,
				Corpus:      req.Corpus,
				ItemKey:     item.ItemKey,
				ContentHash: item.ContentHash,
				Dimensions:  req.Dimensions,
				Embedding:   pgvector.NewHalfVector(item.Vector),
				CreatedAt:   now,
			})
		}

		if err := r.requireVector(ctx); err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			return 0, nil
		}

		c := airetrievaldb.CatalogEmbeddings
		written, err := c.CreateOrSkip(ctx, r.db.Keel(ctx), rows,
			c.ModelKey, c.Corpus, c.ItemKey, c.ContentHash)
		if err != nil {
			return 0, fmt.Errorf("write catalog embeddings: %w", err)
		}

		return int(written), nil
	})
}

func (r *repository) PruneCatalogEmbeddings(
	ctx context.Context,
	req agentports.PruneCatalogEmbeddingsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if err := validateCatalogScope(req.ModelKey, req.Corpus); err != nil {
			return 0, err
		}
		for _, item := range req.Keep {
			if err := validateCatalogRef(item.ItemKey, item.ContentHash); err != nil {
				return 0, err
			}
		}
		if err := r.requireVector(ctx); err != nil {
			return 0, err
		}

		q := catalogScope(req.ModelKey, req.Corpus)
		if len(req.Keep) > 0 {
			q.Where(keel.Not(catalogRefsMatch(req.Keep)))
		}

		removed, err := q.Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("prune catalog embeddings: %w", err)
		}

		return int(removed), nil
	})
}
