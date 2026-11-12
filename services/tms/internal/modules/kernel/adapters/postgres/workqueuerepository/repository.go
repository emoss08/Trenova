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

const (
	dueTenantsReason = "list the tenants with due background work of a kind across every organization"
	dueKindsReason   = "list the tenants and kinds with due background work across every organization"
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
			items = append(items, &workqueue.Item{
				Kind:      req.Kind,
				SubjectID: id,
				DueAt:     req.At,
				Detail:    req.Detail,
			})
		}

		wq := workqueuedb.Items
		_, err := wq.Upsert(ctx, r.db.Keel(ctx), items,
			keel.OnConflict(wq.OrganizationID, wq.BusinessUnitID, wq.Kind, wq.SubjectID).Set(
				wq.DueAt.ToExpr(keel.Raw("GREATEST(? + 1, ?)", wq.DueAt, wq.DueAt.Excluded())),
				wq.Detail.ToExpr(wq.Detail.Excluded()),
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
			wq.Pick(wq.SubjectID, wq.DueAt, wq.Detail.Coalesce("")).
				Where(wq.Kind.Eq(req.Kind), wq.DueAt.Lte(req.Now)).
				Order(wq.DueAt.Asc(), wq.SubjectID.Asc()).
				Limit(req.Limit),
		)
	})
}

func (r *repository) SettleMany(ctx context.Context, req *workqueueports.SettleManyRequest) error {
	if len(req.Done) == 0 && len(req.Retry) == 0 {
		return nil
	}

	ctx = dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wq := workqueuedb.Items
		db := r.db.Keel(ctx)
		if len(req.Done) > 0 {
			claimed := claimedRows("settled", req.Done)
			if _, err := wq.With(claimed).
				Join(claimed, heldClaim(wq, claimed)).
				Where(wq.Kind.Eq(req.Kind)).
				Delete(ctx, db); err != nil {
				return err
			}
		}
		if len(req.Retry) > 0 {
			claimed := claimedRows("retried", req.Retry)
			if _, err := wq.With(claimed).
				Join(claimed, heldClaim(wq, claimed)).
				Where(wq.Kind.Eq(req.Kind)).
				Update(ctx, db, wq.DueAt.To(req.RetryAt)); err != nil {
				return err
			}
		}

		return nil
	})
}

// claimedRows is a page's claimed rows as a table to join: each subject and
// the due time it was claimed at.
func claimedRows(name string, rows []workqueueports.Due) *keel.CTETable {
	subjects := make([]string, 0, len(rows))
	dueAts := make([]int64, 0, len(rows))
	for _, row := range rows {
		subjects = append(subjects, row.SubjectID.String())
		dueAts = append(dueAts, row.DueAt)
	}

	return keel.Unnest(name, []string{"subject_id", "due_at"}, subjects, dueAts)
}

// heldClaim matches a row to its claim only while it still holds the claimed
// due time: a request made while the work ran moved it forward and keeps it.
func heldClaim(wq *workqueuedb.ItemsTable, claimed *keel.CTETable) keel.Pred {
	return keel.And(
		claimed.Col("subject_id").EqCol(wq.SubjectID),
		claimed.Col("due_at").EqCol(wq.DueAt),
	)
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
			tenants = append(
				tenants,
				pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID},
			)
		}

		return tenants, nil
	})
}

func (r *repository) ListDueKinds(
	ctx context.Context,
	req *workqueueports.ListDueKindsRequest,
) ([]workqueueports.TenantKind, error) {
	ctx = dbscope.WithSystem(ctx, dueKindsReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workqueueports.TenantKind, error) {
		wq := workqueuedb.Items
		rows, err := keel.Into[dueKind](ctx, r.db.Keel(ctx),
			wq.Pick(wq.OrganizationID, wq.BusinessUnitID, wq.Kind).
				Where(wq.Kind.HasPrefix(req.Prefix), wq.DueAt.Lte(req.Now)).
				Distinct().
				Limit(req.Limit).
				AllTenants(),
		)
		if err != nil {
			return nil, err
		}

		kinds := make([]workqueueports.TenantKind, 0, len(rows))
		for _, row := range rows {
			kinds = append(kinds, workqueueports.TenantKind{
				TenantInfo: pagination.TenantInfo{
					OrgID: row.OrganizationID,
					BuID:  row.BusinessUnitID,
				},
				Kind: row.Kind,
			})
		}

		return kinds, nil
	})
}

type dueKind struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Kind           string
}

type dueTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}
