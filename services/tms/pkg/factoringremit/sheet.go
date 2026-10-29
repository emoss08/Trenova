package factoringremit

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/pkg/rateimport"
)

var ErrUnsupportedFile = errors.New(
	"a remittance must be a CSV, an Excel workbook (.xlsx) or a PDF",
)

func FormatFor(fileName, contentType string) (factoringremittance.RemittanceFormat, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".csv", ".txt":
		return factoringremittance.RemittanceFormatCsv, nil
	case ".xlsx":
		return factoringremittance.RemittanceFormatXlsx, nil
	case ".pdf":
		return factoringremittance.RemittanceFormatPdf, nil
	}
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])) {
	case "text/csv", "application/csv", "text/plain":
		return factoringremittance.RemittanceFormatCsv, nil
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return factoringremittance.RemittanceFormatXlsx, nil
	case "application/pdf":
		return factoringremittance.RemittanceFormatPdf, nil
	}
	return "", ErrUnsupportedFile
}

func IsRemittanceFile(fileName, contentType string) bool {
	_, err := FormatFor(fileName, contentType)
	return err == nil
}

func ReadSheet(
	data []byte,
	format factoringremittance.RemittanceFormat,
	limits rateimport.Limits,
) (*rateimport.Sheet, error) {
	switch format {
	case factoringremittance.RemittanceFormatCsv:
		return rateimport.ReadCSV(data, limits)
	case factoringremittance.RemittanceFormatXlsx:
		return rateimport.ReadXLSX(data, limits)
	case factoringremittance.RemittanceFormatPdf:
	}
	return nil, ErrUnsupportedFile
}

func ToDomain(lines []*Line) []*factoringremittance.RemittanceLine {
	out := make([]*factoringremittance.RemittanceLine, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		domainLine := &factoringremittance.RemittanceLine{
			RowNumber:         line.RowNumber,
			Cells:             line.Cells,
			Kind:              line.Kind,
			InvoiceNumber:     line.InvoiceNumber,
			NormalizedInvoice: factoringremittance.NormalizeRemittanceReference(line.InvoiceNumber),
			FactorReference:   line.FactorReference,
			DebtorName:        line.Debtor,
			FaceMinor:         line.FaceMinor,
			AdvanceMinor:      line.AdvanceMinor,
			FeeMinor:          line.FeeMinor,
			ReserveMinor:      line.ReserveMinor,
			CashMinor:         line.CashMinor,
			OccurredOn:        line.OccurredOn,
			MatchStatus:       factoringremittance.RemittanceLineUnmatched,
			Suggestions:       []*factoringremittance.RemittanceSuggestion{},
		}
		if line.Err != "" {
			domainLine.MatchStatus = factoringremittance.RemittanceLineError
			domainLine.Error = clip(line.Err, factoringremittance.MaxRemittanceReasonLength)
		}
		out = append(out, domainLine)
	}
	return out
}
