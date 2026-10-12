package businesscentral

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	PageGeneralLedgerEntries     = 20
	PageCustomerLedgerEntries    = 25
	PagePostedSalesInvoice       = 132
	PagePostedSalesCreditMemo    = 134
	PagePostedPurchaseInvoice    = 138
	PagePostedPurchaseCreditMemo = 140
)

type Linker struct {
	webURL string
}

func NewLinker(opts ...Option) *Linker {
	settings := resolveOptions(opts)
	return &Linker{webURL: strings.TrimRight(strings.TrimSpace(settings.webURL), "/")}
}

func DocumentLink(ref CompanyRef, companyName string, page int, number string) string {
	return NewLinker().DocumentLink(ref, companyName, page, number)
}

func (l *Linker) DocumentLink(ref CompanyRef, companyName string, page int, number string) string {
	name := strings.TrimSpace(companyName)
	value := strings.TrimSpace(number)
	if name == "" || value == "" || page <= 0 {
		return ""
	}
	normalized, err := NewCompanyRef(ref.TenantID, ref.Environment, ref.CompanyID)
	if err != nil {
		return ""
	}
	return l.webURL + "/" + normalized.TenantID + "/" + url.PathEscape(normalized.Environment) +
		"/?company=" + url.QueryEscape(name) +
		"&page=" + strconv.Itoa(page) +
		"&filter=" + url.QueryEscape("'No.' IS '"+value+"'")
}
