package dispatchconsoleresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchconsoleportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// denyingPermissionEngine refuses every check, standing in for a role that lacks the
// permission under test.
type denyingPermissionEngine struct {
	mocks.AllowAllPermissionEngine
	request *permissionports.PermissionCheckRequest
}

func (e *denyingPermissionEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.request = req
	return &permissionports.PermissionCheckResult{Allowed: false}, nil
}

func dispatchAuthContext(t *testing.T) context.Context {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
}

// Every console entry point must be gated. A read-only role must not be able to move work
// onto a driver, and the assign and unassign operations are separately permissioned so an
// organization can grant one without the other.
func TestDispatchResolvers_RequireTheCorrectPermission(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		invoke    func(ctx context.Context, r *Deps) error
	}{
		{
			name:      "board requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchBoard(ctx, gqlmodel.DispatchBoardInput{})
				return err
			},
		},
		{
			name:      "a single move requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchMove(ctx, pulid.MustNew("sm_").String())
				return err
			},
		},
		{
			name:      "move candidates require read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchMoveCandidates(
					ctx,
					gqlmodel.DispatchMoveCandidatesInput{MoveID: pulid.MustNew("sm_").String()},
				)
				return err
			},
		},
		{
			name:      "driver moves require read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchDriverMoves(
					ctx,
					gqlmodel.DispatchDriverMovesInput{WorkerID: pulid.MustNew("wrk_").String()},
				)
				return err
			},
		},
		{
			name:      "assignment preview requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchAssignmentPreview(
					ctx,
					gqlmodel.DispatchAssignmentPreviewInput{
						MoveID:   pulid.MustNew("sm_").String(),
						WorkerID: pulid.MustNew("wrk_").String(),
					},
				)
				return err
			},
		},
		{
			name:      "the freight outlook requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchFreightOutlook(ctx)
				return err
			},
		},
		{
			name:      "a freight outlook area requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchFreightArea(ctx, 12971, nil)
				return err
			},
		},
		{
			name:      "sending a driver requires positioning create",
			resource:  permission.ResourceDriverPositioning,
			operation: permission.OpCreate,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&MutationResolver{r}).DispatchPositionDriver(
					ctx,
					gqlmodel.DispatchPositionDriverInput{
						WorkerID:  pulid.MustNew("wrk_").String(),
						ToCell:    12971,
						SlotStart: 1_800_000_000,
					},
				)
				return err
			},
		},
		{
			name:      "calling a positioning off requires positioning cancel",
			resource:  permission.ResourceDriverPositioning,
			operation: permission.OpCancel,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&MutationResolver{r}).DispatchCallOffPositioning(
					ctx,
					pulid.MustNew("dpo_").String(),
				)
				return err
			},
		},
		{
			name:      "the forecast quality requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchForecastQuality(ctx)
				return err
			},
		},
		{
			name:      "assigning moves requires assign",
			operation: permission.OpAssign,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&MutationResolver{r}).DispatchAssignMoves(
					ctx,
					[]*gqlmodel.DispatchAssignMoveInput{{
						MoveID:          pulid.MustNew("sm_").String(),
						PrimaryWorkerID: pulid.MustNew("wrk_").String(),
						TractorID:       pulid.MustNew("trac_").String(),
					}},
				)
				return err
			},
		},
		{
			name:      "unassigning moves requires unassign",
			operation: permission.OpUnassign,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&MutationResolver{r}).DispatchUnassignMoves(
					ctx,
					[]string{pulid.MustNew("sm_").String()},
				)
				return err
			},
		},
		{
			// Planning is a read, but the same call applies the plan. Gating it on assign
			// stops a read-only role triggering writes through the apply flag.
			name:      "planning auto-assign requires assign",
			operation: permission.OpAssign,
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&MutationResolver{r}).DispatchPlanAutoAssign(
					ctx,
					gqlmodel.DispatchPlanInput{},
				)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &denyingPermissionEngine{}
			err := tc.invoke(
				dispatchAuthContext(t),
				&Deps{Core: &resolvercore.Core{PermissionEngine: engine}},
			)

			require.Error(t, err, "a denied permission must not fall through to the service")
			require.NotNil(t, engine.request)
			resource := tc.resource
			if resource == "" {
				resource = permission.ResourceShipmentMove
			}
			assert.Equal(t, resource.String(), engine.request.Resource)
			assert.Equal(t, tc.operation, engine.request.Operation)
		})
	}
}

// denyResourcePermissionEngine allows everything except the named resource, so a test can
// prove a resolver demands more than its primary permission.
type denyResourcePermissionEngine struct {
	mocks.AllowAllPermissionEngine
	deniedResource string
	denied         *permissionports.PermissionCheckRequest
}

func (e *denyResourcePermissionEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	if req.Resource == e.deniedResource {
		e.denied = req
		return &permissionports.PermissionCheckResult{Allowed: false}, nil
	}
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}

// The board and the reverse-match expose driver PII, live position, and HOS clocks, so
// shipment_move:read alone must not be enough.
func TestDispatchResolvers_DriverDataAlsoRequiresWorkerRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		invoke func(ctx context.Context, r *Deps) error
	}{
		{
			name: "board",
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchBoard(ctx, gqlmodel.DispatchBoardInput{})
				return err
			},
		},
		{
			name: "driver moves",
			invoke: func(ctx context.Context, r *Deps) error {
				_, err := (&QueryResolver{r}).DispatchDriverMoves(
					ctx,
					gqlmodel.DispatchDriverMovesInput{WorkerID: pulid.MustNew("wrk_").String()},
				)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &denyResourcePermissionEngine{
				deniedResource: permission.ResourceWorker.String(),
			}
			err := tc.invoke(
				dispatchAuthContext(t),
				&Deps{Core: &resolvercore.Core{PermissionEngine: engine}},
			)

			require.Error(t, err, "denying worker:read must block the query")
			require.NotNil(t, engine.denied)
			assert.Equal(t, permission.ResourceWorker.String(), engine.denied.Resource)
			assert.Equal(t, permission.OpRead, engine.denied.Operation)
		})
	}
}

func TestDispatchResolvers_RejectUnauthenticatedCallers(t *testing.T) {
	t.Parallel()

	resolver := &Deps{
		Core: &resolvercore.Core{PermissionEngine: &resolvertest.RecordingPermissionEngine{}},
	}

	_, err := (&QueryResolver{resolver}).DispatchBoard(
		t.Context(),
		gqlmodel.DispatchBoardInput{},
	)
	require.Error(t, err)
}

func TestSummarizeBulkResults_CountsBothOutcomes(t *testing.T) {
	t.Parallel()

	summary := summarizeBulkResults([]*gqlmodel.DispatchAssignResult{
		{MoveID: "a", Success: true},
		{MoveID: "b", Success: false},
		{MoveID: "c", Success: true},
	})

	assert.Equal(t, 2, summary.Succeeded)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Results, 3, "every move reports its own outcome")
}

// A failed assignment carries its validation errors back so the console renders them with
// the same badges it uses for a pre-flight.
func TestFindingsFromError_MapsValidationErrorsToFindings(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add("hos", errortypes.ErrComplianceViolation, "Driver has no drive time remaining")
	multiErr.Add("mvrDueDate", errortypes.ErrInvalid, "MVR due date has passed")

	findings := findingsFromError(multiErr)

	require.Len(t, findings, 2)
	assert.Equal(t, string(dispatchfinding.SeverityBlock), findings[0].Severity)
	assert.Equal(t, "Driver has no drive time remaining", findings[0].Message)
	assert.Equal(t, string(dispatchfinding.SeverityWarn), findings[1].Severity)
}

func TestFindingsFromError_NonValidationErrorYieldsNoFindings(t *testing.T) {
	t.Parallel()

	assert.Empty(t, findingsFromError(errortypes.NewBusinessError("something broke")))
	assert.Empty(t, findingsFromError(nil))
}

// Raw internal errors must never reach a client; only structured errortypes surface their
// message, everything else collapses to a generic failure.
func TestErrorMessage_OnlySurfacesStructuredErrors(t *testing.T) {
	t.Parallel()

	assert.Nil(t, errorMessage(nil))

	business := errorMessage(errortypes.NewBusinessError("Move cannot be assigned"))
	require.NotNil(t, business)
	assert.Equal(t, "Move cannot be assigned", *business)

	notFound := errorMessage(errortypes.NewNotFoundError("Shipment move not found"))
	require.NotNil(t, notFound)
	assert.Equal(t, "Shipment move not found", *notFound)

	multiErr := errortypes.NewMultiError()
	multiErr.Add("hos", errortypes.ErrComplianceViolation, "No drive time remaining")
	structured := errorMessage(multiErr)
	require.NotNil(t, structured)
	assert.Contains(t, *structured, "No drive time remaining")

	internal := errorMessage(errors.New("pq: connection refused to 10.0.0.5:5432"))
	require.NotNil(t, internal)
	assert.Equal(t, genericDispatchErrorMessage, *internal)
	assert.NotContains(t, *internal, "10.0.0.5")
}

func TestParseDispatchID_ReturnsASafeValidationError(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("sm_")
	parsed, err := parseDispatchID(id.String(), "moveId", "Move ID is not a valid identifier")
	require.NoError(t, err)
	assert.Equal(t, id, parsed)

	_, err = parseDispatchID("garbage", "moveId", "Move ID is not a valid identifier")
	require.Error(t, err)
	message := errorMessage(err)
	require.NotNil(t, message)
	assert.Equal(t, "Move ID is not a valid identifier", *message)
}

type boardService struct {
	dispatchconsoleports.DispatchConsoleService

	board *dispatchconsoleservice.Board
}

func (s *boardService) GetBoard(
	context.Context,
	*dispatchconsoleservice.GetBoardRequest,
) (*dispatchconsoleservice.Board, error) {
	return s.board, nil
}

func boardOf(moves ...*dispatchconsoleservice.BoardMove) *dispatchconsoleservice.Board {
	return &dispatchconsoleservice.Board{
		Moves:   moves,
		Drivers: []*dispatchconsoleservice.BoardDriver{},
		Summary: &dispatchboardports.BoardSummary{},
	}
}

func boardMove(covered bool, version int64) *dispatchconsoleservice.BoardMove {
	return &dispatchconsoleservice.BoardMove{
		BoardMove: &dispatchboardports.BoardMove{
			MoveID:      pulid.MustNew("sm_"),
			MoveVersion: version,
		},
		IsCovered: covered,
	}
}

/*
The board carries each uncovered move's coverage proposals, the one waiting and the last approval
that failed, read once for the whole
board, and only for someone who may see agent proposals: a dispatcher without it sees the
moves and no proposal, rather than the board failing.
*/
func TestDispatchBoard_AttachesEachUncoveredMovesProposalsInOneRead(t *testing.T) {
	t.Parallel()

	uncovered, failed, covered := boardMove(false, 3), boardMove(false, 2), boardMove(true, 1)
	autoAssign := dispatchconsoleportsmock.NewMockDispatchAutoAssignService(t)
	workerID := pulid.MustNew("wrk_")
	failedProposalID := pulid.MustNew("ap_")
	autoAssign.EXPECT().
		CoverageProposals(mock.Anything, mock.MatchedBy(func(req *dispatchconsoleports.CoverageProposalsRequest) bool {
			return len(req.Moves) == 2 &&
				req.Moves[0].MoveID == uncovered.MoveID &&
				req.Moves[0].MoveVersion == 3 &&
				req.Moves[1].MoveID == failed.MoveID
		})).
		Return(map[pulid.ID]*dispatchconsoleports.MoveCoverageProposals{
			uncovered.MoveID: {Pending: &dispatchconsoleports.PendingCoverage{
				ProposalID: pulid.MustNew("ap_"),
				MoveID:     uncovered.MoveID,
				WorkerID:   workerID,
				WorkerName: "Dana Ortiz",
				TractorID:  pulid.MustNew("trc_"),
				Confidence: decimal.NewFromFloat(0.91),
				Stale:      true,
			}},
			failed.MoveID: {Failed: &dispatchconsoleports.FailedCoverage{
				ProposalID: failedProposalID,
				MoveID:     failed.MoveID,
				WorkerID:   workerID,
				WorkerName: "Dana Ortiz",
				Reason:     "Move 1, stop 1: Scheduled window end must be before the next stop's scheduled window start",
				FailedAt:   1_800_000_500,
			}},
		}, nil).
		Once()

	board, err := (&QueryResolver{&Deps{
		Core:                      &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		DispatchConsoleService:    &boardService{board: boardOf(uncovered, failed, covered)},
		DispatchAutoAssignService: autoAssign,
	}}).DispatchBoard(dispatchAuthContext(t), gqlmodel.DispatchBoardInput{})
	require.NoError(t, err)

	require.NotNil(t, board.Moves[0].PendingProposal)
	assert.Equal(t, workerID.String(), board.Moves[0].PendingProposal.WorkerID)
	assert.Equal(t, "Dana Ortiz", board.Moves[0].PendingProposal.WorkerName)
	assert.InDelta(t, 0.91, board.Moves[0].PendingProposal.Confidence, 0.0001)
	assert.True(t, board.Moves[0].PendingProposal.Stale)
	assert.Nil(t, board.Moves[0].FailedProposal)
	assert.Equal(t, &gqlmodel.DispatchMoveFailedProposal{
		ID:         failedProposalID.String(),
		WorkerID:   workerID.String(),
		WorkerName: "Dana Ortiz",
		Reason:     "Move 1, stop 1: Scheduled window end must be before the next stop's scheduled window start",
		FailedAt:   1_800_000_500,
	}, board.Moves[1].FailedProposal, "an approval that failed stays on its move with why")
	assert.Nil(t, board.Moves[2].PendingProposal, "a covered move waits on no proposal")
}

func TestDispatchBoard_LeavesProposalsOffForSomeoneWhoMayNotSeeThem(t *testing.T) {
	t.Parallel()

	autoAssign := dispatchconsoleportsmock.NewMockDispatchAutoAssignService(t)

	board, err := (&QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: &denyResourcePermissionEngine{
			deniedResource: permission.ResourceAgentProposal.String(),
		}},
		DispatchConsoleService:    &boardService{board: boardOf(boardMove(false, 1))},
		DispatchAutoAssignService: autoAssign,
	}}).DispatchBoard(dispatchAuthContext(t), gqlmodel.DispatchBoardInput{})
	require.NoError(t, err)

	assert.Nil(t, board.Moves[0].PendingProposal)
	autoAssign.AssertNotCalled(t, "CoverageProposals", mock.Anything, mock.Anything)
}

func allowAllCore() *resolvercore.Core {
	return &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}
}

type forecastService struct {
	demandforecastports.Service

	qualityTenant pagination.TenantInfo
	quality       *demandforecast.Quality
	refitting     bool
}

func (s *forecastService) Standing(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Standing, error) {
	s.qualityTenant = tenantInfo
	return &demandforecast.Standing{Quality: s.quality, Refitting: s.refitting}, nil
}

type outlookService struct {
	freightoutlookports.Service

	request     *freightoutlookports.Request
	areaRequest *freightoutlookports.AreaRequest
	quality     *demandforecast.Quality
}

func (s *outlookService) Outlook(
	_ context.Context,
	req *freightoutlookports.Request,
) (*freightoutlookports.Outlook, error) {
	s.request = req
	return &freightoutlookports.Outlook{
		Quality:     s.quality,
		GeneratedAt: 1_800_000_000,
		Starts:      []int64{1_800_000_000},
		Areas: []*freightoutlookports.Area{{
			Balance: demandforecast.AreaBalance{Cell: 12971, Label: "Chicago, IL"},
		}},
		AreaCount: 3,
	}, nil
}

func (s *outlookService) Area(
	_ context.Context,
	req *freightoutlookports.AreaRequest,
) (*freightoutlookports.AreaDetail, error) {
	s.areaRequest = req
	return &freightoutlookports.AreaDetail{
		Quality:   s.quality,
		Area:      &freightoutlookports.Area{Balance: demandforecast.AreaBalance{Cell: req.Cell}},
		SlotStart: 1_800_014_400,
	}, nil
}

func TestDispatchFreightOutlook_ReadsTheCallersTenant(t *testing.T) {
	t.Parallel()

	ctx := dispatchAuthContext(t)
	service := &outlookService{
		quality: &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: 1},
	}

	outlook, err := (&QueryResolver{&Deps{
		Core:                  allowAllCore(),
		FreightOutlookService: service,
	}}).DispatchFreightOutlook(ctx)
	require.NoError(t, err)

	authCtx, ok := gqlctx.AuthContext(ctx)
	require.True(t, ok)
	require.NotNil(t, service.request)
	assert.Equal(t, authCtx.OrganizationID, service.request.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, service.request.TenantInfo.BuID)
	assert.Equal(t, gqlmodel.DispatchForecastStatusReady, outlook.Quality.Status)
	assert.Equal(t, []int{1_800_000_000}, outlook.SlotStarts)
	assert.Equal(t, 3, outlook.AreaCount)
	require.Len(t, outlook.Areas, 1)
	assert.Equal(t, "Chicago, IL", outlook.Areas[0].Label)
}

func TestDispatchFreightArea_PassesTheCellAndWindowAsked(t *testing.T) {
	t.Parallel()

	service := &outlookService{quality: &demandforecast.Quality{Status: demandforecast.StatusReady}}
	resolver := &QueryResolver{&Deps{Core: allowAllCore(), FreightOutlookService: service}}

	detail, err := resolver.DispatchFreightArea(dispatchAuthContext(t), 12971, new(1_800_014_400))
	require.NoError(t, err)
	require.NotNil(t, service.areaRequest)
	assert.Equal(t, int64(12971), service.areaRequest.Cell)
	assert.Equal(t, int64(1_800_014_400), service.areaRequest.SlotStart)
	assert.Equal(t, 12971, detail.Area.Cell)

	_, err = resolver.DispatchFreightArea(dispatchAuthContext(t), 12971, nil)
	require.NoError(t, err)
	assert.Zero(
		t,
		service.areaRequest.SlotStart,
		"no window asked leaves the area's worst to the service",
	)
}

func TestDispatchForecastQuality_ReadsTheCallersTenant(t *testing.T) {
	t.Parallel()

	ctx := dispatchAuthContext(t)
	service := &forecastService{
		quality:   &demandforecast.Quality{Status: demandforecast.StatusNoSkill, Skill: 0.02},
		refitting: true,
	}

	quality, err := (&QueryResolver{&Deps{
		Core:                  allowAllCore(),
		DemandForecastService: service,
	}}).DispatchForecastQuality(ctx)
	require.NoError(t, err)

	authCtx, ok := gqlctx.AuthContext(ctx)
	require.True(t, ok)
	assert.Equal(t, authCtx.OrganizationID, service.qualityTenant.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, service.qualityTenant.BuID)
	assert.Equal(t, gqlmodel.DispatchForecastStatusNoSkill, quality.Status)
	assert.InDelta(t, 0.02, quality.Skill, 0.0001)
	assert.Nil(t, quality.FittedAt)
	assert.True(t, quality.Refitting)
}

func TestDispatchForecastResolvers_DeniedReadNeverReachesTheService(t *testing.T) {
	t.Parallel()

	service := &forecastService{}
	outlook := &outlookService{}
	resolver := &QueryResolver{&Deps{
		Core:                  &resolvercore.Core{PermissionEngine: &denyingPermissionEngine{}},
		DemandForecastService: service,
		FreightOutlookService: outlook,
	}}

	_, err := resolver.DispatchFreightOutlook(dispatchAuthContext(t))
	require.Error(t, err)
	_, err = resolver.DispatchFreightArea(dispatchAuthContext(t), 12971, nil)
	require.Error(t, err)
	_, err = resolver.DispatchForecastQuality(dispatchAuthContext(t))
	require.Error(t, err)

	assert.Nil(t, outlook.request)
	assert.Nil(t, outlook.areaRequest)
	assert.True(t, service.qualityTenant.OrgID.IsNil())
}
