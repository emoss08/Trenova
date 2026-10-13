package customerstatementservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

func pdfContext(from *issuer, p *prepared, note string) *documenttemplate.CustomerStatementContext {
	tz := from.org.Timezone
	st := p.statement
	format := func(minor int64) string { return money.FormatMinor(minor, p.currency) }
	// A blank cell reads as "nothing", where a zero would read as a
	// transaction for nothing.
	formatNonZero := func(minor int64) string {
		if minor == 0 {
			return ""
		}
		return format(minor)
	}

	out := &documenttemplate.CustomerStatementContext{
		StatementDate: timeutils.FormatUnixDateIn(p.period.End, tz),
		PeriodStart:   timeutils.FormatUnixDateIn(p.period.Start, tz),
		CurrencyCode:  p.currency,
		CustomerCode:  p.customer.Code,
		Organization: documenttemplate.AddressBlock{
			Name:  from.org.Name,
			Lines: from.org.MailingAddress().Lines(),
		},
		Customer: documenttemplate.AddressBlock{
			Name:  p.customer.DisplayName(),
			Lines: p.customer.MailingAddress().Lines(),
		},
		RemitTo:        remitTo(from, p),
		LogoDataURI:    from.logo,
		OpeningBalance: format(st.OpeningBalanceMinor),
		TotalCharges:   format(st.TotalChargesMinor),
		TotalPayments:  format(st.TotalPaymentsMinor),
		EndingBalance:  format(st.EndingBalanceMinor),
		PastDue:        formatNonZero(p.pastDue),
		Aging: documenttemplate.StatementAging{
			Current:    format(st.Aging.CurrentMinor),
			Days1To30:  format(st.Aging.Days1To30Minor),
			Days31To60: format(st.Aging.Days31To60Minor),
			Days61To90: format(st.Aging.Days61To90Minor),
			Over90:     format(st.Aging.DaysOver90Minor),
		},
		Transactions:       make([]documenttemplate.StatementTransaction, 0, len(st.Transactions)),
		OpenItems:          make([]documenttemplate.StatementOpenItem, 0, len(st.OpenItems)),
		Note:               note,
		InvoiceFooter:      invoiceFooter(from.control),
		NoticeOfAssignment: p.factor.Notice(),
	}
	for _, txn := range st.Transactions {
		if txn == nil {
			continue
		}
		out.Transactions = append(out.Transactions, documenttemplate.StatementTransaction{
			Date:        timeutils.FormatUnixDateIn(txn.TransactionDate, tz),
			Document:    txn.DocumentNumber,
			Description: txn.Description,
			Charge:      formatNonZero(txn.ChargeMinor),
			Payment:     formatNonZero(txn.PaymentMinor),
			Balance:     format(txn.RunningBalanceMinor),
		})
	}
	var totalOpen int64
	for _, item := range st.OpenItems {
		if item == nil {
			continue
		}
		currency := money.CurrencyCode(item.CurrencyCode)
		totalOpen += item.OpenAmountMinor
		out.OpenItems = append(out.OpenItems, documenttemplate.StatementOpenItem{
			InvoiceNumber: item.InvoiceNumber,
			InvoiceDate:   timeutils.FormatUnixDateIn(item.InvoiceDate, tz),
			DueDate:       timeutils.FormatUnixDateIn(item.DueDate, tz),
			DaysPastDue:   max(item.DaysPastDue, 0),
			Total:         money.FormatMinor(item.TotalAmountMinor, currency),
			Open:          money.FormatMinor(item.OpenAmountMinor, currency),
			Disputed:      item.DisputeStatus == string(invoice.DisputeStatusDisputed),
		})
	}
	out.TotalOpen = format(totalOpen)
	return out
}

func emailContext(
	from *issuer,
	p *prepared,
	d *delivery,
) *documenttemplate.CustomerStatementEmailContext {
	tz := from.org.Timezone
	pastDue := ""
	if p.pastDue > 0 {
		pastDue = money.FormatMinor(p.pastDue, p.currency)
	}
	return &documenttemplate.CustomerStatementEmailContext{
		CustomerName:           p.customer.DisplayName(),
		CustomerCode:           p.customer.Code,
		CompanyName:            from.org.Name,
		StatementDate:          timeutils.FormatUnixDateIn(p.period.End, tz),
		PeriodStart:            timeutils.FormatUnixDateIn(p.period.Start, tz),
		CurrencyCode:           p.currency,
		EndingBalance:          money.FormatMinor(p.statement.EndingBalanceMinor, p.currency),
		PastDue:                pastDue,
		OpenItemCount:          len(p.statement.OpenItems),
		IsMonthly:              d.monthly,
		RemittanceInstructions: statementRemittance(from, p),
		Note:                   d.note,
		AttachmentName:         p.fileName,
	}
}

// remitTo is where the statement says to pay: the factor when the
// customer's invoices are assigned to one, otherwise the organization with
// the remittance instructions from billing control beneath it, as the
// invoice prints it.
func remitTo(from *issuer, p *prepared) documenttemplate.AddressBlock {
	if p.factor != nil {
		return documenttemplate.AddressBlock{Name: p.factor.Name, Lines: p.factor.RemitLines()}
	}
	lines := from.org.MailingAddress().Lines()
	lines = append(lines, stringutils.FilterEmpty(strings.Split(remittanceInstructions(from.control), "\n"))...)
	return documenttemplate.AddressBlock{Name: from.org.Name, Lines: lines}
}

func statementRemittance(from *issuer, p *prepared) string {
	if p.factor != nil {
		return p.factor.RemittanceText()
	}
	return remittanceInstructions(from.control)
}

func remittanceInstructions(control *tenant.BillingControl) string {
	if control == nil {
		return ""
	}
	return strings.TrimSpace(control.DefaultInvoiceTerms)
}

func invoiceFooter(control *tenant.BillingControl) string {
	if control == nil {
		return ""
	}
	return strings.TrimSpace(control.DefaultInvoiceFooter)
}
