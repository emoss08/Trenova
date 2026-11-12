package workqueuerepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/workqueue"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/workqueue/workqueuedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const dueTenantsReason = "list the tenants with due background work of a kind across every organization"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workqueueports.Queue {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.work-queue-repository")}
}

func (r *repository) Enqueue(ctx context.Context, req *workqueueports.EnqueueRequest) error {
	ids := sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(req.SubjectIDs), pulid.ID.IsNil))
	if len(ids) == 0 || req.Kind == "" {
		return nil
	}

	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		items := make([]*workqueue.Item, 0, len(ids))
		for _, id := range ids {
			items = append(items, &workqueue.Item{Kind: req.Kind, SubjectID: id, DueAt: req.At})
		}

		wq := workqueuedb.Items
		_, err := wq.Upsert(ctx, r.db.Keel(ctx), items,
			keel.OnConflict(wq.OrganizationID, wq.BusinessUnitID, wq.Kind, wq.SubjectID).Set(
				wq.DueAt.ToExpr(keel.Raw("GREATEST(? + 1, ?)", wq.DueAt, wq.DueAt.Excluded())),
			),
		)

		return err
	})
}

func (r *repository) ListDue(
	ctx context.Context,
	req *workqueueports.ListDueRequest,
) ([]workqueueports.Due, error) {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workqueueports.Due, error) {
		wq := workqueuedb.Items
		return keel.Into[workqueueports.Due](ctx, r.db.Keel(ctx),
			wq.Pick(wq.SubjectID, wq.DueAt).
				Where(wq.Kind.Eq(req.Kind), wq.DueAt.Lte(req.Now)).
				Order(wq.DueAt.Asc(), wq.SubjectID.Asc()).
				Limit(req.Limit),
		)
	})
}

func (r *repository) Settle(ctx context.Context, req *workqueueports.SettleRequest) error {
	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wq := workqueuedb.Items
		claimed := wq.Where(
			wq.Kind.Eq(req.Kind),
			wq.SubjectID.Eq(req.SubjectID),
			wq.DueAt.Eq(req.ClaimedDueAt),
		)
		if req.RetryAt == nil {
			_, err := claimed.Delete(ctx, r.db.Keel(ctx))
			return err
		}
		_, err := claimed.Update(ctx, r.db.Keel(ctx), wq.DueAt.To(*req.RetryAt))

		return err
	})
}

func (r *repository) ListDueTenants(
	ctx context.Context,
	req *workqueueports.ListDueTenantsRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, dueTenantsReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		wq := workqueuedb.Items
		rows, err := keel.Into[dueTenant](ctx, r.db.Keel(ctx),
			wq.Pick(wq.OrganizationID, wq.BusinessUnitID).
				Where(wq.Kind.Eq(req.Kind), wq.DueAt.Lte(req.Now)).
				Distinct().
				Limit(req.Limit).
				AllTenants(),
		)
		if err != nil {
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID})
		}

		return tenants, nil
	})
}

type dueTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}
