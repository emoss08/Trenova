package billingqueueassist

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	readDocumentTool  = "read_document"
	chargeHistoryTool = "charge_history"
	historyInvoices   = 40
)

// reviewReads are what a review may read beyond what it is shown: the rest of
// a long document, and one charge across more of the payer's invoices. Each
// read it makes is kept in log, to be recorded with its answer.
func (s *Service) reviewReads(
	tenantInfo pagination.TenantInfo,
	e *reviewprompt.Evidence,
	log *readLog,
) []llmports.NovaTool {
	reads := []llmports.NovaTool{&chargeHistory{
		repo:    s.repo,
		tenant:  tenantInfo,
		payerID: e.Item.BillToCustomerID,
		payer:   reviewprompt.PayerName(e),
		log:     log,
	}}

	long := make([]string, 0, len(e.Excerpts))
	for _, doc := range e.Documents {
		if e.Excerpts[doc.ID].Parts > 1 {
			long = append(long, doc.ID.String())
		}
	}
	if s.content != nil && len(long) > 0 {
		reads = append(reads, &documentReader{content: s.content, tenant: tenantInfo, ids: long, log: log})
	}

	return reads
}

// readLog is what a review read beyond what it was shown.
type readLog struct {
	parts   map[pulid.ID]int
	charges []reviewprompt.ChargeLookup
}

func (l *readLog) document(id pulid.ID, part int) {
	if l.parts == nil {
		l.parts = map[pulid.ID]int{}
	}
	l.parts[id] = max(l.parts[id], part)
}

func (l *readLog) charge(lookup reviewprompt.ChargeLookup) {
	lookup.Code = strings.ToUpper(lookup.Code)
	same := func(seen reviewprompt.ChargeLookup) bool { return seen.Code == lookup.Code }
	if i := slices.IndexFunc(l.charges, same); i >= 0 {
		l.charges[i] = lookup
		return
	}
	l.charges = append(l.charges, lookup)
}

func (l *readLog) taken() reviewprompt.Taken {
	return reviewprompt.Taken{Parts: l.parts, Charges: l.charges}
}

type documentReader struct {
	content documentports.DocumentContentService
	tenant  pagination.TenantInfo
	ids     []string
	log     *readLog
}

func (r *documentReader) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name: readDocumentTool,
		Description: "Read a further part of a shipment document's text. Part 1 is the excerpt " +
			"you were shown; each part is about " + strconv.Itoa(reviewprompt.ExcerptRunes) + " characters.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"documentId": map[string]any{"type": "string", "enum": r.ids},
				"part":       map[string]any{"type": "integer", "minimum": 2},
			},
			"required":             []string{"documentId", "part"},
			"additionalProperties": false,
		},
	}
}

func (r *documentReader) Call(ctx context.Context, arguments map[string]any) llmports.NovaToolResult {
	id, _ := arguments["documentId"].(string)
	if !slices.Contains(r.ids, id) {
		return refused("Only these documents have more to read: " + strings.Join(r.ids, ", ") + ".")
	}
	part, ok := wholeNumber(arguments["part"])
	if !ok || part < 2 {
		return refused("part is a whole number from 2.")
	}
	docID, err := pulid.Parse(id)
	if err != nil {
		return refused("documentId is not a document id.")
	}

	content, err := r.content.GetContent(ctx, docID, r.tenant)
	if err != nil || content == nil {
		return refused("The text of document " + id + " could not be read.")
	}
	parts := reviewprompt.Parts(strings.TrimSpace(content.ContentText))
	if part > len(parts) {
		return refused(fmt.Sprintf("Document %s has %d parts.", id, len(parts)))
	}
	r.log.document(docID, part)

	return llmports.NovaToolResult{Content: documentPartText(id, part, parts)}
}

func documentPartText(id string, part int, parts []string) string {
	return fmt.Sprintf("Part %d of %d of document %s:\n%s", part, len(parts), id, parts[part-1])
}

type chargeHistory struct {
	repo    billingqueueports.BillingQueueRepository
	tenant  pagination.TenantInfo
	payerID pulid.ID
	payer   string
	log     *readLog
}

func (h *chargeHistory) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name: chargeHistoryTool,
		Description: "Look up one charge on this payer's last " + strconv.Itoa(historyInvoices) +
			" posted invoices: which carried it and for how much, beside each invoice's total. " +
			"Use it when the recent invoices you were shown do not settle whether an amount is usual.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chargeCode": map[string]any{
					"type": "string",
					"description": "A charge code from the invoice or the accessorial catalog, or " +
						reviewprompt.FreightCode + " for the line haul.",
				},
			},
			"required":             []string{"chargeCode"},
			"additionalProperties": false,
		},
	}
}

func (h *chargeHistory) Call(ctx context.Context, arguments map[string]any) llmports.NovaToolResult {
	code, _ := arguments["chargeCode"].(string)
	code = strings.TrimSpace(code)
	if code == "" {
		return refused("chargeCode is required.")
	}

	recent, err := h.repo.RecentCustomerInvoices(ctx, h.tenant, h.payerID, historyInvoices)
	if err != nil {
		return refused("The payer's invoices could not be read.")
	}
	h.log.charge(chargeLookup(recent, code))

	return llmports.NovaToolResult{Content: chargeHistoryText(recent, h.payer, code)}
}

// chargeLookup is how many of the payer's recent invoices carried a charge.
func chargeLookup(recent []*billingqueueports.RecentInvoice, code string) reviewprompt.ChargeLookup {
	keep := chargeMatcher(code)
	carried := 0
	for _, past := range recent {
		if slices.ContainsFunc(past.Charges, keep) {
			carried++
		}
	}

	return reviewprompt.ChargeLookup{Code: code, Carried: carried, Of: len(recent)}
}

func chargeMatcher(code string) func(*billingqueueports.RecentCharge) bool {
	return func(charge *billingqueueports.RecentCharge) bool {
		if strings.EqualFold(code, reviewprompt.FreightCode) {
			return charge.Type == invoice.InvoiceLineTypeFreight
		}

		return strings.EqualFold(charge.Code, code)
	}
}

// chargeHistoryText is one charge across the payer's recent invoices.
func chargeHistoryText(recent []*billingqueueports.RecentInvoice, payer, code string) string {
	keep := chargeMatcher(code)
	lines := make([]string, 0, len(recent))
	for _, past := range recent {
		if slices.ContainsFunc(past.Charges, keep) {
			lines = append(lines, reviewprompt.InvoiceLine(past, keep))
		}
	}
	if len(lines) == 0 {
		return fmt.Sprintf("None of the last %d posted invoices to %s carried %s.", len(recent), payer, code)
	}

	return fmt.Sprintf("%d of the last %d posted invoices to %s carried %s:\n%s",
		len(lines), len(recent), payer, code, strings.Join(lines, "\n"))
}

func wholeNumber(value any) (int, bool) {
	switch number := value.(type) {
	case float64:
		return int(number), number == float64(int(number))
	case int:
		return number, true
	case int64:
		return int(number), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(number))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func refused(content string) llmports.NovaToolResult {
	return llmports.NovaToolResult{Content: content, Failed: true}
}
