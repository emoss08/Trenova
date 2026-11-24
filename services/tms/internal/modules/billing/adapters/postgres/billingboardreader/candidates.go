package billingboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AgainstCandidate struct{ db *postgres.Connection }

func NewAgainstCandidate(p Params) *AgainstCandidate { return &AgainstCandidate{db: p.DB} }

func (*AgainstCandidate) Kind() recordkind.Kind { return receiptKind }
func (*AgainstCandidate) Measure() string       { return billingboard.MeasureAgainstCandidate }

type bestRow struct {
	ReceiptID       pulid.ID
	Kind            string
	Standing        string
	OpenMinor       int64
	DifferenceMinor int64
	InvoiceNumber   string
	Customer        string
}

func shownCandidates() (
	*bankreceiptdb.CandidatesTable,
	keel.Query[*bankreceipt.Candidate],
	keel.Pred,
) {
	c := bankreceiptdb.Candidates
	cp := customerpaymentdb.Payments
	inv := invoicedb.Invoices

	return c, c.
			LeftJoin(cp, cp.ID.EqCol(c.CustomerPaymentID)).
			LeftJoin(inv, inv.ID.EqCol(c.InvoiceID)),
		receivablesql.ShownCandidate(c, cp, inv)
}

func readBest(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) (map[pulid.ID]*bestRow, error) {
	c, joined, shown := shownCandidates()
	cus := customerdb.Customers
	rows, err := keel.Into[bestRow](ctx, db.Keel(ctx), joined.
		LeftJoin(cus, cus.ID.EqCol(c.CustomerID)).
		DistinctOn(c.BankReceiptID).
		Pick(
			c.BankReceiptID, text(c.Kind), text(c.Standing), c.OpenMinor, c.DifferenceMinor,
			text(invoicedb.Invoices.Number), text(cus.Name),
		).
		Where(c.BankReceiptID.In(ids...), shown).
		Order(c.BankReceiptID.Asc(), c.Rank.Asc()))
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]*bestRow, len(rows))
	for idx := range rows {
		out[rows[idx].ReceiptID] = &rows[idx]
	}

	return out, nil
}

func (r *AgainstCandidate) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			receipts, err := readReceipts(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			best, err := readBest(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(receipts))
			for idx := range receipts {
				row := &receipts[idx]
				out[row.ID] = againstReading(ctx, row, best[row.ID])
			}

			return out, nil
		},
	)
}

func againstReading(ctx context.Context, row *receiptRow, best *bestRow) boardreading.Reading {
	if best == nil || bankreceipt.Status(row.Status) == bankreceipt.StatusMatched {
		return receivedReading(ctx, row)
	}
	value := boardcatalog.Money(row.AmountMinor, "")
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
	}
	invoice := bankreceipt.CandidateKind(best.Kind) == bankreceipt.CandidateInvoice
	name := named(ctx, best, invoice)
	switch bankreceipt.Standing(best.Standing) {
	case bankreceipt.StandingExact:
		reading.Tone = boardcatalog.ToneOK
		reading.Fact = textFact(exactWords(ctx, invoice), name)
	case bankreceipt.StandingSameAmount:
		reading.Tone = boardcatalog.ToneInfo
		reading.Fact = textFact(sameWords(ctx, invoice), name)
	case bankreceipt.StandingShort:
		reading.Tone = boardcatalog.ToneWarn
		reading.Fact = gapFact(shortWords(ctx, invoice, name), best.DifferenceMinor)
		reading.Bounded = bounded(row, best)
	case bankreceipt.StandingOver:
		reading.Tone = boardcatalog.ToneWarn
		reading.Fact = gapFact(overWords(ctx, invoice, name), best.DifferenceMinor)
		reading.Bounded = bounded(row, best)
	}

	return reading
}

func named(ctx context.Context, best *bestRow, invoice bool) string {
	if invoice {
		return strings.TrimSpace(best.InvoiceNumber)
	}
	if customer := strings.TrimSpace(best.Customer); customer != "" {
		return customer
	}

	return i18n.T(ctx, "a customer")
}

func exactWords(ctx context.Context, invoice bool) string {
	if invoice {
		return i18n.T(ctx, "Matches {0} to the cent")
	}

	return i18n.T(ctx, "Matches {0}'s payment to the cent")
}

func sameWords(ctx context.Context, invoice bool) string {
	if invoice {
		return i18n.T(ctx, "Same amount as {0}")
	}

	return i18n.T(ctx, "Same amount as {0}'s payment")
}

func shortWords(ctx context.Context, invoice bool, name string) string {
	if invoice {
		return i18n.T(ctx, "{0} short of {1}", slot, name)
	}

	return i18n.T(ctx, "{0} short of {1}'s payment", slot, name)
}

func overWords(ctx context.Context, invoice bool, name string) string {
	if invoice {
		return i18n.T(ctx, "{0} over {1}", slot, name)
	}

	return i18n.T(ctx, "{0} over {1}'s payment", slot, name)
}

func textFact(words, name string) *boardreading.Words {
	text := boardcatalog.Value{Kind: boardcatalog.ValueText, Text: name}

	return &boardreading.Words{Text: words, Value: &text}
}

func gapFact(words string, difference int64) *boardreading.Words {
	money := boardcatalog.Money(max(difference, -difference), "")

	return &boardreading.Words{Text: words, Value: &money}
}

func bounded(row *receiptRow, best *bestRow) *boardreading.BoundedFacts {
	return &boardreading.BoundedFacts{
		Value: decimal.NewFromInt(row.AmountMinor),
		Bound: decimal.NewFromInt(best.OpenMinor),
	}
}
