//go:build integration

package carriersettlementrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *holdFixture) moveOf(t *testing.T, assignmentID pulid.ID) pulid.ID {
	t.Helper()
	moveID, err := keel.QueryValue[pulid.ID](
		f.ctx,
		f.db,
		"SELECT shipment_move_id FROM carrier_assignments WHERE id = $1",
		assignmentID.String(),
	)
	require.NoError(t, err)
	return moveID
}

func matchIDs(matches []*carriersettlement.InvoiceMatch) []pulid.ID {
	ids := make([]pulid.ID, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match.ID)
	}
	return ids
}

func TestListLiveByMoveIDsGroupsEachMovesLiveInvoicesThroughItsAssignment_Integration(t *testing.T) {
	f := setupHoldFixture(t)

	invoicedLoad := f.assignment(t)
	rejectedLoad := f.assignment(t)
	quietLoad := f.assignment(t)
	invoicedMove := f.moveOf(t, invoicedLoad)
	rejectedMove := f.moveOf(t, rejectedLoad)
	quietMove := f.moveOf(t, quietLoad)

	first, err := f.match(t, invoicedLoad, "INV-1", carriersettlement.InvoiceMatchStatusVariance)
	require.NoError(t, err)
	second, err := f.match(t, invoicedLoad, "INV-2", carriersettlement.InvoiceMatchStatusMatched)
	require.NoError(t, err)
	_, err = f.match(t, rejectedLoad, "INV-3", carriersettlement.InvoiceMatchStatusRejected)
	require.NoError(t, err)

	grouped, err := f.matches.ListLiveByMoveIDs(
		f.ctx,
		carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest{
			TenantInfo: f.tenant,
			MoveIDs:    []pulid.ID{invoicedMove, rejectedMove, quietMove},
		},
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{first.ID, second.ID}, matchIDs(grouped[invoicedMove]))
	for _, match := range grouped[invoicedMove] {
		assert.Equal(t, invoicedLoad, match.CarrierAssignmentID)
		assert.NotEmpty(t, match.InvoiceNumber)
	}
	assert.Empty(t, grouped[rejectedMove])
	assert.Empty(t, grouped[quietMove])

	otherTenant := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	elsewhere, err := f.matches.ListLiveByMoveIDs(
		dbscope.WithTenant(f.ctx, otherTenant.DBTenant()),
		carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest{
			TenantInfo: otherTenant,
			MoveIDs:    []pulid.ID{invoicedMove},
		},
	)
	require.NoError(t, err)
	assert.Empty(t, elsewhere)

	none, err := f.matches.ListLiveByMoveIDs(
		f.ctx,
		carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest{TenantInfo: f.tenant},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
