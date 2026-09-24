package airetrievalrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

const (
	maxChunksPerSource = 500

	defaultSearchLimit    = 10
	maxSearchLimit        = 200
	candidatesPerResult   = 4
	maxSearchCandidates   = 1000
	minHNSWEfSearch       = 40
	maxHNSWEfSearch       = 1000
	candidateAlias        = "candidate"
	bestAlias             = "best"
	distanceColumn        = "distance"
	setIterativeScan      = "SET LOCAL hnsw.iterative_scan = relaxed_order"
	setEfSearchStatement  = "SET LOCAL hnsw.ef_search = "
	halfvecCastPrefix     = "::halfvec("
	cosineDistanceOperand = " <=> ?"
)

type storedChunk struct {
	ChunkIndex  int    `bun:"chunk_index"`
	ContentHash string `bun:"content_hash"`
	Dimensions  int    `bun:"dimensions"`
}

func embeddingSourceScope(
	source repositories.AIRetrievalSourceRef,
	modelKey string,
) func(*bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.EmbeddingColumns

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return buncolgen.EmbeddingScopeTenant(q, source.TenantInfo).
			Where(cols.SourceType.Eq(), source.SourceType).
			Where(cols.SourceID.Eq(), source.SourceID).
			Where(cols.ModelKey.Eq(), modelKey)
	}
}

func (r *repository) ListChunkHashes(
	ctx context.Context,
	req repositories.ListEmbeddingChunkHashesRequest,
) ([]repositories.EmbeddingChunkHash, error) {
	if err := validateSource(req.Source); err != nil {
		return nil, err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return nil, err
	}
	if err := r.requireVector(ctx); err != nil {
		return nil, err
	}

	cols := buncolgen.EmbeddingColumns
	hashes := make([]repositories.EmbeddingChunkHash, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		Column(cols.ChunkIndex.String(), cols.ContentHash.String()).
		Apply(embeddingSourceScope(req.Source, req.ModelKey)).
		OrderExpr(cols.ChunkIndex.OrderAsc()).
		Scan(ctx, &hashes); err != nil {
		return nil, fmt.Errorf("list embedding chunk hashes: %w", err)
	}

	return hashes, nil
}

func validateReplace(req repositories.ReplaceEmbeddingChunksRequest) error {
	if err := validateSource(req.Source); err != nil {
		return err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return err
	}
	if err := validateDimensions(req.Dimensions); err != nil {
		return err
	}
	if len(req.Chunks) > maxChunksPerSource {
		return invalid("a source is at most %d chunks", maxChunksPerSource)
	}

	seen := make(map[int]struct{}, len(req.Chunks))
	for _, chunk := range req.Chunks {
		if chunk.ChunkIndex < 0 {
			return invalid("chunk index %d is negative", chunk.ChunkIndex)
		}
		if _, duplicate := seen[chunk.ChunkIndex]; duplicate {
			return invalid("chunk index %d is listed more than once", chunk.ChunkIndex)
		}
		seen[chunk.ChunkIndex] = struct{}{}

		if err := validateContentHash(chunk.ContentHash); err != nil {
			return err
		}
		if chunk.Vector != nil {
			if err := validateVector(chunk.Vector, req.Dimensions); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *repository) ReplaceChunks(
	ctx context.Context,
	req repositories.ReplaceEmbeddingChunksRequest,
) (repositories.ReplaceEmbeddingChunksResult, error) {
	var result repositories.ReplaceEmbeddingChunksResult

	if err := validateReplace(req); err != nil {
		return result, err
	}
	if err := r.requireVector(ctx); err != nil {
		return result, err
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		stored, err := r.storedChunks(ctx, tx, req)
		if err != nil {
			return err
		}

		rows, unchanged, err := planChunkWrites(req, stored, timeutils.NowUnix())
		if err != nil {
			return err
		}
		result.Unchanged = unchanged

		if result.Removed, err = r.removeChunksNotIn(ctx, tx, req); err != nil {
			return err
		}

		if len(rows) == 0 {
			return nil
		}

		result.Written, err = r.upsertChunks(ctx, tx, rows)
		return err
	})
	if err != nil {
		return repositories.ReplaceEmbeddingChunksResult{}, err
	}

	return result, nil
}

func (r *repository) storedChunks(
	ctx context.Context,
	tx bun.Tx,
	req repositories.ReplaceEmbeddingChunksRequest,
) (map[int]storedChunk, error) {
	cols := buncolgen.EmbeddingColumns
	rows := make([]storedChunk, 0, len(req.Chunks))
	if err := tx.NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		Column(cols.ChunkIndex.String(), cols.ContentHash.String(), cols.Dimensions.String()).
		Apply(embeddingSourceScope(req.Source, req.ModelKey)).
		For("UPDATE").
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read stored embedding chunks: %w", err)
	}

	stored := make(map[int]storedChunk, len(rows))
	for _, row := range rows {
		stored[row.ChunkIndex] = row
	}

	return stored, nil
}

func planChunkWrites(
	req repositories.ReplaceEmbeddingChunksRequest,
	stored map[int]storedChunk,
	now int64,
) ([]*airetrieval.Embedding, int, error) {
	rows := make([]*airetrieval.Embedding, 0, len(req.Chunks))
	unchanged := 0

	for _, chunk := range req.Chunks {
		existing, ok := stored[chunk.ChunkIndex]
		current := ok &&
			existing.ContentHash == chunk.ContentHash &&
			existing.Dimensions == req.Dimensions

		if current {
			unchanged++
			continue
		}

		if chunk.Vector == nil {
			return nil, 0, fmt.Errorf(
				"%w: chunk %d",
				airetrieval.ErrChunkEmbeddingMissing,
				chunk.ChunkIndex,
			)
		}

		rows = append(rows, &airetrieval.Embedding{
			OrganizationID: req.Source.TenantInfo.OrgID,
			BusinessUnitID: req.Source.TenantInfo.BuID,
			SourceType:     req.Source.SourceType,
			SourceID:       req.Source.SourceID,
			ChunkIndex:     chunk.ChunkIndex,
			ModelKey:       req.ModelKey,
			Dimensions:     req.Dimensions,
			ContentHash:    chunk.ContentHash,
			Embedding:      pgvector.NewHalfVector(chunk.Vector),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	return rows, unchanged, nil
}

func (r *repository) removeChunksNotIn(
	ctx context.Context,
	tx bun.Tx,
	req repositories.ReplaceEmbeddingChunksRequest,
) (int, error) {
	cols := buncolgen.EmbeddingColumns
	indexes := make([]int, 0, len(req.Chunks))
	for _, chunk := range req.Chunks {
		indexes = append(indexes, chunk.ChunkIndex)
	}

	q := tx.NewDelete().
		Model((*airetrieval.Embedding)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.EmbeddingScopeTenantDelete(dq, req.Source.TenantInfo).
				Where(cols.SourceType.Eq(), req.Source.SourceType).
				Where(cols.SourceID.Eq(), req.Source.SourceID).
				Where(cols.ModelKey.Eq(), req.ModelKey)
		})
	if len(indexes) > 0 {
		q = q.Where(cols.ChunkIndex.NotIn(), bun.List(indexes))
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("remove stale embedding chunks: %w", err)
	}

	return rowsAffected(res)
}

func (r *repository) upsertChunks(
	ctx context.Context,
	tx bun.Tx,
	rows []*airetrieval.Embedding,
) (int, error) {
	cols := buncolgen.EmbeddingColumns
	excluded := excludedEmbeddingColumns

	res, err := tx.NewInsert().
		Model(&rows).
		On(conflictTarget(buncolgen.EmbeddingTable) + " DO UPDATE").
		Set(cols.ContentHash.SetExcluded()).
		Set(cols.Dimensions.SetExcluded()).
		Set(cols.Embedding.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
		Where(buncolgen.Expr("{0} <> {1} OR {2} <> {3}",
			cols.ContentHash, excluded.ContentHash,
			cols.Dimensions, excluded.Dimensions,
		)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("write embedding chunks: %w", err)
	}

	return rowsAffected(res)
}

var excludedEmbeddingColumns = struct {
	ContentHash buncolgen.Column
	Dimensions  buncolgen.Column
}{
	ContentHash: buncolgen.EmbeddingColumns.ContentHash.WithAlias("excluded"),
	Dimensions:  buncolgen.EmbeddingColumns.Dimensions.WithAlias("excluded"),
}

func (r *repository) DeleteSource(
	ctx context.Context,
	source repositories.AIRetrievalSourceRef,
) (repositories.DeleteAIRetrievalSourceResult, error) {
	var result repositories.DeleteAIRetrievalSourceResult

	if err := validateSource(source); err != nil {
		return result, err
	}

	vectorReady, err := r.vectorReady(ctx)
	if err != nil {
		return result, err
	}

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		entries := buncolgen.IndexEntryColumns
		res, txErr := tx.NewDelete().
			Model((*airetrieval.IndexEntry)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.IndexEntryScopeTenantDelete(dq, source.TenantInfo).
					Where(entries.SourceType.Eq(), source.SourceType).
					Where(entries.SourceID.Eq(), source.SourceID)
			}).
			Exec(ctx)
		if txErr != nil {
			return fmt.Errorf("delete index entries: %w", txErr)
		}
		if result.IndexEntries, txErr = rowsAffected(res); txErr != nil {
			return txErr
		}

		if !vectorReady {
			return nil
		}

		embeddings := buncolgen.EmbeddingColumns
		res, txErr = tx.NewDelete().
			Model((*airetrieval.Embedding)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.EmbeddingScopeTenantDelete(dq, source.TenantInfo).
					Where(embeddings.SourceType.Eq(), source.SourceType).
					Where(embeddings.SourceID.Eq(), source.SourceID)
			}).
			Exec(ctx)
		if txErr != nil {
			return fmt.Errorf("delete embeddings: %w", txErr)
		}
		result.Embeddings, txErr = rowsAffected(res)

		return txErr
	})
	if err != nil {
		return repositories.DeleteAIRetrievalSourceResult{}, err
	}

	return result, nil
}

type searchPlan struct {
	sourceTypes []airetrieval.SourceType
	limit       int
	candidates  int
	efSearch    int
	dimensions  string
}

func planSearch(req repositories.VectorSearchRequest) (searchPlan, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return searchPlan{}, err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return searchPlan{}, err
	}
	if err := validateDimensions(req.Dimensions); err != nil {
		return searchPlan{}, err
	}
	if err := validateVector(req.Query, req.Dimensions); err != nil {
		return searchPlan{}, err
	}
	if err := validateSourceTypes(req.SourceTypes); err != nil {
		return searchPlan{}, err
	}
	if req.MinSimilarity < -1 || req.MinSimilarity > 1 {
		return searchPlan{}, invalid("a similarity floor is between -1 and 1")
	}

	plan := searchPlan{
		sourceTypes: req.SourceTypes,
		limit:       req.Limit,
		dimensions:  strconv.Itoa(req.Dimensions),
	}
	if len(plan.sourceTypes) == 0 {
		plan.sourceTypes = airetrieval.AllSourceTypes()
	}
	if plan.limit <= 0 {
		plan.limit = defaultSearchLimit
	}
	plan.limit = min(plan.limit, maxSearchLimit)

	plan.candidates = req.Candidates
	if plan.candidates < plan.limit {
		plan.candidates = plan.limit * candidatesPerResult
	}
	plan.candidates = min(plan.candidates, maxSearchCandidates)
	plan.efSearch = min(max(plan.candidates, minHNSWEfSearch), maxHNSWEfSearch)

	return plan, nil
}

func (r *repository) Search(
	ctx context.Context,
	req repositories.VectorSearchRequest,
) ([]repositories.VectorSearchHit, error) {
	plan, err := planSearch(req)
	if err != nil {
		return nil, err
	}
	if err = r.requireVector(ctx); err != nil {
		return nil, err
	}

	hits := make([]repositories.VectorSearchHit, 0, plan.limit)
	err = r.db.WithTx(
		ctx,
		ports.TxOptions{ReadOnly: true},
		func(ctx context.Context, tx bun.Tx) error {
			if _, txErr := tx.ExecContext(ctx, setIterativeScan); txErr != nil {
				return fmt.Errorf("enable iterative index scan: %w", txErr)
			}
			if _, txErr := tx.ExecContext(
				ctx,
				setEfSearchStatement+strconv.Itoa(plan.efSearch),
			); txErr != nil {
				return fmt.Errorf("set index search breadth: %w", txErr)
			}

			if txErr := searchQuery(tx, req, plan).Scan(ctx, &hits); txErr != nil {
				return fmt.Errorf("search embeddings: %w", txErr)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return hits, nil
}

func searchQuery(
	tx bun.Tx,
	req repositories.VectorSearchRequest,
	plan searchPlan,
) *bun.SelectQuery {
	cols := buncolgen.EmbeddingColumns
	query := pgvector.NewHalfVector(req.Query)
	cast := halfvecCastPrefix + plan.dimensions + ")"
	distance := cols.Embedding.Expr("{}"+cast) + cosineDistanceOperand + cast

	candidates := tx.NewSelect().
		Model((*airetrieval.Embedding)(nil)).
		Column(cols.SourceType.String(), cols.SourceID.String(), cols.ChunkIndex.String()).
		ColumnExpr(distance+" AS "+distanceColumn, query).
		Apply(buncolgen.EmbeddingApplyTenant(req.TenantInfo)).
		Where(cols.SourceType.In(), bun.List(plan.sourceTypes)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Where(cols.Dimensions.Expr("{} = "+plan.dimensions)).
		OrderExpr(distance, query).
		Limit(plan.candidates)

	candidate := candidateColumns
	best := tx.NewSelect().
		TableExpr("(?) AS "+candidateAlias, candidates).
		DistinctOn(buncolgen.Expr("{0}, {1}", candidate.SourceType, candidate.SourceID)).
		ColumnExpr(candidate.SourceType.Qualified()).
		ColumnExpr(candidate.SourceID.Qualified()).
		ColumnExpr(candidate.ChunkIndex.Qualified()).
		ColumnExpr(candidate.Distance).
		OrderExpr(candidate.SourceType.OrderAsc()).
		OrderExpr(candidate.SourceID.OrderAsc()).
		OrderExpr(candidate.Distance + " ASC")

	top := bestColumns
	similarity := "1 - " + top.Distance
	outer := tx.NewSelect().
		TableExpr("(?) AS "+bestAlias, best).
		ColumnExpr(top.SourceType.Qualified()).
		ColumnExpr(top.SourceID.Qualified()).
		ColumnExpr(top.ChunkIndex.Qualified()).
		ColumnExpr(similarity + " AS similarity").
		OrderExpr(top.Distance + " ASC").
		OrderExpr(top.SourceID.OrderAsc()).
		Limit(plan.limit)

	if req.MinSimilarity != 0 {
		outer = outer.Where(similarity+" >= ?", req.MinSimilarity)
	}

	return outer
}

type rankedColumns struct {
	SourceType buncolgen.Column
	SourceID   buncolgen.Column
	ChunkIndex buncolgen.Column
	Distance   string
}

func newRankedColumns(alias string) rankedColumns {
	cols := buncolgen.EmbeddingColumns

	return rankedColumns{
		SourceType: cols.SourceType.WithAlias(alias),
		SourceID:   cols.SourceID.WithAlias(alias),
		ChunkIndex: cols.ChunkIndex.WithAlias(alias),
		Distance:   alias + "." + distanceColumn,
	}
}

var (
	candidateColumns = newRankedColumns(candidateAlias)
	bestColumns      = newRankedColumns(bestAlias)
)
