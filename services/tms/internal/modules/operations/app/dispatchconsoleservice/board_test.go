package dispatchconsoleservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchboardportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/integrationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tractorportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type unroutedLegs struct{}

func (unroutedLegs) EstimateLegs(
	_ context.Context,
	_ pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	return make([]routeestimateports.LegEstimate, len(legs)), nil
}

func TestGetBoard_SaysWhenTheDriverListWasCut(t *testing.T) {
	t.Parallel()

	for _, truncated := range []bool{true, false} {
		t.Run(map[bool]string{true: "truncated", false: "complete"}[truncated], func(t *testing.T) {
			t.Parallel()

			info := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
			console := dispatchboardportsmock.NewMockDispatchConsoleRepository(t)
			controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
			organizations := tenantportsmock.NewMockOrganizationRepository(t)
			integrations := integrationportsmock.NewMockIntegrationRepository(t)
			tractors := tractorportsmock.NewMockTractorRepository(t)

			svc := &Service{
				consoleRepo:         console,
				dispatchControlRepo: controls,
				organizations:       organizations,
				candidates: dispatchcandidateservice.New(dispatchcandidateservice.Params{
					ConsoleRepo:     console,
					IntegrationRepo: integrations,
					TractorRepo:     tractors,
					RouteEstimator:  unroutedLegs{},
					Logger:          zap.NewNop(),
				}),
				prices: dispatchplanportsmock.NewMockPriceRepository(t),
				l:      zap.NewNop(),
			}

			workerID := pulid.MustNew("wrk_")
			tractorID := pulid.MustNew("trac_")
			driver := &dispatchboardports.BoardDriver{
				WorkerID:  workerID,
				FirstName: "Ada",
				LastName:  "Reed",
				TractorID: tractorID,
			}

			controls.EXPECT().
				GetOrCreate(mock.Anything, info.OrgID, info.BuID).
				Return(dispatchcontrol.NewDefaultDispatchControl(info.OrgID, info.BuID), nil)
			organizations.EXPECT().
				GetByID(mock.Anything, mock.Anything).
				Return(&tenant.Organization{Timezone: "America/Chicago"}, nil)
			console.EXPECT().
				ListBoardMoves(mock.Anything, mock.Anything).
				Return([]*dispatchboardports.BoardMove{}, nil)
			console.EXPECT().
				ListBoardDrivers(mock.Anything, mock.Anything).
				Return(&dispatchboardports.BoardDriverPage{
					Drivers:   []*dispatchboardports.BoardDriver{driver},
					Truncated: truncated,
				}, nil)
			integrations.EXPECT().ListByTenant(mock.Anything, info).Return(nil, nil)
			console.EXPECT().ListWorkerCommitments(mock.Anything, mock.Anything).Return(nil, nil)
			console.EXPECT().ListWorkerTimeOff(mock.Anything, mock.Anything).Return(nil, nil)
			console.EXPECT().ListWorkerWorkload(mock.Anything, mock.Anything).Return(nil, nil)
			console.EXPECT().
				ListWorkersByIDs(mock.Anything, info, []pulid.ID{workerID}).
				Return([]*worker.Worker{availableWorker(workerID)}, nil)
			tractors.EXPECT().
				GetByIDs(mock.Anything, mock.Anything).
				Return([]*tractor.Tractor{{
					ID:     tractorID,
					Status: domaintypes.EquipmentStatusAvailable,
				}}, nil)
			console.EXPECT().
				ListEquipmentByIDs(mock.Anything, mock.Anything).
				Return(nil, nil, nil)
			console.EXPECT().ListWorkerOnTimeStats(mock.Anything, mock.Anything).Return(nil, nil)
			console.EXPECT().
				ListWorkerAcceptanceStats(mock.Anything, mock.Anything).
				Return(nil, nil)
			console.EXPECT().
				GetBoardSummary(mock.Anything, mock.Anything).
				Return(&dispatchboardports.BoardSummary{}, nil)

			board, err := svc.GetBoard(t.Context(), &GetBoardRequest{TenantInfo: info})

			require.NoError(t, err)
			assert.Equal(t, truncated, board.DriversTruncated)
			require.Len(t, board.Drivers, 1)
			assert.Equal(t, workerID, board.Drivers[0].WorkerID)
			assert.Empty(t, board.Moves)
		})
	}
}
