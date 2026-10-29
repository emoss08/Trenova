package invoicedocument

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
)

func PDFName(entity *invoice.Invoice) string {
	number := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(entity.Number)
	return "invoice-" + number + ".pdf"
}

func UnixDate(value int64) string {
	if value == 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format("2006-01-02")
}

func UnixDatePtr(value *int64) string {
	if value == nil {
		return ""
	}
	return UnixDate(*value)
}

func MoneyString(currency, amount string) string {
	if currency == "" {
		currency = "USD"
	}
	return currency + " " + amount
}
