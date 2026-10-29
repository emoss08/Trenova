package documentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var documentControlErrors = dberror.Mapping{Entity: "DocumentControl"}

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
) (*documentcontrol.DocumentControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentcontrol.DocumentControl, error) {
			entity, err := r.find(ctx)
			if err != nil {
				r.l.Error(
					"failed to get document control",
					zap.String("orgID", req.TenantInfo.OrgID.String()),
					zap.String("buID", req.TenantInfo.BuID.String()),
					zap.Error(err),
				)
				return nil, documentControlErrors.Map(err, req.TenantInfo.OrgID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentcontrol.DocumentControl,
) (*documentcontrol.DocumentControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentcontrol.DocumentControl, error) {
			if err := tenantdb.DocumentControls.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, documentControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentcontrol.DocumentControl,
) (*documentcontrol.DocumentControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentcontrol.DocumentControl, error) {
			if err := tenantdb.DocumentControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, documentControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*documentcontrol.DocumentControl, error) {
	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*documentcontrol.DocumentControl]{
		Find: func(ctx context.Context) (*documentcontrol.DocumentControl, error) {
			return dbtx.Read(ctx, r.db, r.find)
		},
		Create: func(ctx context.Context) error {
			dc := tenantdb.DocumentControls
			_, err := dc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*documentcontrol.DocumentControl{
					documentcontrol.NewDefaultDocumentControl(orgID, buID),
				},
				dc.OrganizationID,
				dc.BusinessUnitID,
			)
			if err != nil {
				return dberror.MapRetryableTransactionError(
					err,
					"Document control is busy. Retry the request.",
				)
			}

			return nil
		},
		Default: func() *documentcontrol.DocumentControl {
			return documentcontrol.NewDefaultDocumentControl(orgID, buID)
		},
		MapError: func(err error) error {
			return documentControlErrors.Map(err, orgID.String())
		},
	})
}

func (r *repository) find(ctx context.Context) (*documentcontrol.DocumentControl, error) {
	return tenantdb.DocumentControls.Cache(resultcache.Settings()...).One(ctx, r.db.Keel(ctx))
}
