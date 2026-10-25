package triagemessagerepository

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
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

func New(p Params) triageports.TriageMessageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-message-repository"),
	}
}

func (r *repository) ClaimRule(
	ctx context.Context,
	req *triageports.ClaimRuleRequest,
) (*triageports.ClaimedRule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triageports.ClaimedRule, error) {
		m := inboundmessagedb.InboundMessages
		claimed, err := claimRuleQuery(req).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			m.HandledByRuleID.To(req.RuleID),
			m.HandledAction.To(req.Action),
			m.UpdatedAt.To(req.At),
		)
		if err != nil {
			return nil, err
		}
		if len(claimed) == 0 {
			return &triageports.ClaimedRule{}, nil
		}

		return &triageports.ClaimedRule{Claimed: true, Version: claimed[0].Version}, nil
	})
}

func claimRuleQuery(req *triageports.ClaimRuleRequest) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return m.Pick(m.Version).Where(m.ID.Eq(req.MessageID), m.HandledByRuleID.IsNull())
}

func (r *repository) CorrectClassification(
	ctx context.Context,
	req *triageports.CorrectClassificationRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		m := inboundmessagedb.InboundMessages
		sets := []keel.Assignment{
			m.Classification.To(req.Classification),
			m.ClassificationCorrected.To(true),
			m.UpdatedAt.To(req.At),
		}
		if req.TargetSeconds > 0 {
			sets = append(sets, m.DueAt.ToExpr(keel.Raw(
				"CASE WHEN ? THEN ? + ? ELSE ? END",
				m.Status.In(inboundmessage.StatusInReview, inboundmessage.StatusQuarantined),
				m.ReceivedAt,
				req.TargetSeconds,
				m.DueAt,
			)))
		}

		n, err := m.Where(
			m.ID.Eq(req.MessageID),
			m.Classification.IsDistinctFrom(req.Classification),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return false, err
		}

		return n == 1, nil
	})
}

func (r *repository) SaveTriageFields(
	ctx context.Context,
	req *triageports.SaveTriageFieldsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		m := inboundmessagedb.InboundMessages
		sets := make([]keel.Assignment, 0, 2)
		if req.Facts != nil {
			encoded, err := jsonOrNull(*req.Facts)
			if err != nil {
				return fmt.Errorf("encode facts: %w", err)
			}
			sets = append(sets, m.Facts.ToExpr(keel.DynLit(encoded).Cast(keel.TypeJSON)))
		}
		if req.Conflicts != nil {
			encoded, err := jsonOrNull(*req.Conflicts)
			if err != nil {
				return fmt.Errorf("encode conflicts: %w", err)
			}
			sets = append(sets, m.Conflicts.ToExpr(keel.DynLit(encoded).Cast(keel.TypeJSON)))
		}

		if len(sets) == 0 {
			sets = append(sets, m.Version.Add(1))
		}

		n, err := m.Where(m.ID.Eq(req.MessageID), m.Version.Eq(req.Version)).
			Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return err
		}
		if n != 1 {
			return triageports.ErrMessageChanged
		}

		return nil
	})
}

func jsonOrNull[T any](values []T) (*string, error) {
	if len(values) == 0 {
		return nil, nil //nolint:nilnil // an empty list is stored as NULL
	}

	encoded, err := sonic.MarshalString(values)
	if err != nil {
		return nil, err
	}

	return &encoded, nil
}

func (r *repository) OriginalClassification(
	ctx context.Context,
	req triageports.FirstCorrectionRequest,
) (inboundmessage.Classification, bool, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (inboundmessage.Classification, bool, error) {
			a := triagedb.Activities
			classes, err := keel.Into[inboundmessage.Classification](ctx, r.db.Keel(ctx), a.
				Pick(a.FromClassification).
				Where(
					a.MessageID.Eq(req.MessageID),
					a.Kind.Eq(triage.ActivityCorrected),
					a.FromClassification.IsNotNull(),
				).
				Order(a.CreatedAt.Asc(), a.ID.Asc()).
				Limit(1))
			if err != nil {
				return "", false, err
			}
			if len(classes) == 0 {
				return "", false, nil
			}

			return classes[0], true, nil
		},
	)
}
