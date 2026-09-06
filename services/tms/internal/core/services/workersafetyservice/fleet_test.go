package workersafetyservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fleetRepo answers the seven aggregate queries with fixed rows, so the
// composition can be tested without a database.
type fleetRepo struct {
	repositories.WorkerSafetyRepository
	ratings   []repositories.FleetSafetyRatingRow
	terminals []repositories.FleetSafetyTerminalRow
	kinds     []repositories.FleetSafetyKindRow
	basics    []repositories.FleetSafetyBasicRow
	inferred  []repositories.FleetSafetyEventBasicRow
	trend     []repositories.FleetSafetyTrendRow
	ranking   []repositories.FleetSafetyRankRow
	// seen records what each call was asked for, so the request the service
	// built can be asserted on.
	seen []repositories.FleetSafetyRequest
}

func (f *fleetRepo) FleetRatings(
	_ context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyRatingRow, error) {
	f.seen = append(f.seen, *req)
	return f.ratings, nil
}

func (f *fleetRepo) FleetTerminals(
	_ context.Context,
	_ *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyTerminalRow, error) {
	return f.terminals, nil
}

func (f *fleetRepo) FleetKinds(
	_ context.Context,
	_ *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyKindRow, error) {
	return f.kinds, nil
}

func (f *fleetRepo) FleetBasics(
	_ context.Context,
	_ *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyBasicRow, error) {
	return f.basics, nil
}

func (f *fleetRepo) FleetEventBasics(
	_ context.Context,
	_ *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyEventBasicRow, error) {
	return f.inferred, nil
}

func (f *fleetRepo) FleetTrend(
	_ context.Context,
	_ *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyTrendRow, error) {
	return f.trend, nil
}

func (f *fleetRepo) FleetRanking(
	_ context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyRankRow, error) {
	if req.RankBest {
		reversed := make([]repositories.FleetSafetyRankRow, len(f.ranking))
		for i, row := range f.ranking {
			reversed[len(f.ranking)-1-i] = row
		}
		return reversed, nil
	}
	return f.ranking, nil
}

func newFleetService(repo *fleetRepo) *workersafetyservice.Service {
	return workersafetyservice.NewWithDeps(workersafetyservice.Deps{
		Logger: zap.NewNop(),
		Repo:   repo,
	})
}

func fleetTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestFleet_SummarisesTheRoster(t *testing.T) {
	t.Parallel()

	repo := &fleetRepo{
		ratings: []repositories.FleetSafetyRatingRow{
			{Rating: worker.SafetyRatingExcellent, Workers: 6, TotalScore: 582},
			{Rating: worker.SafetyRatingWatch, Workers: 3, TotalScore: 195},
			{Rating: worker.SafetyRatingAtRisk, Workers: 1, TotalScore: 40},
		},
		kinds: []repositories.FleetSafetyKindRow{
			{
				Kind:        worker.SafetyEventAccident,
				Events:      2,
				Points:      12,
				Preventable: 1,
				Open:        1,
			},
			{
				Kind:         worker.SafetyEventInspection,
				Events:       9,
				Points:       5,
				OutOfService: 1,
			},
		},
	}

	summary, err := newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo: fleetTenant(),
		AsOf:       1_800_000_000,
	})
	require.NoError(t, err)

	assert.Equal(t, int32(10), summary.Workers)
	assert.Equal(t, int32(82), summary.AverageScore)
	assert.Equal(t, int32(1), summary.AtRisk)
	assert.Equal(t, int32(3), summary.Watch)

	// The header totals are summed once here so every caller does not have to
	// add up the kind breakdown itself.
	assert.Equal(t, int32(11), summary.TotalEvents)
	assert.Equal(t, int32(17), summary.TotalPoints)
	assert.Equal(t, int32(1), summary.OpenEvents)
	assert.Equal(t, int32(1), summary.OutOfServiceOrders)
}

// A window nobody chose still has to be a window, and one somebody chose far
// too wide has to be trimmed before it reaches the database.
func TestFleet_NormalisesTheWindowAndTheRankLimit(t *testing.T) {
	t.Parallel()

	repo := &fleetRepo{}
	_, err := newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo: fleetTenant(),
	})
	require.NoError(t, err)
	require.Len(t, repo.seen, 1)
	assert.Equal(t, 12, repo.seen[0].WindowMonths)
	assert.Equal(t, 10, repo.seen[0].RankLimit)
	assert.Positive(t, repo.seen[0].Now)

	repo.seen = nil
	_, err = newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo:   fleetTenant(),
		WindowMonths: 500,
		RankLimit:    5000,
	})
	require.NoError(t, err)
	require.Len(t, repo.seen, 1)
	assert.Equal(t, 36, repo.seen[0].WindowMonths)
	assert.Equal(t, 50, repo.seen[0].RankLimit)
}

func TestFleet_FoldsRecordedAndInferredBasics(t *testing.T) {
	t.Parallel()

	repo := &fleetRepo{
		basics: []repositories.FleetSafetyBasicRow{
			{
				Basic:       worker.BasicHOSCompliance,
				Bucket:      0,
				Violations:  2,
				SeveritySum: 10,
			},
		},
		inferred: []repositories.FleetSafetyEventBasicRow{
			{Kind: worker.SafetyEventAccident, Bucket: 0, Events: 1, Points: 6},
		},
	}

	summary, err := newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo: fleetTenant(),
		AsOf:       1_800_000_000,
	})
	require.NoError(t, err)

	require.Len(t, summary.Basics, 7)
	assert.True(t, summary.BasicsInferred)

	var hos, crash worker.FleetSafetyBasic
	for _, basic := range summary.Basics {
		switch basic.Basic {
		case worker.BasicHOSCompliance:
			hos = basic
		case worker.BasicCrashIndicator:
			crash = basic
		default:
		}
	}
	assert.Equal(t, int32(30), hos.WeightedScore)
	assert.False(t, hos.Inferred, "a keyed violation is not an inference")
	assert.Equal(t, int32(18), crash.WeightedScore)
	assert.True(t, crash.Inferred)
}

// The best and worst lists are the same query read from opposite ends, so a
// driver cannot appear at the top of one and the top of the other.
func TestFleet_RanksFromBothEnds(t *testing.T) {
	t.Parallel()

	repo := &fleetRepo{
		ranking: []repositories.FleetSafetyRankRow{
			{WorkerID: "wrk_1", FirstName: "Ada", LastName: "Byrne", Score: 40},
			{WorkerID: "wrk_2", FirstName: "Ben", LastName: "Cole", Score: 70},
			{WorkerID: "wrk_3", FirstName: "Cal", LastName: "Diaz", Score: 98},
		},
	}

	summary, err := newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo: fleetTenant(),
		AsOf:       1_800_000_000,
	})
	require.NoError(t, err)

	require.Len(t, summary.Worst, 3)
	require.Len(t, summary.Best, 3)
	assert.Equal(t, "Ada Byrne", summary.Worst[0].Name)
	assert.Equal(t, "Cal Diaz", summary.Best[0].Name)
}

// A terminal with nobody in it has no average to report; dividing anyway would
// give it a perfect record.
func TestFleet_TerminalAverages(t *testing.T) {
	t.Parallel()

	repo := &fleetRepo{
		terminals: []repositories.FleetSafetyTerminalRow{
			{
				FleetCodeID:   "fc_1",
				FleetCodeCode: "SOUTH",
				Workers:       4,
				AtRisk:        1,
				TotalScore:    300,
			},
			{FleetCodeCode: "", Workers: 0, TotalScore: 0},
		},
	}

	summary, err := newFleetService(repo).Fleet(t.Context(), &workersafetyservice.FleetRequest{
		TenantInfo: fleetTenant(),
		AsOf:       1_800_000_000,
	})
	require.NoError(t, err)

	require.Len(t, summary.Terminals, 2)
	assert.Equal(t, int32(75), summary.Terminals[0].AverageScore)
	assert.Equal(t, int32(0), summary.Terminals[1].AverageScore)
}
