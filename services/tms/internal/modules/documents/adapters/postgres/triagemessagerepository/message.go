package triagemessagerepository

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	req triageports.ClaimRuleRequest,
) (*triageports.ClaimedRule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triageports.ClaimedRule, error) {
		versions := make([]int64, 0, 1)
		if _, err := claimRuleQuery(r.db.DBForContext(ctx), req).Exec(ctx, &versions); err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			return &triageports.ClaimedRule{}, nil
		}

		return &triageports.ClaimedRule{Claimed: true, Version: versions[0]}, nil
	})
}

func claimRuleQuery(db bun.IDB, req triageports.ClaimRuleRequest) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return db.NewUpdate().
		Model((*inboundmessage.InboundMessage)(nil)).
		Set(cols.HandledByRuleID.Set(), req.RuleID).
		Set(cols.HandledAction.Set(), req.Action).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), req.At).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.MessageID).
				Where(cols.HandledByRuleID.IsNull())
		}).
		Returning(cols.Version.Bare())
}

func (r *repository) CorrectClassification(
	ctx context.Context,
	req triageports.CorrectClassificationRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := inboundmessagecol.InboundMessageColumns

		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*inboundmessage.InboundMessage)(nil)).
			Set(cols.Classification.Set(), req.Classification).
			Set(cols.ClassificationCorrected.Set(), true).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), req.At)
		if req.TargetSeconds > 0 {
			query = query.Set(
				cols.DueAt.SetExpr("CASE WHEN status IN (?) THEN received_at + ? ELSE {} END"),
				bun.In([]inboundmessage.Status{
					inboundmessage.StatusInReview,
					inboundmessage.StatusQuarantined,
				}),
				req.TargetSeconds,
			)
		}

		result, err := query.
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.MessageID).
					Where(cols.Classification.Expr("{} IS DISTINCT FROM ?"), req.Classification)
			}).
			Exec(ctx)
		if err != nil {
			return false, err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return false, err
		}

		return affected == 1, nil
	})
}

func (r *repository) SaveTriageFields(
	ctx context.Context,
	req triageports.SaveTriageFieldsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := inboundmessagecol.InboundMessageColumns

		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*inboundmessage.InboundMessage)(nil)).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix())
		if req.Facts != nil {
			encoded, err := jsonOrNull(*req.Facts)
			if err != nil {
				return fmt.Errorf("encode facts: %w", err)
			}
			query = query.Set(cols.Facts.SetExpr("?::jsonb"), encoded)
		}
		if req.Conflicts != nil {
			encoded, err := jsonOrNull(*req.Conflicts)
			if err != nil {
				return fmt.Errorf("encode conflicts: %w", err)
			}
			query = query.Set(cols.Conflicts.SetExpr("?::jsonb"), encoded)
		}

		result, err := query.
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.MessageID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
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
			cols := triagecol.ActivityColumns
			classes := make([]inboundmessage.Classification, 0, 1)

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model((*triage.Activity)(nil)).
				ColumnExpr(cols.FromClassification.Qualified()).
				Apply(triagecol.ActivityApplyTenant(req.TenantInfo)).
				Where(cols.MessageID.Eq(), req.MessageID).
				Where(cols.Kind.Eq(), triage.ActivityCorrected).
				Where(cols.FromClassification.IsNotNull()).
				Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
				Limit(1).
				Scan(ctx, &classes); err != nil {
				return "", false, err
			}
			if len(classes) == 0 {
				return "", false, nil
			}

			return classes[0], true, nil
		},
	)
}
