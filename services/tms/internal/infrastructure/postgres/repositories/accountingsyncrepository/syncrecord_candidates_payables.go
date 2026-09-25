package accountingsyncrepository

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
)

type settlementTable struct {
	model    any
	org      buncolgen.Column
	bu       buncolgen.Column
	id       buncolgen.Column
	number   buncolgen.Column
	status   buncolgen.Column
	net      buncolgen.Column
	postedAt buncolgen.Column
	paidAt   buncolgen.Column
	voidedAt buncolgen.Column
	statuses map[string]string
	scope    func(q *bun.SelectQuery) *bun.SelectQuery
}

func carrierSettlementTable() *settlementTable {
	cols := buncolgen.CarrierSettlementColumns
	return &settlementTable{
		model:    (*carriersettlement.CarrierSettlement)(nil),
		org:      cols.OrganizationID,
		bu:       cols.BusinessUnitID,
		id:       cols.ID,
		number:   cols.SettlementNumber,
		status:   cols.Status,
		net:      cols.NetPayableMinor,
		postedAt: cols.PostedAt,
		paidAt:   cols.PaidAt,
		voidedAt: cols.VoidedAt,
		statuses: map[string]string{
			"posted": carriersettlement.StatusPosted.String(),
			"paid":   carriersettlement.StatusPaid.String(),
			"voided": carriersettlement.StatusVoided.String(),
		},
		scope: func(q *bun.SelectQuery) *bun.SelectQuery { return q },
	}
}

func driverSettlementTable() *settlementTable {
	cols := buncolgen.SettlementColumns
	return &settlementTable{
		model:    (*driversettlement.Settlement)(nil),
		org:      cols.OrganizationID,
		bu:       cols.BusinessUnitID,
		id:       cols.ID,
		number:   cols.SettlementNumber,
		status:   cols.Status,
		net:      cols.NetPayMinor,
		postedAt: cols.PostedAt,
		paidAt:   cols.PaidAt,
		voidedAt: cols.VoidedAt,
		statuses: map[string]string{
			"posted": driversettlement.StatusPosted.String(),
			"paid":   driversettlement.StatusPaid.String(),
			"voided": driversettlement.StatusVoided.String(),
		},
		scope: func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.Classification.Eq(), driverpay.PayeeClassificationOwnerOperator)
		},
	}
}

func settlementSource(
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
) (*candidateSource, error) {
	table := carrierSettlementTable()
	if objectType.IsDriverSettlement() {
		table = driverSettlementTable()
	}
	number := table.number
	source := &candidateSource{
		model:  table.model,
		org:    table.org,
		bu:     table.bu,
		id:     table.id,
		number: &number,
		date:   table.postedAt,
	}

	switch {
	case objectType.IsBill() && operation == accountingsync.SyncOperationCreate:
		source.at = table.postedAt
		source.filter = func(
			q *bun.SelectQuery,
			req *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			q = table.scope(q).
				Where(table.postedAt.IsNotNull()).
				Where(table.status.In(), bun.List([]string{
					table.statuses["posted"],
					table.statuses["paid"],
					table.statuses["voided"],
				}))
			if req.UndoneBefore == nil {
				return q
			}
			return q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(table.status.NotEq(), table.statuses["voided"]).
					WhereOr(table.voidedAt.Gte(), *req.UndoneBefore)
			})
		}
	case objectType.IsBill() && operation == accountingsync.SyncOperationVoid:
		source.at = table.voidedAt
		source.filter = func(
			q *bun.SelectQuery,
			_ *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			return table.scope(q).
				Where(table.status.Eq(), table.statuses["voided"]).
				Where(table.postedAt.IsNotNull()).
				Where(table.voidedAt.IsNotNull())
		}
	case objectType.IsBillPayment() && operation == accountingsync.SyncOperationCreate:
		source.at = table.paidAt
		source.filter = func(
			q *bun.SelectQuery,
			_ *repositories.ListAccountingSyncCandidatesRequest,
		) *bun.SelectQuery {
			return table.scope(q).
				Where(table.status.Eq(), table.statuses["paid"]).
				Where(table.postedAt.IsNotNull()).
				Where(table.paidAt.IsNotNull()).
				Where(table.net.Ne(), 0)
		}
	default:
		return nil, fmt.Errorf("no settlements back %s %s records", objectType, operation)
	}
	return source, nil
}
