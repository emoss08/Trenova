package rules

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/shopspring/decimal"
)

const (
	OpReportFormat  = "report.format"
	OpReportResolve = "report.resolve"
)

var errReportShape = errors.New("the rules module answered for a different number of report values")

type ReportRule struct {
	Op    string              `json:"op"`
	Value decimal.Decimal     `json:"value"`
	Upper decimal.NullDecimal `json:"upper"`
	Tone  string              `json:"tone"`
}

type ReportBand struct {
	Width decimal.NullDecimal `json:"width"`
	Edges []decimal.Decimal   `json:"edges,omitempty"`
}

type ReportDisplay struct {
	Style         string       `json:"style"`
	Decimals      int          `json:"decimals"`
	Grouping      bool         `json:"grouping"`
	Currency      string       `json:"currency"`
	Negative      string       `json:"negative"`
	Notation      string       `json:"notation"`
	Prefix        string       `json:"prefix"`
	Suffix        string       `json:"suffix"`
	DateStyle     string       `json:"dateStyle"`
	BoolStyle     string       `json:"boolStyle"`
	DurationUnit  string       `json:"durationUnit"`
	DurationStyle string       `json:"durationStyle"`
	NullText      string       `json:"nullText"`
	Rules         []ReportRule `json:"rules,omitempty"`
	Band          *ReportBand  `json:"band,omitempty"`
}

type ReportSpec struct {
	Style         string       `json:"style,omitempty"`
	Decimals      *int         `json:"decimals,omitempty"`
	Grouping      *bool        `json:"grouping,omitempty"`
	Currency      string       `json:"currency,omitempty"`
	Negative      string       `json:"negative,omitempty"`
	Notation      string       `json:"notation,omitempty"`
	Prefix        string       `json:"prefix,omitempty"`
	Suffix        string       `json:"suffix,omitempty"`
	DateStyle     string       `json:"dateStyle,omitempty"`
	BoolStyle     string       `json:"boolStyle,omitempty"`
	DurationUnit  string       `json:"durationUnit,omitempty"`
	DurationStyle string       `json:"durationStyle,omitempty"`
	NullText      string       `json:"nullText,omitempty"`
	Rules         []ReportRule `json:"rules,omitempty"`
}

type ReportResolveInput struct {
	Type      string      `json:"type"`
	Hint      string      `json:"hint,omitempty"`
	DateStyle string      `json:"dateStyle,omitempty"`
	Band      *ReportBand `json:"band,omitempty"`
	Spec      *ReportSpec `json:"spec,omitempty"`
}

type ReportValues []any

type ReportColumn struct {
	Display   *ReportDisplay `json:"display,omitempty"`
	ValueType string         `json:"valueType,omitempty"`
	Values    ReportValues   `json:"values"`
}

type ReportCell struct {
	Text string              `json:"text"`
	Date decimal.NullDecimal `json:"date"`
	Tone string              `json:"tone"`
}

type reportFormatRequest struct {
	Columns []ReportColumn `json:"columns"`
}

type reportFormattedColumn struct {
	Values []ReportCell `json:"values"`
}

type reportFormatOut struct {
	Columns []reportFormattedColumn `json:"columns"`
}

type reportResolveRequest struct {
	Columns []ReportResolveInput `json:"columns"`
}

type reportResolveOut struct {
	Displays []ReportDisplay `json:"displays"`
}

func FormatReport(ctx context.Context, columns []ReportColumn) ([][]ReportCell, error) {
	if len(columns) == 0 {
		return nil, nil
	}

	var out reportFormatOut
	if err := Call(ctx, OpReportFormat, &reportFormatRequest{Columns: columns}, &out); err != nil {
		return nil, err
	}
	if len(out.Columns) != len(columns) {
		return nil, fmt.Errorf(
			"%w: %d columns for %d",
			errReportShape,
			len(out.Columns),
			len(columns),
		)
	}

	cells := make([][]ReportCell, len(columns))
	for i := range columns {
		if len(out.Columns[i].Values) != len(columns[i].Values) {
			return nil, fmt.Errorf(
				"%w: %d values for %d in column %d",
				errReportShape,
				len(out.Columns[i].Values),
				len(columns[i].Values),
				i,
			)
		}
		cells[i] = out.Columns[i].Values
	}

	return cells, nil
}

func ResolveReportDisplays(
	ctx context.Context,
	inputs []ReportResolveInput,
) ([]ReportDisplay, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	var out reportResolveOut
	if err := Call(ctx, OpReportResolve, &reportResolveRequest{Columns: inputs}, &out); err != nil {
		return nil, err
	}
	if len(out.Displays) != len(inputs) {
		return nil, fmt.Errorf(
			"%w: %d displays for %d columns",
			errReportShape,
			len(out.Displays),
			len(inputs),
		)
	}

	return out.Displays, nil
}

func (v ReportValues) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 16*len(v)+2)
	buf = append(buf, '[')
	for i, value := range v {
		if i > 0 {
			buf = append(buf, ',')
		}
		var err error
		if buf, err = appendReportValue(buf, value); err != nil {
			return nil, err
		}
	}

	return append(buf, ']'), nil
}

func appendReportValue(buf []byte, value any) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return append(buf, "null"...), nil
	case decimal.Decimal:
		return append(buf, v.String()...), nil
	case decimal.NullDecimal:
		if !v.Valid {
			return append(buf, "null"...), nil
		}
		return append(buf, v.Decimal.String()...), nil
	case int64:
		return strconv.AppendInt(buf, v, 10), nil
	case int:
		return strconv.AppendInt(buf, int64(v), 10), nil
	case int32:
		return strconv.AppendInt(buf, int64(v), 10), nil
	case int16:
		return strconv.AppendInt(buf, int64(v), 10), nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return appendReportText(buf, strconv.FormatFloat(v, 'f', -1, 64))
		}
		return append(buf, decimal.NewFromFloat(v).String()...), nil
	case bool:
		return strconv.AppendBool(buf, v), nil
	case string:
		return appendReportText(buf, v)
	case fmt.Stringer:
		return appendReportText(buf, v.String())
	default:
		return append(buf, "null"...), nil
	}
}

func appendReportText(buf []byte, text string) ([]byte, error) {
	quoted, err := sonic.Marshal(text)
	if err != nil {
		return nil, err
	}

	return append(buf, quoted...), nil
}
