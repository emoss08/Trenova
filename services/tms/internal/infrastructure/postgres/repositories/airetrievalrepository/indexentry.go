package airetrievalrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

	outcomeAlias = "_outcome"

	defaultErroredLimit = 500
	maxErroredLimit     = 2000
)

func (r *repository) MarkStale(
	ctx context.Context,
	req *repositories.MarkAIRetrievalStaleRequest,
) (int, error) {
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

	cols := buncolgen.IndexEntryColumns
	res, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entries).
		On(conflictTarget(buncolgen.IndexEntryTable)+" DO UPDATE").
		Set(cols.Status.Set(), airetrieval.IndexStatusPending).
		Set(cols.Generation.IncConflict(1)).
		Set(cols.Attempts.Set(), 0).
		Set(cols.LastError.SetNull()).
		Set(cols.NextAttemptAt.SetNull()).
		Set(cols.UpdatedAt.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("mark retrieval sources stale: %w", err)
	}

	return rowsAffected(res)
}

func (r *repository) ClaimIndexEntries(
	ctx context.Context,
	req *repositories.ClaimIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
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

	cols := buncolgen.IndexEntryColumns
	dba := r.db.DBForContext(ctx)

	picked := dba.NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		Column(buncolgen.IndexEntryTable.PrimaryKey...).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Where(cols.Status.Eq(), airetrieval.IndexStatusPending).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.LeaseExpiresAt.IsNull()).WhereOr(cols.LeaseExpiresAt.Lte(), now)
		}).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.NextAttemptAt.IsNull()).WhereOr(cols.NextAttemptAt.Lte(), now)
		}).
		OrderExpr(cols.UpdatedAt.OrderAsc()).
		OrderExpr(cols.SourceID.OrderAsc()).
		Limit(limit).
		For("UPDATE SKIP LOCKED")
	if len(req.SourceTypes) > 0 {
		picked = picked.Where(cols.SourceType.In(), bun.List(req.SourceTypes))
	}

	entries := make([]*airetrieval.IndexEntry, 0, limit)
	if err := dba.NewUpdate().
		Model((*airetrieval.IndexEntry)(nil)).
		Set(cols.LeaseExpiresAt.Set(), now+int64(lease/time.Second)).
		Set(cols.Attempts.Inc(1)).
		Set(cols.LastAttemptAt.Set(), now).
		Set(cols.UpdatedAt.Set(), now).
		Where(indexEntryKeyTuple()+" IN (?)", picked).
		Returning(buncolgen.IndexEntryTable.All()).
		Scan(ctx, &entries); err != nil {
		return nil, fmt.Errorf("claim index entries: %w", err)
	}

	slices.SortFunc(entries, func(a, b *airetrieval.IndexEntry) int {
		if c := strings.Compare(a.SourceType.String(), b.SourceType.String()); c != 0 {
			return c
		}
		return strings.Compare(a.SourceID.String(), b.SourceID.String())
	})

	return entries, nil
}

func indexEntryKeyTuple() string {
	cols := buncolgen.IndexEntryColumns

	return buncolgen.Expr("({0}, {1}, {2}, {3}, {4})",
		cols.OrganizationID,
		cols.BusinessUnitID,
		cols.SourceType,
		cols.SourceID,
		cols.ModelKey,
	)
}

type outcomeRow struct {
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100)"`
	SourceType     string   `bun:"source_type,type:VARCHAR(30)"`
	SourceID       pulid.ID `bun:"source_id,type:VARCHAR(100)"`
	ModelKey       string   `bun:"model_key,type:VARCHAR(300)"`
	Generation     int64    `bun:"generation,type:BIGINT"`
	ChunkCount     int      `bun:"chunk_count,type:INTEGER"`
	LastError      *string  `bun:"last_error,type:TEXT"`
	NextAttemptAt  *int64   `bun:"next_attempt_at,type:BIGINT"`
}

var outcomeColumns = struct {
	OrganizationID buncolgen.Column
	BusinessUnitID buncolgen.Column
	SourceType     buncolgen.Column
	SourceID       buncolgen.Column
	ModelKey       buncolgen.Column
	Generation     buncolgen.Column
	ChunkCount     buncolgen.Column
	LastError      buncolgen.Column
	NextAttemptAt  buncolgen.Column
}{
	OrganizationID: buncolgen.IndexEntryColumns.OrganizationID.WithAlias(outcomeAlias),
	BusinessUnitID: buncolgen.IndexEntryColumns.BusinessUnitID.WithAlias(outcomeAlias),
	SourceType:     buncolgen.IndexEntryColumns.SourceType.WithAlias(outcomeAlias),
	SourceID:       buncolgen.IndexEntryColumns.SourceID.WithAlias(outcomeAlias),
	ModelKey:       buncolgen.IndexEntryColumns.ModelKey.WithAlias(outcomeAlias),
	Generation:     buncolgen.IndexEntryColumns.Generation.WithAlias(outcomeAlias),
	ChunkCount:     buncolgen.IndexEntryColumns.ChunkCount.WithAlias(outcomeAlias),
	LastError:      buncolgen.IndexEntryColumns.LastError.WithAlias(outcomeAlias),
	NextAttemptAt:  buncolgen.IndexEntryColumns.NextAttemptAt.WithAlias(outcomeAlias),
}

func matchOutcomeKey(q *bun.UpdateQuery) *bun.UpdateQuery {
	cols := buncolgen.IndexEntryColumns
	o := outcomeColumns

	return q.
		Where(cols.OrganizationID.EqColumn(o.OrganizationID)).
		Where(cols.BusinessUnitID.EqColumn(o.BusinessUnitID)).
		Where(cols.SourceType.EqColumn(o.SourceType)).
		Where(cols.SourceID.EqColumn(o.SourceID)).
		Where(cols.ModelKey.EqColumn(o.ModelKey))
}

func outcomeRows(
	req repositories.MarkIndexEntriesRequest,
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
			OrganizationID: key.OrganizationID,
			BusinessUnitID: key.BusinessUnitID,
			SourceType:     key.SourceType.String(),
			SourceID:       key.SourceID,
			ModelKey:       key.ModelKey,
			Generation:     outcome.Generation,
		}

		switch status {
		case airetrieval.IndexStatusIndexed:
			row.ChunkCount = outcome.ChunkCount
		case airetrieval.IndexStatusFailed:
			if strings.TrimSpace(outcome.Error) == "" {
				return nil, invalid("a failed outcome says why")
			}
			if outcome.RetryAt > 0 {
				retryAt := outcome.RetryAt
				row.NextAttemptAt = &retryAt
			}
		case airetrieval.IndexStatusSkipped:
		case airetrieval.IndexStatusPending:
			return nil, invalid("an outcome records a result, not a pending entry")
		}

		if message := strings.TrimSpace(outcome.Error); message != "" {
			truncated := stringutils.TruncateRunes(message, maxErrorChars)
			row.LastError = &truncated
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (r *repository) MarkIndexed(
	ctx context.Context,
	req repositories.MarkIndexEntriesRequest,
) (repositories.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusIndexed)
}

func (r *repository) MarkFailed(
	ctx context.Context,
	req repositories.MarkIndexEntriesRequest,
) (repositories.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusFailed)
}

func (r *repository) MarkSkipped(
	ctx context.Context,
	req repositories.MarkIndexEntriesRequest,
) (repositories.MarkIndexEntriesResult, error) {
	return r.markEntries(ctx, req, airetrieval.IndexStatusSkipped)
}

func (r *repository) markEntries(
	ctx context.Context,
	req repositories.MarkIndexEntriesRequest,
	status airetrieval.IndexStatus,
) (repositories.MarkIndexEntriesResult, error) {
	var result repositories.MarkIndexEntriesResult

	rows, err := outcomeRows(req, status)
	if err != nil || len(rows) == 0 {
		return result, err
	}

	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		res, txErr := applyOutcome(tx, rows, status, now).Exec(ctx)
		if txErr != nil {
			return fmt.Errorf("record index outcomes: %w", txErr)
		}
		if result.Applied, txErr = rowsAffected(res); txErr != nil {
			return txErr
		}

		res, txErr = releaseSuperseded(tx, rows, now).Exec(ctx)
		if txErr != nil {
			return fmt.Errorf("release superseded index entries: %w", txErr)
		}
		result.Superseded, txErr = rowsAffected(res)

		return txErr
	})
	if err != nil {
		return repositories.MarkIndexEntriesResult{}, err
	}

	return result, nil
}

func applyOutcome(
	tx bun.Tx,
	rows []outcomeRow,
	status airetrieval.IndexStatus,
	now int64,
) *bun.UpdateQuery {
	cols := buncolgen.IndexEntryColumns
	o := outcomeColumns

	q := tx.NewUpdate().
		With(outcomeAlias, tx.NewValues(&rows)).
		Model((*airetrieval.IndexEntry)(nil)).
		TableExpr(outcomeAlias).
		Set(cols.LeaseExpiresAt.SetNull()).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.LastError.SetExpr(o.LastError.Qualified()))

	if status == airetrieval.IndexStatusFailed {
		q = q.
			Set(
				cols.Status.SetExpr(
					"CASE WHEN "+o.NextAttemptAt.Qualified()+" IS NULL THEN ? ELSE ? END",
				),
				airetrieval.IndexStatusFailed,
				airetrieval.IndexStatusPending,
			).
			Set(cols.NextAttemptAt.SetExpr(o.NextAttemptAt.Qualified()))
	} else {
		q = q.
			Set(cols.Status.Set(), status).
			Set(cols.ChunkCount.SetExpr(o.ChunkCount.Qualified())).
			Set(cols.NextAttemptAt.SetNull()).
			Set(cols.IndexedAt.Set(), now)
	}

	return q.
		Apply(matchOutcomeKey).
		Where(cols.Generation.EqColumn(o.Generation))
}

func releaseSuperseded(tx bun.Tx, rows []outcomeRow, now int64) *bun.UpdateQuery {
	cols := buncolgen.IndexEntryColumns

	return tx.NewUpdate().
		With(outcomeAlias, tx.NewValues(&rows)).
		Model((*airetrieval.IndexEntry)(nil)).
		TableExpr(outcomeAlias).
		Set(cols.LeaseExpiresAt.SetNull()).
		Set(cols.UpdatedAt.Set(), now).
		Apply(matchOutcomeKey).
		Where(cols.Generation.Expr("{} <> "+outcomeColumns.Generation.Qualified())).
		Where(cols.Status.Eq(), airetrieval.IndexStatusPending)
}

func (r *repository) CountIndexEntries(
	ctx context.Context,
	req repositories.CountIndexEntriesRequest,
) ([]repositories.IndexEntryCount, error) {
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return nil, err
	}

	cols := buncolgen.IndexEntryColumns
	counts := make(
		[]repositories.IndexEntryCount,
		0,
		len(airetrieval.AllSourceTypes())*len(airetrieval.AllIndexStatuses()),
	)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		Column(cols.SourceType.String(), cols.Status.String()).
		ColumnExpr(buncolgen.Count("count")).
		ColumnExpr(buncolgen.Max(cols.IndexedAt, "last_indexed_at")).
		ColumnExpr(buncolgen.Max(cols.LastAttemptAt, "last_attempt_at")).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		GroupExpr(cols.SourceType.Qualified()).
		GroupExpr(cols.Status.Qualified()).
		OrderExpr(cols.SourceType.OrderAsc()).
		OrderExpr(cols.Status.OrderAsc()).
		Scan(ctx, &counts); err != nil {
		return nil, fmt.Errorf("count index entries: %w", err)
	}

	return counts, nil
}

type sourceTable struct {
	model          any
	id             buncolgen.Column
	organizationID buncolgen.Column
	businessUnitID buncolgen.Column
	updatedAt      buncolgen.Column
	applyTenant    func(pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery
}

var sourceTables = map[airetrieval.SourceType]sourceTable{
	airetrieval.SourceTypeMemory: {
		model:          (*agent.Memory)(nil),
		id:             buncolgen.MemoryColumns.ID,
		organizationID: buncolgen.MemoryColumns.OrganizationID,
		businessUnitID: buncolgen.MemoryColumns.BusinessUnitID,
		updatedAt:      buncolgen.MemoryColumns.UpdatedAt,
		applyTenant:    buncolgen.MemoryApplyTenant,
	},
	airetrieval.SourceTypeDocument: {
		model:          (*document.Document)(nil),
		id:             buncolgen.DocumentColumns.ID,
		organizationID: buncolgen.DocumentColumns.OrganizationID,
		businessUnitID: buncolgen.DocumentColumns.BusinessUnitID,
		updatedAt:      buncolgen.DocumentColumns.UpdatedAt,
		applyTenant:    buncolgen.DocumentApplyTenant,
	},
	airetrieval.SourceTypeInboundMessage: {
		model:          (*inboundmessage.InboundMessage)(nil),
		id:             buncolgen.InboundMessageColumns.ID,
		organizationID: buncolgen.InboundMessageColumns.OrganizationID,
		businessUnitID: buncolgen.InboundMessageColumns.BusinessUnitID,
		updatedAt:      buncolgen.InboundMessageColumns.UpdatedAt,
		applyTenant:    buncolgen.InboundMessageApplyTenant,
	},
}

func (r *repository) FindStaleSources(
	ctx context.Context,
	req *repositories.FindStaleAIRetrievalSourcesRequest,
) ([]pulid.ID, error) {
	if req == nil {
		return nil, invalid("finding stale sources needs a request")
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if err := validateModelKey(req.ModelKey); err != nil {
		return nil, err
	}

	table, ok := sourceTables[req.SourceType]
	if !ok {
		return nil, invalid("source type %q is not one retrieval indexes", req.SourceType)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultStaleLimit
	}
	limit = intutils.Clamp(limit, 1, maxStaleLimit)

	entry := buncolgen.IndexEntryColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(table.model).
		ColumnExpr(table.id.Qualified()).
		Apply(table.applyTenant(req.TenantInfo)).
		Join("LEFT JOIN "+buncolgen.IndexEntryTable.As(buncolgen.IndexEntryTable.Alias)).
		JoinOn(entry.OrganizationID.EqColumn(table.organizationID)).
		JoinOn(entry.BusinessUnitID.EqColumn(table.businessUnitID)).
		JoinOn(entry.SourceID.EqColumn(table.id)).
		JoinOn(entry.SourceType.Eq(), req.SourceType).
		JoinOn(entry.ModelKey.Eq(), req.ModelKey).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(entry.SourceID.IsNull()).
				WhereOr(
					buncolgen.Expr(
						"{0} IN (?) AND {1} >= COALESCE({2}, 0)",
						entry.Status, table.updatedAt, entry.IndexedAt,
					),
					bun.List([]airetrieval.IndexStatus{
						airetrieval.IndexStatusIndexed,
						airetrieval.IndexStatusSkipped,
					}),
				).
				WhereOr(
					buncolgen.Expr(
						"{0} = ? AND {1} >= COALESCE({2}, 0)",
						entry.Status, table.updatedAt, entry.LastAttemptAt,
					),
					airetrieval.IndexStatusFailed,
				)
		}).
		OrderExpr(table.id.OrderAsc()).
		Limit(limit)
	if req.AfterID.IsNotNil() {
		q = q.Where(table.id.Gt(), req.AfterID)
	}

	ids := make([]pulid.ID, 0, limit)
	if err := q.Scan(ctx, &ids); err != nil {
		return nil, fmt.Errorf("find stale %s sources: %w", req.SourceType, err)
	}

	return ids, nil
}

func (r *repository) AverageIndexChunks(
	ctx context.Context,
	req *repositories.AverageIndexChunksRequest,
) (repositories.IndexChunkAverage, error) {
	var average repositories.IndexChunkAverage

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
		return average, invalid("source type %q is not one retrieval indexes", req.SourceType)
	}

	cols := buncolgen.IndexEntryColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*airetrieval.IndexEntry)(nil)).
		ColumnExpr(buncolgen.Count("entries")).
		ColumnExpr(cols.ChunkCount.Expr("COALESCE(AVG({}), 0) AS average_chunks")).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		Where(cols.SourceType.Eq(), req.SourceType).
		Where(cols.ModelKey.Eq(), req.ModelKey).
		Where(cols.Status.Eq(), airetrieval.IndexStatusIndexed).
		Where(cols.ChunkCount.Gt(), 0).
		Scan(ctx, &average); err != nil {
		return repositories.IndexChunkAverage{}, fmt.Errorf(
			"average %s chunks per source: %w",
			req.SourceType,
			err,
		)
	}

	return average, nil
}

func (r *repository) ListErroredIndexEntries(
	ctx context.Context,
	req *repositories.ListErroredIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
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

	cols := buncolgen.IndexEntryColumns
	entries := make([]*airetrieval.IndexEntry, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entries).
		Apply(buncolgen.IndexEntryApplyTenant(req.TenantInfo)).
		Where(cols.ModelKey.In(), bun.List(modelKeys)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.Status.Eq(), airetrieval.IndexStatusFailed).
				WhereOr(
					buncolgen.Expr("{0} = ? AND {1} IS NOT NULL", cols.Status, cols.LastError),
					airetrieval.IndexStatusPending,
				)
		}).
		OrderExpr(cols.LastAttemptAt.Expr("{} DESC NULLS LAST")).
		OrderExpr(cols.SourceType.OrderAsc()).
		OrderExpr(cols.SourceID.OrderAsc()).
		OrderExpr(cols.ModelKey.OrderAsc()).
		Limit(limit)
	if req.SourceType != "" {
		q = q.Where(cols.SourceType.Eq(), req.SourceType)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list errored index entries: %w", err)
	}

	return entries, nil
}
