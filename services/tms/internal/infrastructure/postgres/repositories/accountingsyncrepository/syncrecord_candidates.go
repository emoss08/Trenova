package accountingsyncrepository

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type candidateRow struct {
	ObjectID     pulid.ID `bun:"object_id"`
	ObjectNumber string   `bun:"object_number"`
	DocumentDate int64    `bun:"document_date"`
	PostedAt     int64    `bun:"posted_at"`
}

type candidateSource struct {
	model  any
	org    buncolgen.Column
	bu     buncolgen.Column
	id     buncolgen.Column
	number *buncolgen.Column
	date   buncolgen.Column
	at     buncolgen.Column
	filter func(q *bun.SelectQuery, req *repositories.ListAccountingSyncCandidatesRequest) *bun.SelectQuery
}

func postgresTxOptions() ports.TxOptions {
	return ports.TxOptions{}
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
	default:
		return nil, fmt.Errorf(
			"no posted documents back %s %s records",
			objectType,
			operation,
		)
	}
}

func salesDocumentSource(billType billingqueue.BillType) *candidateSource {
	cols := buncolgen.InvoiceColumns
	return &candidateSource{
		model:  (*invoice.Invoice)(nil),
		org:    cols.OrganizationID,
		bu:     cols.BusinessUnitID,
		id:     cols.ID,
		number: &cols.Number,
		date:   cols.InvoiceDate,
		at:     cols.PostedAt,
		filter: func(
			q *bun.SelectQuery,
			_ *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			return q.
				Where(cols.BillType.Eq(), billType).
				Where(cols.PostedAt.IsNotNull()).
				Where(cols.Status.In(), bun.List([]invoice.Status{
					invoice.StatusPosted,
					invoice.StatusVoided,
				}))
		},
	}
}

func paymentSource(operation accountingsync.SyncOperation) (*candidateSource, error) {
	cols := buncolgen.PaymentColumns
	source := &candidateSource{
		model:  (*customerpayment.Payment)(nil),
		org:    cols.OrganizationID,
		bu:     cols.BusinessUnitID,
		id:     cols.ID,
		number: &cols.ReferenceNumber,
		date:   cols.AccountingDate,
	}
	switch operation {
	case accountingsync.SyncOperationCreate:
		source.at = cols.CreatedAt
		source.filter = func(
			q *bun.SelectQuery,
			req *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			if req.UndoneBefore == nil {
				return q
			}
			return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(cols.Status.NotEq(), customerpayment.StatusReversed).
					WhereOr(cols.ReversedAt.Gte(), *req.UndoneBefore)
			})
		}
	case accountingsync.SyncOperationVoid:
		source.at = cols.ReversedAt
		source.filter = func(
			q *bun.SelectQuery,
			_ *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			return q.
				Where(cols.Status.Eq(), customerpayment.StatusReversed).
				Where(cols.ReversedAt.IsNotNull())
		}
	case accountingsync.SyncOperationUpdate:
		return nil, fmt.Errorf("no posted payments back %s records", operation)
	default:
		return nil, fmt.Errorf("no posted payments back %s records", operation)
	}
	return source, nil
}

func creditApplicationSource(operation accountingsync.SyncOperation) (*candidateSource, error) {
	cols := buncolgen.CreditMemoApplicationColumns
	source := &candidateSource{
		model: (*customerpayment.CreditMemoApplication)(nil),
		org:   cols.OrganizationID,
		bu:    cols.BusinessUnitID,
		id:    cols.ID,
		date:  cols.AccountingDate,
	}
	switch operation {
	case accountingsync.SyncOperationCreate:
		source.at = cols.CreatedAt
		source.filter = func(
			q *bun.SelectQuery,
			req *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			if req.UndoneBefore == nil {
				return q
			}
			return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(cols.Status.NotEq(), customerpayment.CreditApplicationStatusUnapplied).
					WhereOr(cols.UnappliedAt.Gte(), *req.UndoneBefore)
			})
		}
	case accountingsync.SyncOperationVoid:
		source.at = cols.UnappliedAt
		source.filter = func(
			q *bun.SelectQuery,
			_ *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			return q.
				Where(cols.Status.Eq(), customerpayment.CreditApplicationStatusUnapplied).
				Where(cols.UnappliedAt.IsNotNull())
		}
	case accountingsync.SyncOperationUpdate:
		return nil, fmt.Errorf("no credit applications back %s records", operation)
	default:
		return nil, fmt.Errorf("no credit applications back %s records", operation)
	}
	return source, nil
}

func (s *candidateSource) query(
	dba bun.IDB,
	req *repositories.ListAccountingSyncCandidatesRequest,
	limit int,
) *bun.SelectQuery {
	records := buncolgen.AccountingSyncRecordColumns
	queued := dba.NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		ColumnExpr("1").
		Where(records.OrganizationID.EqColumn(s.org)).
		Where(records.BusinessUnitID.EqColumn(s.bu)).
		Where(records.ObjectID.EqColumn(s.id)).
		Where(records.ConnectionID.Eq(), req.ConnectionID).
		Where(records.ObjectType.Eq(), req.ObjectType).
		Where(records.Operation.Eq(), req.Operation)

	numberExpr := "''"
	if s.number != nil {
		numberExpr = s.number.Expr("COALESCE({}, '')")
	}

	q := dba.NewSelect().
		Model(s.model).
		ColumnExpr(s.id.As("object_id")).
		ColumnExpr(numberExpr+" AS object_number").
		ColumnExpr(s.date.As("document_date")).
		ColumnExpr(s.at.As("posted_at")).
		Where(s.org.Eq(), req.TenantInfo.OrgID).
		Where(s.bu.Eq(), req.TenantInfo.BuID).
		Where(s.date.Gte(), req.DatedFrom).
		Where("NOT EXISTS (?)", queued)
	q = s.filter(q, req)
	if req.PostedFrom != nil {
		q = q.Where(s.at.Gte(), *req.PostedFrom)
	}
	if req.PostedBefore != nil {
		q = q.Where(s.at.Lt(), *req.PostedBefore)
	}
	if req.AfterAt > 0 || !req.AfterID.IsNil() {
		q = q.Where(buncolgen.Expr("({0}, {1}) > (?, ?)", s.at, s.id), req.AfterAt, req.AfterID)
	}

	return q.
		Order(s.at.OrderAsc(), s.id.OrderAsc()).
		Limit(limit)
}
