package shipmentvalidator_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	spValidator "github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
)

func newMovement() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		Status: shipment.MoveStatusNew,
		Stops: []*shipment.Stop{
			{
				Type:             shipment.StopTypePickup,
				Sequence:         0,
				Status:           shipment.StopStatusNew,
				PlannedArrival:   100,
				PlannedDeparture: 200,
			},
			{
				Type:             shipment.StopTypePickup,
				Sequence:         1,
				Status:           shipment.StopStatusNew,
				PlannedArrival:   300,
				PlannedDeparture: 400,
			},
			{
				Type:             shipment.StopTypeDelivery,
				Sequence:         2,
				Status:           shipment.StopStatusNew,
				PlannedArrival:   500,
				PlannedDeparture: 600,
			},
			{
				Type:             shipment.StopTypeDelivery,
				Sequence:         3,
				Status:           shipment.StopStatusNew,
				PlannedArrival:   700,
				PlannedDeparture: 800,
			},
		},
	}
}

func TestMoveValidator(t *testing.T) {
	val := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB: ts.DB,
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
				{Field: "moves[0].stops[0].plannedDeparture", Code: errors.ErrInvalid, Message: "Planned departure must be before next stop's planned arrival"},
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
				{Field: "moves[0].stops[0].actualDeparture", Code: errors.ErrInvalid, Message: "Actual departure must be before next stop's actual arrival"},
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
				{Field: "moves[0].stops[0].plannedDeparture", Code: errors.ErrInvalid, Message: "Planned departure must be before next stop's planned arrival"},
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
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "First stop must be a pickup or split pickup"},
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "Delivery stop must be preceded by a pickup or split pickup"},
			},
		},
		{
			name: "last stop must be delivery",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[len(s.Stops)-1].Type = shipment.StopTypePickup
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "moves[0].stops[3].type", Code: errors.ErrInvalid, Message: "Last stop must be a delivery or split delivery"},
			},
		},
		{
			name: "invalid stop type in sequence",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.Stops[1].Type = "INVALID_TYPE"
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "moves[0].stops[1].type", Code: errors.ErrInvalid, Message: "Stop type must be pickup or delivery"},
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
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "First stop must be a pickup or split pickup"},
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "Delivery stop must be preceded by a pickup or split pickup"},
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
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "First stop must be a pickup or split pickup"},
				{Field: "moves[0].stops[0].type", Code: errors.ErrInvalid, Message: "Delivery stop must be preceded by a pickup or split pickup"},
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
				{Field: "moves[0].stops", Code: errors.ErrInvalid, Message: "At least two stops is required in a move"},
				{Field: "moves[0].stops", Code: errors.ErrInvalid, Message: "Movement must have at least one stop"},
			},
		},
		{
			name: "no id on create",
			modifyMove: func(s *shipment.ShipmentMove) {
				s.ID = pulid.MustNew("sm_")
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "moves[0].id", Code: errors.ErrInvalid, Message: "ID cannot be set on create"},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			move := newMovement()
			scenario.modifyMove(move)

			vCtx := validator.NewValidationContext(ctx, &validator.ValidationContext{
				IsCreate: true,
			})

			me := errors.NewMultiError()

			val.Validate(ctx, vCtx, move, me, 0)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
