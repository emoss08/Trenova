package trailerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/stretchr/testify/assert"
)

func TestApplyLastKnownLocationJoin_ReadsTheLocationsCoordinates(t *testing.T) {
	t.Parallel()

	db := newTrailerQueryTestDB()
	q := db.NewSelect().Model((*trailer.Trailer)(nil))
	q = applyLastKnownLocationJoin(q, trailerports.TrailerRelationIncludes{
		IncludeLastKnownLocation: true,
	})

	sql := q.String()
	assert.Contains(t, sql, "lkl.latitude AS last_known_location_latitude")
	assert.Contains(t, sql, "lkl.longitude AS last_known_location_longitude")
	assert.Contains(t, sql, "LEFT JOIN locations AS lkl")
}

func TestApplyLastKnownLocationJoin_LeavesTheJoinOutWhenUnselected(t *testing.T) {
	t.Parallel()

	db := newTrailerQueryTestDB()
	q := applyLastKnownLocationJoin(
		db.NewSelect().Model((*trailer.Trailer)(nil)),
		trailerports.TrailerRelationIncludes{},
	)

	assert.NotContains(t, q.String(), "last_known_location_latitude")
}

func TestApplyTrailerRelations_LoadsTheOwnerOperatorOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	db := newTrailerQueryTestDB()
	with := applyTrailerRelations(
		db.NewSelect().Model((*trailer.Trailer)(nil)),
		trailerports.TrailerRelationIncludes{IncludeOwnerWorker: true},
	)
	without := applyTrailerRelations(
		db.NewSelect().Model((*trailer.Trailer)(nil)),
		trailerports.TrailerRelationIncludes{},
	)

	assert.Contains(t, with.String(), `"owner_worker"`)
	assert.NotContains(t, without.String(), `"owner_worker"`)
}
