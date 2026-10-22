package rateagreementrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/rateagreementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const maxTemplateInputSets = 2_000

func NewTemplateInputsLister(p Params) rateagreementports.TemplateInputsLister {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rateagreement-template-inputs"),
	}
}

type templateInputsRow struct {
	Rule        tarifftypes.Inputs `bun:"rule_inputs"`
	Agreement   tarifftypes.Inputs `bun:"agreement_inputs"`
	ExampleID   pulid.ID           `bun:"example_id"`
	ExampleName string             `bun:"example_name"`
	Count       int                `bun:"count"`
}

func (r *repository) ListInputsByTemplate(
	ctx context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
) ([]*rateagreementports.TemplateInputs, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rateagreementports.TemplateInputs, error) {
			limit := req.Limit
			if limit <= 0 || limit > maxTemplateInputSets {
				limit = maxTemplateInputSets
			}

			rules, err := r.ruleInputSets(ctx, req, limit)
			if err != nil {
				return nil, err
			}
			shipments, err := r.shipmentInputSets(ctx, req, limit-len(rules))
			if err != nil {
				return nil, err
			}

			out := make([]*rateagreementports.TemplateInputs, 0, len(rules)+len(shipments))
			for i := range rules {
				out = append(out, inputSet(rateagreementports.TemplateInputsRule, &rules[i]))
			}
			for i := range shipments {
				out = append(out, inputSet(rateagreementports.TemplateInputsShipment, &shipments[i]))
			}
			return out, nil
		},
	)
}

func inputSet(
	kind rateagreementports.TemplateInputsKind,
	row *templateInputsRow,
) *rateagreementports.TemplateInputs {
	return &rateagreementports.TemplateInputs{
		Kind:        kind,
		ExampleID:   row.ExampleID,
		ExampleName: row.ExampleName,
		Count:       row.Count,
		Rule:        row.Rule,
		Agreement:   row.Agreement,
	}
}

func (r *repository) ruleInputSets(
	ctx context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) ([]templateInputsRow, error) {
	rule := rateagreementcol.RateAgreementRuleColumns
	agreement := rateagreementcol.RateAgreementColumns

	rows := make([]templateInputsRow, 0, 16)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr(
			"? AS ?",
			bun.Ident(rateagreementcol.RateAgreementRuleTable.Name),
			bun.Ident(rateagreementcol.RateAgreementRuleTable.Alias),
		).
		Join(
			"JOIN ? AS ? ON ? AND ? AND ?",
			bun.Ident(rateagreementcol.RateAgreementTable.Name),
			bun.Ident(rateagreementcol.RateAgreementTable.Alias),
			bun.Safe(agreement.ID.EqColumn(rule.RateAgreementID)),
			bun.Safe(agreement.OrganizationID.EqColumn(rule.OrganizationID)),
			bun.Safe(agreement.BusinessUnitID.EqColumn(rule.BusinessUnitID)),
		).
		ColumnExpr(rule.Inputs.As("rule_inputs")).
		ColumnExpr(agreement.Inputs.As("agreement_inputs")).
		ColumnExpr("MIN(?) AS example_id", bun.Safe(rule.ID.Qualified())).
		ColumnExpr("MIN(?) AS example_name", bun.Safe(agreement.Name.Qualified())).
		ColumnExpr("COUNT(*) AS count").
		Where(rule.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(rule.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(rule.FormulaTemplateID.Eq(), req.TemplateID).
		Where(rule.Status.Eq(), rateagreement.RuleStatusActive).
		GroupExpr(rule.Inputs.Qualified()).
		GroupExpr(agreement.Inputs.Qualified()).
		OrderExpr("count DESC").
		Limit(int64(limit)).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to list rule inputs by template", zap.Error(err))
		return nil, err
	}
	return rows, nil
}

func (r *repository) shipmentInputSets(
	ctx context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) ([]templateInputsRow, error) {
	rows := make([]templateInputsRow, 0, 16)
	if limit <= 0 {
		return rows, nil
	}

	cols := shipmentcol.ShipmentColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr(
			"? AS ?",
			bun.Ident(shipmentcol.ShipmentTable.Name),
			bun.Ident(shipmentcol.ShipmentTable.Alias),
		).
		ColumnExpr(cols.TariffInputs.As("rule_inputs")).
		ColumnExpr("MIN(?) AS example_id", bun.Safe(cols.ID.Qualified())).
		ColumnExpr("MIN(?) AS example_name", bun.Safe(cols.ProNumber.Qualified())).
		ColumnExpr("COUNT(*) AS count").
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.FormulaTemplateID.Eq(), req.TemplateID).
		Where(cols.Status.NotIn(), bun.List([]shipment.Status{
			shipment.StatusInvoiced,
			shipment.StatusCanceled,
		})).
		GroupExpr(cols.TariffInputs.Qualified()).
		OrderExpr("count DESC").
		Limit(int64(limit)).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to list shipment inputs by template", zap.Error(err))
		return nil, err
	}
	return rows, nil
}
