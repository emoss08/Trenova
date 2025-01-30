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
	}
}

func TestShipmentValidator(t *testing.T) {
	val := spValidator.NewValidator(spValidator.ValidatorParams{
		DB: ts.DB,
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
