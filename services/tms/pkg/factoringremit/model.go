package factoringremit

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	MaxModelLines      = 400
	maxModelValueRunes = 100
)

type ModelLine struct {
	InvoiceNumber   string `json:"invoiceNumber"`
	FactorReference string `json:"factorReference"`
	Debtor          string `json:"debtor"`
	Kind            string `json:"kind"`
	Face            string `json:"face"`
	Advance         string `json:"advance"`
	Fee             string `json:"fee"`
	Reserve         string `json:"reserve"`
	Cash            string `json:"cash"`
	Date            string `json:"date"`
}

type ModelReply struct {
	Lines []ModelLine `json:"lines"`
}

func ModelMapping() Mapping {
	fields := Fields()
	mapping := make(Mapping, len(fields))
	for idx, field := range fields {
		mapping[field] = idx
	}
	return mapping
}

func ModelSchema() map[string]any {
	text := jsonschemautils.String(maxModelValueRunes)
	kinds := make([]string, 0, len(factoringremittance.AllRemittanceLineKinds()))
	for _, kind := range factoringremittance.AllRemittanceLineKinds() {
		kinds = append(kinds, kind.String())
	}
	line := jsonschemautils.Object(map[string]any{
		"invoiceNumber":   text,
		"factorReference": text,
		"debtor":          text,
		"kind":            map[string]any{"type": "string", "enum": kinds},
		"face":            text,
		"advance":         text,
		"fee":             text,
		"reserve":         text,
		"cash":            text,
		"date":            text,
	}, "invoiceNumber", "factorReference", "debtor", "kind", "face", "advance", "fee",
		"reserve", "cash", "date")
	return jsonschemautils.Object(map[string]any{
		"lines": jsonschemautils.Array(line, MaxModelLines),
	}, "lines")
}

func ModelSheet(lines []ModelLine) *rateimport.Sheet {
	fields := Fields()
	headers := make([]string, len(fields))
	for idx, field := range fields {
		headers[idx] = field.Label()
	}
	rows := make([][]string, 0, min(len(lines), MaxModelLines))
	for idx := range lines {
		if idx >= MaxModelLines {
			break
		}
		line := &lines[idx]
		rows = append(rows, []string{
			line.InvoiceNumber,
			line.FactorReference,
			line.Debtor,
			modelKindText(line.Kind),
			line.Face,
			line.Advance,
			line.Fee,
			line.Reserve,
			line.Cash,
			line.Date,
		})
	}
	return &rateimport.Sheet{Headers: headers, Rows: rows, FirstDataRow: 1}
}

func modelKindText(raw string) string {
	switch factoringremittance.RemittanceLineKind(strings.TrimSpace(raw)) {
	case factoringremittance.RemittanceLineFunding:
		return "funding"
	case factoringremittance.RemittanceLineRejection:
		return "rejected"
	case factoringremittance.RemittanceLineReserveRelease:
		return "reserve release"
	case factoringremittance.RemittanceLineChargeback:
		return "chargeback"
	case factoringremittance.RemittanceLineOther:
		return "adjustment"
	}
	return ""
}
