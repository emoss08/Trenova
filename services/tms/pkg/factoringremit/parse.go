package factoringremit

import (
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	maxCells      = 40
	maxCellRunes  = factoring.MaxRemittanceCellLength
	maxFieldRunes = 100
)

type Line struct {
	RowNumber       int
	Cells           []string
	Kind            factoring.RemittanceLineKind
	InvoiceNumber   string
	FactorReference string
	Debtor          string
	FaceMinor       *int64
	AdvanceMinor    *int64
	FeeMinor        *int64
	ReserveMinor    *int64
	CashMinor       *int64
	OccurredOn      *int64
	Err             string
}

type Options struct {
	DefaultKind factoring.RemittanceLineKind
	Location    *time.Location
}

func Parse(sheet *rateimport.Sheet, mapping Mapping, opts Options) []*Line {
	if sheet == nil {
		return nil
	}
	if !opts.DefaultKind.IsValid() || opts.DefaultKind == factoring.RemittanceLineOther {
		opts.DefaultKind = factoring.RemittanceLineFunding
	}
	lines := make([]*Line, 0, len(sheet.Rows))
	for idx, cells := range sheet.Rows {
		if blank(cells) || totalRow(cells, mapping) {
			continue
		}
		lines = append(lines, parseRow(sheet.FirstDataRow+idx, cells, mapping, opts))
	}
	return lines
}

func parseRow(rowNumber int, cells []string, mapping Mapping, opts Options) *Line {
	line := &Line{
		RowNumber:       rowNumber,
		Cells:           clipCells(cells),
		Kind:            opts.DefaultKind,
		InvoiceNumber:   clip(cellAt(cells, mapping, FieldInvoiceNumber), maxFieldRunes),
		FactorReference: clip(cellAt(cells, mapping, FieldFactorReference), maxFieldRunes),
		Debtor:          clip(cellAt(cells, mapping, FieldDebtor), maxCellRunes),
	}
	if raw := cellAt(cells, mapping, FieldKind); raw != "" {
		line.Kind = KindFromText(raw, opts.DefaultKind)
	}

	problems := make([]string, 0, 2)
	for _, target := range []struct {
		field Field
		dest  **int64
	}{
		{FieldFace, &line.FaceMinor},
		{FieldAdvance, &line.AdvanceMinor},
		{FieldFee, &line.FeeMinor},
		{FieldReserve, &line.ReserveMinor},
		{FieldCash, &line.CashMinor},
	} {
		value, present, err := amountAt(cells, mapping, target.field)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if present {
			*target.dest = &value
		}
	}

	if raw := cellAt(cells, mapping, FieldDate); raw != "" {
		on, err := timeutils.ParseLooseDate(raw, opts.Location)
		if err != nil {
			problems = append(problems, "The date \""+clip(raw, 40)+"\" could not be read")
		} else {
			line.OccurredOn = &on
		}
	}

	if line.InvoiceNumber == "" && line.FactorReference == "" {
		problems = append(problems, "The row has no invoice number or reference")
	}
	if len(problems) == 0 && !line.hasAmount() {
		problems = append(problems, "The row has no amount")
	}
	line.Err = strings.Join(problems, "; ")
	return line
}

func (l *Line) hasAmount() bool {
	return l.CashMinor != nil || l.AdvanceMinor != nil || l.FaceMinor != nil ||
		l.ReserveMinor != nil
}

func amountAt(
	cells []string,
	mapping Mapping,
	field Field,
) (minor int64, present bool, err error) {
	raw := cellAt(cells, mapping, field)
	if raw == "" {
		return 0, false, nil
	}
	value, err := decimalutils.ParseLoose(raw)
	if err != nil {
		if errors.Is(err, decimalutils.ErrNumberMissing) {
			return 0, false, nil
		}
		return 0, false, errors.New(field.Label() + " \"" + clip(raw, 40) + "\" is not an amount")
	}
	return money.MinorUnits(value), true, nil
}

func KindFromText(raw string, fallback factoring.RemittanceLineKind) factoring.RemittanceLineKind {
	text := normalizeHeader(raw)
	if text == "" {
		return fallback
	}
	compact := strings.ReplaceAll(text, " ", "")
	switch {
	case containsAny(compact, "chargeback", "recourse", "repurchase", "buyback"):
		return factoring.RemittanceLineChargeback
	case containsAny(compact, "rebate", "reserve", "release", "escrow"):
		return factoring.RemittanceLineReserveRelease
	case containsAny(compact, "declin", "reject", "denied", "nonpurchase", "notpurchased"):
		return factoring.RemittanceLineRejection
	case containsAny(compact, "purchase", "advance", "fund", "buy"):
		return factoring.RemittanceLineFunding
	case containsAny(compact, "adjust", "interest", "misc", "wire", "ach", "fuel", "deduction"):
		return factoring.RemittanceLineOther
	}
	return fallback
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func cellAt(cells []string, mapping Mapping, field Field) string {
	index, ok := mapping[field]
	if !ok || index < 0 || index >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[index])
}

func blank(cells []string) bool {
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func totalRow(cells []string, mapping Mapping) bool {
	for _, field := range []Field{FieldInvoiceNumber, FieldFactorReference, FieldDebtor} {
		label := normalizeHeader(cellAt(cells, mapping, field))
		if label == "total" || label == "totals" || label == "grand total" ||
			label == "subtotal" || label == "sub total" || label == "report total" {
			return true
		}
	}
	first := ""
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			first = normalizeHeader(cell)
			break
		}
	}
	return first == "total" || first == "totals" || first == "grand total" ||
		strings.HasPrefix(first, "total for") || strings.HasPrefix(first, "subtotal")
}

func clipCells(cells []string) []string {
	limit := min(len(cells), maxCells)
	out := make([]string, limit)
	for i := range limit {
		out[i] = clip(strings.TrimSpace(cells[i]), maxCellRunes)
	}
	return out
}

func clip(value string, limit int) string {
	return stringutils.TruncateRunes(strings.TrimSpace(value), limit)
}
