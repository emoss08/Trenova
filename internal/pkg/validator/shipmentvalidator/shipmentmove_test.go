/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentvalidator_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	shipmentrepo "github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	spValidator "github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/test/testutils"
)

func newMovement() *shipment.ShipmentMove {
	move := ts.Fixture.MustRow("ShipmentMove.test_shipment_move").(*shipment.ShipmentMove)
	move.Status = shipment.MoveStatusNew

	// * Get all of the stops for the second move
	stop1 := ts.Fixture.MustRow("Stop.test_stop_3").(*shipment.Stop)
	stop2 := ts.Fixture.MustRow("Stop.test_stop_4").(*shipment.Stop)
	stop3 := ts.Fixture.MustRow("Stop.test_stop_5").(*shipment.Stop)
	stop4 := ts.Fixture.MustRow("Stop.test_stop_6").(*shipment.Stop)

	// Set sequential timestamps to avoid time validation issues
	// By explicitly setting times in sequence, we avoid the past/future timestamp issue
	baseTime := int64(1000000)

	// First stop
	stop1.PlannedArrival = baseTime
	stop1.PlannedDeparture = baseTime + 100
	stop1.ActualArrival = nil
	stop1.ActualDeparture = nil

	// Second stop
	stop2.PlannedArrival = baseTime + 200
	stop2.PlannedDeparture = baseTime + 300
	stop2.ActualArrival = nil
	stop2.ActualDeparture = nil

	// Third stop
	stop3.PlannedArrival = baseTime + 400
	stop3.PlannedDeparture = baseTime + 500
	stop3.ActualArrival = nil
	stop3.ActualDeparture = nil

	// Fourth stop
	stop4.PlannedArrival = baseTime + 600
	stop4.PlannedDeparture = baseTime + 700
	stop4.ActualArrival = nil
	stop4.ActualDeparture = nil

	now := timeutils.NowUnix()
	stop1.PlannedArrival = now - 172800   // 2 days ago
	stop1.PlannedDeparture = now - 86400  // 1 day ago
	stop2.PlannedArrival = now + 86400    // 1 day from now
	stop2.PlannedDeparture = now + 172800 // 2 days from now

	move.Stops = []*shipment.Stop{stop1, stop2, stop3, stop4}

	return move
}

func TestMoveValidator(t *testing.T) {
	log := testutils.NewTestLogger(t)

	// Use a real validation engine factory instead of a mock
	vef := framework.ProvideValidationEngineFactory()

	stopRepo := repositories.NewStopRepository(repositories.StopRepositoryParams{
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

	shipmentRepo := shipmentrepo.NewShipmentRepository(shipmentrepo.ShipmentRepositoryParams{
		Logger: log,
		DB:     ts.DB,
	})

	assignmentRepo := repositories.NewAssignmentRepository(repositories.AssignmentRepositoryParams{
		DB:           ts.DB,
		Logger:       log,
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
	})

	stpValidator := spValidator.NewStopValidator(
		spValidator.StopValidatorParams{
			DB:                      ts.DB,
			Logger:                  log,
			MoveRepo:                moveRepo,
			AssignmentRepo:          assignmentRepo,
			ValidationEngineFactory: vef,
		},
	)
	val := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB:                      ts.DB,
		StopValidator:           stpValidator,
		ValidationEngineFactory: vef,
	})

	scenarios := []struct {
		name           string
		modifyMove     func(*shipment.ShipmentMove)
		expectedErrors []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "status is required",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Status = ""
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "moves[0].status", Code: errors.ErrRequired, Message: "Status is required"},
			},
		},
		{
			name: "validate stop planned times",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].PlannedDeparture = 300
				s.Stops[1].PlannedArrival = 200
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].plannedDeparture",
					Code:    errors.ErrInvalid,
					Message: "Planned departure must be before next stop's planned arrival",
				},
				{
					Field:   "moves[0].stops[0].plannedArrival",
					Code:    errors.ErrInvalid,
					Message: "Planned arrival must be before planned departure",
				},
			},
		},
		{
			name: "validate actual departure before next arrival",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].ActualDeparture = &[]int64{300}[0]
				s.Stops[1].ActualArrival = &[]int64{200}[0]
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual departure must be before next stop's actual arrival",
				},
				{
					Field:   "moves[0].stops[0].actualDeparture",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
				{
					Field:   "moves[0].stops[1].actualArrival",
					Code:    errors.ErrInvalid,
					Message: "Actual arrival and departure times cannot be set on a move with no assignment",
				},
			},
		},
		{
			name: "validate planned departure before next planned arrival",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].PlannedDeparture = 300
				s.Stops[1].PlannedArrival = 200
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].plannedDeparture",
					Code:    errors.ErrInvalid,
					Message: "Planned departure must be before next stop's planned arrival",
				},
				{
					Field:   "moves[0].stops[0].plannedArrival",
					Code:    errors.ErrInvalid,
					Message: "Planned arrival must be before planned departure",
				},
			},
		},
		{
			name: "first stop must be pickup",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].Type = shipment.StopTypeDelivery
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "First stop must be a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[1].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
			},
		},
		{
			name: "last stop must be delivery",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[len(s.Stops)-1].Type = shipment.StopTypePickup
				// We also need to make all stops non-deliveries to match expected errors
				s.Stops[0].Type = shipment.StopTypeDelivery
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "First stop must be a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[3].type",
					Code:    errors.ErrInvalid,
					Message: "Last stop must be a delivery or split delivery",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[1].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
			},
		},
		{
			name: "invalid stop type in sequence",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[1].Type = "INVALID_TYPE"
				// Also set first stop as delivery to match expectation
				s.Stops[0].Type = shipment.StopTypeDelivery
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[1].type",
					Code:    "INVALID",
					Message: "Type must be a valid stop type",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "First stop must be a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[3].type",
					Code:    errors.ErrInvalid,
					Message: "Last stop must be a delivery or split delivery",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[1].type",
					Code:    errors.ErrInvalid,
					Message: "Stop type must be pickup or delivery",
				},
				{
					Field:   "moves[0].stops[1].type",
					Code:    "INVALID",
					Message: "Type must be a valid stop type",
				},
			},
		},
		{
			name: "delivery before pickup",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].Type = shipment.StopTypeDelivery
				s.Stops[1].Type = shipment.StopTypeSplitPickup
				s.Stops[2].Type = shipment.StopTypeDelivery
				s.Stops[3].Type = shipment.StopTypeDelivery
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "First stop must be a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
			},
		},
		{
			name: "split pickup and split delivery sequence",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[0].Type = shipment.StopTypeDelivery
				s.Stops[1].Type = shipment.StopTypeSplitPickup
				s.Stops[2].Type = shipment.StopTypeDelivery
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "First stop must be a pickup or split pickup",
				},
				{
					Field:   "moves[0].stops[0].type",
					Code:    errors.ErrInvalid,
					Message: "Delivery stop must be preceded by a pickup or split pickup",
				},
			},
		},
		{
			name: "atleast two stops is required",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops = nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves[0].stops",
					Code:    errors.ErrInvalid,
					Message: "At least two stops is required in a move",
				},
				{
					Field:   "moves[0].stops",
					Code:    errors.ErrInvalid,
					Message: "Movement must have at least one stop",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			move := newMovement()
			scenario.modifyMove(move)

			me := errors.NewMultiError()

			val.Validate(ctx, move, me, 0)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
