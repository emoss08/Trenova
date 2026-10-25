package airetrievalrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval/airetrievaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pgvector/pgvector-go"
)

const (
	maxChunksPerSource = 500

	defaultSearchLimit   = 10
	maxSearchLimit       = 200
	candidatesPerResult  = 4
	maxSearchCandidates  = 1000
	minHNSWEfSearch      = 40
	maxHNSWEfSearch      = 1000
	setIterativeScan     = "SET LOCAL hnsw.iterative_scan = relaxed_order"
	setEfSearchStatement = "SET LOCAL hnsw.ef_search = "
)

const (
	colSourceType = "source_type"
	colSourceID   = "source_id"
)

var rankedColumns = []string{colSourceType, colSourceID, "chunk_index", "distance"}

type storedChunk struct {
	ChunkIndex  int
	ContentHash string
	Dimensions  int
}

func embeddingsOf(
	source *agentports.AIRetrievalSourceRef,
	modelKey string,
) *keel.Query[airetrieval.Embedding] {
	e := airetrievaldb.Embeddings

	return e.Where(
		e.SourceType.Eq(source.SourceType),
		e.SourceID.Eq(source.SourceID),
		e.ModelKey.Eq(modelKey),
	)
}

func (r *repository) ListChunkHashes(
	ctx context.Context,
	req *agentports.ListEmbeddingChunkHashesRequest,
) ([]agentports.EmbeddingChunkHash, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentports.EmbeddingChunkHash, error) {
			if req == nil {
				return nil, invalid("listing chunk hashes needs a request")
			}
			if err := validateSource(&req.Source); err != nil {
				return nil, err
			}
			if err := validateModelKey(req.ModelKey); err != nil {
				return nil, err
			}
			if err := r.requireVector(ctx); err != nil {
				return nil, err
			}

			e := airetrievaldb.Embeddings
			hashes, err := keel.Into[agentports.EmbeddingChunkHash](ctx, r.db.Keel(ctx),
				embeddingsOf(&req.Source, req.ModelKey).
					Pick(e.ChunkIndex, e.ContentHash).
					Order(e.ChunkIndex.Asc()))
			if err != nil {
				return nil, fmt.Errorf("list embedding chunk hashes: %w", err)
			}
			if hashes == nil {
				hashes = []agentports.EmbeddingChunkHash{}
			}

			return hashes, nil
		},
	)
}

func validateReplace(req *agentports.ReplaceEmbeddingChunksRequest) error {
	if req == nil {
		return invalid("replacing chunks needs a request")
	}
	if err := validateSource(&req.Source); err != nil {
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
	req *agentports.ReplaceEmbeddingChunksRequest,
) (agentports.ReplaceEmbeddingChunksResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (agentports.ReplaceEmbeddingChunksResult, error) {
			var result agentports.ReplaceEmbeddingChunksResult

			if err := validateReplace(req); err != nil {
				return result, err
			}
			if err := r.requireVector(ctx); err != nil {
				return result, err
			}

			err := r.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
				db := r.db.Keel(ctx)
				stored, err := storedChunks(ctx, db, req)
				if err != nil {
					return err
				}

				rows, unchanged, err := planChunkWrites(req, stored, timeutils.NowUnix())
				if err != nil {
					return err
				}
				result.Unchanged = unchanged

				if result.Removed, err = removeChunksNotIn(ctx, db, req); err != nil {
					return err
				}

				if len(rows) == 0 {
					return nil
				}

				result.Written, err = upsertChunks(ctx, db, rows)
				return err
			})
			if err != nil {
				return agentports.ReplaceEmbeddingChunksResult{}, err
			}

			return result, nil
		},
	)
}

func storedChunks(
	ctx context.Context,
	db *keel.DB,
	req *agentports.ReplaceEmbeddingChunksRequest,
) (map[int]storedChunk, error) {
	e := airetrievaldb.Embeddings
	rows, err := keel.Into[storedChunk](ctx, db, embeddingsOf(&req.Source, req.ModelKey).
		Pick(e.ChunkIndex, e.ContentHash, e.Dimensions).
		Lock(keel.ForUpdate))
	if err != nil {
		return nil, fmt.Errorf("read stored embedding chunks: %w", err)
	}

	stored := make(map[int]storedChunk, len(rows))
	for _, row := range rows {
		stored[row.ChunkIndex] = row
	}

	return stored, nil
}

func planChunkWrites(
	req *agentports.ReplaceEmbeddingChunksRequest,
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

func removeChunksNotIn(
	ctx context.Context,
	db *keel.DB,
	req *agentports.ReplaceEmbeddingChunksRequest,
) (int, error) {
	e := airetrievaldb.Embeddings
	indexes := make([]int64, 0, len(req.Chunks))
	for _, chunk := range req.Chunks {
		indexes = append(indexes, int64(chunk.ChunkIndex))
	}

	q := embeddingsOf(&req.Source, req.ModelKey)
	if len(indexes) > 0 {
		q.Where(e.ChunkIndex.NotIn(indexes...))
	}

	removed, err := q.Delete(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("remove stale embedding chunks: %w", err)
	}

	return int(removed), nil
}

func upsertChunks(
	ctx context.Context,
	db *keel.DB,
	rows []*airetrieval.Embedding,
) (int, error) {
	e := airetrievaldb.Embeddings
	if _, err := e.Upsert(ctx, db, rows,
		keel.OnConflict(
			e.OrganizationID,
			e.BusinessUnitID,
			e.SourceType,
			e.SourceID,
			e.ChunkIndex,
			e.ModelKey,
		).Update(e.ContentHash, e.Dimensions, e.Embedding),
	); err != nil {
		return 0, fmt.Errorf("write embedding chunks: %w", err)
	}

	return len(rows), nil
}

func (r *repository) DeleteSource(
	ctx context.Context,
	source *agentports.AIRetrievalSourceRef,
) (agentports.DeleteAIRetrievalSourceResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (agentports.DeleteAIRetrievalSourceResult, error) {
			var result agentports.DeleteAIRetrievalSourceResult

			if err := validateSource(source); err != nil {
				return result, err
			}

			vectorReady, err := r.vectorReady(ctx)
			if err != nil {
				return result, err
			}

			err = r.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
				db := r.db.Keel(ctx)
				entries := airetrievaldb.IndexEntries
				removed, txErr := entries.Where(
					entries.SourceType.Eq(source.SourceType),
					entries.SourceID.Eq(source.SourceID),
				).Delete(ctx, db)
				if txErr != nil {
					return fmt.Errorf("delete index entries: %w", txErr)
				}
				result.IndexEntries = int(removed)

				if !vectorReady {
					return nil
				}

				embeddings := airetrievaldb.Embeddings
				removed, txErr = embeddings.Where(
					embeddings.SourceType.Eq(source.SourceType),
					embeddings.SourceID.Eq(source.SourceID),
				).Delete(ctx, db)
				if txErr != nil {
					return fmt.Errorf("delete embeddings: %w", txErr)
				}
				result.Embeddings = int(removed)

				return nil
			})
			if err != nil {
				return agentports.DeleteAIRetrievalSourceResult{}, err
			}

			return result, nil
		},
	)
}

type searchPlan struct {
	sourceTypes []airetrieval.SourceType
	limit       int
	candidates  int
	efSearch    int
	dimensions  string
}

func planSearch(req *agentports.VectorSearchRequest) (searchPlan, error) {
	if req == nil {
		return searchPlan{}, invalid("a vector search needs a request")
	}
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
	req *agentports.VectorSearchRequest,
) ([]agentports.VectorSearchHit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]agentports.VectorSearchHit, error) {
		plan, err := planSearch(req)
		if err != nil {
			return nil, err
		}
		if err = r.requireVector(ctx); err != nil {
			return nil, err
		}

		var hits []agentports.VectorSearchHit
		err = r.db.Transact(
			ctx,
			kernelports.TxOptions{ReadOnly: true},
			func(ctx context.Context) error {
				db := r.db.Keel(ctx)
				if _, txErr := keel.Exec(ctx, db, setIterativeScan); txErr != nil {
					return fmt.Errorf("enable iterative index scan: %w", txErr)
				}
				if _, txErr := keel.Exec(
					ctx,
					db,
					setEfSearchStatement+strconv.Itoa(plan.efSearch),
				); txErr != nil {
					return fmt.Errorf("set index search breadth: %w", txErr)
				}

				var txErr error
				if hits, txErr = keel.Into[agentports.VectorSearchHit](
					ctx,
					db,
					searchQuery(req, plan),
				); txErr != nil {
					return fmt.Errorf("search embeddings: %w", txErr)
				}

				return nil
			},
		)
		if err != nil {
			return nil, err
		}
		if hits == nil {
			hits = []agentports.VectorSearchHit{}
		}

		return hits, nil
	})
}

func searchQuery(req *agentports.VectorSearchRequest, plan searchPlan) *keel.Query[keel.Row] {
	e := airetrievaldb.Embeddings
	query := pgvector.NewHalfVector(req.Query)
	cast := "::halfvec(" + plan.dimensions + ")"
	distance := func() keel.Expr {
		return keel.Raw("?"+cast+" <=> ?"+cast, e.Embedding, query)
	}

	candidate := keel.CTE("candidate", rankedColumns, e.
		Where(
			e.SourceType.In(plan.sourceTypes...),
			e.ModelKey.Eq(req.ModelKey),
			keel.Raw("? = "+plan.dimensions, e.Dimensions),
		).
		Pick(e.SourceType, e.SourceID, e.ChunkIndex, distance()).
		Order(keel.Asc(distance())).
		Limit(plan.candidates))

	best := keel.CTE("best", rankedColumns, candidate.Query().
		DistinctOn(candidate.Col(colSourceType), candidate.Col(colSourceID)).
		Pick(
			candidate.Col(colSourceType),
			candidate.Col(colSourceID),
			candidate.Col("chunk_index"),
			candidate.Col("distance"),
		).
		Order(
			candidate.Col(colSourceType).Asc(),
			candidate.Col(colSourceID).Asc(),
			candidate.Col("distance").Asc(),
		))

	similarity := keel.DynLit(1).Sub(best.Col("distance"))
	outer := best.Query().
		With(candidate, best).
		Pick(best.Col(colSourceType), best.Col(colSourceID), best.Col("chunk_index"), similarity).
		Order(best.Col("distance").Asc(), best.Col(colSourceID).Asc()).
		Limit(plan.limit)
	if req.MinSimilarity != 0 {
		outer.Where(similarity.Gte(req.MinSimilarity))
	}

	return outer
}
