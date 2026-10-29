package carrierintelsnapshot_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestSnapshotFreshnessTracksDeeperFetch(t *testing.T) {
	t.Parallel()

	merged := &carrierintelsnapshot.CarrierIntelSnapshot{
		Depth:          carrierintel.LookupDepthFull,
		FetchedDepth:   carrierintel.LookupDepthLite,
		FetchedAt:      partiestest.CarrierIntelNow - timeutils.SecondsPerDay,
		DepthFetchedAt: partiestest.CarrierIntelNow - 29*timeutils.SecondsPerDay,
	}
	fullTTL := int64(30 * timeutils.SecondsPerDay)
	liteTTL := int64(2 * timeutils.SecondsPerDay)

	assert.True(
		t,
		merged.IsFreshForDepth(partiestest.CarrierIntelNow, liteTTL, carrierintel.LookupDepthLite),
	)
	assert.True(
		t,
		merged.IsFreshForDepth(partiestest.CarrierIntelNow, fullTTL, carrierintel.LookupDepthFull),
	)
	later := partiestest.CarrierIntelNow + 2*timeutils.SecondsPerDay
	assert.False(t, merged.IsFreshForDepth(later, fullTTL, carrierintel.LookupDepthFull))
	assert.Equal(t, merged.DepthFetchedAt, merged.AsOfForDepth(carrierintel.LookupDepthFull))

	confirmedAt := partiestest.CarrierIntelNow
	full := &carrierintelsnapshot.CarrierIntelSnapshot{
		Depth:          carrierintel.LookupDepthFull,
		FetchedDepth:   carrierintel.LookupDepthFull,
		FetchedAt:      partiestest.CarrierIntelNow - 40*timeutils.SecondsPerDay,
		DepthFetchedAt: partiestest.CarrierIntelNow - 40*timeutils.SecondsPerDay,
		ConfirmedAt:    &confirmedAt,
	}
	assert.Equal(t, confirmedAt, full.DepthAsOf())
	assert.True(
		t,
		full.IsFreshForDepth(partiestest.CarrierIntelNow, fullTTL, carrierintel.LookupDepthFull),
	)
}
