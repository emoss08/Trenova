package accountingbriefrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief/accountingbriefdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
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
		return r.latest(ctx, req.BriefDate)
	})
}

func (r *repository) latest(ctx context.Context, briefDate string) (*accountingbrief.Brief, error) {
	b := accountingbriefdb.Briefs
	brief, err := b.Where(b.BriefDate.Eq(briefDate)).
		Order(b.Generation.Desc()).
		First(ctx, r.db.Keel(ctx))
	if dberror.IsNotFoundError(err) {
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
		b := accountingbriefdb.Briefs
		inserted, err := b.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*accountingbrief.Brief{brief},
			b.OrganizationID,
			b.BusinessUnitID,
			b.BriefDate,
			b.Generation,
		)
		if err != nil {
			r.l.Error("failed to insert accounting brief", zap.Error(err))
			return nil, fmt.Errorf("insert accounting brief: %w", err)
		}
		if inserted > 0 {
			return brief, nil
		}

		return r.latest(ctx, brief.BriefDate)
	})
}

func (r *repository) MarkVerified(
	ctx context.Context,
	req *accountingbriefports.MarkAccountingBriefVerifiedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		b := accountingbriefdb.Briefs
		_, err := b.Where(b.ID.Eq(req.ID), b.VerifiedAt.Lt(req.VerifiedAt)).
			Update(ctx, r.db.Keel(ctx), b.VerifiedAt.To(req.VerifiedAt))
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

		b := accountingbriefdb.Briefs
		stale := b.As("stale")
		n, err := b.AllTenants().
			Where(b.ID.InQuery(stale.AllTenants().
				Pick(stale.ID).
				Where(stale.BriefDate.Lt(req.BeforeDate)).
				Order(stale.BriefDate.Asc()).
				Limit(limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("delete accounting briefs: %w", err)
		}

		return int(n), nil
	})
}
