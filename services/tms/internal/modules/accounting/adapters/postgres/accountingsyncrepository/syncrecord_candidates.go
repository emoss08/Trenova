package accountingsyncrepository

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/pulid"
)

type candidateRow struct {
	ObjectID     pulid.ID
	ObjectNumber string
	DocumentDate int64
	PostedAt     int64
}

type candidateQueryFunc func(
	picks []keel.Expr,
	preds []keel.Pred,
	orders []keel.Order,
	limit int,
) keel.Projection

type candidateSource struct {
	query  candidateQueryFunc
	id     keelkinds.PulidIDCol
	number *keel.StringCol
	date   keel.Int64Col
	at     keel.Int64Col
	filter func(req *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred
	queued func(
		rec *accountingsyncdb.AccountingSyncRecordsTable,
		req *accountingsyncports.ListAccountingSyncCandidatesRequest,
	) []keel.Pred
}

type candidatePicker[T any] interface {
	Pick(es ...keel.Expr) *keel.Query[T]
}

func candidateQuery[T any](h candidatePicker[T]) candidateQueryFunc {
	return func(picks []keel.Expr, preds []keel.Pred, orders []keel.Order, limit int) keel.Projection {
		return h.Pick(picks...).Where(preds...).Order(orders...).Limit(limit)
	}
}

func sortClaimed(records []*accountingsync.AccountingSyncRecord) {
	slices.SortStableFunc(records, func(a, b *accountingsync.AccountingSyncRecord) int {
		return cmp.Or(
			cmp.Compare(a.ObjectType.DispatchRank(), b.ObjectType.DispatchRank()),
			cmp.Compare(a.QueuedAt, b.QueuedAt),
			cmp.Compare(a.ID.String(), b.ID.String()),
		)
	})
}

func candidateSourceFor(
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
) (*candidateSource, error) {
	switch {
	case objectType.IsSalesDocument() && operation == accountingsync.SyncOperationCreate:
		billType, _ := objectType.BillType()
		return salesDocumentSource(billType), nil
	case objectType == accountingsync.SyncObjectCustomerPayment:
		return paymentSource(operation)
	case objectType == accountingsync.SyncObjectCreditApplication:
		return creditApplicationSource(operation)
	case objectType.IsBill(), objectType.IsBillPayment():
		return settlementSource(objectType, operation)
	case objectType == accountingsync.SyncObjectJournalEntry &&
		operation == accountingsync.SyncOperationCreate:
		return journalSource(nil), nil
	case objectType == accountingsync.SyncObjectJournalSummary &&
		operation == accountingsync.SyncOperationCreate:
		return journalSource(journalDayQueued), nil
	default:
		return nil, fmt.Errorf(
			"no posted documents back %s %s records",
			objectType,
			operation,
		)
	}
}

func salesDocumentSource(billType billingqueue.BillType) *candidateSource {
	inv := invoicedb.Invoices
	return &candidateSource{
		query:  candidateQuery[invoice.Invoice](inv),
		id:     inv.ID,
		number: &inv.Number,
		date:   inv.InvoiceDate,
		at:     inv.PostedAt,
		filter: func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return []keel.Pred{
				inv.BillType.Eq(billType),
				inv.PostedAt.IsNotNull(),
				inv.Status.In(invoice.StatusPosted, invoice.StatusVoided),
			}
		},
	}
}

func paymentSource(operation accountingsync.SyncOperation) (*candidateSource, error) {
	cp := customerpaymentdb.Payments
	source := &candidateSource{
		query:  candidateQuery[customerpayment.Payment](cp),
		id:     cp.ID,
		number: &cp.ReferenceNumber,
		date:   cp.AccountingDate,
	}
	switch operation {
	case accountingsync.SyncOperationCreate:
		source.at = cp.CreatedAt
		source.filter = func(req *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			if req.UndoneBefore == nil {
				return nil
			}
			return []keel.Pred{keel.Or(
				cp.Status.Ne(customerpayment.StatusReversed),
				cp.ReversedAt.Gte(*req.UndoneBefore),
			)}
		}
	case accountingsync.SyncOperationVoid:
		source.at = cp.ReversedAt
		source.filter = func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return []keel.Pred{
				cp.Status.Eq(customerpayment.StatusReversed),
				cp.ReversedAt.IsNotNull(),
			}
		}
	case accountingsync.SyncOperationUpdate:
		return nil, fmt.Errorf("no posted payments back %s records", operation)
	default:
		return nil, fmt.Errorf("no posted payments back %s records", operation)
	}
	return source, nil
}

func creditApplicationSource(operation accountingsync.SyncOperation) (*candidateSource, error) {
	cma := customerpaymentdb.CreditMemoApplications
	source := &candidateSource{
		query: candidateQuery[customerpayment.CreditMemoApplication](cma),
		id:    cma.ID,
		date:  cma.AccountingDate,
	}
	switch operation {
	case accountingsync.SyncOperationCreate:
		source.at = cma.CreatedAt
		source.filter = func(req *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			if req.UndoneBefore == nil {
				return nil
			}
			return []keel.Pred{keel.Or(
				cma.Status.Ne(customerpayment.CreditApplicationStatusUnapplied),
				cma.UnappliedAt.Gte(*req.UndoneBefore),
			)}
		}
	case accountingsync.SyncOperationVoid:
		source.at = cma.UnappliedAt
		source.filter = func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return []keel.Pred{
				cma.Status.Eq(customerpayment.CreditApplicationStatusUnapplied),
				cma.UnappliedAt.IsNotNull(),
			}
		}
	case accountingsync.SyncOperationUpdate:
		return nil, fmt.Errorf("no credit applications back %s records", operation)
	default:
		return nil, fmt.Errorf("no credit applications back %s records", operation)
	}
	return source, nil
}

func (s *candidateSource) build(
	req *accountingsyncports.ListAccountingSyncCandidatesRequest,
	limit int,
) keel.Projection {
	rec := accountingsyncdb.AccountingSyncRecords
	queued := []keel.Pred{
		rec.ConnectionID.Eq(req.ConnectionID),
		rec.ObjectType.Eq(req.ObjectType),
	}
	if s.queued != nil {
		queued = append(queued, s.queued(rec, req)...)
	} else {
		queued = append(queued, rec.ObjectID.EqCol(s.id), rec.Operation.Eq(req.Operation))
	}

	number := keel.Expr(keel.DynLit("").Cast(keel.TypeString))
	if s.number != nil {
		number = keel.DynOf(*s.number).Coalesce("")
	}

	preds := []keel.Pred{
		s.date.Gte(req.DatedFrom),
		keel.NotExists(rec.Where(queued...)),
	}
	preds = append(preds, s.filter(req)...)
	if req.PostedFrom != nil {
		preds = append(preds, s.at.Gte(*req.PostedFrom))
	}
	if req.PostedBefore != nil {
		preds = append(preds, s.at.Lt(*req.PostedBefore))
	}
	if req.AfterAt > 0 || !req.AfterID.IsNil() {
		preds = append(
			preds,
			keel.DynRow(s.at, s.id).Gt(keel.DynRow(req.AfterAt, req.AfterID.String())),
		)
	}

	return s.query(
		[]keel.Expr{s.id, number, s.date, s.at},
		preds,
		[]keel.Order{s.at.Asc(), s.id.Asc()},
		limit,
	)
}

func journalSource(
	queued func(
		*accountingsyncdb.AccountingSyncRecordsTable,
		*accountingsyncports.ListAccountingSyncCandidatesRequest,
	) []keel.Pred,
) *candidateSource {
	je := journalentrydb.JournalEntries
	return &candidateSource{
		query:  candidateQuery[journalentry.JournalEntry](je),
		id:     je.ID,
		number: &je.EntryNumber,
		date:   je.AccountingDate,
		at:     je.PostedAt,
		filter: func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return []keel.Pred{je.IsPosted.IsTrue(), withoutEntryTypes(je, unsentEntryTypes())}
		},
		queued: queued,
	}
}

func journalDayQueued(
	rec *accountingsyncdb.AccountingSyncRecordsTable,
	req *accountingsyncports.ListAccountingSyncCandidatesRequest,
) []keel.Pred {
	je := journalentrydb.JournalEntries
	timezone := req.Timezone
	if timezone == "" {
		timezone = time.UTC.String()
	}
	return []keel.Pred{
		keel.DynOf(rec.ObjectID).Eq(keel.DynLit(accountingsync.JournalDayPrefix).Cast(keel.TypeString).Concat(keel.DynFunc(
			"to_char",
			keel.DynFunc("timezone", keel.DynLit(timezone).Cast(keel.TypeString), keel.DynFunc("to_timestamp", je.AccountingDate)),
			keel.DynLit(accountingsync.JournalDaySQLLayout).Cast(keel.TypeString),
		))),
		keel.DynOf(rec.QueuedAt).Gte(je.PostedAt),
	}
}
