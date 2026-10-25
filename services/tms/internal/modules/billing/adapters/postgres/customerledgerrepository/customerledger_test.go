package customerledgerrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestBalancesLeaveOutNonRecourseFactoredEntries(t *testing.T) {
	cle := customerledgerdb.CustomerLedgerEntries
	keeltest.Snapshot(t, cle.Pick(cle.CustomerID, keel.DynOf(keel.Sum(cle.AmountMinor)).Coalesce(0)).
		Where(keel.Not(nonRecourseFactoredEntry(cle))).Group(cle.CustomerID))
}
