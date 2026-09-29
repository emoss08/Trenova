package statuschange_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/statuschange"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type record struct {
	ID     pulid.ID
	Status string
}

func request(ids []pulid.ID, found []*record, status string) *statuschange.Request[record, string] {
	return &statuschange.Request[record, string]{
		Field:  "recordIds",
		Kind:   "record",
		IDs:    ids,
		Found:  found,
		Status: status,
		Valid:  status == "Active" || status == "Inactive",
		IDOf:   func(entity *record) pulid.ID { return entity.ID },
		Set:    func(entity *record, status string) { entity.Status = status },
	}
}

func TestPlan_ChangesACopyOfEachRecordAndLeavesTheOriginal(t *testing.T) {
	t.Parallel()

	first := &record{ID: pulid.MustNew("rec_"), Status: "Active"}
	second := &record{ID: pulid.MustNew("rec_"), Status: "Active"}

	changes, err := statuschange.Plan(request(
		[]pulid.ID{first.ID, second.ID}, []*record{first, second}, "Inactive",
	))
	require.NoError(t, err)
	require.Len(t, changes, 2)
	assert.Equal(t, "Active", changes[0].Before.Status)
	assert.Equal(t, "Inactive", changes[0].After.Status)
	assert.Equal(t, "Active", first.Status)
	assert.Equal(t, second.ID, changes[1].After.ID)
}

func TestPlan_RefusesWhatTheWriteCannotDo(t *testing.T) {
	t.Parallel()

	known := &record{ID: pulid.MustNew("rec_"), Status: "Active"}
	missing := pulid.MustNew("rec_")

	for name, tc := range map[string]struct {
		req   *statuschange.Request[record, string]
		field string
	}{
		"no records":        {request(nil, nil, "Inactive"), "recordIds"},
		"a status unknown":  {request([]pulid.ID{known.ID}, []*record{known}, "Gone"), "status"},
		"a record not here": {request([]pulid.ID{known.ID, missing}, []*record{known}, "Inactive"), "recordIds"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := statuschange.Plan(tc.req)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.NotEmpty(t, multiErr.Errors)
			assert.Equal(t, tc.field, multiErr.Errors[0].Field)
		})
	}
}
