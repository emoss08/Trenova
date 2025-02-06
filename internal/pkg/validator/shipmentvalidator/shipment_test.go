package shipmentvalidator_test

import (
	"context"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	spValidator "github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/shopspring/decimal"
)

var (
	ts  *testutils.TestSetup
	ctx = context.Background()
)

func TestMain(m *testing.M) {
	setup, err := testutils.NewTestSetup(ctx)
	if err != nil {
		panic(err)
	}

	ts = setup

	os.Exit(m.Run())
}

func newShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ProNumber:           "123456",
		Status:              shipment.StatusNew,
		ShipmentTypeID:      pulid.MustNew("st_"),
		CustomerID:          pulid.MustNew("cust_"),
		BOL:                 "1234567890",
		RatingMethod:        shipment.RatingMethodFlatRate,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		ReadyToBill:         false,
		Moves: []*shipment.ShipmentMove{
			{
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
			},
		},
	}
}

func TestShipmentValidator(t *testing.T) { //nolint: funlen // Tests
	sv := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB: ts.DB,
	})

	mv := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB:            ts.DB,
		StopValidator: sv,
	})

	val := spValidator.NewValidator(spValidator.ValidatorParams{
		DB:            ts.DB,
		MoveValidator: mv,
	})

	scenarios := []struct {
		name           string
		modifyShipment func(*shipment.Shipment)
		expectedErrors []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "cannot mark ready to bill when status is not completed",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Status = shipment.StatusNew
				shp.ReadyToBill = true
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "readyToBill",
					Code:    errors.ErrInvalid,
					Message: "Shipment must be completed to be marked as ready to bill",
				},
				{
					Field:   "actualDeliveryDate",
					Code:    errors.ErrInvalid,
					Message: "Actual delivery date is required to mark shipment as ready to bill",
				},
			},
		},
		{
			name: "customer is required",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.CustomerID = pulid.Nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "customerId",
					Code:    errors.ErrRequired,
					Message: "Customer is required",
				},
			},
		},
		{
			name: "shipment type is required",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.ShipmentTypeID = pulid.Nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "shipmentTypeId",
					Code:    errors.ErrRequired,
					Message: "Shipment Type is required",
				},
			},
		},
		{
			name: "bol is required",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.BOL = ""
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "bol",
					Code:    errors.ErrRequired,
					Message: "BOL is required",
				},
			},
		},
		{
			name: "freight charge amount is required when rating method is flat",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.RatingMethod = shipment.RatingMethodFlatRate
				shp.FreightChargeAmount = decimal.NullDecimal{}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "freightChargeAmount",
					Code:    errors.ErrRequired,
					Message: "Freight Charge Amount is required when rating method is Flat",
				},
			},
		},
		{
			name: "weight is required when rating method is per pound",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.RatingMethod = shipment.RatingMethodPerPound
				shp.Weight = nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "weight",
					Code:    errors.ErrRequired,
					Message: "Weight is required when rating method is Per Pound",
				},
			},
		},
		{
			name: "rating unit is required when rating method is per mile",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.RatingMethod = shipment.RatingMethodPerMile
				shp.RatingUnit = 0
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "ratingUnit",
					Code:    errors.ErrRequired,
					Message: "Rating Unit is required when rating method is Per Mile",
				},
			},
		},
		{
			name: "shipment must have at last one move",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Moves = []*shipment.ShipmentMove{}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "moves",
					Code:    errors.ErrInvalid,
					Message: "Shipment must have at least one move",
				},
			},
		},
		{
			name: "shipment ready to bill state validation",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.ReadyToBill = false

				// Failure case
				shp.ReadyToBillDate = &[]int64{100}[0]
				shp.SentToBilling = true
				shp.SentToBillingDate = &[]int64{100}[0]
				shp.Billed = true
				shp.BillDate = &[]int64{100}[0]
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "readyToBillDate",
					Code:    errors.ErrInvalid,
					Message: "Ready to bill date cannot be set when shipment is not ready to bill",
				},
				{
					Field:   "sentToBilling",
					Code:    errors.ErrInvalid,
					Message: "Cannot be sent to billing when shipment is not ready to bill",
				},
				{
					Field:   "sentToBillingDate",
					Code:    errors.ErrInvalid,
					Message: "Sent to billing date cannot be set when shipment is not ready to bill",
				},
				{
					Field:   "billDate",
					Code:    errors.ErrInvalid,
					Message: "Bill date cannot be set when shipment is not ready to bill",
				},
				{
					Field:   "billed",
					Code:    errors.ErrInvalid,
					Message: "Cannot be marked as billed when shipment is not ready to bill",
				},
			},
		},
		{
			name: "shipment sent to billing state validation",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.SentToBilling = false
				shp.Status = shipment.StatusCompleted
				shp.ReadyToBill = true
				shp.ReadyToBillDate = &[]int64{100}[0]
				shp.ActualDeliveryDate = &[]int64{100}[0]

				// Failure case
				shp.SentToBillingDate = &[]int64{100}[0]
				shp.Billed = true
				shp.BillDate = &[]int64{100}[0]
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "sentToBillingDate",
					Code:    errors.ErrInvalid,
					Message: "Sent to billing date cannot be set when not sent to billing",
				},
				{
					Field:   "billed",
					Code:    errors.ErrInvalid,
					Message: "Cannot be marked as billed when not sent to billing",
				},
				{
					Field:   "billDate",
					Code:    errors.ErrInvalid,
					Message: "Bill date cannot be set when not sent to billing",
				},
			},
		},
		{
			name: "shipment billed state validation",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.SentToBilling = false
				shp.Status = shipment.StatusCompleted
				shp.ReadyToBill = true
				shp.ReadyToBillDate = &[]int64{100}[0]
				shp.ActualDeliveryDate = &[]int64{100}[0]
				shp.SentToBilling = true
				shp.SentToBillingDate = &[]int64{100}[0]

				// Failure case
				shp.Billed = false
				shp.BillDate = &[]int64{100}[0]
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "billDate",
					Code:    errors.ErrInvalid,
					Message: "Bill date cannot be set when not billed",
				},
			},
		},
		{
			name: "shipment date sequence validation",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Status = shipment.StatusCompleted
				shp.SentToBilling = true
				shp.ReadyToBill = true
				shp.ActualDeliveryDate = &[]int64{100}[0]
				shp.Billed = true

				// Failure case
				shp.ReadyToBillDate = &[]int64{500}[0]
				shp.BillDate = &[]int64{100}[0]
				shp.SentToBillingDate = &[]int64{400}[0]
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "sentToBillingDate",
					Code:    errors.ErrInvalid,
					Message: "Sent to billing date cannot be before ready to bill date",
				},
				{
					Field:   "billDate",
					Code:    errors.ErrInvalid,
					Message: "Bill date cannot be before sent to billing date",
				},
			},
		},
		{
			name: "shipment charge amount validation",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Status = shipment.StatusCompleted
				shp.SentToBilling = true
				shp.ReadyToBill = true
				shp.ActualDeliveryDate = &[]int64{100}[0]
				shp.Billed = true

				// Failure case
				shp.FreightChargeAmount = decimal.NullDecimal{}
				shp.OtherChargeAmount = decimal.NullDecimal{}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "freightChargeAmount",
					Code:    errors.ErrRequired,
					Message: "Freight charge amount is required when shipment is billed",
				},
				{
					Field:   "freightChargeAmount",
					Code:    errors.ErrRequired,
					Message: "Freight Charge Amount is required when rating method is Flat",
				},
			},
		},
		{
			name: "validate total charge amount",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Status = shipment.StatusCompleted
				shp.SentToBilling = true
				shp.ReadyToBill = true
				shp.ActualDeliveryDate = &[]int64{100}[0]
				shp.Billed = true

				// Failure case
				shp.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1000))
				shp.OtherChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1000))
				shp.TotalChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1000))
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "totalChargeAmount",
					Code:    errors.ErrInvalid,
					Message: "Total charge amount must equal freight charge plus other charges",
				},
			},
		},
		{
			name: "shipment must be delivered before ready to bill",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.Status = shipment.StatusCompleted
				shp.SentToBilling = true

				// Failure case
				shp.ReadyToBill = true
				shp.ActualDeliveryDate = nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "actualDeliveryDate",
					Code:    errors.ErrInvalid,
					Message: "Actual delivery date is required to mark shipment as ready to bill",
				},
			},
		},
		{
			name: "temperature min must be less than temperature max",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.TemperatureMin = decimal.NewNullDecimal(decimal.NewFromInt(100))
				shp.TemperatureMax = decimal.NewNullDecimal(decimal.NewFromInt(99))
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "temperatureMin",
					Code:    errors.ErrInvalid,
					Message: "Temperature Min must be less than Temperature Max",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			vCtx := validator.NewValidationContext(ctx, &validator.ValidationContext{
				IsCreate: true,
			})

			shp := newShipment()

			scenario.modifyShipment(shp)

			me := val.Validate(ctx, vCtx, shp)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}

func TestShipmentCancelValidation(t *testing.T) {
	sv := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB: ts.DB,
	})

	mv := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB:            ts.DB,
		StopValidator: sv,
	})

	val := spValidator.NewValidator(spValidator.ValidatorParams{
		DB:            ts.DB,
		MoveValidator: mv,
	})
	scenarios := []struct {
		name           string
		modifyShipment func(*shipment.Shipment)
		expectedErrors []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "cannot cancel shipment in status completed",
			modifyShipment: func(s *shipment.Shipment) {
				s.Status = shipment.StatusCompleted
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "status",
					Code:    errors.ErrInvalid,
					Message: "Cannot cancel shipment in status `Completed`",
				},
			},
		},
		{
			name: "cannot cancel shipment in status billed",
			modifyShipment: func(s *shipment.Shipment) {
				s.Status = shipment.StatusBilled
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "status",
					Code:    errors.ErrInvalid,
					Message: "Cannot cancel shipment in status `Billed`",
				},
			},
		},
		{
			name: "cannot cancel shipment in status canceled",
			modifyShipment: func(s *shipment.Shipment) {
				s.Status = shipment.StatusCanceled
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "status",
					Code:    errors.ErrInvalid,
					Message: "Cannot cancel shipment in status `Canceled`",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			shp := newShipment()

			scenario.modifyShipment(shp)

			me := val.ValidateCancel(shp)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
