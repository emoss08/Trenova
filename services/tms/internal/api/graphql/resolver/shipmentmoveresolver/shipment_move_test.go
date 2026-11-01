package shipmentmoveresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/trackingeventportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShipmentMoveResolvers_CheckTheShipmentMovePermission(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_").String()
	stopID := pulid.MustNew("stp_").String()
	assignment := gqlmodel.ShipmentMoveAssignmentInput{
		PrimaryWorkerID: pulid.MustNew("wrk_").String(),
		TractorID:       pulid.MustNew("tr_").String(),
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"assignShipmentMove", permission.OpAssign, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).AssignShipmentMove(ctx, moveID, assignment)
			return err
		}},
		{"reassignShipmentMove", permission.OpAssign, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReassignShipmentMove(ctx, moveID, assignment)
			return err
		}},
		{
			"unassignShipmentMove",
			permission.OpUnassign,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UnassignShipmentMove(ctx, moveID)
				return err
			},
		},
		{"splitShipmentMove", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SplitShipmentMove(
				ctx,
				moveID,
				gqlmodel.SplitShipmentMoveInput{},
			)
			return err
		}},
		{
			"recordShipmentStopActual",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RecordShipmentStopActual(
					ctx,
					moveID,
					stopID,
					gqlmodel.RecordShipmentStopActualInput{
						Action: shipmentmoveports.StopActualActionArrive,
					},
				)
				return err
			},
		},
		{"shipmentAssignment", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).ShipmentAssignment(ctx, pulid.MustNew("asn_").String())
			return err
		}},
		{"checkWorkerCompliance", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).CheckWorkerCompliance(
				ctx,
				gqlmodel.CheckWorkerComplianceInput{
					ShipmentMoveID:  moveID,
					PrimaryWorkerID: assignment.PrimaryWorkerID,
				},
			)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceShipmentMove,
				call.operation,
			)
		})
	}
}

func TestAssignShipmentMove_SendsTheMoveAndCrewInTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	moveID := pulid.MustNew("smv_")
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("tr_")
	trailerID := pulid.MustNew("trl_")
	assignmentID := pulid.MustNew("asn_")

	service := assignmentportsmock.NewMockAssignmentService(t)
	service.EXPECT().
		AssignToMove(mock.Anything, mock.MatchedBy(func(req *assignmentports.AssignShipmentMoveRequest) bool {
			return req.ShipmentMoveID == moveID &&
				req.PrimaryWorkerID == workerID &&
				req.TractorID == tractorID &&
				req.TrailerID != nil && *req.TrailerID == trailerID &&
				req.SecondaryWorkerID == nil &&
				req.TenantInfo.OrgID == authCtx.OrganizationID &&
				req.TenantInfo.BuID == authCtx.BusinessUnitID &&
				req.TenantInfo.UserID == authCtx.UserID
		})).
		Return(&shipment.Assignment{
			ID:              assignmentID,
			ShipmentMoveID:  moveID,
			PrimaryWorkerID: &workerID,
			TractorID:       &tractorID,
			TrailerID:       &trailerID,
			Status:          shipmentvocab.AssignmentStatusNew,
			Version:         3,
		}, nil)

	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AssignmentService: service,
	}
	trailerRaw := trailerID.String()
	result, err := (&MutationResolver{deps}).AssignShipmentMove(
		resolvertest.AuthedContext(t, authCtx),
		moveID.String(),
		gqlmodel.ShipmentMoveAssignmentInput{
			PrimaryWorkerID: workerID.String(),
			TractorID:       tractorID.String(),
			TrailerID:       &trailerRaw,
		},
	)
	require.NoError(t, err)

	require.NotNil(t, result.ID)
	assert.Equal(t, assignmentID.String(), *result.ID)
	require.NotNil(t, result.TrailerID)
	assert.Equal(t, trailerID.String(), *result.TrailerID)
	assert.Nil(t, result.SecondaryWorkerID)
	assert.Equal(t, 3, result.Version)
}

func TestRecordShipmentStopActual_RecordsADispatcherReportAndReturnsTheMove(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	moveID := pulid.MustNew("smv_")
	stopID := pulid.MustNew("stp_")
	occurredAt := 1_767_787_200

	service := trackingeventportsmock.NewMockTrackingEventService(t)
	service.EXPECT().
		RecordReported(mock.Anything, mock.MatchedBy(func(params *trackingeventports.RecordReportedStopEventParams) bool {
			return params.MoveID == moveID &&
				params.StopID == stopID &&
				params.Action == shipmentmoveports.StopActualActionDepart &&
				params.OccurredAt != nil && *params.OccurredAt == int64(occurredAt) &&
				params.Source == trackingevent.SourceDispatcher &&
				params.ReportedByID == authCtx.UserID &&
				params.TenantInfo.OrgID == authCtx.OrganizationID
		})).
		Return(&trackingeventports.StopEventResult{
			Move: &shipment.ShipmentMove{
				ID:     moveID,
				Status: shipmentvocab.MoveStatusCompleted,
				Stops:  []*shipment.Stop{{ID: stopID}},
			},
		}, nil)

	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		TrackingEventService: service,
	}
	result, err := (&MutationResolver{deps}).RecordShipmentStopActual(
		resolvertest.AuthedContext(t, authCtx),
		moveID.String(),
		stopID.String(),
		gqlmodel.RecordShipmentStopActualInput{
			Action:     shipmentmoveports.StopActualActionDepart,
			OccurredAt: &occurredAt,
		},
	)
	require.NoError(t, err)

	require.NotNil(t, result.ID)
	assert.Equal(t, moveID.String(), *result.ID)
	assert.Equal(t, gqlmodel.MoveStatusCompleted, result.Status)
	require.Len(t, result.Stops, 1)
}

func TestSplitMoveRequestFromInput_MapsBothStopWindows(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	locationID := pulid.MustNew("loc_")
	end := 1_700_003_600
	pieces := 4

	req, err := splitMoveRequestFromInput(moveID.String(), &gqlmodel.SplitShipmentMoveInput{
		NewDeliveryLocationID: locationID.String(),
		SplitPickupTimes: &gqlmodel.ShipmentMoveSplitStopTimesInput{
			ScheduledWindowStart: 1_700_000_000,
			ScheduledWindowEnd:   &end,
		},
		NewDeliveryTimes: &gqlmodel.ShipmentMoveSplitStopTimesInput{
			ScheduledWindowStart: 1_699_990_000,
		},
		Pieces: &pieces,
	})
	require.NoError(t, err)

	assert.Equal(t, moveID, req.MoveID)
	assert.Equal(t, locationID, req.NewDeliveryLocationID)
	assert.Equal(t, int64(1_700_000_000), req.SplitPickupTimes.ScheduledWindowStart)
	require.NotNil(t, req.SplitPickupTimes.ScheduledWindowEnd)
	assert.Equal(t, int64(end), *req.SplitPickupTimes.ScheduledWindowEnd)
	assert.Equal(t, int64(1_699_990_000), req.NewDeliveryTimes.ScheduledWindowStart)
	assert.Nil(t, req.NewDeliveryTimes.ScheduledWindowEnd)
	require.NotNil(t, req.Pieces)
	assert.Equal(t, int64(4), *req.Pieces)
	assert.Nil(t, req.Weight)
}

func TestAssignmentArgsFromInput_NamesAMalformedTractor(t *testing.T) {
	t.Parallel()

	_, err := assignmentArgsFromInput(
		pulid.MustNew("smv_").String(),
		&gqlmodel.ShipmentMoveAssignmentInput{
			PrimaryWorkerID: pulid.MustNew("wrk_").String(),
			TractorID:       "not-an-id",
		},
	)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "tractorId", fieldErr.Field)
}
