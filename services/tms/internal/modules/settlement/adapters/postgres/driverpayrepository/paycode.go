package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var payCodeErrors = dberror.Mapping{Entity: "PayCode"}

type payCodeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayCode(p Params) driverpayports.PayCodeRepository {
	return &payCodeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-code-repository"),
	}
}

func (r *payCodeRepository) List(
	ctx context.Context,
	req *driverpayports.ListPayCodesRequest,
) (*pagination.ListResult[*paycode.PayCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*paycode.PayCode], error) {
			pc := driverpaydb.PayCodes
			q := pc.Where(
				searchPred(req.Filter.Query, pc.Code, pc.Name),
				when(req.Direction != "", pc.Direction.Eq(req.Direction)),
			).Load(pc.GLAccount()).Order(pc.Direction.Asc(), pc.Code.Asc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list pay codes: %w", err)
			}

			return result, nil
		},
	)
}

func (r *payCodeRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListPayCodeConnectionRequest,
) (*pagination.CursorListResult[*paycode.PayCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*paycode.PayCode], error) {
			pc := driverpaydb.PayCodes
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				pc.Query(),
				&listquery.Request[paycode.PayCode]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*paycode.PayCode]) *keel.Query[*paycode.PayCode] {
						return q.Load(pc.GLAccount())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list pay codes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *payCodeRepository) ListActive(
	ctx context.Context,
	req driverpayports.ListActivePayCodesRequest,
) ([]*paycode.PayCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*paycode.PayCode, error) {
		pc := driverpaydb.PayCodes
		items, err := pc.Where(
			pc.Status.Eq(domaintypes.StatusActive),
			when(req.Direction != "", pc.Direction.Eq(req.Direction)),
		).Order(pc.Code.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list active pay codes: %w", err)
		}

		return items, nil
	})
}

func (r *payCodeRepository) SelectOptions(
	ctx context.Context,
	req *driverpayports.PayCodeSelectOptionsRequest,
) (*pagination.ListResult[*paycode.PayCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*paycode.PayCode], error) {
			pc := driverpaydb.PayCodes
			q := pc.Pick(
				pc.ID,
				pc.Code,
				pc.Name,
				pc.Description,
				pc.Direction,
				pc.Taxable,
				pc.CountsTowardGuarantee,
				pc.DefaultAmountMinor,
				pc.CreatedAt,
			).Where(
				pc.Status.Eq(domaintypes.StatusActive),
				when(req.Direction != "", pc.Direction.Eq(req.Direction)),
			).Order(pc.Code.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						pc.Code.ContainsFold(s),
						pc.Name.ContainsFold(s),
						pc.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *payCodeRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetPayCodeByIDRequest,
) (*paycode.PayCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*paycode.PayCode, error) {
		pc := driverpaydb.PayCodes
		entity, err := pc.Where(pc.ID.Eq(req.ID)).
			Load(pc.GLAccount()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, payCodeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *payCodeRepository) GetByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*paycode.PayCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*paycode.PayCode, error) {
		if len(ids) == 0 {
			return []*paycode.PayCode{}, nil
		}

		pc := driverpaydb.PayCodes
		items, err := pc.Where(pc.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("get pay codes by ids: %w", err)
		}

		return items, nil
	})
}

func (r *payCodeRepository) Create(
	ctx context.Context,
	entity *paycode.PayCode,
) (*paycode.PayCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*paycode.PayCode, error) {
		if err := driverpaydb.PayCodes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, payCodeErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetPayCodeByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *payCodeRepository) Update(
	ctx context.Context,
	entity *paycode.PayCode,
) (*paycode.PayCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*paycode.PayCode, error) {
		pc := driverpaydb.PayCodes
		if err := pc.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			pc.Status,
			pc.Code,
			pc.Name,
			pc.Description,
			pc.Taxable,
			pc.CountsTowardGuarantee,
			pc.GLAccountID,
			pc.DefaultAmountMinor,
		); err != nil {
			return nil, payCodeErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetPayCodeByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *payCodeRepository) EnsureSystemDefaults(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		defs := paycode.SystemPayCodes()
		rows := make([]*paycode.PayCode, 0, len(defs))
		for _, def := range defs {
			rows = append(rows, &paycode.PayCode{
				BusinessUnitID:        tenantInfo.BuID,
				OrganizationID:        tenantInfo.OrgID,
				Status:                domaintypes.StatusActive,
				Direction:             def.Direction,
				Code:                  def.Code,
				Name:                  def.Name,
				Taxable:               def.Taxable,
				CountsTowardGuarantee: true,
				IsSystem:              true,
			})
		}

		pc := driverpaydb.PayCodes
		if _, err := pc.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			rows,
			pc.OrganizationID,
			pc.BusinessUnitID,
			pc.Direction,
			pc.Code,
		); err != nil {
			return fmt.Errorf("ensure system pay codes: %w", err)
		}

		return nil
	})
}
