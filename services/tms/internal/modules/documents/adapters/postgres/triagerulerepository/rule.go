package triagerulerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const ruleEntity = "TriageRule"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) triageports.TriageRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-rule-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req triageports.ListRulesRequest,
) ([]*triage.Rule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.Rule, error) {
		cols := triagecol.RuleColumns
		rules := make([]*triage.Rule, 0, 16)

		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rules).
			Apply(triagecol.RuleApplyTenant(req.TenantInfo))
		if req.EnabledOnly {
			query = query.Where(cols.Enabled.IsTrue())
		}

		if err := query.
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(triage.MaxRulesPerTenant).
			Scan(ctx); err != nil {
			return nil, err
		}

		return rules, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req triageports.GetRuleRequest,
) (*triage.Rule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		rule := new(triage.Rule)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(rule).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return triagecol.RuleScopeTenant(sq, req.TenantInfo).
					Where(triagecol.RuleColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, ruleEntity)
		}

		return rule, nil
	})
}

func (r *repository) FindBySentence(
	ctx context.Context,
	req triageports.FindRuleBySentenceRequest,
) (*triage.Rule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		rules := make([]*triage.Rule, 0, 1)
		if err := sentenceQuery(
			r.db.DBForContext(ctx).NewSelect().Model(&rules),
			req,
		).Limit(1).Scan(ctx); err != nil {
			return nil, err
		}
		if len(rules) == 0 {
			return nil, nil //nolint:nilnil // an absent rule is an answer, not a failure
		}

		return rules[0], nil
	})
}

func sentenceQuery(
	sq *bun.SelectQuery,
	req triageports.FindRuleBySentenceRequest,
) *bun.SelectQuery {
	cols := triagecol.RuleColumns
	sentence := req.Sentence

	sq = triagecol.RuleScopeTenant(sq, req.TenantInfo).
		Where(cols.Classification.Eq(), sentence.Classification).
		Where(cols.SenderScope.Eq(), sentence.SenderScope).
		Where(cols.Sender.Expr("COALESCE({}, '') = ?"), sentence.Sender).
		Where(cols.MailboxID.Expr("COALESCE({}, '') = ?"), sentence.MailboxID.String()).
		Where(cols.Action.Eq(), sentence.Action)
	if sentence.ID.IsNotNil() {
		sq = sq.Where(cols.ID.NotEq(), sentence.ID)
	}

	return sq
}

func (r *repository) Create(
	ctx context.Context,
	rule *triage.Rule,
	limit int,
) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
			tenantInfo := pagination.TenantInfo{OrgID: rule.OrganizationID, BuID: rule.BusinessUnitID}
			if _, err := tx.NewRaw(
				"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
				"triage-rules:"+rule.OrganizationID.String()+":"+rule.BusinessUnitID.String(),
			).Exec(ctx); err != nil {
				return fmt.Errorf("hold the organization's rules: %w", err)
			}

			count, err := tx.NewSelect().
				Model((*triage.Rule)(nil)).
				Apply(triagecol.RuleApplyTenant(tenantInfo)).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count the organization's rules: %w", err)
			}
			if count >= int64(limit) {
				return triageports.ErrTooManyRules
			}

			if _, err = tx.NewInsert().Model(rule).Returning("*").Exec(ctx); err != nil {
				r.l.Error("failed to create a triage rule", zap.Error(err))

				return err
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		return rule, nil
	})
}

func (r *repository) Update(ctx context.Context, rule *triage.Rule) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		cols := triagecol.RuleColumns
		previous := rule.Version
		rule.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(rule).
			Column(
				cols.Classification.Bare(),
				cols.SenderScope.Bare(),
				cols.Sender.Bare(),
				cols.MailboxID.Bare(),
				cols.Action.Bare(),
				cols.AssigneeID.Bare(),
				cols.Enabled.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), previous).
			Returning("*").
			Exec(ctx)
		if err != nil {
			rule.Version = previous

			return nil, err
		}
		if err = dberror.CheckRowsAffected(result, ruleEntity, rule.ID.String()); err != nil {
			rule.Version = previous

			return nil, err
		}

		return rule, nil
	})
}

func (r *repository) SetEnabled(
	ctx context.Context,
	req triageports.SetRuleEnabledRequest,
) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		cols := triagecol.RuleColumns
		rule := new(triage.Rule)

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(rule).
			Set(cols.Enabled.Set(), req.Enabled).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return triagecol.RuleScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckRowsAffected(result, ruleEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return rule, nil
	})
}

func (r *repository) Delete(ctx context.Context, req triageports.GetRuleRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*triage.Rule)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return triagecol.RuleScopeTenantDelete(dq, req.TenantInfo).
					Where(triagecol.RuleColumns.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(result, ruleEntity, req.ID.String())
	})
}

func (r *repository) RecordMatch(
	ctx context.Context,
	req *triageports.RecordRuleMatchRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := triagecol.RuleColumns
		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*triage.Rule)(nil)).
			Set(cols.MatchCount.Inc(1)).
			Set(cols.LastMatchedAt.SetExpr("GREATEST(COALESCE({}, 0), ?)"), req.At).
			Set(cols.LastMessageID.Set(), req.MessageID).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return triagecol.RuleScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.RuleID)
			}).
			Exec(ctx)

		return err
	})
}

func (r *repository) CountSettledSince(
	ctx context.Context,
	req *triageports.CountRuleSettlementsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(req.RuleIDs))
		if len(req.RuleIDs) == 0 {
			return counts, nil
		}

		cols := inboundmessagecol.InboundMessageColumns
		rows := make([]struct {
			RuleID pulid.ID `bun:"rule_id"`
			Count  int      `bun:"count"`
		}, 0, len(req.RuleIDs))

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(cols.HandledByRuleID.As("rule_id")).
			ColumnExpr("COUNT(*) AS count").
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.HandledByRuleID.In(), bun.In(req.RuleIDs)).
			Where(cols.ReceivedAt.Gte(), req.Since).
			GroupExpr(cols.HandledByRuleID.Qualified()).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}

		for _, row := range rows {
			counts[row.RuleID] = row.Count
		}

		return counts, nil
	})
}
