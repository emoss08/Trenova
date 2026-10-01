package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repeatedIDs(n int) []string {
	id := pulid.MustNew("acctsr_").String()
	out := make([]string, n)
	for idx := range out {
		out[idx] = id
	}
	return out
}

func TestAccountingSyncRecordIDsCapsABatch(t *testing.T) {
	t.Parallel()

	ids, err := accountingSyncRecordIDs(repeatedIDs(maxAccountingSyncActionIDs))
	require.NoError(t, err)
	assert.Len(t, ids, maxAccountingSyncActionIDs)

	_, err = accountingSyncRecordIDs(repeatedIDs(maxAccountingSyncActionIDs + 1))
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestAccountingSyncRecordIDsRejectsAMalformedID(t *testing.T) {
	t.Parallel()

	_, err := accountingSyncRecordIDs([]string{"not-an-id"})
	require.Error(t, err)
}

func TestAccountingSyncRecordIDsAllowsNone(t *testing.T) {
	t.Parallel()

	ids, err := accountingSyncRecordIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestAccountingSyncObjectIDsRefusesMoreThanTheSchemaAllows(t *testing.T) {
	t.Parallel()

	_, err := accountingSyncObjectIDs(repeatedIDs(maxAccountingSyncStateIDs + 1))
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	ids, err := accountingSyncObjectIDs(repeatedIDs(maxAccountingSyncStateIDs))
	require.NoError(t, err)
	assert.Len(t, ids, maxAccountingSyncStateIDs)
}

func TestOrderedAccountingSyncStatesFollowsTheRequestAndDropsUnknowns(t *testing.T) {
	t.Parallel()

	first, second, never := pulid.MustNew("inv_"), pulid.MustNew("inv_"), pulid.MustNew("inv_")
	states := map[pulid.ID]*services.AccountingSyncObjectState{
		first:  {ObjectID: first, ObjectType: accountingsync.SyncObjectInvoice},
		second: {ObjectID: second, ObjectType: accountingsync.SyncObjectInvoice},
	}

	ordered := orderedAccountingSyncStates([]pulid.ID{second, never, first, second}, states)
	require.Len(t, ordered, 2)
	assert.Equal(t, second, ordered[0].ObjectID)
	assert.Equal(t, first, ordered[1].ObjectID)
}
