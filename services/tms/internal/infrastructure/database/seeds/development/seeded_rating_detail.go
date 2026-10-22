package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	seededBillCurrency = "USD"
	seededFreightLabel = "Linehaul"
)

var seededRateUnits = map[string]string{
	formulatemplate.StandardPerMile: rules.UnitMoney + "/mi",
	formulatemplate.StandardPerStop: rules.UnitMoney + "/stop",
}

func seededTariffInputs(templateName string, rate decimal.Decimal) tarifftypes.Inputs {
	unit, ok := seededRateUnits[templateName]
	if !ok {
		return tarifftypes.Inputs{formulatemplate.StandardInputRate: rules.MoneyValue(rate)}
	}
	return tarifftypes.Inputs{formulatemplate.StandardInputRate: rules.QuantityValue(rate, unit)}
}

var seededUnitRates = map[string]decimal.Decimal{
	formulatemplate.StandardPerMile: decimal.RequireFromString("2.50"),
	formulatemplate.StandardPerStop: decimal.NewFromInt(150),
}

func seededRate(templateName string, freight decimal.Decimal) decimal.Decimal {
	if rate, ok := seededUnitRates[templateName]; ok {
		return rate
	}
	return freight
}

func seededFreightLines(shp *shipment.Shipment) []shipment.FreightLine {
	return []shipment.FreightLine{{Label: seededFreightLabel, Amount: shp.FreightChargeAmount.Decimal}}
}

func seededRatingDetail(
	shp *shipment.Shipment,
	template *formulatemplate.FormulaTemplate,
	now int64,
) *shipment.RatingDetail {
	freight := shp.FreightChargeAmount.Decimal
	quote := ratequote.RateQuote{Outcome: ratequote.OutcomeFormulaFallback}

	return &shipment.RatingDetail{
		Bill: &tarifftypes.Bill{
			Currency: seededBillCurrency,
			Lines: []tarifftypes.Line{{
				Kind:    tarifftypes.LineCharge,
				Label:   seededFreightLabel,
				Freight: true,
				Amount:  freight,
			}},
			FreightTotal:     freight,
			AccessorialTotal: decimal.Zero,
			Total:            freight,
		},
		Source:              string(ratequote.OutcomeFormulaFallback),
		RatedAt:             seededRatedAt(shp, now),
		FormulaTemplateID:   template.ID.String(),
		FormulaTemplateName: template.Name,
		Inputs:              shp.TariffInputs.Clone().OrEmpty(),
		Skipped:             []shipment.RatingSkip{},
		Explanation:         quote.Explanation(),
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
	cols := shipmentcol.ShipmentColumns
	var shipments []*shipment.Shipment
	if err := tx.NewSelect().
		Model(&shipments).
		Column(
			cols.ID.String(),
			cols.BusinessUnitID.String(),
			cols.OrganizationID.String(),
			cols.FormulaTemplateID.String(),
			cols.FreightChargeAmount.String(),
			cols.TariffInputs.String(),
			cols.ActualShipDate.String(),
		).
		Where(cols.OrganizationID.Eq(), p.orgID).
		Where(cols.BusinessUnitID.Eq(), p.buID).
		Where(cols.ProNumber.Like(), p.proPrefix+"%").
		Where(cols.RatingDetail.IsNull()).
		Where(cols.FormulaTemplateID.IsNotNull()).
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

	templateCols := formulatemplatecol.FormulaTemplateColumns
	var templates []*formulatemplate.FormulaTemplate
	if err := tx.NewSelect().
		Model(&templates).
		Column(templateCols.ID.String(), templateCols.Name.String()).
		Where(templateCols.OrganizationID.Eq(), p.orgID).
		Where(templateCols.BusinessUnitID.Eq(), p.buID).
		Where(templateCols.ID.In(), bun.List(templateIDs)).
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
			Column(cols.RatingDetail.String()).
			WherePK().
			Exec(ctx); err != nil {
			return 0, fmt.Errorf("backfill rating detail for shipment %s: %w", shp.ID, err)
		}
	}

	return len(shipments), nil
}
