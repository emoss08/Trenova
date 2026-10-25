package accountingsyncrepository

import (
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/pkg/keelkinds"
)

type settlementTable struct {
	query    candidateQueryFunc
	id       keelkinds.PulidIDCol
	number   keel.StringCol
	status   keel.StringCol
	net      keel.Int64Col
	postedAt keel.Int64Col
	paidAt   keel.Int64Col
	voidedAt keel.Int64Col
	posted   string
	paid     string
	voided   string
	scope    []keel.Pred
}

func carrierSettlementTable() *settlementTable {
	cs := carriersettlementdb.CarrierSettlements
	return &settlementTable{
		query:    candidateQuery[carriersettlement.CarrierSettlement](cs),
		id:       cs.ID,
		number:   cs.SettlementNumber,
		status:   cs.Status.StringCol,
		net:      cs.NetPayableMinor,
		postedAt: cs.PostedAt,
		paidAt:   cs.PaidAt,
		voidedAt: cs.VoidedAt,
		posted:   carriersettlement.StatusPosted.String(),
		paid:     carriersettlement.StatusPaid.String(),
		voided:   carriersettlement.StatusVoided.String(),
	}
}

func driverSettlementTable() *settlementTable {
	ds := driversettlementdb.Settlements
	return &settlementTable{
		query:    candidateQuery[driversettlement.Settlement](ds),
		id:       ds.ID,
		number:   ds.SettlementNumber,
		status:   ds.Status.StringCol,
		net:      ds.NetPayMinor,
		postedAt: ds.PostedAt,
		paidAt:   ds.PaidAt,
		voidedAt: ds.VoidedAt,
		posted:   driversettlement.StatusPosted.String(),
		paid:     driversettlement.StatusPaid.String(),
		voided:   driversettlement.StatusVoided.String(),
		scope:    []keel.Pred{ds.Classification.Eq(driverpay.PayeeClassificationOwnerOperator)},
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
		query:  table.query,
		id:     table.id,
		number: &number,
		date:   table.postedAt,
	}

	switch {
	case objectType.IsBill() && operation == accountingsync.SyncOperationCreate:
		source.at = table.postedAt
		source.filter = func(req *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			preds := append([]keel.Pred{
				table.postedAt.IsNotNull(),
				table.status.In(table.posted, table.paid, table.voided),
			}, table.scope...)
			if req.UndoneBefore == nil {
				return preds
			}
			return append(preds, keel.Or(
				table.status.Ne(table.voided),
				table.voidedAt.Gte(*req.UndoneBefore),
			))
		}
	case objectType.IsBill() && operation == accountingsync.SyncOperationVoid:
		source.at = table.voidedAt
		source.filter = func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return append([]keel.Pred{
				table.status.Eq(table.voided),
				table.postedAt.IsNotNull(),
				table.voidedAt.IsNotNull(),
			}, table.scope...)
		}
	case objectType.IsBillPayment() && operation == accountingsync.SyncOperationCreate:
		source.at = table.paidAt
		source.filter = func(_ *accountingsyncports.ListAccountingSyncCandidatesRequest) []keel.Pred {
			return append([]keel.Pred{
				table.status.Eq(table.paid),
				table.postedAt.IsNotNull(),
				table.paidAt.IsNotNull(),
				table.net.Ne(0),
			}, table.scope...)
		}
	default:
		return nil, fmt.Errorf("no settlements back %s %s records", objectType, operation)
	}
	return source, nil
}
