package invoicerunrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun/invoicerundb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var runErrors = dberror.Mapping{
	Entity: "Invoice run",
	Unique: []dberror.Constraint{
		{
			Name:    "uk_invoice_runs_number",
			Field:   "number",
			Message: "Invoice run with this number already exists in your organization",
		},
		{
			Name:    "uq_invoice_runs_scheduled_period",
			Field:   "periodEnd",
			Message: "A scheduled invoice run already exists for this cycle and period",
		},
	},
}

var groupErrors = dberror.Mapping{Entity: "Invoice run group"}

var itemErrors = dberror.Mapping{Entity: "Invoice run group item"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) invoicerunports.InvoiceRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-run-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *invoicerunports.ListInvoiceRunsRequest,
) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				invoicerundb.InvoiceRuns.Query(),
				&listquery.Request[invoicerun.InvoiceRun]{Filter: req.Filter},
			)
			if err != nil {
				r.l.Error("failed to list invoice runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *invoicerunports.ListInvoiceRunConnectionRequest,
) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				invoicerundb.InvoiceRuns.Query(),
				&listquery.Request[invoicerun.InvoiceRun]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list invoice runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func runQuery(includeGroups, includeItems bool) *keel.Query[invoicerun.InvoiceRun] {
	run := invoicerundb.InvoiceRuns
	if !includeGroups && !includeItems {
		return run.Query()
	}

	groups := run.Groups()
	items := groups.Items()
	nested := []keel.Loader{}
	if includeGroups {
		nested = append(nested, groups.Customer())
	}
	if includeItems {
		nested = append(nested, items.Order(items.SortKey.Asc()))
	}

	return run.Load(groups.Order(groups.GroupLabel.Asc()).Load(nested...))
}

func (r *repository) GetByID(
	ctx context.Context,
	req invoicerunports.GetInvoiceRunByIDRequest,
) (*invoicerun.InvoiceRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoicerun.InvoiceRun, error) {
		entity, err := runQuery(req.IncludeGroups, req.IncludeItems).
			Where(invoicerundb.InvoiceRuns.ID.Eq(req.ID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, runErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetOpenScheduledRun(
	ctx context.Context,
	req invoicerunports.GetOpenScheduledRunRequest,
) (*invoicerun.InvoiceRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoicerun.InvoiceRun, error) {
		run := invoicerundb.InvoiceRuns
		entity, err := run.Where(
			run.Source.Eq(invoicerun.SourceScheduled),
			run.Cycle.Eq(customer.BillingCycle(req.Cycle)),
			run.PeriodEnd.Eq(req.PeriodEnd),
			run.Status.Ne(invoicerun.StatusCanceled),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, runErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoicerun.InvoiceRun,
) (*invoicerun.InvoiceRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoicerun.InvoiceRun, error) {
		if err := invoicerundb.InvoiceRuns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create invoice run", zap.Error(err))
			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoicerun.InvoiceRun,
) (*invoicerun.InvoiceRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoicerun.InvoiceRun, error) {
		if err := invoicerundb.InvoiceRuns.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update invoice run", zap.Error(err))
			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ReplaceGroups(
	ctx context.Context,
	req *invoicerunports.ReplaceGroupsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		grp := invoicerundb.InvoiceRunGroups
		if _, err := grp.Where(grp.RunID.Eq(req.RunID)).Delete(ctx, db); err != nil {
			return groupErrors.Map(err, req.RunID.String())
		}

		if len(req.Groups) == 0 {
			return nil
		}

		if err := grp.CreateMany(ctx, db, req.Groups); err != nil {
			r.l.Error("failed to create invoice run groups", zap.Error(err))
			return groupErrors.Map(err, req.RunID.String())
		}

		items := make([]*invoicerun.InvoiceRunGroupItem, 0, len(req.Groups))
		for _, group := range req.Groups {
			for _, item := range group.Items {
				if item == nil {
					continue
				}
				item.RunID = req.RunID
				item.GroupID = group.ID
				item.SyncMinorAmount()
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			return nil
		}

		if err := invoicerundb.InvoiceRunGroupItems.CreateMany(ctx, db, items); err != nil {
			r.l.Error("failed to create invoice run group items", zap.Error(err))
			return itemErrors.Map(err, req.RunID.String())
		}

		return nil
	})
}

func (r *repository) UpdateGroup(
	ctx context.Context,
	entity *invoicerun.InvoiceRunGroup,
) (*invoicerun.InvoiceRunGroup, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoicerun.InvoiceRunGroup, error) {
		if err := invoicerundb.InvoiceRunGroups.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update invoice run group", zap.Error(err))
			return nil, groupErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateItems(
	ctx context.Context,
	_ pagination.TenantInfo,
	items []*invoicerun.InvoiceRunGroupItem,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(items) == 0 {
			return nil
		}

		ids := make([]pulid.ID, len(items))
		groups := make([]pulid.ID, len(items))
		excluded := make([]bool, len(items))
		reasons := make([]string, len(items))
		sortKeys := make([]int64, len(items))
		for i, item := range items {
			item.SyncMinorAmount()
			ids[i] = item.ID
			groups[i] = item.GroupID
			excluded[i] = item.Excluded
			reasons[i] = item.ExclusionReason
			sortKeys[i] = int64(item.SortKey)
		}

		itm := invoicerundb.InvoiceRunGroupItems
		edits := keel.Unnest(
			"edits",
			[]string{"id", "group_id", "excluded", "exclusion_reason", "sort_key"},
			ids, groups, excluded, reasons, sortKeys,
		)
		_, err := itm.With(edits).
			Join(edits, edits.Col("id").EqCol(itm.ID)).
			Where(keel.True()).
			Update(ctx, r.db.Keel(ctx),
				itm.GroupID.ToExpr(edits.Col("group_id")),
				itm.Excluded.ToExpr(edits.Col("excluded")),
				itm.ExclusionReason.ToExpr(keel.DynOf(edits.Col("exclusion_reason")).NullIf("")),
				itm.SortKey.ToExpr(edits.Col("sort_key")),
				itm.Version.ToExpr(itm.Version),
			)
		if err != nil {
			r.l.Error("failed to update invoice run group items", zap.Error(err))
			return itemErrors.Map(err, "")
		}

		return nil
	})
}
