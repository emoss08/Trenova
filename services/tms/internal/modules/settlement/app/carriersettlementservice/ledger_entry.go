package carriersettlementservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/shared/pulid"
)

type ledgerEntryParams struct {
	EntryType       carriersettlement.LedgerEntryType
	SourceEvent     accountingcontrol.JournalSourceEventType
	JournalBatchID  *pulid.ID
	AmountMinor     int64
	TransactionDate int64
	Actor           *permissionports.RequestActor
}
