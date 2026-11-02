package tariff

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

var _ tariffports.RatedShipments = (*RatedShipments)(nil)

type RatedShipmentsParams struct {
	fx.In

	Shipments shipmentports.ShipmentRepository
	Hydrator  ratequoteports.ShipmentRatingHydrator
	Records   *RecordSource
	Clock     timeutils.Clock
}

// RatedShipments reads back the shipments a formula template last rated, each with the record,
// inputs, currency and template version its rating used, so a candidate formula can be run over
// the same facts.
type RatedShipments struct {
	shipments shipmentports.ShipmentRepository
	hydrator  ratequoteports.ShipmentRatingHydrator
	records   *RecordSource
	now       timeutils.Clock
}

//nolint:gocritic // fx param structs are passed by value
func NewRatedShipments(p RatedShipmentsParams) *RatedShipments {
	return &RatedShipments{
		shipments: p.Shipments,
		hydrator:  p.Hydrator,
		records:   p.Records,
		now:       p.Clock,
	}
}

func (s *RatedShipments) RatedByTemplate(
	ctx context.Context,
	req *tariffports.RatedByTemplateRequest,
) ([]*tariffports.RatedShipment, error) {
	entities, err := s.shipments.ListRatedByFormulaTemplate(
		ctx,
		&shipmentports.ListRatedByFormulaTemplateRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			Limit:      req.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	rated := make([]*shipment.Shipment, 0, len(entities))
	for _, entity := range entities {
		if ratedByTemplate(entity, req.TemplateID.String()) {
			rated = append(rated, entity)
		}
	}
	if len(rated) == 0 {
		return []*tariffports.RatedShipment{}, nil
	}

	if err = s.hydrator.HydrateShipments(ctx, &ratequoteports.HydrateShipmentsRequest{
		Shipments:  rated,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	out := make([]*tariffports.RatedShipment, 0, len(rated))
	for _, entity := range rated {
		record, buildErr := s.record(ctx, req, entity)
		if buildErr != nil {
			return nil, buildErr
		}
		out = append(out, record)
	}

	return out, nil
}

// ratedByTemplate keeps a shipment whose last rating produced a bill from this template. A
// refusal has nothing to compare against, and a shipment whose template was changed after it
// was rated by an agreement rule was not rated by this one.
func ratedByTemplate(entity *shipment.Shipment, templateID string) bool {
	detail := entity.RatingDetail
	if detail == nil || detail.Bill == nil {
		return false
	}

	return detail.FormulaTemplateID == "" || detail.FormulaTemplateID == templateID
}

func (s *RatedShipments) record(
	ctx context.Context,
	req *tariffports.RatedByTemplateRequest,
	entity *shipment.Shipment,
) (*tariffports.RatedShipment, error) {
	detail := entity.RatingDetail
	asOf := detail.RatedAt
	if asOf == 0 {
		asOf = shipmentrating.AsOf(entity, s.now())
	}

	encoded, err := s.records.Build(ctx, &RecordRequest{
		TenantInfo: req.TenantInfo,
		Shipment:   entity,
		Currency:   BillingCurrency(entity),
		AsOf:       asOf,
		CustomerID: entity.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	inputs := detail.Inputs
	if inputs == nil {
		inputs = entity.TariffInputs
	}

	currency, exchange := ratedCurrency(detail)

	return &tariffports.RatedShipment{
		ShipmentID:    entity.ID,
		ProNumber:     entity.ProNumber,
		Record:        encoded,
		Inputs:        inputs,
		Currency:      currency,
		Exchange:      exchange,
		VersionNumber: detail.VersionNumber,
		Billed:        billedAmount(entity),
		RatedAt:       detail.RatedAt,
	}, nil
}

// ratedCurrency is the currency the template rated in and the exchange that carried its bill
// into the billing currency. A converted bill keeps the template's currency on its exchange and
// is itself in the billing currency.
func ratedCurrency(detail *shipment.RatingDetail) (string, *rules.TariffExchange) {
	bill := detail.Bill
	if bill.Exchange == nil {
		return bill.Currency, nil
	}

	return bill.Exchange.Currency, &rules.TariffExchange{
		Rate:     bill.Exchange.Rate,
		Currency: bill.Currency,
	}
}

// billedAmount is what the customer was billed, once the shipment has been billed; before that
// the backtest recomputes the current amount from the version that rated it.
func billedAmount(entity *shipment.Shipment) decimal.NullDecimal {
	if entity.BilledAt == nil {
		return decimal.NullDecimal{}
	}

	return entity.TotalChargeAmount
}
