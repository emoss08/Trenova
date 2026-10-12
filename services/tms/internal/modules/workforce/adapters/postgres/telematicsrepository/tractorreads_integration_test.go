//go:build integration

package telematicsrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *stopReviewFixture) seededTractors(t *testing.T, count int) []pulid.ID {
	t.Helper()

	ids := make([]pulid.ID, 0, count)
	require.NoError(t, f.db.NewSelect().
		TableExpr("tractors").
		Column("id").
		Where("organization_id = ?", f.tenantInfo.OrgID).
		Where("business_unit_id = ?", f.tenantInfo.BuID).
		OrderExpr("id").
		Limit(int64(count)).
		Scan(f.ctx, &ids))
	require.Len(t, ids, count)
	return ids
}

func (f *stopReviewFixture) insertInspection(
	t *testing.T,
	id pulid.ID,
	tractorID pulid.ID,
	startedAt int64,
) {
	t.Helper()

	_, err := f.db.NewInsert().Model(&telematics.VehicleInspection{
		ID:             id,
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Provider:       "Samsara",
		ProviderDvirID: id.String(),
		TractorID:      tractorID,
		InspectionType: "preTrip",
		SafetyStatus:   "safe",
		StartedAt:      startedAt,
		EndedAt:        startedAt + 600,
		Signed:         true,
		CreatedAt:      startedAt + 600,
	}).Exec(f.ctx)
	require.NoError(t, err)
}

func TestListVehicleInspections_PagesByStartedAtAndIDWithoutLosingTies(t *testing.T) {
	f := setupStopReview(t)
	tractors := f.seededTractors(t, 2)
	tractorID := tractors[0]

	ids := []pulid.ID{
		pulid.ID("vinsp_01JAAAAAAAAAAAAAAAAAAAAAA1"),
		pulid.ID("vinsp_01JAAAAAAAAAAAAAAAAAAAAAA2"),
		pulid.ID("vinsp_01JAAAAAAAAAAAAAAAAAAAAAA3"),
		pulid.ID("vinsp_01JAAAAAAAAAAAAAAAAAAAAAA4"),
		pulid.ID("vinsp_01JAAAAAAAAAAAAAAAAAAAAAA5"),
	}
	f.insertInspection(t, ids[0], tractorID, 1_800_000_000)
	f.insertInspection(t, ids[1], tractorID, 1_800_003_600)
	f.insertInspection(t, ids[2], tractorID, 1_800_003_600)
	f.insertInspection(t, ids[3], tractorID, 1_800_007_200)
	f.insertInspection(t, ids[4], tractors[1], 1_800_010_000)

	want := []pulid.ID{ids[3], ids[2], ids[1], ids[0]}
	var got []pulid.ID
	req := &telematicsports.ListVehicleInspectionsRequest{
		TenantInfo: f.tenantInfo,
		TractorID:  tractorID,
		Limit:      2,
	}
	for range 4 {
		page, err := f.repo.ListVehicleInspections(f.ctx, req)
		require.NoError(t, err)
		if len(page) == 0 {
			break
		}
		for _, inspection := range page {
			got = append(got, inspection.ID)
		}
		last := page[len(page)-1]
		req.Before = last.StartedAt
		req.BeforeID = last.ID
	}

	assert.Equal(t, want, got)
}

func TestListVehicleInspections_BeforeWithoutIDKeepsStrictlyOlder(t *testing.T) {
	f := setupStopReview(t)
	tractorID := f.seededTractors(t, 1)[0]

	older := pulid.ID("vinsp_01JBBBBBBBBBBBBBBBBBBBBBB1")
	same := pulid.ID("vinsp_01JBBBBBBBBBBBBBBBBBBBBBB2")
	f.insertInspection(t, older, tractorID, 1_800_000_000)
	f.insertInspection(t, same, tractorID, 1_800_003_600)

	page, err := f.repo.ListVehicleInspections(f.ctx, &telematicsports.ListVehicleInspectionsRequest{
		TenantInfo: f.tenantInfo,
		TractorID:  tractorID,
		Before:     1_800_003_600,
	})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, older, page[0].ID)
}

func TestListVehiclePositions_ByTractorReturnsOnlyThatTractor(t *testing.T) {
	f := setupStopReview(t)
	tractors := f.seededTractors(t, 2)

	positions := make([]*telematics.VehiclePosition, 0, len(tractors))
	for index, tractorID := range tractors {
		positions = append(positions, &telematics.VehiclePosition{
			OrganizationID:    f.tenantInfo.OrgID,
			BusinessUnitID:    f.tenantInfo.BuID,
			TractorID:         tractorID,
			Provider:          "Samsara",
			ProviderVehicleID: tractorID.String(),
			Latitude:          32.7767 + float64(index),
			Longitude:         -96.797,
			SpeedMph:          54,
			RecordedAt:        1_800_000_000 + int64(index),
			ReceivedAt:        1_800_000_005 + int64(index),
		})
	}
	require.NoError(t, f.repo.UpsertVehiclePositions(f.ctx, positions))

	got, err := f.repo.ListVehiclePositions(f.ctx, &telematicsports.ListVehiclePositionsRequest{
		TenantInfo:    f.tenantInfo,
		TractorIDs:    []pulid.ID{tractors[1]},
		IncludeWorker: true,
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, tractors[1], got[0].TractorID)
	assert.InDelta(t, 33.7767, got[0].Latitude, 0.0001)
}
