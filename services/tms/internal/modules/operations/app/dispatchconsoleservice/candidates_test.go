package dispatchconsoleservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/positioningtest"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchboardportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/integrationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tractorportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetMoveCandidates_ReadsTheDriversNearestThePickup(t *testing.T) {
	t.Parallel()

	latitude, longitude := 32.7767, -96.7970
	cases := map[string]struct {
		move *dispatchboardports.BoardMove
		near *geoutils.Point
	}{
		"located pickup": {
			move: &dispatchboardports.BoardMove{
				OriginLatitude:  &latitude,
				OriginLongitude: &longitude,
			},
			near: &geoutils.Point{Latitude: latitude, Longitude: longitude},
		},
		"pickup with no coordinates": {
			move: &dispatchboardports.BoardMove{},
			near: nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			info := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
			console := dispatchboardportsmock.NewMockDispatchConsoleRepository(t)
			controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
			integrations := integrationportsmock.NewMockIntegrationRepository(t)
			svc := &Service{
				consoleRepo:         console,
				dispatchControlRepo: controls,
				candidates: dispatchcandidateservice.New(dispatchcandidateservice.Params{
					Positionings:    positioningtest.None{},
					ConsoleRepo:     console,
					IntegrationRepo: integrations,
					TractorRepo:     tractorportsmock.NewMockTractorRepository(t),
					RouteEstimator:  unroutedLegs{},
					Logger:          zap.NewNop(),
				}),
				l: zap.NewNop(),
			}
			moveID := pulid.MustNew("smv_")
			tc.move.MoveID = moveID

			controls.EXPECT().
				GetOrCreate(mock.Anything, info.OrgID, info.BuID).
				Return(dispatchcontrol.NewDefaultDispatchControl(info.OrgID, info.BuID), nil)
			console.EXPECT().
				ListBoardMoves(mock.Anything, mock.Anything).
				Return([]*dispatchboardports.BoardMove{tc.move}, nil)
			var asked *dispatchboardports.DispatchBoardFilter
			console.EXPECT().
				ListBoardDrivers(mock.Anything, mock.Anything).
				RunAndReturn(func(
					_ context.Context,
					filter *dispatchboardports.DispatchBoardFilter,
				) (*dispatchboardports.BoardDriverPage, error) {
					asked = filter
					return &dispatchboardports.BoardDriverPage{}, nil
				})
			integrations.EXPECT().ListByTenant(mock.Anything, info).Return(nil, nil)

			candidates, err := svc.GetMoveCandidates(t.Context(), &MoveCandidatesRequest{
				TenantInfo: info,
				MoveID:     moveID,
			})

			require.NoError(t, err)
			assert.Empty(t, candidates)
			require.NotNil(t, asked)
			assert.Equal(t, tc.near, asked.Near)
		})
	}
}
