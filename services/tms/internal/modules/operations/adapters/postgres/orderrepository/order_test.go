package orderrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestEmptyOrder_ExcludesEveryReferencedOrder(t *testing.T) {
	t.Parallel()

	repo := &repository{references: crossref.OrderReferences()}
	keeltest.Snapshot(t, repo.emptyOrder(pulid.ID("ord_01")))
}
