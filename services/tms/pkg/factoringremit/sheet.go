package factoringremit

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/rateimport"
)

var ErrUnsupportedFile = errors.New(
	"a remittance must be a CSV, an Excel workbook (.xlsx) or a PDF",
)

func FormatFor(fileName, contentType string) (factoring.RemittanceFormat, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".csv", ".txt":
		return factoring.RemittanceFormatCsv, nil
	case ".xlsx":
		return factoring.RemittanceFormatXlsx, nil
	case ".pdf":
		return factoring.RemittanceFormatPdf, nil
	}
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])) {
	case "text/csv", "application/csv", "text/plain":
		return factoring.RemittanceFormatCsv, nil
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return factoring.RemittanceFormatXlsx, nil
	case "application/pdf":
		return factoring.RemittanceFormatPdf, nil
	}
	return "", ErrUnsupportedFile
}

func IsRemittanceFile(fileName, contentType string) bool {
	_, err := FormatFor(fileName, contentType)
	return err == nil
}

func ReadSheet(data []byte, format factoring.RemittanceFormat) (*rateimport.Sheet, error) {
	switch format {
	case factoring.RemittanceFormatCsv:
		return rateimport.ReadCSV(data)
	case factoring.RemittanceFormatXlsx:
		return rateimport.ReadXLSX(data)
	case factoring.RemittanceFormatPdf:
	}
	return nil, ErrUnsupportedFile
}

func ToDomain(lines []*Line) []*factoring.RemittanceLine {
	out := make([]*factoring.RemittanceLine, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		domainLine := &factoring.RemittanceLine{
			RowNumber:         line.RowNumber,
			Cells:             line.Cells,
			Kind:              line.Kind,
			InvoiceNumber:     line.InvoiceNumber,
			NormalizedInvoice: factoring.NormalizeRemittanceReference(line.InvoiceNumber),
			FactorReference:   line.FactorReference,
			DebtorName:        line.Debtor,
			FaceMinor:         line.FaceMinor,
			AdvanceMinor:      line.AdvanceMinor,
			FeeMinor:          line.FeeMinor,
			ReserveMinor:      line.ReserveMinor,
			CashMinor:         line.CashMinor,
			OccurredOn:        line.OccurredOn,
			MatchStatus:       factoring.RemittanceLineUnmatched,
			Suggestions:       []*factoring.RemittanceSuggestion{},
		}
		if line.Err != "" {
			domainLine.MatchStatus = factoring.RemittanceLineError
			domainLine.Error = clip(line.Err, factoring.MaxRemittanceReasonLength)
		}
		out = append(out, domainLine)
	}
	return out
}
