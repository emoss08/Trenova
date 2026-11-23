package telematicsmapping_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/telematicsmapping"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapWorkerHOSState_CarriesWhenTheDutyStatusBegan(t *testing.T) {
	t.Parallel()

	since := int64(1_800_000_000)
	state := &telematics.WorkerHOSState{
		WorkerID:        pulid.MustNew("wrk_"),
		DutyStatus:      telematics.DutyStatusDriving,
		DutyStatusSince: &since,
	}

	mapped := telematicsmapping.MapWorkerHOSState(state)
	require.NotNil(t, mapped.DutyStatusSince)
	assert.Equal(t, 1_800_000_000, *mapped.DutyStatusSince)

	state.DutyStatusSince = nil
	assert.Nil(t, telematicsmapping.MapWorkerHOSState(state).DutyStatusSince)
}
