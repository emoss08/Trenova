package invoicerunrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.InvoiceRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-run-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListInvoiceRunsRequest,
) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
	entities := make([]*invoicerun.InvoiceRun, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFilters(
				sq,
				buncolgen.InvoiceRunTable.Alias,
				req.Filter,
				(*invoicerun.InvoiceRun)(nil),
			).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list invoice runs", zap.Error(err))
		return nil, fmt.Errorf("list invoice runs: %w", err)
	}

	return &pagination.ListResult[*invoicerun.InvoiceRun]{Items: entities, Total: total}, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListInvoiceRunConnectionRequest,
) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error) {
	dba := r.db.DBForContext(ctx)

	// Only pay for the COUNT when the caller actually selected totalCount.
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*invoicerun.InvoiceRun)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return querybuilder.ApplyFilters(
					sq,
					buncolgen.InvoiceRunTable.Alias,
					req.Filter,
					(*invoicerun.InvoiceRun)(nil),
				)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count invoice runs", zap.Error(err))
			return nil, fmt.Errorf("count invoice runs: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*invoicerun.InvoiceRun]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*invoicerun.InvoiceRun) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.InvoiceRunTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyFilters(
					sq,
					buncolgen.InvoiceRunTable.Alias,
					req.Filter,
					(*invoicerun.InvoiceRun)(nil),
				), nil
			},
		})
	if err != nil {
		r.l.Error("failed to scan invoice runs", zap.Error(err))
		return nil, fmt.Errorf("scan invoice runs: %w", err)
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceRunByIDRequest,
) (*invoicerun.InvoiceRun, error) {
	run := buncolgen.InvoiceRunColumns

	entity := new(invoicerun.InvoiceRun)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(run.ID.Eq(), req.ID)

	if req.IncludeGroups {
		q = q.Relation("Groups", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.InvoiceRunGroupColumns.GroupLabel.OrderAsc())
		}).Relation("Groups.Customer")
	}
	if req.IncludeItems {
		q = q.Relation("Groups.Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.InvoiceRunGroupItemColumns.SortKey.OrderAsc())
		})
	}

	if err := buncolgen.InvoiceRunScopeTenant(q, req.TenantInfo).Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice run")
	}

	return entity, nil
}

func (r *repository) GetOpenScheduledRun(
	ctx context.Context,
	req repositories.GetOpenScheduledRunRequest,
) (*invoicerun.InvoiceRun, error) {
	run := buncolgen.InvoiceRunColumns

	entity := new(invoicerun.InvoiceRun)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(run.Source.Eq(), invoicerun.SourceScheduled).
		Where(run.Cycle.Eq(), req.Cycle).
		Where(run.PeriodEnd.Eq(), req.PeriodEnd).
		Where(run.Status.Ne(), invoicerun.StatusCanceled)

	if err := buncolgen.InvoiceRunScopeTenant(q, req.TenantInfo).Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice run")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoicerun.InvoiceRun,
) (*invoicerun.InvoiceRun, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create invoice run", zap.Error(err))
		return nil, fmt.Errorf("create invoice run: %w", err)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoicerun.InvoiceRun,
) (*invoicerun.InvoiceRun, error) {
	run := buncolgen.InvoiceRunColumns
	version := entity.Version
	entity.Version++

	result, err := buncolgen.InvoiceRunScopeTenantUpdate(
		r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(run.Version.Eq(), version).
			OmitZero(),
		pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	).Returning("*").Exec(ctx)
	if err != nil {
		r.l.Error("failed to update invoice run", zap.Error(err))
		return nil, fmt.Errorf("update invoice run: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Invoice run", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// ReplaceGroups swaps the run's whole proposal. Building is not incremental — a
// rebuild must not leave a stale group behind — so the old groups are deleted
// and the new set written in one transaction.
func (r *repository) ReplaceGroups(
	ctx context.Context,
	req *repositories.ReplaceGroupsRequest,
) error {
	grp := buncolgen.InvoiceRunGroupColumns

	return r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		// The item rows cascade from the group delete.
		if _, err := tx.NewDelete().
			Model((*invoicerun.InvoiceRunGroup)(nil)).
			Where(grp.RunID.Eq(), req.RunID).
			Where(grp.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(grp.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Exec(txCtx); err != nil {
			return fmt.Errorf("clear invoice run groups: %w", err)
		}

		if len(req.Groups) == 0 {
			return nil
		}

		if _, err := tx.NewInsert().Model(&req.Groups).Exec(txCtx); err != nil {
			return fmt.Errorf("insert invoice run groups: %w", err)
		}

		items := make([]*invoicerun.InvoiceRunGroupItem, 0, len(req.Groups))
		for _, group := range req.Groups {
			for _, item := range group.Items {
				if item == nil {
					continue
				}
				item.RunID = req.RunID
				item.GroupID = group.ID
				item.OrganizationID = group.OrganizationID
				item.BusinessUnitID = group.BusinessUnitID
				item.SyncMinorAmount()
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			return nil
		}

		if _, err := tx.NewInsert().Model(&items).Exec(txCtx); err != nil {
			return fmt.Errorf("insert invoice run group items: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateGroup(
	ctx context.Context,
	entity *invoicerun.InvoiceRunGroup,
) (*invoicerun.InvoiceRunGroup, error) {
	grp := buncolgen.InvoiceRunGroupColumns
	version := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(grp.Version.Eq(), version).
		Where(grp.OrganizationID.Eq(), entity.OrganizationID).
		Where(grp.BusinessUnitID.Eq(), entity.BusinessUnitID).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update invoice run group", zap.Error(err))
		return nil, fmt.Errorf("update invoice run group: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Invoice run group", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// UpdateItems writes membership changes for a whole edit in one statement, so an
// operator's move-and-exclude lands atomically rather than row by row.
func (r *repository) UpdateItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	items []*invoicerun.InvoiceRunGroupItem,
) error {
	if len(items) == 0 {
		return nil
	}

	itm := buncolgen.InvoiceRunGroupItemColumns
	for _, item := range items {
		item.SyncMinorAmount()
	}

	if _, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(&items).
		Column(
			itm.GroupID.Name,
			itm.Excluded.Name,
			itm.ExclusionReason.Name,
			itm.SortKey.Name,
			itm.UpdatedAt.Name,
		).
		Bulk().
		Where(itm.OrganizationID.Qualified()+" = ?", tenantInfo.OrgID).
		Where(itm.BusinessUnitID.Qualified()+" = ?", tenantInfo.BuID).
		Exec(ctx); err != nil {
		r.l.Error("failed to update invoice run group items", zap.Error(err))
		return fmt.Errorf("update invoice run group items: %w", err)
	}

	return nil
}
