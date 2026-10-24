package customerledgerrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestDump(t *testing.T) {
	cle := customerledgerdb.CustomerLedgerEntries
	sql, args := keeltest.Render(t, cle.Pick(cle.CustomerID, keel.Raw("COALESCE(?, 0)", keel.Sum(cle.AmountMinor))).
		Where(keel.Not(nonRecourseFactoredEntry(cle))).Group(cle.CustomerID))
	t.Log(sql)
	t.Log(args)
}
