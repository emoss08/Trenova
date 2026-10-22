package shipmentcommercial

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
)

type ContractCharges struct {
	FreightLines []shipment.FreightLine
	Charges      []*shipment.AdditionalCharge
}

func (c *Calculator) ContractCharges(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
) (*ContractCharges, error) {
	out := &ContractCharges{
		FreightLines: []shipment.FreightLine{},
		Charges:      []*shipment.AdditionalCharge{},
	}
	if rated == nil || !rated.Outcome.Priced() || rated.Bill == nil {
		return out, nil
	}
	out.FreightLines = billFreightLines(ctx, rated.Bill)

	pass, err := c.newRatingPass(ctx, entity, rated)
	if err != nil {
		return nil, err
	}

	var applicable []*rateagreement.RateAgreementAccessorial
	if rated.Agreement != nil {
		applicable = applicableAccessorials(rated, entity, ratingDate(entity, c.now), pass)
	}
	coded := rated.Bill.CodedLines()
	if len(coded) == 0 && len(applicable) == 0 {
		return out, nil
	}
	if err = c.loadCatalog(ctx, entity, pass); err != nil {
		return nil, err
	}

	if err = appendTariffCharges(out, entity, coded, pass); err != nil {
		return nil, err
	}

	priced, err := shipment.PriceAgreementAccessorials(
		ctx,
		entity.AccessorialMeasures(),
		applicable,
		rated.Amount,
	)
	if err != nil {
		return nil, err
	}
	for _, entry := range priced {
		charge := agreementCharge(entity, entry, nil)
		charge.AccessorialCharge = pass.definition(charge.AccessorialChargeID)
		out.Charges = append(out.Charges, charge)
	}
	return out, nil
}

func appendTariffCharges(
	out *ContractCharges,
	entity *shipment.Shipment,
	coded []tarifftypes.Line,
	pass *ratingPass,
) error {
	for index := range coded {
		line := &coded[index]
		if !line.Amount.IsPositive() {
			continue
		}
		definition, ok := pass.byCode[accessorialcharge.NormalizeCode(line.Code)]
		if !ok {
			return unknownCodeError(line.Code)
		}
		out.Charges = append(
			out.Charges,
			tariffCharge(entity, &shipment.AdditionalCharge{}, definition, line),
		)
	}
	return nil
}

func unknownCodeError(code string) error {
	return errortypes.NewBusinessError(
		"The tariff billed the accessorial code {0}, which is not in the accessorial charge catalog",
		code,
	)
}
