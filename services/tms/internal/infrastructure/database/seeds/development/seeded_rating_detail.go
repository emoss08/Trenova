package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func seededRatingDetail(
	shp *shipment.Shipment,
	template *formulatemplate.FormulaTemplate,
	now int64,
) *shipment.RatingDetail {
	amount, _ := shp.FreightChargeAmount.Decimal.Float64()
	linehaul := ratetypes.ComponentKindLinehaul.String()
	quote := ratequote.RateQuote{Outcome: ratequote.OutcomeFormulaFallback}

	return &shipment.RatingDetail{
		FormulaTemplateID:   template.ID.String(),
		FormulaTemplateName: template.Name,
		Expression:          template.Expression,
		ResolvedVariables:   map[string]any{},
		Result:              amount,
		RatedAt:             seededRatedAt(shp, now),
		Breakdown: []shipment.RatingBreakdownItem{
			{Name: linehaul, Label: linehaul, Amount: amount},
		},
		Source:      string(ratequote.OutcomeFormulaFallback),
		Explanation: quote.Explanation(),
	}
}

func seededRatedAt(shp *shipment.Shipment, now int64) int64 {
	if shp.ActualShipDate != nil && *shp.ActualShipDate > 0 {
		return *shp.ActualShipDate
	}

	return now
}

type seededRatingBackfill struct {
	orgID     pulid.ID
	buID      pulid.ID
	proPrefix string
	now       int64
}

func backfillSeededRatingDetails(
	ctx context.Context,
	tx bun.Tx,
	p seededRatingBackfill,
) (int, error) {
	var shipments []*shipment.Shipment
	if err := tx.NewSelect().
		Model(&shipments).
		Column(
			"id",
			"business_unit_id",
			"organization_id",
			"formula_template_id",
			"freight_charge_amount",
			"actual_ship_date",
		).
		Where("organization_id = ?", p.orgID).
		Where("business_unit_id = ?", p.buID).
		Where("pro_number LIKE ?", p.proPrefix+"%").
		Where("rating_detail IS NULL").
		Scan(ctx); err != nil {
		return 0, fmt.Errorf("load seeded shipments without a rating detail: %w", err)
	}
	if len(shipments) == 0 {
		return 0, nil
	}

	templateIDs := make([]pulid.ID, 0, len(shipments))
	seen := make(map[pulid.ID]struct{}, len(shipments))
	for _, shp := range shipments {
		if _, ok := seen[shp.FormulaTemplateID]; ok {
			continue
		}
		seen[shp.FormulaTemplateID] = struct{}{}
		templateIDs = append(templateIDs, shp.FormulaTemplateID)
	}

	var templates []*formulatemplate.FormulaTemplate
	if err := tx.NewSelect().
		Model(&templates).
		Column("id", "name", "expression").
		Where("organization_id = ?", p.orgID).
		Where("business_unit_id = ?", p.buID).
		Where("id IN (?)", bun.List(templateIDs)).
		Scan(ctx); err != nil {
		return 0, fmt.Errorf("load formula templates for seeded shipments: %w", err)
	}

	byID := make(map[pulid.ID]*formulatemplate.FormulaTemplate, len(templates))
	for _, template := range templates {
		byID[template.ID] = template
	}

	for _, shp := range shipments {
		template, ok := byID[shp.FormulaTemplateID]
		if !ok {
			return 0, fmt.Errorf(
				"formula template %s for seeded shipment %s not found",
				shp.FormulaTemplateID,
				shp.ID,
			)
		}

		shp.RatingDetail = seededRatingDetail(shp, template, p.now)
		if _, err := tx.NewUpdate().
			Model(shp).
			Column("rating_detail").
			WherePK().
			Exec(ctx); err != nil {
			return 0, fmt.Errorf("backfill rating detail for shipment %s: %w", shp.ID, err)
		}
	}

	return len(shipments), nil
}
