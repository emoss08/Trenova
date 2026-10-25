package triagerulerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ruleErrors = dberror.Mapping{Entity: "TriageRule"}

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
		ru := triagedb.Rules
		rules, err := ru.Where(keel.When(req.EnabledOnly, ru.Enabled.IsTrue())).
			Order(ru.CreatedAt.Asc(), ru.ID.Asc()).
			Limit(triage.MaxRulesPerTenant).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if rules == nil {
			rules = []*triage.Rule{}
		}

		return rules, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req triageports.GetRuleRequest,
) (*triage.Rule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		rule, err := triagedb.Rules.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, ruleErrors.Map(err, req.ID.String())
		}

		return rule, nil
	})
}

func (r *repository) FindBySentence(
	ctx context.Context,
	req triageports.FindRuleBySentenceRequest,
) (*triage.Rule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		rules, err := sentenceQuery(req).Limit(1).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if len(rules) == 0 {
			return nil, nil
		}

		return rules[0], nil
	})
}

func sentenceQuery(req triageports.FindRuleBySentenceRequest) *keel.Query[triage.Rule] {
	ru := triagedb.Rules
	sentence := req.Sentence

	return ru.Where(
		ru.Classification.Eq(sentence.Classification),
		ru.SenderScope.Eq(sentence.SenderScope),
		keel.DynOf(ru.Sender).Coalesce(keel.DynConst("")).Eq(sentence.Sender),
		keel.DynOf(ru.MailboxID).Coalesce(keel.DynConst("")).Eq(sentence.MailboxID.String()),
		ru.Action.Eq(sentence.Action),
		keel.When(sentence.ID.IsNotNil(), ru.ID.Ne(sentence.ID)),
	)
}

func (r *repository) Create(
	ctx context.Context,
	rule *triage.Rule,
	limit int,
) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		db := r.db.Keel(ctx)
		ru := triagedb.Rules

		if err := keel.LockKey(
			ctx,
			db,
			"triage-rules:"+rule.OrganizationID.String()+":"+rule.BusinessUnitID.String(),
		); err != nil {
			return nil, fmt.Errorf("hold the organization's rules: %w", err)
		}

		count, err := ru.Query().Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("count the organization's rules: %w", err)
		}
		if count >= int64(limit) {
			return nil, triageports.ErrTooManyRules
		}

		if err = ru.Create(ctx, db, rule); err != nil {
			r.l.Error("failed to create a triage rule", zap.Error(err))
			return nil, ruleErrors.Map(err, rule.ID.String())
		}

		return rule, nil
	})
}

func (r *repository) Update(ctx context.Context, rule *triage.Rule) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		ru := triagedb.Rules
		if err := ru.Save(
			ctx,
			r.db.Keel(ctx),
			rule,
			ru.Classification,
			ru.SenderScope,
			ru.Sender,
			ru.MailboxID,
			ru.Action,
			ru.AssigneeID,
			ru.Enabled,
		); err != nil {
			return nil, ruleErrors.Map(err, rule.ID.String())
		}

		return rule, nil
	})
}

func (r *repository) SetEnabled(
	ctx context.Context,
	req triageports.SetRuleEnabledRequest,
) (*triage.Rule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Rule, error) {
		ru := triagedb.Rules
		rules, err := ru.Where(ru.ID.Eq(req.ID)).
			UpdateReturning(ctx, r.db.Keel(ctx), ru.Enabled.To(req.Enabled))
		if err != nil {
			return nil, ruleErrors.Map(err, req.ID.String())
		}
		if err = dberror.CheckAffected(
			int64(len(rules)),
			"TriageRule",
			req.ID.String(),
		); err != nil {
			return nil, err
		}

		return rules[0], nil
	})
}

func (r *repository) Delete(ctx context.Context, req triageports.GetRuleRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ru := triagedb.Rules
		n, err := ru.Where(ru.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return ruleErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "TriageRule", req.ID.String())
	})
}

func (r *repository) RecordMatch(
	ctx context.Context,
	req *triageports.RecordRuleMatchRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ru := triagedb.Rules
		_, err := ru.Where(ru.ID.Eq(req.RuleID)).Update(
			ctx,
			r.db.Keel(ctx),
			ru.MatchCount.Add(1),
			ru.LastMatchedAt.ToExpr(
				keel.DynFunc("greatest", keel.DynOf(ru.LastMatchedAt).Coalesce(0), req.At),
			),
			ru.LastMessageID.To(req.MessageID),
			ru.Version.ToExpr(ru.Version),
			ru.UpdatedAt.ToExpr(ru.UpdatedAt),
		)

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

		type settled struct {
			RuleID pulid.ID
			Count  int64
		}

		m := inboundmessagedb.InboundMessages
		rows, err := keel.Into[settled](ctx, r.db.Keel(ctx), m.
			Where(m.HandledByRuleID.In(req.RuleIDs...), m.ReceivedAt.Gte(req.Since)).
			Group(m.HandledByRuleID).
			Pick(m.HandledByRuleID, keel.Count()))
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			counts[row.RuleID] = int(row.Count)
		}

		return counts, nil
	})
}
