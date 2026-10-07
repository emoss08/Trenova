package aituneuprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const entityName = "AITuneUp"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AITuneUpRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*aituneup.TuneUp, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aituneup.TuneUp, error) {
		cols := buncolgen.TuneUpColumns
		tuneUps := make([]*aituneup.TuneUp, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&tuneUps).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TuneUpScopeTenant(sq, tenantInfo)
			}).
			Order(cols.CreatedAt.OrderAsc()).
			Order(cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list ai tune-ups: %w", err)
		}
		return tuneUps, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		entity := new(aituneup.TuneUp)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TuneUpScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.TuneUpColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}
		return entity, nil
	})
}

func (r *repository) Save(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *aituneup.Plan,
) error {
	if plan.Empty() {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)
		if len(plan.Delete) > 0 {
			if err := r.delete(ctx, db, tenantInfo, plan.Delete); err != nil {
				return err
			}
		}
		for _, tuneUp := range plan.Update {
			if err := r.refresh(ctx, db, tenantInfo, tuneUp); err != nil {
				return err
			}
		}
		if len(plan.Insert) > 0 {
			for _, tuneUp := range plan.Insert {
				tuneUp.OrganizationID = tenantInfo.OrgID
				tuneUp.BusinessUnitID = tenantInfo.BuID
			}
			if _, err := db.NewInsert().
				Model(&plan.Insert).
				On("CONFLICT (organization_id, business_unit_id, fingerprint) DO NOTHING").
				Exec(ctx); err != nil {
				return fmt.Errorf("insert ai tune-ups: %w", err)
			}
		}
		return nil
	})
}

func (r *repository) delete(
	ctx context.Context,
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) error {
	_, err := db.NewDelete().
		Model((*aituneup.TuneUp)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.TuneUpScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.TuneUpColumns.ID.In(), bun.In(ids))
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete ai tune-ups: %w", err)
	}
	return nil
}

func (r *repository) refresh(
	ctx context.Context,
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	tuneUp *aituneup.TuneUp,
) error {
	cols := buncolgen.TuneUpColumns
	_, err := db.NewUpdate().
		Model(tuneUp).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TuneUpScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.Eq(), tuneUp.ID).
				Where(cols.Version.Eq(), tuneUp.Version)
		}).
		Set(cols.AgentDefinitionID.Set(), tuneUp.AgentDefinitionID).
		Set(cols.ProviderID.Set(), tuneUp.ProviderID).
		Set(cols.OtherProviderID.Set(), tuneUp.OtherProviderID).
		Set(cols.ToolName.Set(), tuneUp.ToolName).
		Set(cols.Task.Set(), tuneUp.Task).
		Set(cols.Evidence.Set(), tuneUp.Evidence).
		Set(cols.Status.Set(), tuneUp.Status).
		Set(cols.DismissedUntil.Set(), tuneUp.DismissedUntil).
		Set(cols.DecidedByID.Set(), tuneUp.DecidedByID).
		Set(cols.DecidedAt.Set(), tuneUp.DecidedAt).
		Set(cols.ComputedAt.Set(), tuneUp.ComputedAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), tuneUp.Version+1).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("refresh ai tune-up: %w", err)
	}
	return nil
}

func (r *repository) Decide(
	ctx context.Context,
	req *repositories.DecideAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		cols := buncolgen.TuneUpColumns
		entity := new(aituneup.TuneUp)
		decidedAt := req.DecidedAt
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.TuneUpScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.DismissedUntil.Set(), req.DismissedUntil).
			Set(cols.DecidedByID.Set(), req.DecidedByID).
			Set(cols.DecidedAt.Set(), &decidedAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Set(), req.Version+1).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("decide ai tune-up: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, entityName, req.ID.String()); err != nil {
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) Reopen(
	ctx context.Context,
	req *repositories.ReopenAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		cols := buncolgen.TuneUpColumns
		entity := new(aituneup.TuneUp)
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.TuneUpScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version).
					Where(cols.Status.Eq(), aituneup.StatusDismissed)
			}).
			Set(cols.Status.Set(), aituneup.StatusOpen).
			Set(cols.DismissedUntil.Set(), nil).
			Set(cols.DecidedByID.Set(), nil).
			Set(cols.DecidedAt.Set(), nil).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Set(), req.Version+1).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("reopen ai tune-up: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, entityName, req.ID.String()); err != nil {
			return nil, err
		}
		return entity, nil
	})
}
