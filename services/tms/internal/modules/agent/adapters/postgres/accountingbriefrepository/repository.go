package accountingbriefrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/accountingbriefcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDeleteBatch = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) accountingbriefports.AccountingBriefRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-brief-repository"),
	}
}

func (r *repository) Latest(
	ctx context.Context,
	req *accountingbriefports.GetLatestAccountingBriefRequest,
) (*accountingbrief.Brief, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*accountingbrief.Brief, error) {
		return latest(ctx, r.db.DBForContext(ctx), req)
	})
}

func latest(
	ctx context.Context,
	db bun.IDB,
	req *accountingbriefports.GetLatestAccountingBriefRequest,
) (*accountingbrief.Brief, error) {
	cols := accountingbriefcol.BriefColumns
	brief := new(accountingbrief.Brief)
	err := db.NewSelect().
		Model(brief).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.BriefDate.Eq(), req.BriefDate).
		Order(cols.Generation.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no brief has been written for the day yet
	}
	if err != nil {
		return nil, fmt.Errorf("read latest accounting brief: %w", err)
	}

	return brief, nil
}

func (r *repository) Insert(
	ctx context.Context,
	brief *accountingbrief.Brief,
) (*accountingbrief.Brief, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accountingbrief.Brief, error) {
		cols := accountingbriefcol.BriefColumns
		db := r.db.DBForContext(ctx)
		result, err := db.NewInsert().
			Model(brief).
			On("CONFLICT (" + cols.OrganizationID.Bare() + ", " + cols.BusinessUnitID.Bare() +
				", " + cols.BriefDate.Bare() + ", " + cols.Generation.Bare() + ") DO NOTHING").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to insert accounting brief", zap.Error(err))
			return nil, fmt.Errorf("insert accounting brief: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			return brief, nil
		}

		return latest(ctx, db, &accountingbriefports.GetLatestAccountingBriefRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: brief.OrganizationID,
				BuID:  brief.BusinessUnitID,
			},
			BriefDate: brief.BriefDate,
		})
	})
}

func (r *repository) MarkVerified(
	ctx context.Context,
	req *accountingbriefports.MarkAccountingBriefVerifiedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := accountingbriefcol.BriefColumns
		_, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*accountingbrief.Brief)(nil)).
			Set(cols.VerifiedAt.Set(), req.VerifiedAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return accountingbriefcol.BriefScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.VerifiedAt.Lt(), req.VerifiedAt)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("mark accounting brief verified: %w", err)
		}

		return nil
	})
}

func (r *repository) DeleteBefore(
	ctx context.Context,
	req *accountingbriefports.DeleteAccountingBriefsBeforeRequest,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete accounting briefs past retention across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxDeleteBatch {
			limit = maxDeleteBatch
		}

		cols := accountingbriefcol.BriefColumns
		res, err := r.db.DBForContext(ctx).NewDelete().
			Model((*accountingbrief.Brief)(nil)).
			Where(
				cols.ID.Qualified()+" IN (SELECT "+cols.ID.Qualified()+" FROM "+
					accountingbriefcol.BriefTable.Name+" AS "+accountingbriefcol.BriefTable.Alias+
					" WHERE "+cols.BriefDate.Qualified()+" < ? ORDER BY "+
					cols.BriefDate.Qualified()+" LIMIT ?)",
				req.BeforeDate,
				limit,
			).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("delete accounting briefs: %w", err)
		}
		affected, _ := res.RowsAffected()

		return int(affected), nil
	})
}
