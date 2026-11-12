package recordcaseservice

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/shopspring/decimal"
)

const (
	secondsPerDay    = 86_400
	secondsPerMinute = 60
)

func invoiceSubject(
	invoices invoiceports.InvoiceRepository,
	disputes invoiceports.InvoiceDisputeRepository,
) subjectReader {
	return func(ctx context.Context, read *subjectRead) (*subject, error) {
		entity, err := readInvoice(ctx, invoices, read, read.id)
		if err != nil || entity == nil {
			return nil, err
		}
		dispute, err := disputes.GetOpenByInvoiceID(ctx, invoiceports.GetOpenInvoiceDisputeRequest{
			InvoiceID:  entity.ID,
			TenantInfo: read.tenant,
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}

		return (&invoiceCase{read: read, invoice: entity, dispute: dispute}).build(false), nil
	}
}

func invoiceStand(disputes invoiceports.InvoiceDisputeRepository) standReader {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
	) (recordkind.Kind, pulid.ID, error) {
		dispute, err := disputes.GetOpenByInvoiceID(ctx, invoiceports.GetOpenInvoiceDisputeRequest{
			InvoiceID:  id,
			TenantInfo: tenant,
		})
		if err != nil || dispute == nil {
			return "", pulid.Nil, err
		}

		return recordkind.Kind(permission.ResourceInvoiceDispute), dispute.ID, nil
	}
}

func disputeSubject(
	invoices invoiceports.InvoiceRepository,
	disputes invoiceports.InvoiceDisputeRepository,
) subjectReader {
	return func(ctx context.Context, read *subjectRead) (*subject, error) {
		dispute, err := disputes.GetByID(ctx, invoiceports.GetInvoiceDisputeByIDRequest{
			ID:         read.id,
			TenantInfo: read.tenant,
		})
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return &subject{}, nil
			}

			return nil, err
		}
		entity, err := readInvoice(ctx, invoices, read, dispute.InvoiceID)
		if err != nil || entity == nil {
			return nil, err
		}

		return (&invoiceCase{read: read, invoice: entity, dispute: dispute}).build(true), nil
	}
}

func readInvoice(
	ctx context.Context,
	invoices invoiceports.InvoiceRepository,
	read *subjectRead,
	id pulid.ID,
) (*invoice.Invoice, error) {
	found, err := invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: read.tenant,
		InvoiceIDs: []pulid.ID{id},
	})
	if err != nil {
		return nil, err
	}
	for _, entity := range found {
		if entity != nil {
			return entity, nil
		}
	}

	return nil, nil //nolint:nilnil // a record with no invoice behind it draws a plain case
}

type invoiceCase struct {
	read    *subjectRead
	invoice *invoice.Invoice
	dispute *invoicedispute.InvoiceDispute
}

func (c *invoiceCase) amount(minor int64) string {
	if c.invoice.CurrencyCode == "" {
		return money.FormatDollars(decimal.New(minor, -2))
	}

	return money.Display(c.invoice.CurrencyCode, decimal.New(minor, -2))
}

func (c *invoiceCase) terms() string {
	words := textcase.HumanizeCamelCaseSentence(string(c.invoice.PaymentTerm))
	var spaced strings.Builder
	for i, r := range words {
		if unicode.IsDigit(r) && i > 0 && unicode.IsLetter(rune(words[i-1])) {
			spaced.WriteByte(' ')
		}
		spaced.WriteRune(r)
	}

	return spaced.String()
}

func (c *invoiceCase) open() int64 {
	return max(c.invoice.TotalAmountMinor-c.invoice.AppliedAmountMinor, 0)
}

func (c *invoiceCase) pastDueDays() int64 {
	if c.invoice.DueDate == nil || *c.invoice.DueDate <= 0 ||
		c.invoice.SettlementStatus == invoice.SettlementStatusPaid ||
		c.invoice.Status != invoice.StatusPosted {
		return 0
	}

	return max((c.read.now-*c.invoice.DueDate)/secondsPerDay, 0)
}

func (c *invoiceCase) disputed() bool {
	return c.dispute != nil && c.dispute.Status == invoicedispute.DisputeCaseStatusOpen
}

func (c *invoiceCase) build(aboutDispute bool) *subject {
	out := &subject{
		head:     c.head(aboutDispute),
		absorbs:  map[watchtower.SourceKind]struct{}{},
		customer: c.invoice.BillToName,
	}
	out.exhibits = append(out.exhibits, c.owed())
	if c.dispute != nil {
		out.exhibits = append(out.exhibits, c.disputeFacts())
	}
	out.exhibits = append(out.exhibits, c.history())
	out.unchanged = c.unchanged()
	out.starters = c.starters()

	return out
}

func (c *invoiceCase) head(aboutDispute bool) recordcase.Head {
	head := recordcase.Head{StatusTone: recordcase.ToneNeutral}
	head.References = nonEmpty(c.invoice.BillToName)
	if aboutDispute {
		head.References = append(head.References, "Invoice "+c.invoice.Number)
	}
	if c.invoice.ShipmentProNumber != "" {
		head.References = append(head.References, "PRO "+c.invoice.ShipmentProNumber)
	}
	if c.invoice.PONumber != "" {
		head.References = append(head.References, "PO "+c.invoice.PONumber)
	}

	total := c.amount(c.invoice.TotalAmountMinor)
	head.Title = total
	if c.invoice.BillToName != "" {
		head.Title = total + " billed to " + c.invoice.BillToName
	}
	if aboutDispute {
		head.Title = c.amount(c.dispute.DisputedAmountMinor) + " disputed: " +
			lowerReason(c.dispute.ReasonCode)
	}

	head.Status, head.StatusTone = c.status()
	head.Details = append(head.Details, c.terms())
	if c.invoice.DueDate != nil && *c.invoice.DueDate > 0 {
		head.Details = append(head.Details, "Due "+c.read.day(*c.invoice.DueDate))
	}
	if c.invoice.SentAt != nil && *c.invoice.SentAt > 0 {
		head.Details = append(head.Details, "Sent "+c.read.day(*c.invoice.SentAt))
	} else if c.invoice.Status == invoice.StatusPosted {
		head.Details = append(head.Details, "Not sent yet")
	}

	return head
}

func lowerReason(reason invoicedispute.DisputeReasonCode) string {
	return textcase.LowerFirst(textcase.HumanizeCamelCaseSentence(string(reason)))
}

func (c *invoiceCase) status() (string, recordcase.Tone) {
	status := textcase.HumanizeCamelCaseSentence(string(c.invoice.Status))
	switch {
	case c.invoice.Status == invoice.StatusVoided:
		return status, recordcase.ToneNeutral
	case c.invoice.SettlementStatus == invoice.SettlementStatusPaid:
		return status + " · paid", recordcase.ToneGood
	case c.disputed():
		return status + " · disputed", recordcase.ToneWarning
	case c.pastDueDays() > 0:
		return fmt.Sprintf("%s · %d days past due", status, c.pastDueDays()), recordcase.ToneBad
	case c.invoice.SettlementStatus == invoice.SettlementStatusPartiallyPaid:
		return status + " · partly paid", recordcase.ToneWarning
	case c.invoice.Status == invoice.StatusDraft:
		return status + " · not posted", recordcase.ToneWarning
	default:
		return status + " · unpaid", recordcase.ToneNeutral
	}
}

func (c *invoiceCase) owed() recordcase.Exhibit {
	total := c.invoice.TotalAmountMinor
	exhibit := recordcase.Exhibit{
		Key:    "owed",
		Kind:   recordcase.ExhibitMeters,
		Title:  "What is owed",
		Source: "Receivables",
		At:     c.invoice.UpdatedAt,
		Read:   true,
		Meters: []recordcase.Meter{{
			Label:    "Paid",
			Value:    c.amount(c.invoice.AppliedAmountMinor) + " of " + c.amount(total),
			Fraction: fraction(c.invoice.AppliedAmountMinor, total),
			Tone:     recordcase.ToneNeutral,
		}},
	}
	exhibit.Claims = []string{
		fmt.Sprintf("%s paid of %s; %s still open",
			c.amount(c.invoice.AppliedAmountMinor), c.amount(total), c.amount(c.open())),
	}
	if c.disputed() {
		exhibit.Meters = append(exhibit.Meters, recordcase.Meter{
			Label:    "Disputed",
			Value:    c.amount(c.dispute.DisputedAmountMinor) + " of " + c.amount(total),
			Fraction: fraction(c.dispute.DisputedAmountMinor, total),
			Tone:     recordcase.ToneWarning,
		})
	}
	if days := c.pastDueDays(); days > 0 {
		exhibit.Note = fmt.Sprintf("%s is %d days past due.", c.amount(c.open()), days)
		exhibit.Claims = append(exhibit.Claims, exhibit.Note)
	}

	return exhibit
}

func (c *invoiceCase) disputeFacts() recordcase.Exhibit {
	opened := c.dispute.OpenedAt
	exhibit := recordcase.Exhibit{
		Key:    "dispute",
		Kind:   recordcase.ExhibitFacts,
		Title:  "The dispute",
		Source: "Invoice disputes",
		At:     c.dispute.OpenedAt,
		Facts: []recordcase.Fact{
			{
				Label: "Reason",
				Value: textcase.HumanizeCamelCaseSentence(string(c.dispute.ReasonCode)),
			},
			{
				Label: "Amount",
				Value: c.amount(c.dispute.DisputedAmountMinor),
				Tone:  recordcase.ToneWarning,
			},
			{Label: "Opened", At: &opened},
			{Label: "Status", Value: textcase.HumanizeCamelCaseSentence(string(c.dispute.Status))},
		},
		Note: c.dispute.Notes,
	}
	if c.dispute.Resolution != "" {
		exhibit.Facts = append(exhibit.Facts, recordcase.Fact{
			Label: "Resolution",
			Value: textcase.HumanizeCamelCaseSentence(string(c.dispute.Resolution)),
		})
	}
	exhibit.Claims = []string{
		fmt.Sprintf("%s disputed for %s, opened %s",
			c.amount(c.dispute.DisputedAmountMinor),
			lowerReason(c.dispute.ReasonCode),
			c.read.day(c.dispute.OpenedAt),
		),
		c.dispute.Notes,
	}

	return exhibit
}

func (c *invoiceCase) history() recordcase.Exhibit {
	exhibit := recordcase.Exhibit{
		Key:    "history",
		Kind:   recordcase.ExhibitTimeline,
		Title:  "Invoice " + c.invoice.Number,
		Source: "Billing",
		At:     c.invoice.UpdatedAt,
	}
	exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
		Label: "Invoiced",
		When:  c.read.day(c.invoice.InvoiceDate),
		State: recordcase.PointDone,
	})
	if c.invoice.PostedAt != nil && *c.invoice.PostedAt > 0 {
		exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
			Label: "Posted",
			When:  c.read.day(*c.invoice.PostedAt),
			State: recordcase.PointDone,
		})
	}
	if c.invoice.SentAt != nil && *c.invoice.SentAt > 0 {
		exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
			Label: "Sent to " + stringutils.FirstNonEmptyTrimmed(
				c.invoice.BillToName,
				"the customer",
			),
			When:  c.read.day(*c.invoice.SentAt),
			State: recordcase.PointDone,
		})
	}
	if c.dispute != nil {
		exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
			Label: "Disputed",
			Sub:   textcase.HumanizeCamelCaseSentence(string(c.dispute.ReasonCode)),
			When:  c.read.day(c.dispute.OpenedAt),
			State: recordcase.PointDone,
			Tone:  recordcase.ToneWarning,
		})
	}
	if c.invoice.DueDate != nil && *c.invoice.DueDate > 0 {
		row := recordcase.TimelineRow{
			Label: "Due",
			Sub:   c.terms(),
			When:  c.read.day(*c.invoice.DueDate),
			State: recordcase.PointAhead,
		}
		switch {
		case c.invoice.SettlementStatus == invoice.SettlementStatusPaid:
			row.State, row.Tone = recordcase.PointDone, recordcase.ToneGood
		case c.pastDueDays() > 0:
			row.Tone = recordcase.ToneBad
		}
		if chosen := c.read.chosen; chosen != nil && chosen.Kind == recordcase.OptionKindPlan &&
			chosen.Outcome != "" && row.State == recordcase.PointAhead {
			if step := chosen.Projection; step != nil && step.Step != nil {
				exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
					Label:    step.Step.Label,
					When:     step.Step.When,
					State:    recordcase.PointProposed,
					Proposed: true,
				})
			}
		}
		exhibit.Timeline = append(exhibit.Timeline, row)
		exhibit.Claims = append(exhibit.Claims, "due "+row.When)
	}

	return exhibit
}

func (c *invoiceCase) unchanged() string {
	switch {
	case c.disputed():
		return fmt.Sprintf(
			"The dispute stays open and %s stays out of collections.",
			c.amount(c.dispute.DisputedAmountMinor),
		)
	case c.pastDueDays() > 0:
		return fmt.Sprintf(
			"%s stays unpaid, now %d days past due.",
			c.amount(c.open()),
			c.pastDueDays(),
		)
	default:
		return ""
	}
}

func (c *invoiceCase) starters() []string {
	who := stringutils.FirstNonEmptyTrimmed(c.invoice.BillToName, "the customer")
	if c.disputed() {
		return []string{
			"What does the dispute say?",
			"What would a credit memo cost us?",
			"Draft a reply to " + who,
		}
	}

	return []string{
		"Why hasn't " + who + " paid this?",
		"What else does " + who + " owe?",
		"Draft a reminder to " + who,
	}
}
