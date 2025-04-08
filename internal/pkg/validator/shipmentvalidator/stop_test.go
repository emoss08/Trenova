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

func newStop() *shipment.Stop {
	return &shipment.Stop{
		Type:             shipment.StopTypePickup,
		Sequence:         1,
		Status:           shipment.StopStatusNew,
		PlannedArrival:   100,
		PlannedDeparture: 200,
	}
}

func TestStopValidator(t *testing.T) {
	val := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB: ts.DB,
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
				{Field: "stops[0].plannedArrival", Code: errors.ErrRequired, Message: "Planned arrival is required"},
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
				{Field: "stops[0].plannedDeparture", Code: errors.ErrRequired, Message: "Planned departure is required"},
				{Field: "stops[0].plannedArrival", Code: errors.ErrInvalid, Message: "Planned arrival must be before planned departure"},
				{Field: "stops[0].plannedDeparture", Code: errors.ErrInvalid, Message: "Planned departure must be after planned arrival"},
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
				{Field: "stops[0].actualArrival", Code: errors.ErrInvalid, Message: "Actual arrival must be before actual departure"},
				{Field: "stops[0].actualDeparture", Code: errors.ErrInvalid, Message: "Actual departure must be after actual arrival"},
			},
		},
		{
			name: "no id on create",
			modifyStop: func(s *shipment.Stop) {
				s.ID = pulid.MustNew("stp_")
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{Field: "stops[0].id", Code: errors.ErrInvalid, Message: "ID cannot be set on create"},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			stop := newStop()
			scenario.modifyStop(stop)

			vCtx := validator.NewValidationContext(&validator.ValidationContext{
				IsCreate: true,
			})

			me := errors.NewMultiError()

			val.Validate(ctx, vCtx, stop, spValidator.WithIndexedMultiError(me, 0))

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
