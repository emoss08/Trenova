package dispatchjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_767_225_600)

type recordingPublisher struct {
	events []portservices.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event portservices.AgentEvent) {
	r.events = append(r.events, event)
}

type recordingProjector struct {
	items []portservices.WatchtowerItemInput
}

func (r *recordingProjector) Upsert(_ context.Context, item portservices.WatchtowerItemInput) {
	r.items = append(r.items, item)
}

func (r *recordingProjector) Resolve(
	context.Context,
	pagination.TenantInfo,
	watchtower.SourceKind,
	string,
) {
}

func coverageActivities(
	t *testing.T,
	window int16,
) (*Activities, *recordingPublisher, *recordingProjector) {
	t.Helper()

	controls := mocks.NewMockDispatchControlRepository(t)
	controls.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&dispatchcontrol.DispatchControl{CoverageRiskWindowHours: window}, nil).
		Maybe()

	publisher := &recordingPublisher{}
	projector := &recordingProjector{}
	activities := NewActivities(ActivitiesParams{
		DispatchControlRepo: controls,
		ProposalRepo:        new(stubProposalRepo),
		AutoAssign:          mocks.NewMockDispatchAutoAssignService(t),
		Watchtower:          projector,
		Publisher:           publisher,
		Logger:              zap.NewNop(),
	})
	activities.now = func() int64 { return testNow }

	return activities, publisher, projector
}

func uncoveredAt(hoursOut int64) *portservices.DispatchUncoveredMove {
	return &portservices.DispatchUncoveredMove{
		MoveID:    pulid.MustNew("mov_"),
		ProNumber: "PRO-1",
		Reason:    "No eligible driver was available for this move",
		StartsAt:  testNow + hoursOut*secondsPerHour,
	}
}

func TestRaiseCoverageRisk_OnlyMovesInsideTheWindow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		window    int16
		hoursOut  []int64
		wantCount int
	}{
		{
			name:      "a move inside the window is raised",
			window:    12,
			hoursOut:  []int64{6},
			wantCount: 1,
		},
		{
			name:      "a move exactly on the window is raised",
			window:    12,
			hoursOut:  []int64{12},
			wantCount: 1,
		},
		{
			name:      "a move an hour past the window is left to the planner",
			window:    12,
			hoursOut:  []int64{13},
			wantCount: 0,
		},
		{
			name:      "a wider window reaches further out",
			window:    48,
			hoursOut:  []int64{13, 30, 49},
			wantCount: 2,
		},
		{
			name:      "an unset window falls back to the default twelve hours",
			window:    0,
			hoursOut:  []int64{6, 20},
			wantCount: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			activities, publisher, projector := coverageActivities(t, tc.window)
			moves := make([]*portservices.DispatchUncoveredMove, 0, len(tc.hoursOut))
			for _, hours := range tc.hoursOut {
				moves = append(moves, uncoveredAt(hours))
			}

			raised := activities.raiseCoverageRisk(t.Context(), tenant(), moves)

			require.Equal(t, tc.wantCount, raised)
			require.Len(t, publisher.events, tc.wantCount)
			require.Len(t, projector.items, tc.wantCount)
			for _, event := range publisher.events {
				require.Equal(t, agent.EventShipmentMoveCoverageAtRisk, event.Kind)
			}
			for _, item := range projector.items {
				require.Equal(t, watchtower.SourceMoveCoverage, item.SourceKind)
				require.Equal(t, agent.SubjectShipmentMove, item.SubjectType)
			}
		})
	}
}

func TestRaiseCoverageRisk_LeavesUndatedMovesAlone(t *testing.T) {
	t.Parallel()

	activities, publisher, projector := coverageActivities(t, 12)

	raised := activities.raiseCoverageRisk(t.Context(), tenant(), []*portservices.DispatchUncoveredMove{
		nil,
		{MoveID: pulid.MustNew("mov_"), StartsAt: 0},
	})

	require.Zero(t, raised)
	require.Empty(t, publisher.events)
	require.Empty(t, projector.items)
}

func TestRaiseCoverageRisk_SeverityRisesAsTheMoveNears(t *testing.T) {
	t.Parallel()

	activities, _, projector := coverageActivities(t, 12)

	raised := activities.raiseCoverageRisk(t.Context(), tenant(), []*portservices.DispatchUncoveredMove{
		uncoveredAt(2),
		uncoveredAt(10),
	})

	require.Equal(t, 2, raised)
	require.Equal(t, watchtower.SeverityCritical, projector.items[0].Severity)
	require.Equal(t, watchtower.SeverityWarning, projector.items[1].Severity)
}
