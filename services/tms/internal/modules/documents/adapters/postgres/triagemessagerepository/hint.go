package triagemessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
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

func (r *hintRepository) Upsert(
	ctx context.Context,
	req *triageports.UpsertSenderHintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		h := triagedb.SenderHints
		hint := &triage.SenderHint{
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			SenderDomain:    req.SenderDomain,
			Classification:  req.Classification,
			Corrections:     1,
			LastCorrectedAt: req.At,
		}

		_, err := h.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*triage.SenderHint{hint},
			keel.OnConflict(h.OrganizationID, h.BusinessUnitID, h.SenderDomain, h.Classification).
				Update(h.LastCorrectedAt).
				Set(h.Corrections.Add(1)),
		)
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
		h := triagedb.SenderHints
		hints, err := h.Where(h.SenderDomain.Eq(req.SenderDomain)).
			Order(h.Corrections.Desc(), h.LastCorrectedAt.Desc()).
			Limit(max(req.Limit, 1)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if hints == nil {
			hints = []*triage.SenderHint{}
		}

		return hints, nil
	})
}
