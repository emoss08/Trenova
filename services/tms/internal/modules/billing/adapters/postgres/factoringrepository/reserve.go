package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const reserveReversalIndex = "uq_factoring_reserve_entries_reversal"

type reserveEntryRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewReserveEntryRepository(p Params) factoringports.ReserveEntryRepository {
	return &reserveEntryRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-reserve-entry"),
	}
}

func (r *reserveEntryRepository) Create(
	ctx context.Context,
	entity *factoring.ReserveEntry,
) (*factoring.ReserveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.ReserveEntry, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) &&
				dberror.ExtractConstraintName(err) == reserveReversalIndex {
				return nil, errortypes.NewConflictError(
					"That chargeback has already been reversed; reload to see the reserve",
				)
			}
			return nil, fmt.Errorf("create factoring reserve entry: %w", err)
		}
		return entity, nil
	})
}

func (r *reserveEntryRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListReserveEntryConnectionRequest,
) (*pagination.CursorListResult[*factoring.ReserveEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.ReserveEntry], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.ReserveEntryTable.Alias
			filter, term := withoutQuery(req.Filter)

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.ReserveEntry)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.scope(
							querybuilder.ApplyFiltersWithoutSort(
								sq,
								alias,
								filter,
								(*factoring.ReserveEntry)(nil),
							),
							req,
							term,
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count factoring reserve entries: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoring.ReserveEntry]{
					Filter:     filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoring.ReserveEntry) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(factoringcol.ReserveEntryTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						q, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							alias,
							filter,
							req.Cursor,
							(*factoring.ReserveEntry)(nil),
						)
						if applyErr != nil {
							return nil, applyErr
						}
						return r.scope(q, req, term), nil
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list factoring reserve entries: %w", err)
			}

			return result, nil
		},
	)
}

func (r *reserveEntryRepository) ListForItem(
	ctx context.Context,
	req factoringports.ListReserveEntriesForItemRequest,
) ([]*factoring.ReserveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.ReserveEntry, error) {
		cols := factoringcol.ReserveEntryColumns
		rows := make([]*factoring.ReserveEntry, 0, 4)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.ReserveEntryScopeTenant(sq, req.TenantInfo).
					Where(cols.SubmissionItemID.Eq(), req.SubmissionItemID)
			}).
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list factoring reserve entries for item: %w", err)
		}
		return rows, nil
	})
}

func (r *reserveEntryRepository) GetBalance(
	ctx context.Context,
	req factoringports.GetReserveBalanceRequest,
) (*factoringports.ReserveBalance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringports.ReserveBalance, error) {
		cols := factoringcol.ReserveEntryColumns
		balance := &factoringports.ReserveBalance{FactoringCompanyID: req.FactoringCompanyID}
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*factoring.ReserveEntry)(nil)).
			ColumnExpr(cols.ReserveMinor.Expr("COALESCE(SUM({}), 0) AS reserve_minor")).
			ColumnExpr(cols.LiabilityMinor.Expr("COALESCE(SUM({}), 0) AS liability_minor")).
			ColumnExpr(
				buncolgen.Expr(
					"COALESCE(SUM({0}) FILTER (WHERE {1} = ?), 0) AS advanced_minor",
					cols.CashMinor,
					cols.Kind,
				),
				factoring.ReserveEntryFunding,
			).
			ColumnExpr(
				buncolgen.Expr(
					"COALESCE(SUM({0}) FILTER (WHERE {1} = ?), 0) AS released_minor",
					cols.CashMinor,
					cols.Kind,
				),
				factoring.ReserveEntryRelease,
			).
			ColumnExpr(
				buncolgen.Expr(
					"COALESCE(-SUM({0}) FILTER (WHERE {1} IN (?, ?)), 0) AS charged_back_minor",
					cols.CashMinor,
					cols.Kind,
				),
				factoring.ReserveEntryChargeback,
				factoring.ReserveEntryChargebackReversal,
			).
			ColumnExpr(cols.FeeMinor.Expr("COALESCE(SUM({}), 0) AS fee_minor")).
			ColumnExpr("COUNT(*) AS entry_count").
			ColumnExpr(cols.OccurredOn.Expr("MAX({}) AS last_occurred_on")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.ReserveEntryScopeTenant(sq, req.TenantInfo).
					Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
			}).
			Scan(ctx, balance); err != nil {
			return nil, fmt.Errorf("get factoring reserve balance: %w", err)
		}
		balance.FactoringCompanyID = req.FactoringCompanyID
		return balance, nil
	})
}

func (r *reserveEntryRepository) scope(
	q *bun.SelectQuery,
	req *factoringports.ListReserveEntryConnectionRequest,
	term string,
) *bun.SelectQuery {
	cols := factoringcol.ReserveEntryColumns
	if term != "" {
		pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.Memo.ILike(), pattern).
				WhereOr(cols.Kind.TextILike(), pattern)
		})
	}
	if !req.FactoringCompanyID.IsNil() {
		q = q.Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
	}
	if !req.SubmissionID.IsNil() {
		q = q.Where(cols.SubmissionID.Eq(), req.SubmissionID)
	}
	if !req.InvoiceID.IsNil() {
		q = q.Where(cols.InvoiceID.Eq(), req.InvoiceID)
	}
	if len(req.Kinds) > 0 {
		q = q.Where(cols.Kind.In(), bun.List(req.Kinds))
	}
	return q
}
