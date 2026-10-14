package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	invoiceStateEntity  = "FactoringInvoiceState"
	maxStateBatch       = 500
	maxCandidateBatch   = 1000
	invoiceStateUpsertC = "CONFLICT (invoice_id, organization_id, business_unit_id) DO UPDATE"
)

type invoiceStateRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewInvoiceStateRepository(p Params) factoringports.InvoiceStateRepository {
	return &invoiceStateRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-invoice-state"),
	}
}

func (r *invoiceStateRepository) SetInvoiceOverride(
	ctx context.Context,
	req *factoringports.SetInvoiceOverrideRequest,
) error {
	_, err := dbtx.Write(ctx, r.db, func(ctx context.Context) (struct{}, error) {
		inv := invoicecol.InvoiceColumns
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*invoice.Invoice)(nil)).
			Set(inv.FactoringOverride.Set(), req.Override).
			Set(inv.FactoringCompanyID.Set(), req.FactoringCompanyID).
			Set(inv.FactoringAgreementID.Set(), req.FactoringAgreementID).
			Set(inv.Version.Inc(1)).
			Set(inv.UpdatedAt.Set(), timeutils.NowUnix()).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return invoicecol.InvoiceScopeTenantUpdate(uq, req.TenantInfo).
					Where(inv.ID.Eq(), req.InvoiceID)
			}).
			Exec(ctx)
		if err != nil {
			return struct{}{}, fmt.Errorf("set invoice factoring override: %w", err)
		}
		return struct{}{}, dberror.CheckFound(res, "Invoice")
	})
	return err
}

func (r *invoiceStateRepository) Get(
	ctx context.Context,
	req factoringports.GetInvoiceStateRequest,
) (*factoring.InvoiceState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.InvoiceState, error) {
		entity := new(factoring.InvoiceState)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.InvoiceStateScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.InvoiceStateColumns.InvoiceID.Eq(), req.InvoiceID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, invoiceStateEntity)
		}

		return entity, nil
	})
}

func (r *invoiceStateRepository) ListByInvoiceIDs(
	ctx context.Context,
	req *factoringports.ListInvoiceStatesByInvoiceIDsRequest,
) (map[pulid.ID]*factoring.InvoiceState, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoring.InvoiceState, error) {
			result := make(map[pulid.ID]*factoring.InvoiceState, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			rows := make([]*factoring.InvoiceState, 0, len(req.InvoiceIDs))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model(&rows).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.InvoiceStateScopeTenant(sq, req.TenantInfo).
						Where(factoringcol.InvoiceStateColumns.InvoiceID.In(), bun.List(req.InvoiceIDs))
				}).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list factoring invoice states: %w", err)
			}

			for _, row := range rows {
				result[row.InvoiceID] = row
			}

			return result, nil
		},
	)
}

func (r *invoiceStateRepository) Upsert(
	ctx context.Context,
	entity *factoring.InvoiceState,
) error {
	_, err := dbtx.Write(ctx, r.db, func(ctx context.Context) (struct{}, error) {
		cols := factoringcol.InvoiceStateColumns
		alias := factoringcol.InvoiceStateTable.Alias
		stillReady := "EXCLUDED.verdict = '" + string(factoring.VerdictReady) + "' AND " +
			alias + ".verdict = '" + string(factoring.VerdictReady) + "'"

		if _, err := r.db.DBForContext(ctx).NewInsert().
			Model(entity).
			On(invoiceStateUpsertC).
			Set(cols.FactoringCompanyID.SetExcluded()).
			Set(cols.CustomerID.SetExcluded()).
			Set(cols.Verdict.SetExcluded()).
			Set(cols.Checks.SetExcluded()).
			Set(cols.FailedChecks.SetExcluded()).
			Set(cols.OpenBalanceMinor.SetExcluded()).
			Set(cols.EvaluatedAt.SetExcluded()).
			Set(cols.ReadySince.SetExpr(
				"CASE WHEN " + stillReady + " THEN " + cols.ReadySince.Qualified() +
					" ELSE EXCLUDED.ready_since END",
			)).
			Set(cols.ReadyAnnouncedAt.SetExpr(
				"CASE WHEN " + stillReady + " THEN " + cols.ReadyAnnouncedAt.Qualified() +
					" ELSE EXCLUDED.ready_announced_at END",
			)).
			Returning("*").
			Exec(ctx); err != nil {
			return struct{}{}, fmt.Errorf("upsert factoring invoice state: %w", err)
		}

		return struct{}{}, nil
	})
	return err
}

func (r *invoiceStateRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListInvoiceStateConnectionRequest,
) (*pagination.CursorListResult[*factoring.InvoiceState], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.InvoiceState], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.InvoiceStateTable.Alias

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.InvoiceState)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.scope(
							querybuilder.ApplyFiltersWithoutSort(
								sq,
								alias,
								req.Filter,
								(*factoring.InvoiceState)(nil),
							),
							req,
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count factoring invoice states: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoring.InvoiceState]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoring.InvoiceState) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(factoringcol.InvoiceStateTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						q, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							alias,
							req.Filter,
							req.Cursor,
							(*factoring.InvoiceState)(nil),
						)
						if applyErr != nil {
							return nil, applyErr
						}
						return r.scope(q, req), nil
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list factoring invoice states: %w", err)
			}

			return result, nil
		},
	)
}

func (r *invoiceStateRepository) scope(
	q *bun.SelectQuery,
	req *factoringports.ListInvoiceStateConnectionRequest,
) *bun.SelectQuery {
	cols := factoringcol.InvoiceStateColumns
	if len(req.Verdicts) > 0 {
		q = q.Where(cols.Verdict.In(), bun.List(req.Verdicts))
	}
	if !req.FactoringCompanyID.IsNil() {
		q = q.Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
	}
	if !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), req.CustomerID)
	}
	return q
}

func (r *invoiceStateRepository) ClaimReadyAnnouncement(
	ctx context.Context,
	req *factoringports.ClaimReadyAnnouncementRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := factoringcol.InvoiceStateColumns
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*factoring.InvoiceState)(nil)).
			Set(cols.ReadyAnnouncedAt.Set(), req.At).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return factoringcol.InvoiceStateScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.InvoiceID.Eq(), req.InvoiceID).
					Where(cols.Verdict.Eq(), factoring.VerdictReady).
					Where(cols.ReadySince.Eq(), req.ReadySince).
					Where(cols.ReadyAnnouncedAt.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("claim factoring ready announcement: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("claim factoring ready announcement: %w", err)
		}

		return rows > 0, nil
	})
}

func (r *invoiceStateRepository) ListReadyUnsubmitted(
	ctx context.Context,
	req factoringports.ListReadyUnsubmittedRequest,
) ([]*factoring.InvoiceState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.InvoiceState, error) {
		cols := factoringcol.InvoiceStateColumns
		limit := boundedLimit(req.Limit, maxStateBatch)

		rows := make([]*factoring.InvoiceState, 0, limit)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.InvoiceStateScopeTenant(sq, req.TenantInfo).
					Where(cols.Verdict.Eq(), factoring.VerdictReady).
					Where(cols.ReadySince.Lte(), req.ReadyBefore)
			}).
			Order(cols.ReadySince.OrderAsc(), cols.InvoiceID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list ready factoring invoices: %w", err)
		}

		return rows, nil
	})
}

func (r *invoiceStateRepository) ListCandidateInvoiceIDs(
	ctx context.Context,
	req factoringports.ListCandidateInvoicesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		dba := r.db.DBForContext(ctx)
		limit := boundedLimit(req.Limit, maxCandidateBatch)
		inv := invoicecol.InvoiceColumns
		st := factoringcol.InvoiceStateColumns

		factored := dba.NewSelect().
			Model((*invoice.Invoice)(nil)).
			ColumnExpr(inv.ID.As("id")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				q := openFactoredInvoices(invoicecol.InvoiceScopeTenant(sq, req.TenantInfo))
				if !req.AfterInvoiceID.IsNil() {
					q = q.Where(inv.ID.Gt(), req.AfterInvoiceID)
				}
				return q
			})
		tracked := dba.NewSelect().
			Model((*factoring.InvoiceState)(nil)).
			ColumnExpr(st.InvoiceID.As("id")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				q := factoringcol.InvoiceStateScopeTenant(sq, req.TenantInfo).
					Where(st.Verdict.Ne(), factoring.VerdictExcluded)
				if !req.AfterInvoiceID.IsNil() {
					q = q.Where(st.InvoiceID.Gt(), req.AfterInvoiceID)
				}
				return q
			})

		ids := make([]pulid.ID, 0, limit)
		if err := dba.NewSelect().
			TableExpr("(?) AS candidates", factored.Union(tracked)).
			ColumnExpr("candidates.id").
			OrderExpr("candidates.id ASC").
			Limit(int64(limit)).
			Scan(ctx, &ids); err != nil {
			return nil, fmt.Errorf("list factoring candidate invoices: %w", err)
		}

		return ids, nil
	})
}

func (r *invoiceStateRepository) ListCustomerInvoiceIDs(
	ctx context.Context,
	req *factoringports.ListCustomerInvoiceIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.CustomerIDs) == 0 && req.FactoringCompanyID.IsNil() {
			return nil, nil
		}

		inv := invoicecol.InvoiceColumns
		limit := boundedLimit(req.Limit, maxCandidateBatch)
		ids := make([]pulid.ID, 0, limit)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model((*invoice.Invoice)(nil)).
			Column(inv.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				q := openFactoredInvoices(invoicecol.InvoiceScopeTenant(sq, req.TenantInfo))
				if len(req.CustomerIDs) > 0 {
					q = q.Where(inv.CustomerID.In(), bun.List(req.CustomerIDs))
				}
				if !req.FactoringCompanyID.IsNil() {
					q = q.Where(inv.FactoringCompanyID.Eq(), req.FactoringCompanyID)
				}
				return q
			}).
			Order(inv.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx, &ids); err != nil {
			return nil, fmt.Errorf("list customer factoring invoices: %w", err)
		}

		return ids, nil
	})
}

func (r *invoiceStateRepository) GetSummary(
	ctx context.Context,
	req factoringports.GetSummaryRequest,
) (*factoringports.Summary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringports.Summary, error) {
		cols := factoringcol.InvoiceStateColumns
		type row struct {
			Verdict     factoring.Verdict `bun:"verdict"`
			Count       int               `bun:"count"`
			AmountMinor int64             `bun:"amount_minor"`
			LateCount   int               `bun:"late_count"`
			LateMinor   int64             `bun:"late_minor"`
		}

		late := cols.ReadySince.Qualified() + " <= ?"
		rows := make([]row, 0, 3)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model((*factoring.InvoiceState)(nil)).
			Column(cols.Verdict.Bare()).
			ColumnExpr("COUNT(*) AS count").
			ColumnExpr("COALESCE(SUM(?), 0) AS amount_minor", bun.Safe(cols.OpenBalanceMinor.Qualified())).
			ColumnExpr("COUNT(*) FILTER (WHERE "+late+") AS late_count", req.ReadyBefore).
			ColumnExpr(
				"COALESCE(SUM(?) FILTER (WHERE "+late+"), 0) AS late_minor",
				bun.Safe(cols.OpenBalanceMinor.Qualified()),
				req.ReadyBefore,
			).
			Apply(factoringcol.InvoiceStateApplyTenant(req.TenantInfo)).
			Where(cols.Verdict.In(), bun.List([]factoring.Verdict{
				factoring.VerdictReady,
				factoring.VerdictNotReady,
			})).
			Group(cols.Verdict.Bare()).
			Scan(ctx, &rows); err != nil {
			return nil, fmt.Errorf("summarize factoring invoices: %w", err)
		}

		summary := new(factoringports.Summary)
		for _, rw := range rows {
			totals := factoringports.VerdictTotals{Count: rw.Count, AmountMinor: rw.AmountMinor}
			switch rw.Verdict {
			case factoring.VerdictReady:
				summary.Ready = totals
				summary.ReadyUnsubmitted = factoringports.VerdictTotals{
					Count:       rw.LateCount,
					AmountMinor: rw.LateMinor,
				}
			case factoring.VerdictNotReady:
				summary.NotReady = totals
			case factoring.VerdictExcluded:
			}
		}

		return summary, nil
	})
}

func openFactoredInvoices(q *bun.SelectQuery) *bun.SelectQuery {
	inv := invoicecol.InvoiceColumns
	return q.
		Where(inv.Status.Eq(), invoice.StatusPosted).
		Where(inv.FactoringCompanyID.IsNotNull()).
		Where(inv.BillType.Ne(), billingqueue.BillTypeCreditMemo).
		Where(inv.TotalAmountMinor.Qualified() + " > " + inv.AppliedAmountMinor.Qualified())
}

func boundedLimit(limit, ceiling int) int {
	if limit <= 0 || limit > ceiling {
		return ceiling
	}
	return limit
}
