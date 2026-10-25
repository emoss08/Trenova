package iftarepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func accumulationFixture() accumulationSource {
	return newAccumulationSource(&iftaports.AccumulateMilesRequest{Start: 100, End: 200})
}

func TestAccumulationRouteRows(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, accumulationFixture().routeRows())
}

func TestAccumulationNoTractor(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, accumulationFixture().noTractor())
}

func TestAccumulationUnattributed(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, accumulationFixture().unattributed())
}

func TestAccumulationMismatches(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, accumulationFixture().mismatches())
}
