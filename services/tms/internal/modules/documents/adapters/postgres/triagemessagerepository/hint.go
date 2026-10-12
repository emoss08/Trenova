package triagemessagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"go.uber.org/zap"
)

type hintRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSenderHintRepository(p Params) triageports.SenderHintRepository {
	return &hintRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sender-hint-repository"),
	}
}

func hintConflict() string {
	cols := triagecol.SenderHintColumns

	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.SenderDomain.Bare(),
		cols.Classification.Bare(),
	}, ", ") + ") DO UPDATE"
}

func (r *hintRepository) Upsert(
	ctx context.Context,
	req *triageports.UpsertSenderHintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := triagecol.SenderHintColumns
		hint := &triage.SenderHint{
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			SenderDomain:    req.SenderDomain,
			Classification:  req.Classification,
			Corrections:     1,
			LastCorrectedAt: req.At,
		}

		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(hint).
			On(hintConflict()).
			Set(cols.Corrections.IncConflict(1)).
			Set(cols.LastCorrectedAt.SetExcluded()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to record a sender hint",
				zap.String("senderDomain", req.SenderDomain), zap.Error(err))
		}

		return err
	})
}

func (r *hintRepository) ListByDomain(
	ctx context.Context,
	req triageports.ListSenderHintsRequest,
) ([]*triage.SenderHint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.SenderHint, error) {
		cols := triagecol.SenderHintColumns
		hints := make([]*triage.SenderHint, 0, req.Limit)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&hints).
			Apply(triagecol.SenderHintApplyTenant(req.TenantInfo)).
			Where(cols.SenderDomain.Eq(), req.SenderDomain).
			Order(cols.Corrections.OrderDesc(), cols.LastCorrectedAt.OrderDesc()).
			Limit(int64(req.Limit)).
			Scan(ctx); err != nil {
			return nil, err
		}

		return hints, nil
	})
}
