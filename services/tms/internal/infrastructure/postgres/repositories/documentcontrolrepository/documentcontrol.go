package documentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.DocumentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.documentcontrol-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetDocumentControlRequest,
) (*tenant.DocumentControl, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
		zap.String("buID", req.TenantInfo.BuID.String()),
	)

	entity := new(tenant.DocumentControl)
	if err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentControlScopeTenant(sq, req.TenantInfo)
		}).
		Scan(ctx); err != nil {
		log.Error("failed to get document control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentControl")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tenant.DocumentControl,
) (*tenant.DocumentControl, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DocumentControl,
) (*tenant.DocumentControl, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.DocumentControlColumns

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "DocumentControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*tenant.DocumentControl, error) {
	entity, err := r.Get(ctx, repositories.GetDocumentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	})
	if err == nil {
		return entity, nil
	}
	if !dberror.IsNotFoundError(err) {
		return nil, err
	}

	defaultEntity := tenant.NewDefaultDocumentControl(orgID, buID)
	created, createErr := r.Create(ctx, defaultEntity)
	if createErr == nil {
		return created, nil
	}

	entity = new(tenant.DocumentControl)
	if err = r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentControlScopeTenant(sq, pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			})
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "DocumentControl")
	}

	return entity, nil
}
