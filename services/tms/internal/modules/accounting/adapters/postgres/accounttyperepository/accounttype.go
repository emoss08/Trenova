package accounttyperepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accounttypeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var accountTypeErrors = dberror.Mapping{
	Entity: "AccountType",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_account_types_unique_code",
			Field:   "code",
			Message: "Account type with this code already exists in your organization",
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

func New(p Params) accounttypeports.AccountTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounttype-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *accounttypeports.ListAccountTypesRequest,
) (*pagination.ListResult[*accounttype.AccountType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*accounttype.AccountType], error) {
			at := accounttypedb.AccountTypes
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				at.Query(),
				&listquery.Request[accounttype.AccountType]{
					Filter: req.Filter,
					Then:   []keel.Order{at.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list account types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req accounttypeports.GetAccountTypeByIDRequest,
) (*accounttype.AccountType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*accounttype.AccountType, error) {
		entity, err := accounttypedb.AccountTypes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, accountTypeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *accounttypeports.ListAccountTypesConnectionRequest,
) (*pagination.CursorListResult[*accounttype.AccountType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accounttype.AccountType], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accounttypedb.AccountTypes.Query(),
				&listquery.Request[accounttype.AccountType]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.AccountTypeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list account types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *accounttype.AccountType,
) (*accounttype.AccountType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accounttype.AccountType, error) {
		if err := accounttypedb.AccountTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create account type",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, accountTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *accounttype.AccountType,
) (*accounttype.AccountType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accounttype.AccountType, error) {
		if err := accounttypedb.AccountTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update account type",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, accountTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *accounttypeports.BulkUpdateAccountTypeStatusRequest,
) ([]*accounttype.AccountType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*accounttype.AccountType, error) {
		at := accounttypedb.AccountTypes
		entities, err := at.Where(at.ID.In(req.AccountTypeIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), at.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update account type status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "AccountType", req.AccountTypeIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req accounttypeports.GetAccountTypesByIDsRequest,
) ([]*accounttype.AccountType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*accounttype.AccountType, error) {
		if len(req.AccountTypeIDs) == 0 {
			return []*accounttype.AccountType{}, nil
		}

		at := accounttypedb.AccountTypes
		entities, err := at.Where(at.ID.In(req.AccountTypeIDs...)).
			Limit(len(req.AccountTypeIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get account types", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *accounttypeports.AccountTypeSelectOptionsRequest,
) (*pagination.ListResult[*accounttype.AccountType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*accounttype.AccountType], error) {
			at := accounttypedb.AccountTypes
			q := at.Pick(at.ID, at.Code, at.Name, at.Category).
				Where(at.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(at.Code.ContainsFold(s), at.Name.ContainsFold(s))
				},
			)
		},
	)
}
