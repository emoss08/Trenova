package glaccountrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var glAccountErrors = dberror.Mapping{
	Entity: "GLAccount",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_gl_accounts_code",
			Field:   "accountCode",
			Message: "GL account with this account code already exists in your organization",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_gl_accounts_account_type",
			Field:   "accountTypeId",
			Message: "Account type does not exist in your organization",
		},
		{
			Name:    "fk_gl_accounts_parent",
			Field:   "parentId",
			Message: "Parent account does not exist or still has child accounts",
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

func New(p Params) glaccountports.GLAccountRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.glaccount-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *glaccountports.ListGLAccountsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*glaccount.GLAccount], error) {
			gla := glaccountdb.GLAccounts
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				gla.Query(),
				&listquery.Request[glaccount.GLAccount]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[glaccount.GLAccount]) *keel.Query[glaccount.GLAccount] {
						return q.Load(gla.AccountType())
					},
					Then: []keel.Order{gla.AccountCode.Asc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list gl accounts", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req glaccountports.GetGLAccountByIDRequest,
) (*glaccount.GLAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*glaccount.GLAccount, error) {
		gla := glaccountdb.GLAccounts
		entity, err := gla.Where(gla.ID.Eq(req.ID)).
			Load(gla.AccountType()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, glAccountErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*glaccount.GLAccount, error) {
		if err := glaccountdb.GLAccounts.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create gl account",
				zap.String("accountCode", entity.AccountCode),
				zap.Error(err),
			)
			return nil, glAccountErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *glaccount.GLAccount,
) (*glaccount.GLAccount, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*glaccount.GLAccount, error) {
		gla := glaccountdb.GLAccounts
		err := gla.SaveExcept(
			ctx,
			r.db.Keel(ctx),
			entity,
			gla.CurrentBalance,
			gla.DebitBalance,
			gla.CreditBalance,
		)
		if err != nil {
			r.l.Error(
				"failed to update gl account",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, glAccountErrors.Map(err, entity.ID.String())
		}

		stored, err := gla.Get(ctx, r.db.Keel(ctx), entity.ID)
		if err != nil {
			return nil, glAccountErrors.Map(err, entity.ID.String())
		}

		return stored, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *glaccountports.BulkUpdateGLAccountStatusRequest,
) ([]*glaccount.GLAccount, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*glaccount.GLAccount, error) {
		gla := glaccountdb.GLAccounts
		entities, err := gla.Where(gla.ID.In(req.GLAccountIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), gla.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update gl account status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "GLAccount", req.GLAccountIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req glaccountports.GetGLAccountsByIDsRequest,
) ([]*glaccount.GLAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*glaccount.GLAccount, error) {
		if len(req.GLAccountIDs) == 0 {
			return []*glaccount.GLAccount{}, nil
		}

		gla := glaccountdb.GLAccounts
		q := gla.Where(gla.ID.In(req.GLAccountIDs...)).Limit(len(req.GLAccountIDs))
		if req.IncludeAccountType {
			q = q.Load(gla.AccountType())
		}

		entities, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get gl accounts", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req glaccountports.DeleteGLAccountRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := glaccountdb.GLAccounts.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&glaccount.GLAccount{ID: req.ID},
		)
		if err != nil {
			r.l.Error(
				"failed to delete gl account",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return glAccountErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *glaccountports.GLAccountSelectOptionsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*glaccount.GLAccount], error) {
			gla := glaccountdb.GLAccounts
			q := gla.Pick(gla.ID, gla.AccountCode, gla.Name).
				Where(gla.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(gla.AccountCode.ContainsFold(s), gla.Name.ContainsFold(s))
				},
			)
		},
	)
}
