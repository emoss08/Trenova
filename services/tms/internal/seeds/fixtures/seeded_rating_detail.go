package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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
	db *keel.DB,
	p seededRatingBackfill,
) (int, error) {
	sh := shipmentdb.Shipments
	shipments, err := sh.AllTenants().Pick(
		sh.ID,
		sh.BusinessUnitID,
		sh.OrganizationID,
		sh.FormulaTemplateID,
		sh.FreightChargeAmount,
		sh.TariffInputs,
		sh.ActualShipDate,
	).Where(
		sh.OrganizationID.Eq(p.orgID),
		sh.BusinessUnitID.Eq(p.buID),
		sh.ProNumber.HasPrefix(p.proPrefix),
		sh.RatingDetail.IsNull(),
		sh.FormulaTemplateID.IsNotNull(),
	).All(ctx, db)
	if err != nil {
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

	ft := formulatemplatedb.FormulaTemplates
	templates, err := ft.AllTenants().Pick(ft.ID, ft.Name).Where(
		ft.OrganizationID.Eq(p.orgID),
		ft.BusinessUnitID.Eq(p.buID),
		ft.ID.In(templateIDs...),
	).All(ctx, db)
	if err != nil {
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
		if _, err = sh.AllTenants().Where(
			sh.ID.Eq(shp.ID),
			sh.OrganizationID.Eq(shp.OrganizationID),
			sh.BusinessUnitID.Eq(shp.BusinessUnitID),
		).Update(ctx, db, sh.RatingDetail.To(shp.RatingDetail)); err != nil {
			return 0, fmt.Errorf("backfill rating detail for shipment %s: %w", shp.ID, err)
		}
	}

	return len(shipments), nil
}
