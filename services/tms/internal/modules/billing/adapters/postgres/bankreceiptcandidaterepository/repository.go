package bankreceiptcandidaterepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	invoicePrefix = "inv_"
	paymentPrefix = "cpay_"
	receiptPrefix = "brcpt_"
	lockPrefix    = "bank-receipt-candidates:"
	maxTouched    = 5000
)

type Params struct {
	fx.In

	DB           *postgres.Connection
	Entitlements entitlementports.Filter
	Logger       *zap.Logger
}

type repository struct {
	db           *postgres.Connection
	entitlements entitlementports.Filter
	l            *zap.Logger
}

func New(p Params) bankreceiptports.CandidateRepository {
	return &repository{
		db:           p.DB,
		entitlements: p.Entitlements,
		l:            p.Logger.Named("postgres.bank-receipt-candidate-repository"),
	}
}

func unapplied(br *bankreceiptdb.BankReceiptsTable) keel.Pred {
	return br.Status.In(bankreceipt.StatusImported, bankreceipt.StatusException)
}

type subjects struct {
	receipts []pulid.ID
	invoices []pulid.ID
	payments []pulid.ID
}

func split(ids []pulid.ID) subjects {
	var out subjects
	for _, id := range ids {
		switch value := id.String(); {
		case strings.HasPrefix(value, receiptPrefix):
			out.receipts = append(out.receipts, id)
		case strings.HasPrefix(value, invoicePrefix):
			out.invoices = append(out.invoices, id)
		case strings.HasPrefix(value, paymentPrefix):
			out.payments = append(out.payments, id)
		}
	}

	return out
}

type paymentWindow struct {
	AmountMinor int64
	PaymentDate int64
}

func (r *repository) ReceiptsTouching(
	ctx context.Context,
	req *bankreceiptports.ReceiptsTouchingRequest,
) ([]*bankreceipt.BankReceipt, error) {
	named := split(req.SubjectIDs)
	if len(named.receipts)+len(named.invoices)+len(named.payments) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bankreceipt.BankReceipt, error) {
		db := r.db.Keel(ctx)
		listing, numbers, windows, err := r.itemFacts(ctx, db, named)
		if err != nil {
			return nil, err
		}
		br := bankreceiptdb.BankReceipts
		reached := make([]keel.Pred, 0, len(numbers)*2+len(windows))
		for _, number := range numbers {
			reached = append(reached,
				br.ReferenceNumber.ContainsFold(number), br.Memo.ContainsFold(number))
		}
		for _, window := range windows {
			reached = append(reached, keel.And(
				br.AmountMinor.Between(
					window.AmountMinor-bankreceipt.NearAmountMinor,
					window.AmountMinor+bankreceipt.NearAmountMinor,
				),
				br.ReceiptDate.Between(
					window.PaymentDate-bankreceipt.PaymentWindowSeconds,
					window.PaymentDate+bankreceipt.PaymentWindowSeconds,
				),
			))
		}
		direct := slices.Concat(named.receipts, listing)
		var named0, matched []*bankreceipt.BankReceipt
		batch := []keel.BatchItem{
			keel.ListInto(br.Where(br.ID.In(direct...)).Limit(maxTouched), &named0),
		}
		if len(reached) > 0 {
			batch = append(batch, keel.ListInto(
				br.Where(unapplied(br), keel.Or(reached...)).
					Order(br.ReceiptDate.Desc()).
					Limit(maxTouched),
				&matched,
			))
		}
		if err = keel.Batch(ctx, db, batch...); err != nil {
			return nil, fmt.Errorf("read bank receipts a change touches: %w", err)
		}

		return dedupe(named0, matched), nil
	})
}

func (r *repository) itemFacts(
	ctx context.Context,
	db *keel.DB,
	named subjects,
) (listing []pulid.ID, numbers []string, windows []paymentWindow, err error) {
	items := slices.Concat(named.invoices, named.payments)
	if len(items) == 0 {
		return nil, nil, nil, nil
	}
	c := bankreceiptdb.Candidates
	inv := invoicedb.Invoices
	cp := customerpaymentdb.Payments
	batch := []keel.BatchItem{
		keel.IntoSlice(c.Where(keel.Or(
			c.InvoiceID.In(items...),
			c.CustomerPaymentID.In(items...),
		)).Group(c.BankReceiptID).Pick(c.BankReceiptID), &listing),
	}
	if len(named.invoices) > 0 {
		batch = append(batch, keel.IntoSlice(
			inv.Where(inv.ID.In(named.invoices...), receivablesql.Open(inv)).Pick(inv.Number),
			&numbers,
		))
	}
	if len(named.payments) > 0 {
		batch = append(batch, keel.IntoSlice(
			cp.Where(cp.ID.In(named.payments...), cp.Status.Eq(customerpayment.StatusPosted)).
				Pick(cp.AmountMinor, cp.PaymentDate),
			&windows,
		))
	}
	if err = keel.Batch(ctx, db, batch...); err != nil {
		return nil, nil, nil, fmt.Errorf("read changed open items: %w", err)
	}
	numbers = slices.DeleteFunc(numbers, func(number string) bool {
		return strings.TrimSpace(number) == ""
	})

	return listing, numbers, windows, nil
}

func dedupe(groups ...[]*bankreceipt.BankReceipt) []*bankreceipt.BankReceipt {
	seen := make(map[pulid.ID]bool)
	out := make([]*bankreceipt.BankReceipt, 0)
	for _, group := range groups {
		for _, receipt := range group {
			if receipt == nil || seen[receipt.ID] {
				continue
			}
			seen[receipt.ID] = true
			out = append(out, receipt)
		}
	}

	return out
}

func (r *repository) PaymentEvidence(
	ctx context.Context,
	receipts []*bankreceipt.BankReceipt,
) (map[pulid.ID][]bankreceipt.PaymentEvidence, error) {
	out := make(map[pulid.ID][]bankreceipt.PaymentEvidence, len(receipts))
	if len(receipts) == 0 {
		return out, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]bankreceipt.PaymentEvidence, error) {
			cp := customerpaymentdb.Payments
			found := make([][]*customerpayment.Payment, len(receipts))
			batch := make([]keel.BatchItem, 0, len(receipts))
			for idx, receipt := range receipts {
				batch = append(batch, keel.ListInto(cp.Where(
					cp.Status.Eq(customerpayment.StatusPosted),
					receivablesql.UnmatchedPayment(cp),
					keel.When(receipt.ReferenceNumber != "", keel.Or(
						cp.ReferenceNumber.Eq(receipt.ReferenceNumber),
						cp.ReferenceNumber.ContainsFold(receipt.ReferenceNumber),
					)),
					cp.AmountMinor.Between(
						receipt.AmountMinor-bankreceipt.NearAmountMinor,
						receipt.AmountMinor+bankreceipt.NearAmountMinor,
					),
					cp.PaymentDate.Between(
						receipt.ReceiptDate-bankreceipt.PaymentWindowSeconds,
						receipt.ReceiptDate+bankreceipt.PaymentWindowSeconds,
					),
				).Order(cp.CreatedAt.Desc()).Limit(bankreceipt.MaxCandidates), &found[idx]))
			}
			if err := keel.Batch(ctx, r.db.Keel(ctx), batch...); err != nil {
				return nil, fmt.Errorf("read payments bank receipts resemble: %w", err)
			}
			for idx, receipt := range receipts {
				evidence := make([]bankreceipt.PaymentEvidence, 0, len(found[idx]))
				for _, payment := range found[idx] {
					evidence = append(evidence, bankreceipt.PaymentEvidence{
						ID:          payment.ID,
						CustomerID:  payment.CustomerID,
						Reference:   payment.ReferenceNumber,
						AmountMinor: payment.AmountMinor,
						PaymentDate: payment.PaymentDate,
					})
				}
				out[receipt.ID] = evidence
			}

			return out, nil
		},
	)
}

type invoiceRow struct {
	ID         pulid.ID
	CustomerID pulid.ID
	Number     string
	OpenMinor  int64
}

func (r *repository) OpenInvoicesByNumber(
	ctx context.Context,
	numbers []string,
) ([]bankreceipt.InvoiceEvidence, error) {
	if len(numbers) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]bankreceipt.InvoiceEvidence, error) {
		inv := invoicedb.Invoices
		rows, err := keel.Into[invoiceRow](ctx, r.db.Keel(ctx), inv.
			Pick(inv.ID, inv.CustomerID, inv.Number, receivablesql.OpenMinor(inv)).
			Where(inv.Number.In(numbers...), receivablesql.Open(inv)).
			Limit(len(numbers)))
		if err != nil {
			return nil, fmt.Errorf("read open invoices by number: %w", err)
		}
		out := make([]bankreceipt.InvoiceEvidence, 0, len(rows))
		for _, row := range rows {
			out = append(out, bankreceipt.InvoiceEvidence(row))
		}

		return out, nil
	})
}

func (r *repository) Replace(
	ctx context.Context,
	req *bankreceiptports.ReplaceCandidatesRequest,
) error {
	if len(req.ReceiptIDs) == 0 {
		return nil
	}
	ids := slices.Clone(req.ReceiptIDs)
	slices.SortFunc(ids, func(a, b pulid.ID) int { return strings.Compare(a.String(), b.String()) })
	ids = slices.Compact(ids)

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		for _, id := range ids {
			if err := keel.LockKey(ctx, db, lockPrefix+id.String()); err != nil {
				return fmt.Errorf("lock bank receipt candidates: %w", err)
			}
		}
		c := bankreceiptdb.Candidates
		if _, err := c.Where(c.BankReceiptID.In(ids...)).Delete(ctx, db); err != nil {
			return fmt.Errorf("clear bank receipt candidates: %w", err)
		}
		if len(req.Candidates) == 0 {
			return nil
		}
		now := timeutils.NowUnix()
		for _, candidate := range req.Candidates {
			candidate.ComputedAt = now
		}
		if err := c.CreateMany(ctx, db, req.Candidates); err != nil {
			return fmt.Errorf("write bank receipt candidates: %w", err)
		}

		return nil
	})
}

type viewRow struct {
	Rank              int
	Kind              bankreceipt.CandidateKind
	CustomerPaymentID pulid.ID
	InvoiceID         pulid.ID
	CustomerID        pulid.ID
	Score             int
	Evidence          []string
	Standing          bankreceipt.Standing
	OpenMinor         int64
	DifferenceMinor   int64
	ComputedAt        int64
	Reference         string
	InvoiceNumber     string
	CustomerName      string
}

func (r *repository) ListForReceipt(
	ctx context.Context,
	req *bankreceiptports.ListCandidatesRequest,
) ([]*bankreceiptports.CandidateView, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*bankreceiptports.CandidateView, error) {
			c := bankreceiptdb.Candidates
			cp := customerpaymentdb.Payments
			inv := invoicedb.Invoices
			cus := customerdb.Customers
			rows, err := keel.Into[viewRow](ctx, r.db.Keel(ctx), c.
				LeftJoin(cp, cp.ID.EqCol(c.CustomerPaymentID)).
				LeftJoin(inv, inv.ID.EqCol(c.InvoiceID)).
				LeftJoin(cus, cus.ID.EqCol(c.CustomerID)).
				Pick(
					c.Rank, c.Kind, c.CustomerPaymentID, c.InvoiceID, c.CustomerID, c.Score,
					c.Evidence, c.Standing, c.OpenMinor, c.DifferenceMinor, c.ComputedAt,
					keel.CastText(cp.ReferenceNumber).Coalesce(""),
					keel.CastText(inv.Number).Coalesce(""),
					keel.CastText(cus.Name).Coalesce(""),
				).
				Where(
					c.BankReceiptID.Eq(req.ReceiptID),
					keel.When(len(req.Kinds) > 0, c.Kind.In(req.Kinds...)),
					receivablesql.ShownCandidate(c, cp, inv),
				).
				Order(c.Rank.Asc()).
				Limit(bankreceipt.MaxCandidates))
			if err != nil {
				return nil, fmt.Errorf("list bank receipt candidates: %w", err)
			}
			out := make([]*bankreceiptports.CandidateView, 0, len(rows))
			for idx := range rows {
				row := &rows[idx]
				out = append(out, &bankreceiptports.CandidateView{
					Candidate: &bankreceipt.Candidate{
						OrganizationID:    req.TenantInfo.OrgID,
						BusinessUnitID:    req.TenantInfo.BuID,
						BankReceiptID:     req.ReceiptID,
						Rank:              row.Rank,
						Kind:              row.Kind,
						CustomerPaymentID: row.CustomerPaymentID,
						InvoiceID:         row.InvoiceID,
						CustomerID:        row.CustomerID,
						Score:             row.Score,
						Evidence:          row.Evidence,
						Standing:          row.Standing,
						OpenMinor:         row.OpenMinor,
						DifferenceMinor:   row.DifferenceMinor,
						ComputedAt:        row.ComputedAt,
					},
					Reference:     row.Reference,
					InvoiceNumber: row.InvoiceNumber,
					CustomerName:  row.CustomerName,
				})
			}

			return out, nil
		},
	)
}
