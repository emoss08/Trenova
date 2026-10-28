package airetrievalrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval/airetrievaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	maxStaleSourcesPerCall = 1000
	maxModelKeysPerCall    = 4

	defaultClaimLimit = 50
	maxClaimLimit     = 500
	defaultClaimLease = 5 * time.Minute
	maxClaimLease     = time.Hour

	maxOutcomesPerCall = 1000
	maxErrorChars      = 4000

	defaultStaleLimit = 500
	maxStaleLimit     = 5000

	defaultErroredLimit = 500
	maxErroredLimit     = 2000
)

var outcomeColumnNames = []string{
	"organization_id",
	"business_unit_id",
	colSourceType,
	colSourceID,
	"model_key",
	"generation",
	"chunk_count",
	"last_error",
	"next_attempt_at",
}

func (r *repository) MarkStale(
	ctx context.Context,
	req *retrievalports.MarkAIRetrievalStaleRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if req == nil {
			return 0, invalid("marking sources stale needs a request")
		}
		if err := validateTenant(req.TenantInfo); err != nil {
			return 0, err
		}
		if !req.SourceType.IsValid() {
			return 0, invalid("source type %q is not one retrieval indexes", req.SourceType)
		}
		if len(req.SourceIDs) > maxStaleSourcesPerCall {
			return 0, invalid("at most %d sources per call", maxStaleSourcesPerCall)
		}
		if len(req.ModelKeys) > maxModelKeysPerCall {
			return 0, invalid("at most %d model keys per call", maxModelKeysPerCall)
		}
		for _, key := range req.ModelKeys {
			if err := validateModelKey(key); err != nil {
				return 0, err
			}
		}

		sourceIDs := sliceutils.Dedupe(req.SourceIDs)
		modelKeys := sliceutils.Dedupe(req.ModelKeys)
		if slices.ContainsFunc(sourceIDs, pulid.ID.IsNil) {
			return 0, invalid("a source id is required")
		}
		if len(sourceIDs) == 0 || len(modelKeys) == 0 {
			return 0, nil
		}
		slices.Sort(sourceIDs)
		slices.Sort(modelKeys)

		now := req.Now
		if now == 0 {
			now = timeutils.NowUnix()
		}

		entries := make([]*airetrieval.IndexEntry, 0, len(sourceIDs)*len(modelKeys))
		for _, modelKey := range modelKeys {
			for _, sourceID := range sourceIDs {
				entries = append(entries, &airetrieval.IndexEntry{
					OrganizationID: req.TenantInfo.OrgID,
					BusinessUnitID: req.TenantInfo.BuID,
					SourceType:     req.SourceType,
					SourceID:       sourceID,
					ModelKey:       modelKey,
					Status:         airetrieval.IndexStatusPending,
					Generation:     1,
					CreatedAt:      now,
					UpdatedAt:      now,
				})
			}
		}

		e := airetrievaldb.IndexEntries
		if _, err := e.Upsert(
			ctx,
			r.db.Keel(ctx),
			entries,
			keel.OnConflict(e.OrganizationID, e.BusinessUnitID, e.SourceType, e.SourceID, e.ModelKey).
				Set(
					e.Status.To(airetrieval.IndexStatusPending),
					e.Generation.Increment(1),
					e.Attempts.To(0),
					e.LastError.ToNull(),
					e.NextAttemptAt.ToNull(),
				),
		); err != nil {
			return 0, fmt.Errorf("mark retrieval sources stale: %w", err)
		}

		return len(entries), nil
	})
}

func (r *repository) ClaimIndexEntries(
	ctx context.Context,
	req *retrievalports.ClaimIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*airetrieval.IndexEntry, error) {
		if req == nil {
			return nil, invalid("claiming index entries needs a request")
		}
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}
		if err := validateModelKey(req.ModelKey); err != nil {
			return nil, err
		}
		if err := validateSourceTypes(req.SourceTypes); err != nil {
			return nil, err
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultClaimLimit
		}
		limit = intutils.Clamp(limit, 1, maxClaimLimit)

		lease := req.Lease
		if lease <= 0 {
			lease = defaultClaimLease
		}
		lease = min(max(lease, time.Second), maxClaimLease)

		now := req.Now
		if now == 0 {
			now = timeutils.NowUnix()
		}

		e := airetrievaldb.IndexEntries
		pickedEntries := airetrievaldb.IndexEntries.As("picked_entry")
		pickedQuery := pickedEntries.
			Where(
				pickedEntries.ModelKey.Eq(req.ModelKey),
				pickedEntries.Status.Eq(airetrieval.IndexStatusPending),
				keel.Or(
					pickedEntries.LeaseExpiresAt.IsNull(),
					pickedEntries.LeaseExpiresAt.Lte(now),
				),
				keel.Or(pickedEntries.NextAttemptAt.IsNull(), pickedEntries.NextAttemptAt.Lte(now)),
			).
			Pick(pickedEntries.SourceType, pickedEntries.SourceID).
			Order(pickedEntries.UpdatedAt.Asc(), pickedEntries.SourceID.Asc()).
			Limit(limit).
			Lock(keel.ForUpdate, keel.SkipLocked)
		if len(req.SourceTypes) > 0 {
			pickedQuery.Where(pickedEntries.SourceType.In(req.SourceTypes...))
		}
		picked := keel.CTE("picked", []string{colSourceType, colSourceID}, pickedQuery)

		entries, err := e.With(picked).
			Join(picked,
				picked.Col(colSourceType).EqCol(e.SourceType),
				picked.Col(colSourceID).EqCol(e.SourceID),
			).
			Where(e.ModelKey.Eq(req.ModelKey)).
			UpdateReturning(ctx, r.db.Keel(ctx),
				e.LeaseExpiresAt.To(now+int64(lease/time.Second)),
				e.Attempts.Increment(1),
				e.LastAttemptAt.To(now),
				e.UpdatedAt.To(now),
			)
		if err != nil {
			return nil, fmt.Errorf("claim index entries: %w", err)
		}

		slices.SortFunc(entries, func(a, b *airetrieval.IndexEntry) int {
			if c := strings.Compare(a.SourceType.String(), b.SourceType.String()); c != 0 {
				return c
			}
			return strings.Compare(a.SourceID.String(), b.SourceID.String())
		})

		return entries, nil
	})
}

type outcomeRow struct {
	OrganizationID string
	BusinessUnitID string
	SourceType     string
	SourceID       string
	ModelKey       string
	Generation     int64
	ChunkCount     int64
	LastError      string
	NextAttemptAt  int64
}

func outcomeTable(rows []outcomeRow) *keel.CTETable {
	orgs := make([]string, 0, len(rows))
	bus := make([]string, 0, len(rows))
	types := make([]string, 0, len(rows))
	ids := make([]string, 0, len(rows))
	keys := make([]string, 0, len(rows))
	generations := make([]int64, 0, len(rows))
	chunks := make([]int64, 0, len(rows))
	errs := make([]string, 0, len(rows))
	retries := make([]int64, 0, len(rows))
	for index := range rows {
		row := &rows[index]
		orgs = append(orgs, row.OrganizationID)
		bus = append(bus, row.BusinessUnitID)
		types = append(types, row.SourceType)
		ids = append(ids, row.SourceID)
		keys = append(keys, row.ModelKey)
		generations = append(generations, row.Generation)
		chunks = append(chunks, row.ChunkCount)
		errs = append(errs, row.LastError)
		retries = append(retries, row.NextAttemptAt)
	}

	return keel.Unnest("outcome", outcomeColumnNames,
		orgs, bus, types, ids, keys, generations, chunks, errs, retries)
}

func matchOutcome(
	q *keel.Query[*airetrieval.IndexEntry],
	outcome *keel.CTETable,
) *keel.Query[*airetrieval.IndexEntry] {
	e := airetrievaldb.IndexEntries

	return q.With(outcome).Join(outcome,
		outcome.Col("organization_id").EqCol(e.OrganizationID),
		outcome.Col("business_unit_id").EqCol(e.BusinessUnitID),
		outcome.Col(colSourceType).EqCol(e.SourceType),
		outcome.Col(colSourceID).EqCol(e.SourceID),
		outcome.Col("model_key").EqCol(e.ModelKey),
	)
}

func outcomeRows(
	req retrievalports.MarkIndexEntriesRequest,
	status airetrieval.IndexStatus,
) ([]outcomeRow, error) {
	if len(req.Outcomes) > maxOutcomesPerCall {
		return nil, invalid("at most %d outcomes per call", maxOutcomesPerCall)
	}

	rows := make([]outcomeRow, 0, len(req.Outcomes))
	seen := make(map[airetrieval.IndexEntryKey]struct{}, len(req.Outcomes))
	for _, outcome := range req.Outcomes {
		key := outcome.Key
		if err := validateTenant(key.TenantInfo()); err != nil {
			return nil, err
		}
		if !key.SourceType.IsValid() || key.SourceID.IsNil() {
			return nil, invalid("an outcome names a valid source")
		}
		if err := validateModelKey(key.ModelKey); err != nil {
			return nil, err
		}
		if outcome.Generation < 1 || outcome.ChunkCount < 0 || outcome.RetryAt < 0 {
			return nil, invalid("an outcome has a generation and no negative counts")
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, invalid("source %s is reported more than once", key.SourceID)
		}
		seen[key] = struct{}{}

		row := outcomeRow{
			OrganizationID: key.OrganizationID.String(),
			BusinessUnitID: key.BusinessUnitID.String(),
			SourceType:     key.SourceType.String(),
			SourceID:       key.SourceID.String(),
			ModelKey:       key.ModelKey,
			Generation:     outcome.Generation,
		}

		switch status {
		case airetrieval.IndexStatusIndexed:
			row.ChunkCount = int64(outcome.ChunkCount)
		case airetrieval.IndexStatusFailed:
			if strings.TrimSpace(outcome.Error) == "" {
				return nil, invalid("a failed outcome says why")
			}
			if outcome.RetryAt > 0 {
				row.NextAttemptAt = outcome.RetryAt
			}
		case airetrieval.IndexStatusSkipped:
		case airetrieval.IndexStatusPending:
			return nil, invalid("an outcome records a result, not a pending entry")
		}

		if message := strings.TrimSpace(outcome.Error); message != "" {
			row.LastError = stringutils.TruncateRunes(message, maxErrorChars)
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (r *repository) MarkIndexed(
	ctx context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusIndexed)
}

func (r *repository) MarkFailed(
	ctx context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusFailed)
}

func (r *repository) MarkSkipped(
	ctx context.Context,
	req retrievalports.MarkIndexEntriesRequest,
) (retrievalports.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusSkipped)
}

func (r *repository) markEntries(
	ctx context.Context,
	req retrievalports.MarkIndexEntriesRequest,
	status airetrieval.IndexStatus,
) (retrievalports.MarkIndexEntriesResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (retrievalports.MarkIndexEntriesResult, error) {
			var result retrievalports.MarkIndexEntriesResult

			rows, err := outcomeRows(req, status)
			if err != nil || len(rows) == 0 {
				return result, err
			}

			now := req.Now
			if now == 0 {
				now = timeutils.NowUnix()
			}

			err = r.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
				db := r.db.Keel(ctx)
				applied, txErr := applyOutcome(ctx, db, rows, status, now)
				if txErr != nil {
					return fmt.Errorf("record index outcomes: %w", txErr)
				}
				result.Applied = int(applied)

				superseded, txErr := releaseSuperseded(ctx, db, rows, now)
				if txErr != nil {
					return fmt.Errorf("release superseded index entries: %w", txErr)
				}
				result.Superseded = int(superseded)

				return nil
			})
			if err != nil {
				return retrievalports.MarkIndexEntriesResult{}, err
			}

			return result, nil
		},
	)
}

func applyOutcome(
	ctx context.Context,
	db *keel.DB,
	rows []outcomeRow,
	status airetrieval.IndexStatus,
	now int64,
) (int64, error) {
	e := airetrievaldb.IndexEntries
	outcome := outcomeTable(rows)
	sets := []keel.Assignment{
		e.LeaseExpiresAt.ToNull(),
		e.UpdatedAt.To(now),
		e.LastError.ToExpr(keel.StringOf(outcome.Col("last_error")).NullIf("")),
	}
	if status == airetrieval.IndexStatusFailed {
		sets = append(sets,
			e.Status.ToExpr(keel.CaseString().
				When(
					keel.Int64Of(outcome.Col("next_attempt_at")).Eq(0),
					keel.Text(string(airetrieval.IndexStatusFailed)),
				).
				Else(keel.Text(string(airetrieval.IndexStatusPending)))),
			e.NextAttemptAt.ToExpr(keel.Int64Of(outcome.Col("next_attempt_at")).NullIf(0)),
		)
	} else {
		sets = append(sets,
			e.Status.To(status),
			e.ChunkCount.ToExpr(outcome.Col("chunk_count")),
			e.NextAttemptAt.ToNull(),
			e.IndexedAt.To(now),
		)
	}

	return matchOutcome(e.Query(), outcome).
		Where(e.Generation.EqCol(outcome.Col("generation"))).
		Update(ctx, db, sets...)
}

func releaseSuperseded(
	ctx context.Context,
	db *keel.DB,
	rows []outcomeRow,
	now int64,
) (int64, error) {
	e := airetrievaldb.IndexEntries
	outcome := outcomeTable(rows)

	return matchOutcome(e.Query(), outcome).
		Where(
			e.Generation.NeCol(outcome.Col("generation")),
			e.Status.Eq(airetrieval.IndexStatusPending),
		).
		Update(ctx, db, e.LeaseExpiresAt.ToNull(), e.UpdatedAt.To(now))
}

func (r *repository) CountIndexEntries(
	ctx context.Context,
	req retrievalports.CountIndexEntriesRequest,
) ([]retrievalports.IndexEntryCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]retrievalports.IndexEntryCount, error) {
			if err := validateTenant(req.TenantInfo); err != nil {
				return nil, err
			}
			if err := validateModelKey(req.ModelKey); err != nil {
				return nil, err
			}

			e := airetrievaldb.IndexEntries
			counts, err := keel.Into[retrievalports.IndexEntryCount](ctx, r.db.Keel(ctx), e.
				Where(e.ModelKey.Eq(req.ModelKey)).
				Group(e.SourceType, e.Status).
				Pick(e.SourceType, e.Status, keel.Count(), e.IndexedAt.Max(), e.LastAttemptAt.Max()).
				Order(e.SourceType.Asc(), e.Status.Asc()))
			if err != nil {
				return nil, fmt.Errorf("count index entries: %w", err)
			}
			if counts == nil {
				counts = []retrievalports.IndexEntryCount{}
			}

			return counts, nil
		},
	)
}

func staleIndexEntry(
	entry *airetrievaldb.IndexEntriesTable,
	updatedAt keel.Int64Col,
) keel.Pred {
	return keel.Or(
		entry.SourceID.IsNull(),
		keel.And(
			entry.Status.In(airetrieval.IndexStatusIndexed, airetrieval.IndexStatusSkipped),
			keel.Int64Of(updatedAt).GteExpr(entry.IndexedAt.Coalesce(0)),
		),
		keel.And(
			entry.Status.Eq(airetrieval.IndexStatusFailed),
			keel.Int64Of(updatedAt).GteExpr(entry.LastAttemptAt.Coalesce(0)),
		),
	)
}

type staleSource struct {
	ID        keelkinds.PulidIDCol
	UpdatedAt keel.Int64Col
}

func staleSources[T any](
	q *keel.Query[*T],
	source staleSource,
	req *retrievalports.FindStaleAIRetrievalSourcesRequest,
	limit int,
) keel.Projection {
	entry := airetrievaldb.IndexEntries.As("stale_entry")
	q.LeftJoin(entry,
		entry.SourceID.EqCol(source.ID),
		entry.SourceType.Eq(req.SourceType),
		entry.ModelKey.Eq(req.ModelKey),
	).
		Where(staleIndexEntry(entry, source.UpdatedAt)).
		Order(source.ID.Asc()).
		Limit(limit)
	if req.AfterID.IsNotNil() {
		q.Where(source.ID.Gt(req.AfterID))
	}

	return q.Pick(source.ID)
}

func staleSourceQuery(
	req *retrievalports.FindStaleAIRetrievalSourcesRequest,
	limit int,
) (keel.Projection, bool) {
	switch req.SourceType {
	case airetrieval.SourceTypeMemory:
		m := agentdb.Memories
		return staleSources(
			m.Query(),
			staleSource{ID: m.ID, UpdatedAt: m.UpdatedAt},
			req,
			limit,
		), true
	case airetrieval.SourceTypeDocument:
		d := documentdb.Documents
		return staleSources(
			d.Query(),
			staleSource{ID: d.ID, UpdatedAt: d.UpdatedAt},
			req,
			limit,
		), true
	case airetrieval.SourceTypeInboundMessage:
		m := inboundmessagedb.InboundMessages
		return staleSources(
			m.Query(),
			staleSource{ID: m.ID, UpdatedAt: m.UpdatedAt},
			req,
			limit,
		), true
	default:
		return nil, false
	}
}

func (r *repository) FindStaleSources(
	ctx context.Context,
	req *retrievalports.FindStaleAIRetrievalSourcesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req == nil {
			return nil, invalid("finding stale sources needs a request")
		}
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}
		if err := validateModelKey(req.ModelKey); err != nil {
			return nil, err
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultStaleLimit
		}
		limit = intutils.Clamp(limit, 1, maxStaleLimit)

		q, ok := staleSourceQuery(req, limit)
		if !ok {
			return nil, invalid("source type %q is not one retrieval indexes", req.SourceType)
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("find stale %s sources: %w", req.SourceType, err)
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func (r *repository) AverageIndexChunks(
	ctx context.Context,
	req *retrievalports.AverageIndexChunksRequest,
) (retrievalports.IndexChunkAverage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (retrievalports.IndexChunkAverage, error) {
			var average retrievalports.IndexChunkAverage

			if req == nil {
				return average, invalid("an average of index chunks needs a request")
			}
			if err := validateTenant(req.TenantInfo); err != nil {
				return average, err
			}
			if err := validateModelKey(req.ModelKey); err != nil {
				return average, err
			}
			if !req.SourceType.IsValid() {
				return average, invalid(
					"source type %q is not one retrieval indexes",
					req.SourceType,
				)
			}

			e := airetrievaldb.IndexEntries
			average, err := keel.IntoOne[retrievalports.IndexChunkAverage](ctx, r.db.Keel(ctx), e.
				Where(
					e.SourceType.Eq(req.SourceType),
					e.ModelKey.Eq(req.ModelKey),
					e.Status.Eq(airetrieval.IndexStatusIndexed),
					e.ChunkCount.Gt(0),
				).
				Pick(keel.Count(), keel.CastFloat64(keel.NumericOf(e.ChunkCount.Avg()).Coalesce(0))),
			)
			if err != nil {
				return retrievalports.IndexChunkAverage{}, fmt.Errorf(
					"average %s chunks per source: %w",
					req.SourceType,
					err,
				)
			}

			return average, nil
		},
	)
}

func (r *repository) ListErroredIndexEntries(
	ctx context.Context,
	req *retrievalports.ListErroredIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*airetrieval.IndexEntry, error) {
		if req == nil {
			return nil, invalid("a list of failed index entries needs a request")
		}
		if err := validateTenant(req.TenantInfo); err != nil {
			return nil, err
		}
		if req.SourceType != "" && !req.SourceType.IsValid() {
			return nil, invalid("source type %q is not one retrieval indexes", req.SourceType)
		}
		if len(req.ModelKeys) > maxModelKeysPerCall {
			return nil, invalid("at most %d model keys per call", maxModelKeysPerCall)
		}
		for _, key := range req.ModelKeys {
			if err := validateModelKey(key); err != nil {
				return nil, err
			}
		}

		modelKeys := sliceutils.Dedupe(req.ModelKeys)
		if len(modelKeys) == 0 {
			return []*airetrieval.IndexEntry{}, nil
		}

		limit := req.Limit
		if limit <= 0 {
			limit = defaultErroredLimit
		}
		limit = intutils.Clamp(limit, 1, maxErroredLimit)

		e := airetrievaldb.IndexEntries
		q := e.Where(
			e.ModelKey.In(modelKeys...),
			keel.Or(
				e.Status.Eq(airetrieval.IndexStatusFailed),
				keel.And(e.Status.Eq(airetrieval.IndexStatusPending), e.LastError.IsNotNull()),
			),
		).
			Order(
				e.LastAttemptAt.Desc().NullsLast(),
				e.SourceType.Asc(),
				e.SourceID.Asc(),
				e.ModelKey.Asc(),
			).
			Limit(limit)
		if req.SourceType != "" {
			q.Where(e.SourceType.Eq(req.SourceType))
		}

		entries, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list errored index entries: %w", err)
		}

		return entries, nil
	})
}
