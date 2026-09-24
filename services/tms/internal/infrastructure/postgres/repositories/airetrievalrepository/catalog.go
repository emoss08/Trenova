package airetrievalrepository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
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

func catalogRefTuples(refs []airetrieval.CatalogItemRef) [][]any {
	tuples := make([][]any, 0, len(refs))
	for _, ref := range refs {
		tuples = append(tuples, []any{ref.ItemKey, ref.ContentHash})
	}

	return tuples
}

func catalogScope(
	modelKey string,
	corpus airetrieval.CatalogCorpus,
) func(*bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.CatalogEmbeddingColumns

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where(cols.ModelKey.Eq(), modelKey).Where(cols.Corpus.Eq(), corpus)
	}
}

func (r *repository) GetCatalogEmbeddings(
	ctx context.Context,
	req repositories.GetCatalogEmbeddingsRequest,
) ([]*airetrieval.CatalogEmbedding, error) {
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

	cols := buncolgen.CatalogEmbeddingColumns
	entities := make([]*airetrieval.CatalogEmbedding, 0, len(req.Items))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(catalogScope(req.ModelKey, req.Corpus)).
		Where(
			buncolgen.Expr("({0}, {1}) IN (?)", cols.ItemKey, cols.ContentHash),
			bun.In(catalogRefTuples(req.Items)),
		).
		OrderExpr(cols.ItemKey.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read catalog embeddings: %w", err)
	}

	return entities, nil
}

func (r *repository) PutCatalogEmbeddings(
	ctx context.Context,
	req repositories.PutCatalogEmbeddingsRequest,
) (int, error) {
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

	res, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&rows).
		On(conflictTarget(buncolgen.CatalogEmbeddingTable) + " DO NOTHING").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("write catalog embeddings: %w", err)
	}

	return rowsAffected(res)
}

func (r *repository) PruneCatalogEmbeddings(
	ctx context.Context,
	req repositories.PruneCatalogEmbeddingsRequest,
) (int, error) {
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

	cols := buncolgen.CatalogEmbeddingColumns
	q := r.db.DBForContext(ctx).
		NewDelete().
		Model((*airetrieval.CatalogEmbedding)(nil)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Where(cols.Corpus.Eq(), req.Corpus)
	if len(req.Keep) > 0 {
		q = q.Where(
			buncolgen.Expr("({0}, {1}) NOT IN (?)", cols.ItemKey, cols.ContentHash),
			bun.In(catalogRefTuples(req.Keep)),
		)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("prune catalog embeddings: %w", err)
	}

	return rowsAffected(res)
}

func rowsAffected(res sql.Result) (int, error) {
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows: %w", err)
	}

	return int(affected), nil
}
