package documentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
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

func New(p Params) tenantports.DocumentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.documentcontrol-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req tenantports.GetDocumentControlRequest,
) (*tenant.DocumentControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.DocumentControl, error) {
		log := r.l.With(
			zap.String("operation", "Get"),
			zap.String("orgID", req.TenantInfo.OrgID.String()),
			zap.String("buID", req.TenantInfo.BuID.String()),
		)

		entity := new(tenant.DocumentControl)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return tenantcol.DocumentControlScopeTenant(sq, req.TenantInfo)
			}).
			Scan(ctx); err != nil {
			log.Error("failed to get document control", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "DocumentControl")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *tenant.DocumentControl,
) (*tenant.DocumentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.DocumentControl, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DocumentControl,
) (*tenant.DocumentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.DocumentControl, error) {
		ov := entity.Version
		entity.Version++
		cols := tenantcol.DocumentControlColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			result,
			"DocumentControl",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*tenant.DocumentControl, error) {
	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.DocumentControl]{
		Find: func(ctx context.Context) (*tenant.DocumentControl, error) {
			found := new(tenant.DocumentControl)
			if scanErr := r.db.DBForContext(ctx).
				NewSelect().
				Model(found).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return tenantcol.DocumentControlScopeTenant(sq, pagination.TenantInfo{
						OrgID: orgID,
						BuID:  buID,
					})
				}).
				Scan(ctx); scanErr != nil {
				return nil, scanErr
			}

			return found, nil
		},
		Create: func(ctx context.Context) error {
			_, insertErr := r.db.DBForContext(ctx).
				NewInsert().
				Model(tenant.NewDefaultDocumentControl(orgID, buID)).
				On(`CONFLICT ("organization_id", "business_unit_id") DO NOTHING`).
				Exec(ctx)
			if insertErr != nil {
				return dberror.MapRetryableTransactionError(
					insertErr,
					"Document control is busy. Retry the request.",
				)
			}

			return nil
		},
		Default: func() *tenant.DocumentControl {
			return tenant.NewDefaultDocumentControl(orgID, buID)
		},
		MapError: func(err error) error {
			return dberror.HandleNotFoundError(err, "DocumentControl")
		},
	})
}
