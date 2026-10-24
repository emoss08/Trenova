package accessorialchargerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var accessorialChargeErrors = dberror.Mapping{
	Entity: "AccessorialCharge",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_accessorial_charges_code",
			Field:   "code",
			Message: "Accessorial charge with this code already exists in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) accessorialchargeports.AccessorialChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accessorial-charge-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *accessorialchargeports.ListAccessorialChargeRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
			ac := accessorialchargedb.AccessorialCharges
			result, err := listquery.List(ctx, r.db.Keel(ctx), ac.Query(),
				&listquery.Request[accessorialcharge.AccessorialCharge]{
					Filter: req.Filter,
					Then:   []keel.Order{ac.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list accessorial charges", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *accessorialchargeports.ListAccessorialChargeConnectionRequest,
) (*pagination.CursorListResult[*accessorialcharge.AccessorialCharge], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accessorialcharge.AccessorialCharge], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accessorialchargedb.AccessorialCharges.Query(),
				&listquery.Request[accessorialcharge.AccessorialCharge]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.AccessorialChargeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list accessorial charges", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accessorialcharge.AccessorialCharge, error) {
			err := accessorialchargedb.AccessorialCharges.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create accessorial charge",
					zap.String("code", entity.Code),
					zap.Error(err),
				)
				return nil, accessorialChargeErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) (*accessorialcharge.AccessorialCharge, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accessorialcharge.AccessorialCharge, error) {
			err := accessorialchargedb.AccessorialCharges.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update accessorial charge",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, accessorialChargeErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req accessorialchargeports.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accessorialcharge.AccessorialCharge, error) {
			entity, err := accessorialchargedb.AccessorialCharges.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, accessorialChargeErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListCatalog(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*accessorialcharge.AccessorialCharge, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accessorialcharge.AccessorialCharge, error) {
			ac := accessorialchargedb.AccessorialCharges
			entities, err := ac.Pick(
				ac.ID,
				ac.OrganizationID,
				ac.BusinessUnitID,
				ac.Code,
				ac.Description,
				ac.Status,
				ac.Method,
				ac.RateUnit,
				ac.Amount,
			).
				Order(ac.Code.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list the accessorial charge catalog", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
			ac := accessorialchargedb.AccessorialCharges
			q := ac.Pick(
				ac.ID,
				ac.Code,
				ac.Description,
				ac.Status,
				ac.Method,
				ac.RateUnit,
				ac.Amount,
			).Where(ac.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req,
				func(s string) keel.Pred {
					return keel.Or(ac.Code.ContainsFold(s), ac.Description.ContainsFold(s))
				},
			)
		},
	)
}
