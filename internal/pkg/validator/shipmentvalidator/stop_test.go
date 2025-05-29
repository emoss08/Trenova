package shipmentvalidator_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	spValidator "github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/test/testutils"
)

func newStop() *shipment.Stop {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	sm := ts.Fixture.MustRow("ShipmentMove.test_shipment_move").(*shipment.ShipmentMove)

	return &shipment.Stop{
		OrganizationID:   org.ID,
		BusinessUnitID:   bu.ID,
		Type:             shipment.StopTypePickup,
		Sequence:         1,
		Status:           shipment.StopStatusNew,
		ShipmentMoveID:   sm.ID,
		ShipmentMove:     sm,
		PlannedArrival:   100,
		PlannedDeparture: 200,
	}
}

func TestStopValidator(t *testing.T) {
	log := testutils.NewTestLogger(t)

	// Create a real validation engine factory (not mock)
	vef := framework.ProvideValidationEngineFactory()

	stopRepo := repositories.NewStopRepository(repositories.StopRepositoryParams{
		Logger: log,
		DB:     ts.DB,
	})

	shipmentRepo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger: log,
		DB:     ts.DB,
	})

	shipmentControlRepo := repositories.NewShipmentControlRepository(
		repositories.ShipmentControlRepositoryParams{
			Logger: log,
			DB:     ts.DB,
		},
	)

	moveRepo := repositories.NewShipmentMoveRepository(repositories.ShipmentMoveRepositoryParams{
		Logger:                    log,
		DB:                        ts.DB,
		StopRepository:            stopRepo,
		ShipmentControlRepository: shipmentControlRepo,
	})

	assignmentRepo := repositories.NewAssignmentRepository(repositories.AssignmentRepositoryParams{
		DB:           ts.DB,
		Logger:       log,
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
	})

	val := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB:                      ts.DB,
		MoveRepo:                moveRepo,
		Logger:                  log,
		AssignmentRepo:          assignmentRepo,
		ValidationEngineFactory: vef,
	})

	scenarios := []struct {
		name           string
		modifyStop     func(*shipment.Stop)
		expectedErrors []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "type is required",
			modifyStop: func(s *shipment.Stop) {
				s.Type = ""
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "stops[0].type", Code: errors.ErrRequired, Message: "Type is required"},
			},
		},
		{
			name: "status is required",
			modifyStop: func(s *shipment.Stop) {
				s.Status = ""
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "stops[0].status", Code: errors.ErrRequired, Message: "Status is required"},
			},
		},
		{
			name: "planned arrival is required",
			modifyStop: func(s *shipment.Stop) {
				s.PlannedArrival = 0
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].plannedArrival",
					Code:    errors.ErrRequired,
					Message: "Planned arrival is required",
				},
			},
		},
		{
			name: "planned times are invalid",
			modifyStop: func(s *shipment.Stop) {
				s.PlannedDeparture = 0
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].plannedDeparture",
					Code:    errors.ErrRequired,
					Message: "Planned departure is required",
				},
				{
					Field:   "stops[0].plannedArrival",
					Code:    errors.ErrInvalid,
					Message: "Planned arrival must be before planned departure",
				},
			},
		},
		{
			name: "arrival times are invalid",
			modifyStop: func(s *shipment.Stop) {
				arrival := int64(200)
				departure := int64(100)

				s.ActualArrival = &arrival
				s.ActualDeparture = &departure
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival must be before actual departure",
				},
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
		{
			name: "actual times cannot be set on a stop with no assignment",
			modifyStop: func(s *shipment.Stop) {
				s.ActualArrival = intutils.SafeInt64Ptr(100, true)
				s.ActualDeparture = intutils.SafeInt64Ptr(200, true)
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
		{
			name: "actual arrival time cannot be in the future",
			modifyStop: func(s *shipment.Stop) {
				futureTime := timeutils.NowUnix() + 3600 // 1 hour in the future
				s.ActualArrival = &futureTime
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival time cannot be in the future",
				},
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
		{
			name: "actual departure time cannot be in the future",
			modifyStop: func(s *shipment.Stop) {
				futureTime := timeutils.NowUnix() + 3600 // 1 hour in the future
				s.ActualDeparture = &futureTime
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual departure time cannot be in the future",
				},
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
		{
			name: "both actual arrival and departure times cannot be in the future",
			modifyStop: func(s *shipment.Stop) {
				futureArrival := timeutils.NowUnix() + 3600   // 1 hour in the future
				futureDeparture := timeutils.NowUnix() + 7200 // 2 hours in the future
				s.ActualArrival = &futureArrival
				s.ActualDeparture = &futureDeparture
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival time cannot be in the future",
				},
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual departure time cannot be in the future",
				},
				{
					Field:   "stops[0].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
				{
					Field:   "stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			stop := newStop()
			scenario.modifyStop(stop)

			me := errors.NewMultiError()

			val.Validate(ctx, stop, 0, me)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
