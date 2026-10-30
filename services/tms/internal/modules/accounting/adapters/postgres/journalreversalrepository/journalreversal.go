package journalreversalrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var reversalErrors = dberror.Mapping{Entity: "JournalReversal"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) journalreversalports.JournalReversalRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-reversal-repository")}
}

func withEntries(
	q *keel.Query[*journalreversal.Reversal],
) *keel.Query[*journalreversal.Reversal] {
	jr := journalreversaldb.Reversals
	return q.Load(jr.OriginalJournalEntry(), jr.ReversalJournalEntry())
}

func (r *repository) attachEntries(
	ctx context.Context,
	reversals []*journalreversal.Reversal,
) error {
	ids := make([]pulid.ID, 0, 2*len(reversals))
	for _, reversal := range reversals {
		ids = append(ids, reversal.OriginalJournalEntryID)
		if reversal.ReversalJournalEntryID.IsNotNil() {
			ids = append(ids, reversal.ReversalJournalEntryID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	je := journalentrydb.JournalEntries
	entries, err := je.Where(je.ID.In(ids...)).All(ctx, r.db.Keel(ctx))
	if err != nil {
		r.l.Error("failed to load reversed journal entries", zap.Error(err))
		return err
	}

	byID := make(map[pulid.ID]*journalentry.JournalEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	for _, reversal := range reversals {
		reversal.OriginalJournalEntry = byID[reversal.OriginalJournalEntryID]
		reversal.ReversalJournalEntry = byID[reversal.ReversalJournalEntryID]
	}

	return nil
}

func (r *repository) List(
	ctx context.Context,
	req *journalreversalports.ListJournalReversalsRequest,
) (*pagination.ListResult[*journalreversal.Reversal], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*journalreversal.Reversal], error) {
			jr := journalreversaldb.Reversals
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				jr.Query(),
				&listquery.Request[journalreversal.Reversal]{
					Filter: req.Filter,
					Then:   []keel.Order{jr.CreatedAt.Desc()},
				},
			)
			if err != nil {
				return nil, err
			}
			if err = r.attachEntries(ctx, result.Items); err != nil {
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *journalreversalports.ListJournalReversalConnectionRequest,
) (*pagination.CursorListResult[*journalreversal.Reversal], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*journalreversal.Reversal], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				journalreversaldb.Reversals.Query(),
				&listquery.Request[journalreversal.Reversal]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list journal reversals", zap.Error(err))
				return nil, err
			}
			if err = r.attachEntries(ctx, result.Items); err != nil {
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req journalreversalports.GetJournalReversalByIDRequest,
) (*journalreversal.Reversal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*journalreversal.Reversal, error) {
		return r.get(ctx, req.ID)
	})
}

func (r *repository) get(ctx context.Context, id pulid.ID) (*journalreversal.Reversal, error) {
	jr := journalreversaldb.Reversals
	entity, err := withEntries(jr.Where(jr.ID.Eq(id))).One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, reversalErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) ListByJournalEntry(
	ctx context.Context,
	req journalreversalports.ListJournalReversalsByEntryRequest,
) ([]*journalreversal.Reversal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*journalreversal.Reversal, error) {
		jr := journalreversaldb.Reversals
		return withEntries(jr.Where(keel.Or(
			jr.OriginalJournalEntryID.Eq(req.JournalEntryID),
			jr.ReversalJournalEntryID.Eq(req.JournalEntryID),
		))).
			Order(jr.CreatedAt.Desc(), jr.ID.Desc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *journalreversal.Reversal,
) (*journalreversal.Reversal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*journalreversal.Reversal, error) {
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew(journalreversal.IDPrefix)
		}
		if err := journalreversaldb.Reversals.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, reversalErrors.Map(err, entity.ID.String())
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *journalreversal.Reversal,
) (*journalreversal.Reversal, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*journalreversal.Reversal, error) {
		jr := journalreversaldb.Reversals
		err := jr.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			jr.ReversalJournalEntryID,
			jr.PostedBatchID,
			jr.Status,
			jr.ResolvedFiscalYearID,
			jr.ResolvedFiscalPeriodID,
			jr.ApprovedByID,
			jr.ApprovedAt,
			jr.RejectedByID,
			jr.RejectedAt,
			jr.RejectionReason,
			jr.CancelledByID,
			jr.CancelledAt,
			jr.CancelReason,
			jr.PostedByID,
			jr.PostedAt,
		)
		if err != nil {
			return nil, reversalErrors.Map(err, entity.ID.String())
		}

		return r.get(ctx, entity.ID)
	})
}
