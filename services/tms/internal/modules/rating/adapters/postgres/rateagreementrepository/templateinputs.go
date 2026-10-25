package rateagreementrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
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
	Rule        tarifftypes.Inputs
	Agreement   tarifftypes.Inputs
	ExampleID   pulid.ID
	ExampleName string
	Count       int64
}

type shipmentInputsRow struct {
	Rule        tarifftypes.Inputs
	ExampleID   pulid.ID
	ExampleName string
	Count       int64
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
				row := &shipments[i]
				out = append(
					out,
					inputSet(rateagreementports.TemplateInputsShipment, &templateInputsRow{
						Rule:        row.Rule,
						ExampleID:   row.ExampleID,
						ExampleName: row.ExampleName,
						Count:       row.Count,
					}),
				)
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
		Count:       int(row.Count),
		Rule:        row.Rule,
		Agreement:   row.Agreement,
	}
}

func ruleInputsQuery(
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules
	ra := rateagreementdb.RateAgreements

	return rr.Join(ra, ra.ID.EqCol(rr.RateAgreementID)).
		Where(
			rr.FormulaTemplateID.Eq(req.TemplateID),
			rr.Status.Eq(rateagreement.RuleStatusActive),
		).
		Group(rr.Inputs, ra.Inputs).
		Pick(rr.Inputs, ra.Inputs, keel.Min(rr.ID), keel.Min(ra.Name), keel.Count()).
		Order(keel.Count().Desc()).
		Limit(limit)
}

func (r *repository) ruleInputSets(
	ctx context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) ([]templateInputsRow, error) {
	rows, err := keel.Into[templateInputsRow](ctx, r.db.Keel(ctx), ruleInputsQuery(req, limit))
	if err != nil {
		r.l.Error("failed to list rule inputs by template", zap.Error(err))
		return nil, err
	}

	return rows, nil
}

func shipmentInputsQuery(
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	return sp.Where(
		sp.FormulaTemplateID.Eq(req.TemplateID),
		sp.Status.NotIn(shipment.StatusInvoiced, shipment.StatusCanceled),
	).
		Group(sp.TariffInputs).
		Pick(sp.TariffInputs, keel.Min(sp.ID), keel.Min(sp.ProNumber), keel.Count()).
		Order(keel.Count().Desc()).
		Limit(limit)
}

func (r *repository) shipmentInputSets(
	ctx context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
	limit int,
) ([]shipmentInputsRow, error) {
	if limit <= 0 {
		return []shipmentInputsRow{}, nil
	}

	rows, err := keel.Into[shipmentInputsRow](ctx, r.db.Keel(ctx), shipmentInputsQuery(req, limit))
	if err != nil {
		r.l.Error("failed to list shipment inputs by template", zap.Error(err))
		return nil, err
	}

	return rows, nil
}
