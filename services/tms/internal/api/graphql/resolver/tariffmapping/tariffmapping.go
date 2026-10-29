package tariffmapping

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
)

func TariffInputsToModel(inputs tarifftypes.Inputs) (map[string]any, error) {
	return inputs.OrEmpty().Values()
}

func TariffInputsFromModel(field string, values map[string]any) (tarifftypes.Inputs, error) {
	inputs, err := tarifftypes.InputsFromValues(values)
	if err != nil {
		return nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Tariff inputs must be Tariff wire values",
		)
	}
	return inputs, nil
}

var exactTrace = sonic.Config{UseNumber: true}.Froze()

func TariffTraceToModel(trace *shipmentdomain.RatingTrace) (map[string]any, error) {
	if trace == nil {
		return nil, nil //nolint:nilnil // a rating with no trace maps to null
	}
	encoded, err := sonic.Marshal(trace)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err = exactTrace.Unmarshal(encoded, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func FreightLinesToModel(lines []shipmentdomain.FreightLine) []*gqlmodel.ShipmentFreightLine {
	out := make([]*gqlmodel.ShipmentFreightLine, 0, len(lines))
	for i := range lines {
		line := &lines[i]
		out = append(out, &gqlmodel.ShipmentFreightLine{
			Label:        line.Label,
			Quantity:     gqldecimal.DecimalPtrString(line.Quantity),
			QuantityUnit: line.QuantityUnit,
			Rate:         gqldecimal.DecimalPtrString(line.Rate),
			RateUnit:     line.RateUnit,
			Amount:       line.Amount.String(),
		})
	}
	return out
}

func TariffBillToModel(bill *tarifftypes.Bill) *gqlmodel.TariffBill {
	if bill == nil {
		return nil
	}
	lines := make([]*gqlmodel.TariffBillLine, 0, len(bill.Lines))
	for i := range bill.Lines {
		line := &bill.Lines[i]
		model := &gqlmodel.TariffBillLine{
			Kind:         string(line.Kind),
			Label:        line.Label,
			Code:         line.Code,
			Quantity:     gqldecimal.DecimalPtrString(line.Quantity),
			QuantityUnit: line.QuantityUnit,
			Rate:         gqldecimal.DecimalPtrString(line.Rate),
			RateUnit:     line.RateUnit,
			Amount:       line.Amount.String(),
			On:           line.On,
		}
		if line.Limit != nil {
			model.Limit = &gqlmodel.TariffLineLimit{
				Kind:     string(line.Limit.Kind),
				Computed: line.Limit.Computed.String(),
			}
		}
		lines = append(lines, model)
	}
	notes := make([]string, 0, len(bill.Notes))
	for _, note := range bill.Notes {
		notes = append(notes, note.Text)
	}
	reviews := make([]string, 0, len(bill.Reviews))
	for _, review := range bill.Reviews {
		reviews = append(reviews, review.Text)
	}
	out := &gqlmodel.TariffBill{
		Currency:         bill.Currency,
		Lines:            lines,
		Notes:            notes,
		Reviews:          reviews,
		FreightTotal:     bill.FreightTotal.String(),
		AccessorialTotal: bill.AccessorialTotal.String(),
		Total:            bill.Total.String(),
	}
	if bill.Exchange != nil {
		out.Exchange = &gqlmodel.TariffBillExchange{
			Currency: bill.Exchange.Currency,
			Rate:     bill.Exchange.Rate.String(),
		}
	}
	return out
}

func TariffChoicesToModel(choices []tarifftypes.Choice) []*gqlmodel.TariffChoice {
	out := make([]*gqlmodel.TariffChoice, 0, len(choices))
	for i := range choices {
		choice := &choices[i]
		options := make([]*gqlmodel.TariffChoiceOption, 0, len(choice.Options))
		for j := range choice.Options {
			option := &choice.Options[j]
			options = append(options, &gqlmodel.TariffChoiceOption{
				Label:    option.Label,
				Status:   string(option.Status),
				Subtotal: gqldecimal.DecimalPtrString(option.Subtotal),
				Refusal:  gqlscalar.StringPtrFromValue(option.Refusal),
			})
		}
		out = append(out, &gqlmodel.TariffChoice{
			Label:    choice.Label,
			Kind:     string(choice.Kind),
			Currency: choice.Currency,
			Chosen:   choice.Chosen,
			Options:  options,
		})
	}
	return out
}
