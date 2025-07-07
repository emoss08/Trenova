package consolidationvalidator_test

import (
	"context"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	cgValidator "github.com/emoss08/trenova/internal/pkg/validator/consolidationvalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
)

var (
	ts  *testutils.TestSetup
	ctx context.Context
)

func TestMain(m *testing.M) {
	ctx = context.Background()

	var err error
	ts, err = testutils.NewTestSetup(ctx)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	ts.Cleanup()
	os.Exit(code)
}

func newConsolidationGroup() *consolidation.ConsolidationGroup {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)

	return &consolidation.ConsolidationGroup{
		ID:                  pulid.MustNew("cg_"),
		OrganizationID:      org.ID,
		BusinessUnitID:      bu.ID,
		ConsolidationNumber: "TEST-CG-001",
		Status:              consolidation.GroupStatusNew,
		Shipments:           []*shipment.Shipment{},
	}
}

func newShipment() *shipment.Shipment {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)

	return &shipment.Shipment{
		ID:                   pulid.MustNew("shp_"),
		OrganizationID:       org.ID,
		BusinessUnitID:       bu.ID,
		CustomerID:           pulid.MustNew("cust_"), // Required field
		ServiceTypeID:        pulid.MustNew("st_"),   // Required field
		ShipmentTypeID:       pulid.MustNew("sht_"),  // Required field
		ProNumber:            "TEST-PRO-001",
		BOL:                  "TEST-BOL-001",
		Status:               shipment.StatusNew,
		RatingMethod:         shipment.RatingMethodFlatRate,
		ConsolidationGroupID: nil, // Initially not in any consolidation group
	}
}

func newShipmentInConsolidationGroup(consolidationGroupID pulid.ID) *shipment.Shipment {
	shp := newShipment()
	shp.ConsolidationGroupID = &consolidationGroupID
	return shp
}

func TestConsolidationValidator_ValidateShipments(t *testing.T) {
	log := testutils.NewTestLogger(t)

	// Create a real validation engine factory (not mock)
	vef := framework.ProvideValidationEngineFactory()

	shipmentRepo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger: log,
		DB:     ts.DB,
	})

	val := cgValidator.NewValidator(cgValidator.ValidatorParams{
		DB:                      ts.DB,
		ShipmentRepo:            shipmentRepo,
		ValidationEngineFactory: vef,
	})

	scenarios := []struct {
		name           string
		isCreate       bool
		setupData      func() (*consolidation.ConsolidationGroup, []*shipment.Shipment)
		expectedErrors []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name:     "create with no shipments - should pass",
			isCreate: true,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()
				cg.Shipments = []*shipment.Shipment{}
				return cg, nil
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{},
		},
		{
			name:     "create with shipments not in any consolidation group - should pass",
			isCreate: true,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()

				// Create shipments that are not in any consolidation group
				shp1 := newShipment()
				shp1.ID = pulid.MustNew("shp_")
				shp1.ProNumber = "TEST-PRO-001"
				shp1.BOL = "TEST-BOL-001"

				shp2 := newShipment()
				shp2.ID = pulid.MustNew("shp_")
				shp2.ProNumber = "TEST-PRO-002"
				shp2.BOL = "TEST-BOL-002"

				cg.Shipments = []*shipment.Shipment{shp1, shp2}

				// Insert shipments into database without consolidation group
				return cg, []*shipment.Shipment{shp1, shp2}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{},
		},
		{
			name:     "create with shipments already in another consolidation group - should fail",
			isCreate: true,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()

				// Create another consolidation group
				existingCG := newConsolidationGroup()
				existingCG.ID = pulid.MustNew("cg_")
				existingCG.ConsolidationNumber = "EXISTING-CG-001"

				// Create shipments that are already in the existing consolidation group
				shp1 := newShipmentInConsolidationGroup(existingCG.ID)
				shp1.ID = pulid.MustNew("shp_")
				shp1.ProNumber = "TEST-PRO-003"
				shp1.BOL = "TEST-BOL-003"

				shp2 := newShipmentInConsolidationGroup(existingCG.ID)
				shp2.ID = pulid.MustNew("shp_")
				shp2.ProNumber = "TEST-PRO-004"
				shp2.BOL = "TEST-BOL-004"

				cg.Shipments = []*shipment.Shipment{shp1, shp2}

				// Insert shipments into database with existing consolidation group
				return cg, []*shipment.Shipment{shp1, shp2}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "shipments",
					Code:    errors.ErrInvalid,
					Message: "shipment shp_* is already in consolidation group cg_*",
				},
				{
					Field:   "shipments",
					Code:    errors.ErrInvalid,
					Message: "shipment shp_* is already in consolidation group cg_*",
				},
			},
		},
		{
			name:     "update with shipments in same consolidation group - should pass",
			isCreate: false,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()
				cg.ID = pulid.MustNew("cg_")

				// Create shipments that are already in the current consolidation group
				shp1 := newShipmentInConsolidationGroup(cg.ID)
				shp1.ID = pulid.MustNew("shp_")
				shp1.ProNumber = "TEST-PRO-005"
				shp1.BOL = "TEST-BOL-005"

				shp2 := newShipmentInConsolidationGroup(cg.ID)
				shp2.ID = pulid.MustNew("shp_")
				shp2.ProNumber = "TEST-PRO-006"
				shp2.BOL = "TEST-BOL-006"

				cg.Shipments = []*shipment.Shipment{shp1, shp2}

				// Insert shipments into database with current consolidation group
				return cg, []*shipment.Shipment{shp1, shp2}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{},
		},
		{
			name:     "update with shipments in different consolidation group - should fail",
			isCreate: false,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()
				cg.ID = pulid.MustNew("cg_")

				// Create another consolidation group
				otherCG := newConsolidationGroup()
				otherCG.ID = pulid.MustNew("cg_")
				otherCG.ConsolidationNumber = "OTHER-CG-001"

				// Create shipments that are in the other consolidation group
				shp1 := newShipmentInConsolidationGroup(otherCG.ID)
				shp1.ID = pulid.MustNew("shp_")
				shp1.ProNumber = "TEST-PRO-007"
				shp1.BOL = "TEST-BOL-007"

				shp2 := newShipmentInConsolidationGroup(otherCG.ID)
				shp2.ID = pulid.MustNew("shp_")
				shp2.ProNumber = "TEST-PRO-008"
				shp2.BOL = "TEST-BOL-008"

				cg.Shipments = []*shipment.Shipment{shp1, shp2}

				// Insert shipments into database with other consolidation group
				return cg, []*shipment.Shipment{shp1, shp2}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "shipments",
					Code:    errors.ErrInvalid,
					Message: "shipment shp_* is already in another consolidation group cg_*",
				},
				{
					Field:   "shipments",
					Code:    errors.ErrorCode(errors.ErrInvalid),
					Message: "shipment shp_* is already in another consolidation group cg_*",
				},
			},
		},
		{
			name:     "update with mixed shipments - some in current, some in other - should fail for others",
			isCreate: false,
			setupData: func() (*consolidation.ConsolidationGroup, []*shipment.Shipment) {
				cg := newConsolidationGroup()
				cg.ID = pulid.MustNew("cg_")

				// Create another consolidation group
				otherCG := newConsolidationGroup()
				otherCG.ID = pulid.MustNew("cg_")
				otherCG.ConsolidationNumber = "OTHER-CG-002"

				// Create one shipment in current consolidation group
				shp1 := newShipmentInConsolidationGroup(cg.ID)
				shp1.ID = pulid.MustNew("shp_")
				shp1.ProNumber = "TEST-PRO-009"
				shp1.BOL = "TEST-BOL-009"

				// Create one shipment in other consolidation group
				shp2 := newShipmentInConsolidationGroup(otherCG.ID)
				shp2.ID = pulid.MustNew("shp_")
				shp2.ProNumber = "TEST-PRO-010"
				shp2.BOL = "TEST-BOL-010"

				cg.Shipments = []*shipment.Shipment{shp1, shp2}

				// Insert shipments into database
				return cg, []*shipment.Shipment{shp1, shp2}
			},
			expectedErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "shipments",
					Code:    errors.ErrInvalid,
					Message: "shipment shp_* is already in another consolidation group cg_*",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup test data
			cg, shipmentsToInsert := scenario.setupData()

			// Insert shipments into database if needed
			if shipmentsToInsert != nil {
				for _, shp := range shipmentsToInsert {
					writeDB, err := ts.DB.WriteDB(ctx)
					if err != nil {
						t.Fatalf("Failed to get write DB: %v", err)
					}
					_, err = writeDB.NewInsert().
						Model(shp).
						Exec(ctx)
					if err != nil {
						t.Fatalf("Failed to insert test shipment: %v", err)
					}
				}
			}

			// Create validation context
			valCtx := &validator.ValidationContext{
				IsCreate: scenario.isCreate,
			}

			// Run validation
			multiErr := val.Validate(ctx, valCtx, cg)

			// Clean up inserted shipments
			if shipmentsToInsert != nil {
				for _, shp := range shipmentsToInsert {
					writeDB, _ := ts.DB.WriteDB(ctx)
					if writeDB != nil {
						_, _ = writeDB.NewDelete().
							Model((*shipment.Shipment)(nil)).
							Where("id = ?", shp.ID).
							Exec(ctx)
					}
				}
			}

			// Check results
			if len(scenario.expectedErrors) == 0 {
				if multiErr != nil && multiErr.HasErrors() {
					t.Errorf("Expected no errors, but got: %v", multiErr.Errors)
				}
			} else {
				if multiErr == nil || !multiErr.HasErrors() {
					t.Errorf("Expected errors but got none")
					return
				}

				// For scenarios with dynamic IDs, we need to check patterns
				for _, expectedErr := range scenario.expectedErrors {
					found := false
					for _, actualErr := range multiErr.Errors {
						if actualErr.Field == expectedErr.Field && actualErr.Code == expectedErr.Code {
							// For messages with dynamic IDs, check if the pattern matches
							if expectedErr.Message == "shipment shp_* is already in consolidation group cg_*" ||
								expectedErr.Message == "shipment shp_* is already in another consolidation group cg_*" {
								// Check if the message contains the expected pattern
								if len(actualErr.Message) > 0 &&
									(actualErr.Message[:8] == "shipment" &&
										(actualErr.Message[len(actualErr.Message)-20:] == "consolidation group" ||
											len(actualErr.Message) > 40)) {
									found = true
									break
								}
							} else if actualErr.Message == expectedErr.Message {
								found = true
								break
							}
						}
					}
					if !found {
						t.Errorf("Expected error not found: Field=%s, Code=%s, Message=%s",
							expectedErr.Field, expectedErr.Code, expectedErr.Message)
					}
				}
			}
		})
	}
}

func TestConsolidationValidator_ValidateShipments_DirectCall(t *testing.T) {
	log := testutils.NewTestLogger(t)
	vef := framework.ProvideValidationEngineFactory()

	shipmentRepo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger: log,
		DB:     ts.DB,
	})

	val := cgValidator.NewValidator(cgValidator.ValidatorParams{
		DB:                      ts.DB,
		ShipmentRepo:            shipmentRepo,
		ValidationEngineFactory: vef,
	})

	t.Run("direct call to ValidateShipments method", func(t *testing.T) {
		// Setup test data
		cg := newConsolidationGroup()
		cg.ID = pulid.MustNew("cg_")

		// Create a shipment in another consolidation group
		otherCG := pulid.MustNew("cg_")
		shp := newShipmentInConsolidationGroup(otherCG)
		shp.ID = pulid.MustNew("shp_")
		shp.ProNumber = "DIRECT-TEST-001"
		shp.BOL = "DIRECT-BOL-001"

		cg.Shipments = []*shipment.Shipment{shp}

		// Insert shipment into database
		writeDB, err := ts.DB.WriteDB(ctx)
		if err != nil {
			t.Fatalf("Failed to get write DB: %v", err)
		}
		_, err = writeDB.NewInsert().
			Model(shp).
			Exec(ctx)
		if err != nil {
			t.Fatalf("Failed to insert test shipment: %v", err)
		}

		// Clean up after test
		defer func() {
			writeDB, _ := ts.DB.WriteDB(ctx)
			if writeDB != nil {
				_, _ = writeDB.NewDelete().
					Model((*shipment.Shipment)(nil)).
					Where("id = ?", shp.ID).
					Exec(ctx)
			}
		}()

		// Test create operation
		valCtx := &validator.ValidationContext{IsCreate: true}
		multiErr := errors.NewMultiError()

		err = val.ValidateShipments(ctx, valCtx, cg, multiErr)

		if err != nil {
			t.Errorf("ValidateShipments returned error: %v", err)
		}

		if !multiErr.HasErrors() {
			t.Error("Expected validation errors but got none")
		}

		// Check that the error message contains the expected pattern
		found := false
		for _, validationErr := range multiErr.Errors {
			if validationErr.Field == "shipments" &&
				validationErr.Code == errors.ErrInvalid &&
				len(validationErr.Message) > 20 {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected shipment validation error not found")
		}
	})
}
