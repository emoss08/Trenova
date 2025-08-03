/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentvalidator_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	shipmentrepo "github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/hazmatsegreationrulevalidator"
	spValidator "github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/mocks"
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
	shp := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)

	// * Set Random data on the shipment
	shp.ProNumber = fmt.Sprintf("S%d", rand.Intn(100000))
	shp.RatingUnit = 100
	shp.BOL = fmt.Sprintf("TEST%d", rand.Intn(100000))
	shp.CustomerID = ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer).ID
	shp.ShipmentTypeID = ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType).ID
	shp.ServiceTypeID = ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType).ID

	// * Get all of the moves for the shipment
	move1 := ts.Fixture.MustRow("ShipmentMove.test_shipment_move").(*shipment.ShipmentMove)
	move2 := ts.Fixture.MustRow("ShipmentMove.test_shipment_move_2").(*shipment.ShipmentMove)
	move3 := ts.Fixture.MustRow("ShipmentMove.test_shipment_move_3").(*shipment.ShipmentMove)

	// * Set the moves on the shipment
	shp.Moves = []*shipment.ShipmentMove{move1, move2, move3}

	// * Get all of the stops for the first move
	stop1 := ts.Fixture.MustRow("Stop.test_stop").(*shipment.Stop)
	stop2 := ts.Fixture.MustRow("Stop.test_stop_2").(*shipment.Stop)
	move1.Stops = []*shipment.Stop{stop1, stop2}

	// * Get all of the stops for the second move
	stop3 := ts.Fixture.MustRow("Stop.test_stop_3").(*shipment.Stop)
	stop4 := ts.Fixture.MustRow("Stop.test_stop_4").(*shipment.Stop)
	stop5 := ts.Fixture.MustRow("Stop.test_stop_5").(*shipment.Stop)
	stop6 := ts.Fixture.MustRow("Stop.test_stop_6").(*shipment.Stop)
	move2.Stops = []*shipment.Stop{stop3, stop4, stop5, stop6}

	// * Get all of the stops for the third move
	stop7 := ts.Fixture.MustRow("Stop.test_stop_7").(*shipment.Stop)
	stop8 := ts.Fixture.MustRow("Stop.test_stop_8").(*shipment.Stop)

	// Fix the timing issue between stop7 and stop8
	// In the fixture, stop7 uses daysAgo and stop8 uses daysFromNow
	// which creates a validation issue due to how Unix timestamps work
	// Adjust the timestamps to ensure stop7's departure is before stop8's arrival
	now := timeutils.NowUnix()
	stop7.PlannedArrival = now - 172800   // 2 days ago
	stop7.PlannedDeparture = now - 86400  // 1 day ago
	stop8.PlannedArrival = now + 86400    // 1 day from now
	stop8.PlannedDeparture = now + 172800 // 2 days from now

	move3.Stops = []*shipment.Stop{stop7, stop8}

	return shp
}

func TestShipmentValidator(t *testing.T) {
	log := testutils.NewTestLogger(t)
	mockVef := &mocks.MockValidationEngineFactory{}

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

	sv := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB:                      ts.DB,
		MoveRepo:                moveRepo,
		Logger:                  log,
		AssignmentRepo:          assignmentRepo,
		ValidationEngineFactory: mockVef,
	})

	mv := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB:                      ts.DB,
		StopValidator:           sv,
		ValidationEngineFactory: mockVef,
	})

	hs := hazmatsegreationrulevalidator.NewValidator(hazmatsegreationrulevalidator.ValidatorParams{
		DB: ts.DB,
	})

	val := spValidator.NewValidator(spValidator.ValidatorParams{
		DB:                         ts.DB,
		MoveValidator:              mv,
		ShipmentControlRepo:        shipmentControlRepo,
		HazmatSegregationValidator: hs,
		ValidationEngineFactory:    mockVef,
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
			name: "shipment must have at least one move",
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
			name: "temperature min must be less than temperature max",
			modifyShipment: func(shp *shipment.Shipment) {
				shp.TemperatureMin = intutils.SafeInt16Ptr(100, true)
				shp.TemperatureMax = intutils.SafeInt16Ptr(99, true)
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
				{
					Field:   "temperatureMax",
					Code:    errors.ErrInvalid,
					Message: "Temperature Max must be greater than Temperature Min",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			vCtx := validator.NewValidationContext(&validator.ValidationContext{
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
	log := testutils.NewTestLogger(t)

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

	sv := spValidator.NewStopValidator(spValidator.StopValidatorParams{
		DB:             ts.DB,
		MoveRepo:       moveRepo,
		Logger:         log,
		AssignmentRepo: assignmentRepo,
	})

	mv := spValidator.NewMoveValidator(spValidator.MoveValidatorParams{
		DB:            ts.DB,
		StopValidator: sv,
	})

	hs := hazmatsegreationrulevalidator.NewValidator(hazmatsegreationrulevalidator.ValidatorParams{
		DB: ts.DB,
	})

	// Create a mock validation engine factory
	mockVef := &mocks.MockValidationEngineFactory{}

	val := spValidator.NewValidator(spValidator.ValidatorParams{
		DB:                         ts.DB,
		MoveValidator:              mv,
		ShipmentControlRepo:        shipmentControlRepo,
		HazmatSegregationValidator: hs,
		ValidationEngineFactory:    mockVef,
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
					Field:   "__all__",
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
					Field:   "__all__",
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

			me := val.ValidateCancellation(shp)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
