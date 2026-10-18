package agentsubjectservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeShipments struct {
	shipmentports.ShipmentService

	entity *shipment.Shipment
}

func (f *fakeShipments) Get(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	entity := *f.entity
	entity.ID = req.ID

	return &entity, nil
}

type fakeConsole struct {
	shipmentports.DispatchConsoleRepository

	moves   []*shipmentports.BoardMove
	filters []*shipmentports.DispatchBoardFilter
}

func (f *fakeConsole) ListBoardMoves(
	_ context.Context,
	filter *shipmentports.DispatchBoardFilter,
) ([]*shipmentports.BoardMove, error) {
	f.filters = append(f.filters, filter)

	return f.moves, nil
}

type fakeDispatchControls struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
	err     error
}

func (f *fakeDispatchControls) GetByOrgID(
	_ context.Context,
	_ dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	return f.control, f.err
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestService_MarksAShipmentEnteredByEDIAsWrittenOutside(t *testing.T) {
	t.Parallel()

	cases := map[shipment.EntryMethod]agent.TaintSource{
		shipment.EntryMethodEDI:    agent.TaintSourceEDI,
		shipment.EntryMethodManual: "",
	}
	for method, want := range cases {
		t.Run(string(method), func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				shipments: &fakeShipments{entity: &shipment.Shipment{
					ProNumber:   "S-2001",
					EntryMethod: method,
				}},
				logger: zap.NewNop(),
			}
			shipmentID := pulid.MustNew("shp_")

			subject, err := svc.Describe(
				t.Context(), testTenant(), agent.SubjectShipment, shipmentID,
			)
			require.NoError(t, err)
			require.NotNil(t, subject)
			assert.Equal(t, "Shipment PRO S-2001", subject.Label)
			assert.Equal(t, shipmentID.String(), subject.ID)
			assert.Equal(t, want, subject.OutsideAuthored)
		})
	}
}

func TestService_SaysWhetherAMoveStartsInsideTheCoverageWindow(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	cases := []struct {
		name     string
		startsIn int64
		inside   bool
	}{
		{name: "inside", startsIn: 2 * 3600, inside: true},
		{name: "outside", startsIn: 30 * 3600, inside: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			moveID := pulid.MustNew("smv_")
			svc := &Service{
				console: &fakeConsole{moves: []*shipmentports.BoardMove{{
					MoveID:            moveID,
					ProNumber:         "S-2002",
					OriginWindowStart: now + tc.startsIn,
				}}},
				dispatch: &fakeDispatchControls{control: &dispatchcontrol.DispatchControl{
					CoverageRiskWindowHours: 12,
				}},
				logger: zap.NewNop(),
			}

			subject, err := svc.Describe(
				t.Context(), testTenant(), agent.SubjectShipmentMove, moveID,
			)
			require.NoError(t, err)

			var notes struct {
				MoveID   string `json:"moveId"`
				Coverage *struct {
					WindowHours int16 `json:"windowHours"`
					StartsAt    int64 `json:"startsAt"`
					Inside      bool  `json:"startsInsideWindow"`
				} `json:"coverage"`
			}
			require.NoError(t, sonic.UnmarshalString(subject.Notes, &notes))
			assert.Equal(t, moveID.String(), notes.MoveID, "the move itself is still described")
			require.NotNil(t, notes.Coverage)
			assert.Equal(t, int16(12), notes.Coverage.WindowHours)
			assert.Equal(t, now+tc.startsIn, notes.Coverage.StartsAt)
			assert.Equal(t, tc.inside, notes.Coverage.Inside)
		})
	}
}

func TestService_AMoveWithoutDispatchControlsStillDescribesTheMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	svc := &Service{
		console: &fakeConsole{moves: []*shipmentports.BoardMove{{
			MoveID:            moveID,
			ProNumber:         "S-2003",
			OriginWindowStart: timeutils.NowUnix(),
		}}},
		dispatch: &fakeDispatchControls{err: assert.AnError},
		logger:   zap.NewNop(),
	}

	subject, err := svc.Describe(t.Context(), testTenant(), agent.SubjectShipmentMove, moveID)
	require.NoError(t, err)
	assert.Equal(t, "Shipment move for PRO S-2003", subject.Label)
	assert.Contains(t, subject.Notes, moveID.String())
	assert.NotContains(t, subject.Notes, `"coverage":`)
}

func TestService_ReadsACoveredOrFinishedMoveOffTheBoard(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	console := &fakeConsole{moves: []*shipmentports.BoardMove{{
		MoveID:     moveID,
		ProNumber:  "S-2004",
		MoveStatus: shipment.MoveStatusCompleted,
	}}}
	svc := &Service{console: console, logger: zap.NewNop()}

	subject, err := svc.Describe(t.Context(), testTenant(), agent.SubjectShipmentMove, moveID)
	require.NoError(t, err)
	assert.Equal(t, "Shipment move for PRO S-2004", subject.Label)

	require.Len(t, console.filters, 1)
	filter := console.filters[0]
	assert.Equal(t, []pulid.ID{moveID}, filter.MoveIDs)
	assert.True(t, filter.IncludeCovered, "an arrival or departure is on a move with a driver")
	assert.ElementsMatch(t, shipment.MoveStatuses(), filter.MoveStatuses,
		"a delivered move is Completed, which the board leaves out by default")
}

func TestService_ShipmentSubjectReadsEveryOneOfItsMoves(t *testing.T) {
	t.Parallel()

	first, last := pulid.MustNew("smv_"), pulid.MustNew("smv_")
	console := &fakeConsole{}
	svc := &Service{
		shipments: &fakeShipments{entity: &shipment.Shipment{
			ProNumber: "S-2005",
			Moves:     []*shipment.ShipmentMove{{ID: first}, {ID: last}},
		}},
		console: console,
		logger:  zap.NewNop(),
	}

	_, err := svc.Describe(t.Context(), testTenant(), agent.SubjectShipment, pulid.MustNew("shp_"))
	require.NoError(t, err)

	require.Len(t, console.filters, 1)
	filter := console.filters[0]
	assert.Equal(t, []pulid.ID{first, last}, filter.MoveIDs)
	assert.Equal(t, 2, filter.Limit)
	assert.True(t, filter.IncludeCovered)
	assert.ElementsMatch(t, shipment.MoveStatuses(), filter.MoveStatuses)
}
