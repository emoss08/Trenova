package billingqueueassist

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// readAheadParts is how far into a long document a batched review reads: the
// excerpt it is shown and the parts after it, up to this one.
const readAheadParts = 3

// readAhead makes, for a page of reviews sent as a batch, the reads a review
// cannot make there: a batch takes one answer, so each review is given what
// it would have read. That is every charge on its invoice across the payer's
// last invoices, and the next parts of each long document. A payer's
// invoices are read once for the page.
type readAhead struct {
	s      *Service
	tenant pagination.TenantInfo
	recent map[pulid.ID][]*billingqueueports.RecentInvoice
}

func (s *Service) newReadAhead(tenantInfo pagination.TenantInfo) *readAhead {
	return &readAhead{
		s:      s,
		tenant: tenantInfo,
		recent: map[pulid.ID][]*billingqueueports.RecentInvoice{},
	}
}

// sections are what the review would have read, as untrusted context: past
// invoices carry descriptions people typed, and documents carry their text.
func (r *readAhead) sections(
	ctx context.Context,
	e *reviewprompt.Evidence,
) []llmports.ContextSection {
	return append(r.history(ctx, e), r.documents(ctx, e)...)
}

func (r *readAhead) history(
	ctx context.Context,
	e *reviewprompt.Evidence,
) []llmports.ContextSection {
	codes := reviewprompt.ChargeCodes(e)
	if len(codes) == 0 {
		return nil
	}
	recent := r.payerInvoices(ctx, e.Item.BillToCustomerID)
	if len(recent) == 0 {
		return nil
	}

	payer := reviewprompt.PayerName(e)
	sections := make([]llmports.ContextSection, 0, len(codes))
	for _, code := range codes {
		sections = append(sections, llmports.ContextSection{
			Title:   "What " + chargeHistoryTool + " found for " + code,
			Content: chargeHistoryText(recent, payer, code),
		})
	}

	return sections
}

// payerInvoices are the payer's last invoices, read once for the page.
func (r *readAhead) payerInvoices(
	ctx context.Context,
	payerID pulid.ID,
) []*billingqueueports.RecentInvoice {
	recent, read := r.recent[payerID]
	if read {
		return recent
	}
	recent, err := r.s.repo.RecentCustomerInvoices(ctx, r.tenant, payerID, historyInvoices)
	if err != nil {
		r.s.l.Warn("a payer's invoices could not be read for a batched review",
			zap.String("customerId", payerID.String()), zap.Error(err))
	}
	r.recent[payerID] = recent

	return recent
}

func (r *readAhead) documents(
	ctx context.Context,
	e *reviewprompt.Evidence,
) []llmports.ContextSection {
	var sections []llmports.ContextSection
	for _, doc := range e.Documents {
		if e.Excerpts[doc.ID].Parts < 2 {
			continue
		}
		content, err := r.s.content.GetContent(ctx, doc.ID, r.tenant)
		if err != nil || content == nil {
			continue
		}
		parts := reviewprompt.Parts(strings.TrimSpace(content.ContentText))
		for part := 2; part <= min(len(parts), readAheadParts); part++ {
			sections = append(sections, llmports.ContextSection{
				Title:   "What " + readDocumentTool + " found in document " + doc.ID.String(),
				Content: documentPartText(doc.ID.String(), part, parts),
			})
		}
	}

	return sections
}

// taken is what the read-ahead gave the review of e, as the reads it would
// have made: each charge on its invoice across the payer's invoices, when it
// has any, and the parts after each long document's excerpt.
func (r *readAhead) taken(ctx context.Context, e *reviewprompt.Evidence) reviewprompt.Taken {
	taken := reviewprompt.Taken{}
	if codes := reviewprompt.ChargeCodes(e); len(codes) > 0 {
		if recent := r.payerInvoices(ctx, e.Item.BillToCustomerID); len(recent) > 0 {
			for _, code := range codes {
				taken.Charges = append(taken.Charges, chargeLookup(recent, code))
			}
		}
	}
	for _, doc := range e.Documents {
		if parts := e.Excerpts[doc.ID].Parts; parts > 1 {
			if taken.Parts == nil {
				taken.Parts = map[pulid.ID]int{}
			}
			taken.Parts[doc.ID] = min(parts, readAheadParts)
		}
	}

	return taken
}
