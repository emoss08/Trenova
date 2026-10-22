package reportfmt

import (
	"context"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

type ResolveInput struct {
	Type      reportcatalog.FieldType
	Hint      reportcatalog.FormatHint
	DateStyle DateStyle
	Band      *Band
	Spec      *Spec
}

func Resolve(ctx context.Context, inputs []ResolveInput) ([]Resolved, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	request := make([]rules.ReportResolveInput, len(inputs))
	for i := range inputs {
		request[i] = rules.ReportResolveInput{
			Type:      string(inputs[i].Type),
			Hint:      string(inputs[i].Hint),
			DateStyle: string(inputs[i].DateStyle),
			Band:      inputs[i].Band.wire(),
			Spec:      inputs[i].Spec.wire(),
		}
	}

	displays, err := rules.ResolveReportDisplays(ctx, request)
	if err != nil {
		return nil, err
	}

	out := make([]Resolved, len(displays))
	for i := range displays {
		out[i] = resolvedFromWire(&displays[i])
	}

	return out, nil
}

func (r *Resolved) wire() *rules.ReportDisplay {
	if r == nil {
		return nil
	}

	return &rules.ReportDisplay{
		Style:         string(r.Style),
		Decimals:      r.Decimals,
		Grouping:      r.Grouping,
		Currency:      r.Currency,
		Negative:      string(r.Negative),
		Notation:      string(r.Notation),
		Prefix:        r.Prefix,
		Suffix:        r.Suffix,
		DateStyle:     string(r.DateStyle),
		BoolStyle:     string(r.BoolStyle),
		DurationUnit:  string(r.DurationUnit),
		DurationStyle: string(r.DurationStyle),
		NullText:      r.NullText,
		Rules:         rulesToWire(r.Rules),
		Band:          r.Band.wire(),
	}
}

func resolvedFromWire(d *rules.ReportDisplay) Resolved {
	return Resolved{
		Style:         Style(d.Style),
		Decimals:      d.Decimals,
		Grouping:      d.Grouping,
		Currency:      d.Currency,
		Negative:      Negative(d.Negative),
		Notation:      Notation(d.Notation),
		Prefix:        d.Prefix,
		Suffix:        d.Suffix,
		DateStyle:     DateStyle(d.DateStyle),
		BoolStyle:     BoolStyle(d.BoolStyle),
		DurationUnit:  DurationUnit(d.DurationUnit),
		DurationStyle: DurationStyle(d.DurationStyle),
		NullText:      d.NullText,
		Rules:         rulesFromWire(d.Rules),
		Band:          bandFromWire(d.Band),
	}
}

func (s *Spec) wire() *rules.ReportSpec {
	if s == nil {
		return nil
	}

	return &rules.ReportSpec{
		Style:         string(s.Style),
		Decimals:      s.Decimals,
		Grouping:      s.Grouping,
		Currency:      s.Currency,
		Negative:      string(s.Negative),
		Notation:      string(s.Notation),
		Prefix:        s.Prefix,
		Suffix:        s.Suffix,
		DateStyle:     string(s.DateStyle),
		BoolStyle:     string(s.BoolStyle),
		DurationUnit:  string(s.DurationUnit),
		DurationStyle: string(s.DurationStyle),
		NullText:      s.NullText,
		Rules:         rulesToWire(s.Rules),
	}
}

func (b *Band) wire() *rules.ReportBand {
	if b.IsEmpty() {
		return nil
	}

	band := &rules.ReportBand{}
	if b.Width != 0 {
		band.Width = decimal.NewNullDecimal(decimal.NewFromFloat(b.Width))
	}
	if len(b.Edges) > 0 {
		band.Edges = make([]decimal.Decimal, len(b.Edges))
		for i, edge := range b.Edges {
			band.Edges[i] = decimal.NewFromFloat(edge)
		}
	}

	return band
}

func bandFromWire(band *rules.ReportBand) *Band {
	if band == nil {
		return nil
	}

	out := &Band{}
	if band.Width.Valid {
		out.Width = band.Width.Decimal.InexactFloat64()
	}
	if len(band.Edges) > 0 {
		out.Edges = make([]float64, len(band.Edges))
		for i, edge := range band.Edges {
			out.Edges[i] = edge.InexactFloat64()
		}
	}

	return out
}

func rulesToWire(in []Rule) []rules.ReportRule {
	if len(in) == 0 {
		return nil
	}

	out := make([]rules.ReportRule, len(in))
	for i := range in {
		out[i] = rules.ReportRule{
			Op:    string(in[i].Op),
			Value: decimal.NewFromFloat(in[i].Value),
			Tone:  string(in[i].Tone),
		}
		if in[i].Op.NeedsUpper() {
			out[i].Upper = decimal.NewNullDecimal(decimal.NewFromFloat(in[i].Upper))
		}
	}

	return out
}

func rulesFromWire(in []rules.ReportRule) []Rule {
	if len(in) == 0 {
		return nil
	}

	out := make([]Rule, len(in))
	for i := range in {
		out[i] = Rule{
			Op:    RuleOp(in[i].Op),
			Value: in[i].Value.InexactFloat64(),
			Tone:  Tone(in[i].Tone),
		}
		if in[i].Upper.Valid {
			out[i].Upper = in[i].Upper.Decimal.InexactFloat64()
		}
	}

	return out
}
