package iftajobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func moveIDs(n int) []pulid.ID {
	ids := make([]pulid.ID, 0, n)
	for range n {
		ids = append(ids, pulid.MustNew("sm_"))
	}
	return ids
}

func TestChunkMoveIDs(t *testing.T) {
	t.Parallel()

	chunks := chunkMoveIDs(moveIDs(45), AttributeBatchSize)
	require.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 20)
	assert.Len(t, chunks[1], 20)
	assert.Len(t, chunks[2], 5)

	assert.Empty(t, chunkMoveIDs(nil, AttributeBatchSize))
	assert.Empty(t, chunkMoveIDs(moveIDs(3), 0))
	assert.Len(t, chunkMoveIDs(moveIDs(20), AttributeBatchSize), 1)
}

func TestBackfillInputEffectiveMaxMoves(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultBackfillMaxMoves, BackfillInput{}.EffectiveMaxMoves())
	assert.Equal(t, DefaultBackfillMaxMoves, BackfillInput{MaxMoves: -1}.EffectiveMaxMoves())
	assert.Equal(t, 30, BackfillInput{MaxMoves: 30}.EffectiveMaxMoves())
}

func TestAttributeMovesActivityCountsResults(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	ids := moveIDs(3)

	distanceCalc := mocks.NewMockDistanceCalculationService(t)
	distanceCalc.EXPECT().
		RecalculateMoveJurisdictionMiles(mock.Anything, services.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo:     tenantInfo,
			ShipmentMoveID: ids[0],
			UserID:         tenantInfo.UserID,
		}).
		Return([]*shipment.ShipmentMoveJurisdictionMile{
			{JurisdictionCode: "TX", Distance: 100, DistanceUnits: "Miles"},
			{JurisdictionCode: "OK", Distance: 160.9344, DistanceUnits: "Kilometers"},
		}, nil)
	distanceCalc.EXPECT().
		RecalculateMoveJurisdictionMiles(mock.Anything, services.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo:     tenantInfo,
			ShipmentMoveID: ids[1],
			UserID:         tenantInfo.UserID,
		}).
		Return(nil, assert.AnError)
	distanceCalc.EXPECT().
		RecalculateMoveJurisdictionMiles(mock.Anything, services.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo:     tenantInfo,
			ShipmentMoveID: ids[2],
			UserID:         tenantInfo.UserID,
		}).
		Return([]*shipment.ShipmentMoveJurisdictionMile{
			{JurisdictionCode: "NM", Distance: 50, DistanceUnits: "Miles"},
		}, nil)

	activities := &Activities{
		distanceCalculation: distanceCalc,
		logger:              zap.NewNop(),
	}
	env.RegisterActivity(activities.AttributeMovesActivity)

	future, err := env.ExecuteActivity(activities.AttributeMovesActivity, AttributeMovesInput{
		TenantInfo: tenantInfo,
		MoveIDs:    ids,
	})
	require.NoError(t, err)

	var result *AttributeMovesResult
	require.NoError(t, future.Get(&result))
	assert.Equal(t, 2, result.Processed)
	assert.Equal(t, 1, result.Failed)
	assert.True(t, result.Miles.Equal(decimal.NewFromInt(250)), result.Miles.String())
}

func TestListMovesMissingBreakdownActivityMapsPage(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ids := moveIDs(2)
	repo := mocks.NewMockShipmentMoveJurisdictionMileRepository(t)
	repo.EXPECT().
		ListUnattributedMoves(mock.Anything, repositories.UnattributedMovesRequest{
			TenantInfo: tenantInfo,
			Start:      100,
			End:        200,
			Limit:      ListMovesPageSize,
			Offset:     3,
		}).
		Return(&repositories.UnattributedMovesPage{
			MoveIDs:    ids,
			TotalMoves: 5,
			TotalMiles: decimal.NewFromFloat(812.5),
		}, nil)

	activities := &Activities{
		jurisdictionMileRepo: repo,
		logger:               zap.NewNop(),
	}
	env.RegisterActivity(activities.ListMovesMissingBreakdownActivity)

	future, err := env.ExecuteActivity(
		activities.ListMovesMissingBreakdownActivity,
		ListMovesMissingBreakdownInput{
			TenantInfo: tenantInfo,
			Start:      100,
			End:        200,
			Limit:      ListMovesPageSize,
			Offset:     3,
		},
	)
	require.NoError(t, err)

	var result *ListMovesMissingBreakdownResult
	require.NoError(t, future.Get(&result))
	assert.Equal(t, ids, result.MoveIDs)
	assert.Equal(t, 5, result.TotalMoves)
	assert.True(t, result.TotalMiles.Equal(decimal.NewFromFloat(812.5)))
}
