//go:build integration

package telematicsrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crewedTractor struct {
	TractorID pulid.ID
	WorkerID  pulid.ID
}

func (f *stopReviewFixture) crewedTractor(t *testing.T) crewedTractor {
	t.Helper()

	tr := tractordb.Tractors
	rows, err := keel.Into[crewedTractor](f.ctx, f.db, tr.
		Pick(tr.ID, tr.PrimaryWorkerID).
		Where(tr.PrimaryWorkerID.IsNotNull()).
		Order(tr.ID.Asc()).
		Limit(1))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	return rows[0]
}

func (f *stopReviewFixture) storedWorker(t *testing.T, id pulid.ID) *worker.Worker {
	t.Helper()

	w := workerdb.Workers
	found, err := w.Where(w.ID.Eq(id)).One(f.ctx, f.db)
	require.NoError(t, err)

	return found
}

func (f *stopReviewFixture) placeHome(t *testing.T, workerID pulid.ID, lat, lon float64) {
	t.Helper()

	w := workerdb.Workers
	_, err := w.Where(w.ID.Eq(workerID)).Update(f.ctx, f.db, w.HomeLatitude.To(lat), w.HomeLongitude.To(lon))
	require.NoError(t, err)
}

func TestListTractorHomes_ReturnsOnlyTractorsWhoseDriverHasAHome_Integration(t *testing.T) {
	f := setupStopReview(t)
	crewed := f.crewedTractor(t)

	homes, err := f.repo.ListTractorHomes(f.ctx, []pulid.ID{crewed.TractorID})
	require.NoError(t, err)
	assert.Empty(t, homes, "a driver with no home recorded gives the poll nothing to compare")

	f.placeHome(t, crewed.WorkerID, 41.52, -88.08)
	homes, err = f.repo.ListTractorHomes(f.ctx, []pulid.ID{crewed.TractorID})
	require.NoError(t, err)
	require.Len(t, homes, 1)
	assert.Equal(t, crewed.WorkerID, homes[0].WorkerID)
	assert.InDelta(t, 41.52, homes[0].HomeLatitude, 0.0001)
	assert.InDelta(t, -88.08, homes[0].HomeLongitude, 0.0001)
}

func TestRecordHomeVisits_KeepsTheLatestVisitWithoutBumpingTheVersion_Integration(t *testing.T) {
	f := setupStopReview(t)
	crewed := f.crewedTractor(t)
	before := f.storedWorker(t, crewed.WorkerID)

	require.NoError(t, f.repo.RecordHomeVisits(f.ctx, []telematicsports.HomeVisit{
		{WorkerID: crewed.WorkerID, At: 1_800_000_000},
	}))
	require.NoError(t, f.repo.RecordHomeVisits(f.ctx, []telematicsports.HomeVisit{
		{WorkerID: crewed.WorkerID, At: 1_700_000_000},
	}))

	after := f.storedWorker(t, crewed.WorkerID)
	require.NotNil(t, after.LastHomeAt)
	assert.Equal(t, int64(1_800_000_000), *after.LastHomeAt, "an older ping never moves last home back")
	assert.Equal(t, before.Version, after.Version, "a stamp does not conflict with an open edit")
}
