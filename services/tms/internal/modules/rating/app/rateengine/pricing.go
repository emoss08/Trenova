package rateengine

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrNoTemplate = errors.New("rateengine: the rule names no formula template")

type pricing struct {
	templateID pulid.ID
	inputs     tarifftypes.Inputs
	currency   string
	agreement  *rateagreement.RateAgreement
	rule       *rateagreement.RateAgreementRule
}

type priced struct {
	template   *tariffports.Template
	bill       *tarifftypes.Bill
	refusal    string
	conditions map[pulid.ID]bool
	inputs     tarifftypes.Inputs
}

func resolvedInputs(
	rule *rateagreement.RateAgreementRule,
	agreement *rateagreement.RateAgreement,
) tarifftypes.Inputs {
	out := make(tarifftypes.Inputs, 8)
	if agreement != nil {
		for name, value := range agreement.Inputs {
			out[name] = value
		}
	}
	if rule != nil {
		for name, value := range rule.Inputs {
			out[name] = value
		}
	}
	return out
}

func (s *Service) price(
	ctx context.Context,
	rateCtx *RateContext,
	p *pricing,
	trace *ratetypes.Trace,
) (*priced, error) {
	if p.templateID.IsNil() {
		return nil, ErrNoTemplate
	}
	template, err := s.resolver.ResolveBilling(ctx, &tariffports.ResolveRequest{
		TenantInfo: rateCtx.TenantInfo,
		TemplateID: p.templateID,
		AsOf:       rateCtx.AsOf,
	})
	if err != nil {
		return nil, err
	}
	inputs := declared(template, p.inputs)
	trace.TemplateID = template.TemplateID.String()
	trace.TemplateName = template.Name
	trace.TemplateVersion = template.VersionNumber
	trace.TariffInputs = inputs

	currency := p.currency
	if currency == "" {
		currency = rateCtx.BillingCurrency
	}
	exchange, err := s.exchangeFor(ctx, rateCtx, currency, trace)
	if err != nil {
		return nil, err
	}

	encoded, err := s.records.Build(ctx, &tariff.RecordRequest{
		TenantInfo:       rateCtx.TenantInfo,
		Shipment:         rateCtx.Entity,
		OriginZones:      rateCtx.OriginZoneCodes,
		DestinationZones: rateCtx.DestinationZoneCodes,
		ZonesResolved:    rateCtx.ZonesResolved,
		Mode:             modeOf(rateCtx),
		SellTotal:        rateCtx.SellTotal,
		Currency:         currency,
		AsOf:             rateCtx.AsOf,
		CustomerID:       rateCtx.Entity.CustomerID,
		FuelOverride:     fuelOverrideOf(p.agreement),
	})
	if err != nil {
		return nil, err
	}

	accessorials := conditionalAccessorials(rateCtx, p.agreement)
	conditions := make([]tariffports.Condition, 0, len(accessorials))
	for _, accessorial := range accessorials {
		conditions = append(conditions, tariffports.Condition{
			ID:     accessorial.ID.String(),
			Source: accessorial.Condition,
		})
	}

	result, err := s.rater.Rate(ctx, &tariffports.RateRequest{
		TenantInfo: rateCtx.TenantInfo,
		Template:   template,
		Inputs:     inputs,
		Conditions: conditions,
		Record:     encoded,
		Currency:   currency,
		Exchange:   exchange,
		Trace:      true,
	})
	if err != nil {
		return nil, err
	}

	out := &priced{
		template:   template,
		bill:       result.Bill,
		conditions: make(map[pulid.ID]bool, len(accessorials)),
		inputs:     inputs,
	}
	for _, accessorial := range accessorials {
		out.conditions[accessorial.ID] = result.Conditions[accessorial.ID.String()]
	}
	trace.Statements = result.Trace
	trace.Lookups = result.Lookups
	if result.Refusal != nil {
		out.refusal = result.Refusal.Localize(i18n.FromContext(ctx))
		trace.Refusal = out.refusal
		return out, nil
	}
	trace.Bill = result.Bill
	addBillComponents(trace, result.Bill, template)
	return out, nil
}

func declared(template *tariffports.Template, inputs tarifftypes.Inputs) tarifftypes.Inputs {
	if len(template.Inputs) == 0 {
		return inputs.Clone()
	}
	out := make(tarifftypes.Inputs, len(template.Inputs))
	for i := range template.Inputs {
		if value, ok := inputs[template.Inputs[i].Name]; ok {
			out[template.Inputs[i].Name] = value
		}
	}
	return out
}

func modeOf(rateCtx *RateContext) string {
	if rateCtx.PartyType == rateagreement.PartyTypeCarrier {
		return record.ModeBuy
	}
	return record.ModeSell
}

func fuelOverrideOf(
	agreement *rateagreement.RateAgreement,
) *fuelsurchargeports.FuelProgramOverride {
	if agreement == nil || agreement.FuelBinding == nil {
		return nil
	}
	binding := agreement.FuelBinding
	return &fuelsurchargeports.FuelProgramOverride{
		AgreementID:   agreement.ID,
		ProgramID:     binding.FuelSurchargeProgramID,
		Waived:        binding.Waived,
		PegPrice:      binding.PegPriceOverride,
		IncrementRate: binding.IncrementRateOverride,
		CapAmount:     binding.CapAmount,
	}
}

func conditionalAccessorials(
	rateCtx *RateContext,
	agreement *rateagreement.RateAgreement,
) []*rateagreement.RateAgreementAccessorial {
	if agreement == nil {
		return nil
	}
	scoped := agreement.AutoApplyAccessorials(rateagreement.AccessorialFacts{
		At:             rateCtx.AsOf,
		ServiceTypeID:  rateCtx.ServiceTypeID,
		ShipmentTypeID: rateCtx.ShipmentTypeID,
	})
	out := make([]*rateagreement.RateAgreementAccessorial, 0, len(scoped))
	for _, accessorial := range scoped {
		if strings.TrimSpace(accessorial.Condition) != "" {
			out = append(out, accessorial)
		}
	}
	return out
}

func addBillComponents(
	trace *ratetypes.Trace,
	bill *tarifftypes.Bill,
	template *tariffports.Template,
) {
	if bill == nil {
		return
	}
	for i := range bill.Lines {
		line := &bill.Lines[i]
		component := &ratetypes.Component{
			Kind:       componentKind(line),
			Code:       line.Code,
			Label:      line.Label,
			Amount:     line.Amount,
			Source:     ratetypes.ComponentSourceTariff,
			SourceID:   template.TemplateID.String(),
			SourceName: template.Name,
		}
		if line.Quantity != nil {
			component.Quantity = decimal.NewNullDecimal(*line.Quantity)
		}
		if line.Rate != nil {
			component.Rate = decimal.NewNullDecimal(*line.Rate)
		}
		trace.AddComponent(component)
	}
}

func componentKind(line *tarifftypes.Line) ratetypes.ComponentKind {
	switch line.Kind {
	case tarifftypes.LineMinimum:
		return ratetypes.ComponentKindMinimumCharge
	case tarifftypes.LineMaximum:
		return ratetypes.ComponentKindMaximumCharge
	case tarifftypes.LineRounding:
		return ratetypes.ComponentKindRounding
	case tarifftypes.LineCharge:
		if line.Code != "" {
			return ratetypes.ComponentKindAccessorial
		}
	}
	return ratetypes.ComponentKindFreight
}

func refusalError(refusal string, rule *rateagreement.RateAgreementRule) error {
	if rule != nil && rule.Label != "" {
		return errortypes.NewBusinessError(
			"The rate on {0} refused to rate this shipment: {1}",
			rule.Label,
			refusal,
		)
	}
	return errortypes.NewBusinessError("The tariff refused to rate this shipment: {0}", refusal)
}
