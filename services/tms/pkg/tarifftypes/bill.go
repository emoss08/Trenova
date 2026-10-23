package tarifftypes

import (
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

var ErrBillShape = errors.New("tarifftypes: the bill does not have the expected shape")

var lineKinds = map[rules.TariffLineKind]LineKind{
	rules.TariffLineCharge:   LineCharge,
	rules.TariffLineMinimum:  LineMinimum,
	rules.TariffLineMaximum:  LineMaximum,
	rules.TariffLineRounding: LineRounding,
}

func BillFrom(bill *rules.TariffBill, currency string) (*Bill, error) {
	if bill == nil {
		return nil, ErrBillShape
	}
	converted := bill.Converted
	if converted != nil && len(converted.Lines) != len(bill.Lines) {
		return nil, fmt.Errorf("%w: %d converted lines for %d lines",
			ErrBillShape, len(converted.Lines), len(bill.Lines))
	}
	out := &Bill{
		Currency:         currency,
		Lines:            make([]Line, 0, len(bill.Lines)),
		FreightTotal:     bill.FreightTotal.Value,
		AccessorialTotal: bill.AccessorialTotal.Value,
		Total:            bill.Total.Value,
	}
	for i := range bill.Lines {
		line, err := lineFrom(&bill.Lines[i])
		if err != nil {
			return nil, err
		}
		if converted != nil {
			line.Amount = converted.Lines[i].Value
		}
		out.Lines = append(out.Lines, line)
	}
	for i := range bill.Notes {
		out.Notes = append(out.Notes, Note{Text: bill.Notes[i].Text})
	}
	for i := range bill.Reviews {
		out.Reviews = append(out.Reviews, Note{Text: bill.Reviews[i].Text})
	}
	if converted != nil {
		out.Currency = converted.Currency
		out.Total = converted.Total.Value
		out.Exchange = &Exchange{Currency: currency, Rate: converted.Rate}
		out.FreightTotal, out.AccessorialTotal = decimal.Zero, decimal.Zero
		for i := range out.Lines {
			if out.Lines[i].Code == "" {
				out.FreightTotal = out.FreightTotal.Add(out.Lines[i].Amount)
			} else {
				out.AccessorialTotal = out.AccessorialTotal.Add(out.Lines[i].Amount)
			}
		}
	}
	return out, nil
}

func lineFrom(line *rules.TariffLine) (Line, error) {
	kind, ok := lineKinds[line.Kind]
	if !ok {
		return Line{}, fmt.Errorf("%w: line kind %q", ErrBillShape, line.Kind)
	}
	out := Line{Kind: kind, Freight: line.Freight, Amount: line.Amount.Value}
	if line.Label != nil {
		out.Label = *line.Label
	}
	if line.Code != nil {
		out.Code = *line.Code
	}
	if len(line.Quantity) > 0 && string(line.Quantity) != jsonNull {
		quantity, err := quantityOrNumber(line.Quantity)
		if err != nil {
			return Line{}, err
		}
		out.Quantity, out.QuantityUnit = &quantity.Value, quantity.Unit
	}
	if len(line.Rate) > 0 && string(line.Rate) != jsonNull {
		rate, err := quantityOrNumber(line.Rate)
		if err != nil {
			return Line{}, err
		}
		out.Rate, out.RateUnit = &rate.Value, rate.Unit
	}
	return out, nil
}

func quantityOrNumber(raw rules.RawJSON) (rules.TariffQuantity, error) {
	if quantity, err := rules.DecodeQuantity(raw); err == nil {
		return quantity, nil
	}
	number, err := rules.DecodeNumber(raw)
	if err != nil {
		return rules.TariffQuantity{}, fmt.Errorf("%w: %w", ErrBillShape, err)
	}
	return rules.TariffQuantity{Value: number}, nil
}
