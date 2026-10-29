package nsdocsync

import (
	"maps"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/shopspring/decimal"
)

const (
	moneyPlaces     = 2
	maxTranIDLength = 45
	maxMemoLength   = 999
	DefaultLineText = "Charges"
	DateLayout      = "2006-01-02"
)

func unapplyLines(ids []string) *nsbinding.Sublist[nsbinding.ApplyLine] {
	lines := make([]nsbinding.ApplyLine, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, nsbinding.ApplyLine{Doc: nsbinding.RecordRef{ID: id}, Apply: false})
	}
	return &nsbinding.Sublist[nsbinding.ApplyLine]{Items: lines}
}

func lineText(description string) string {
	if text := strings.TrimSpace(description); text != "" {
		return text
	}
	return DefaultLineText
}

func linePurpose(idx int, description string) string {
	return "Line " + strconv.Itoa(idx+1) + " (" + lineText(description) + ")"
}

func sameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func newResult(refs map[string]string) *accountingconnectorports.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+4)
	maps.Copy(copied, refs)
	return &accountingconnectorports.AccountingDocumentResult{Refs: copied}
}
