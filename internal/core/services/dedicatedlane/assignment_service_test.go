// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package dedicatedlane

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/mocks"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

// Mock repositories for testing

func TestNewAssignmentService(t *testing.T) {
	log := logger.NewLogger(testutils.NewTestConfig())

	mockAssignRepo := &mocks.MockAssignmentRepository{}
	mockShipRepo := &mocks.MockShipmentRepository{}
	mockTractorRepo := &mocks.MockTractorRepository{}

	params := AssignmentServiceParams{
		DB:             ts.DB,
		AssignmentRepo: mockAssignRepo,
		ShipmentRepo:   mockShipRepo,
		TractorRepo:    mockTractorRepo,
		Logger:         log,
	}

	service := NewAssignmentService(params)

	require.NotNil(t, service)
	require.Equal(t, ts.DB, service.db)
	require.Equal(t, mockAssignRepo, service.assignmentRepo)
	require.Equal(t, mockShipRepo, service.shipmentRepo)
	require.Equal(t, mockTractorRepo, service.tractorRepo)
	require.NotNil(t, service.l)
}

func TestExtractLocations(t *testing.T) {
	mockAssignRepo := &mocks.MockAssignmentRepository{}
	mockShipRepo := &mocks.MockShipmentRepository{}
	mockTractorRepo := &mocks.MockTractorRepository{}
	log := logger.NewLogger(testutils.NewTestConfig())

	service := &AssignmentService{
		db:             ts.DB,
		assignmentRepo: mockAssignRepo,
		shipmentRepo:   mockShipRepo,
		tractorRepo:    mockTractorRepo,
		l:              log.Logger,
	}

	tests := []struct {
		name                string
		shipment            *shipment.Shipment
		expectedOrigin      pulid.ID
		expectedDestination pulid.ID
		description         string
	}{
		{
			name: "no moves",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{},
			},
			expectedOrigin:      "",
			expectedDestination: "",
			description:         "should return empty IDs when no moves exist",
		},
		{
			name: "move with no stops",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{},
					},
				},
			},
			expectedOrigin:      "",
			expectedDestination: "",
			description:         "should return empty IDs when no stops exist",
		},
		{
			name: "single move with one stop",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{
							{LocationID: pulid.ID("loc_origin")},
						},
					},
				},
			},
			expectedOrigin:      pulid.ID("loc_origin"),
			expectedDestination: "",
			description:         "should extract origin but no destination for single stop",
		},
		{
			name: "single move with multiple stops",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{
							{LocationID: pulid.ID("loc_origin")},
							{LocationID: pulid.ID("loc_middle")},
							{LocationID: pulid.ID("loc_destination")},
						},
					},
				},
			},
			expectedOrigin:      pulid.ID("loc_origin"),
			expectedDestination: pulid.ID("loc_destination"),
			description:         "should extract first and last stops from single move",
		},
		{
			name: "multiple moves",
			shipment: &shipment.Shipment{
				Moves: []*shipment.ShipmentMove{
					{
						Stops: []*shipment.Stop{
							{LocationID: pulid.ID("loc_origin")},
							{LocationID: pulid.ID("loc_middle1")},
						},
					},
					{
						Stops: []*shipment.Stop{
							{LocationID: pulid.ID("loc_middle2")},
							{LocationID: pulid.ID("loc_destination")},
						},
					},
				},
			},
			expectedOrigin:      pulid.ID("loc_origin"),
			expectedDestination: pulid.ID("loc_destination"),
			description:         "should extract origin from first move and destination from last move",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			origin, destination := service.extractLocations(test.shipment)
			require.Equal(t, test.expectedOrigin, origin, test.description)
			require.Equal(t, test.expectedDestination, destination, test.description)
		})
	}
}

func TestHandleDedicatedLaneOperations(t *testing.T) {
	// Get test data from fixtures
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	loc1 := ts.Fixture.MustRow("Location.test_location").(*location.Location)
	loc2 := ts.Fixture.MustRow("Location.test_location_2").(*location.Location)
	cus := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	serviceType := ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType)
	shipmentType := ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType)
	tractorType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	containerType := ts.Fixture.MustRow("EquipmentType.container_equip_type").(*equipmenttype.EquipmentType)
	worker1 := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	worker2 := ts.Fixture.MustRow("Worker.worker_2").(*worker.Worker)

	mockAssignRepo := &mocks.MockAssignmentRepository{}
	mockShipRepo := &mocks.MockShipmentRepository{}
	mockTractorRepo := &mocks.MockTractorRepository{}
	mockDedicatedLaneRepo := &mocks.MockDedicatedLaneRepository{}
	log := logger.NewLogger(testutils.NewTestConfig())

	service := &AssignmentService{
		db:                ts.DB,
		assignmentRepo:    mockAssignRepo,
		shipmentRepo:      mockShipRepo,
		tractorRepo:       mockTractorRepo,
		l:                 log.Logger,
		dedicatedLaneRepo: mockDedicatedLaneRepo,
	}

	t.Run("no moves", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			Moves:          []*shipment.ShipmentMove{},
		}

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)
	})

	t.Run("insufficient location data", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{},
				},
			},
		}

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)
	})

	t.Run("no dedicated lane found", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  &tractorType.ID,
			TrailerTypeID:  nil, // This will cause no match with existing dedicated lane
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Setup mock expectation for FindByShipment to return no dedicated lane found
		mockDedicatedLaneRepo.On("FindByShipment", ctx, mock.MatchedBy(func(req *repositories.FindDedicatedLaneByShipmentRequest) bool {
			return req.OrganizationID == org.ID &&
				req.BusinessUnitID == bu.ID &&
				req.CustomerID == cus.ID &&
				req.OriginLocationID == loc1.ID &&
				req.DestinationLocationID == loc2.ID
		})).
			Return((*dedicatedlane.DedicatedLane)(nil), sql.ErrNoRows)

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		mockDedicatedLaneRepo.AssertExpectations(t)
	})

	t.Run("dedicated lane found but auto assign disabled", func(t *testing.T) {
		// The existing dedicated lane fixture has auto_assign: false
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  &tractorType.ID,
			TrailerTypeID:  &containerType.ID,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Create a dedicated lane with auto assign disabled
		mockDedicatedLane := &dedicatedlane.DedicatedLane{
			ID:                    pulid.ID("dl_test"),
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			CustomerID:            cus.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			AutoAssign:            false, // Auto assign disabled
		}

		// Setup mock expectation for FindByShipment to return dedicated lane with auto assign disabled
		mockDedicatedLaneRepo.On("FindByShipment", ctx, mock.MatchedBy(func(req *repositories.FindDedicatedLaneByShipmentRequest) bool {
			return req.OrganizationID == org.ID &&
				req.BusinessUnitID == bu.ID &&
				req.CustomerID == cus.ID &&
				req.OriginLocationID == loc1.ID &&
				req.DestinationLocationID == loc2.ID
		})).
			Return(mockDedicatedLane, nil)

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		mockDedicatedLaneRepo.AssertExpectations(t)
	})

	t.Run("shipment with null tractor and trailer types", func(t *testing.T) {
		// Create shipment with null tractor and trailer types to match
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_null_types"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  nil, // This should match the NULL in the database
			TrailerTypeID:  nil, // This should match the NULL in the database
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Create a dedicated lane with null tractor and trailer types and auto assign disabled
		mockDedicatedLaneNullTypes := &dedicatedlane.DedicatedLane{
			ID:                    pulid.ID("dl_test_null"),
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			CustomerID:            cus.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         nil, // NULL tractor type
			TrailerTypeID:         nil, // NULL trailer type
			AutoAssign:            false,
		}

		// Setup mock expectation for FindByShipment to return dedicated lane with null types
		mockDedicatedLaneRepo.On("FindByShipment", ctx, mock.MatchedBy(func(req *repositories.FindDedicatedLaneByShipmentRequest) bool {
			return req.OrganizationID == org.ID &&
				req.BusinessUnitID == bu.ID &&
				req.CustomerID == cus.ID &&
				req.OriginLocationID == loc1.ID &&
				req.DestinationLocationID == loc2.ID &&
				req.TractorTypeID == nil &&
				req.TrailerTypeID == nil
		})).
			Return(mockDedicatedLaneNullTypes, nil)

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		mockDedicatedLaneRepo.AssertExpectations(t)
	})

	t.Run("auto assign success", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  &tractorType.ID,
			TrailerTypeID:  &containerType.ID,
			Status:         shipment.StatusNew,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Create a dedicated lane with auto assign enabled
		mockDedicatedLaneAuto := &dedicatedlane.DedicatedLane{
			ID:                    pulid.ID("dl_test_auto"),
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			CustomerID:            cus.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			PrimaryWorkerID:       &worker1.ID,
			SecondaryWorkerID:     &worker2.ID,
			AutoAssign:            true, // Auto assign enabled
		}

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Create fresh mocks for this test to avoid conflicts
		mockAssignRepoAuto := &mocks.MockAssignmentRepository{}
		mockShipRepoAuto := &mocks.MockShipmentRepository{}
		mockTractorRepoAuto := &mocks.MockTractorRepository{}
		mockDedicatedLaneRepoAuto := &mocks.MockDedicatedLaneRepository{}

		serviceAuto := &AssignmentService{
			db:                ts.DB,
			assignmentRepo:    mockAssignRepoAuto,
			shipmentRepo:      mockShipRepoAuto,
			tractorRepo:       mockTractorRepoAuto,
			dedicatedLaneRepo: mockDedicatedLaneRepoAuto,
			l:                 log.Logger,
		}

		// Setup mock expectations
		mockDedicatedLaneRepoAuto.On("FindByShipment", ctx, mock.Anything).
			Return(mockDedicatedLaneAuto, nil)

		mockTractorRepoAuto.On("GetByPrimaryWorkerID", ctx, mock.Anything).
			Return(mockTractor, nil)

		mockAssignRepoAuto.On("BulkAssign", ctx, mock.Anything).
			Return([]*shipment.Assignment{}, nil)

		mockShipRepoAuto.On("UpdateStatus", ctx, mock.Anything).
			Return(shp, nil)

		err := serviceAuto.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		// Verify all mock expectations were met
		mockDedicatedLaneRepoAuto.AssertExpectations(t)
		mockAssignRepoAuto.AssertExpectations(t)
		mockShipRepoAuto.AssertExpectations(t)
		mockTractorRepoAuto.AssertExpectations(t)
	})

	t.Run("auto assign bulk assign failure", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  &tractorType.ID,
			TrailerTypeID:  &containerType.ID,
			Status:         shipment.StatusNew,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Create a dedicated lane with auto_assign: true
		dl := &dedicatedlane.DedicatedLane{
			ID:                    pulid.ID("dl_test_fail"),
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			Name:                  "Test Auto Assign Lane Failure",
			Status:                domain.StatusActive,
			CustomerID:            cus.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			PrimaryWorkerID:       &worker1.ID,
			SecondaryWorkerID:     &worker2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			AutoAssign:            true,
		}

		// Create new mocks for this test
		mockAssignRepoFail := &mocks.MockAssignmentRepository{}
		mockShipRepoFail := &mocks.MockShipmentRepository{}
		mockTractorRepoFail := &mocks.MockTractorRepository{}
		mockDedicatedLaneRepoFail := &mocks.MockDedicatedLaneRepository{}

		serviceFail := &AssignmentService{
			db:                ts.DB,
			assignmentRepo:    mockAssignRepoFail,
			shipmentRepo:      mockShipRepoFail,
			tractorRepo:       mockTractorRepoFail,
			dedicatedLaneRepo: mockDedicatedLaneRepoFail,
			l:                 log.Logger,
		}

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Setup mock expectations
		mockDedicatedLaneRepoFail.On("FindByShipment", ctx, mock.Anything).
			Return(dl, nil)
		mockTractorRepoFail.On("GetByPrimaryWorkerID", ctx, mock.Anything).
			Return(mockTractor, nil)
		mockAssignRepoFail.On("BulkAssign", ctx, mock.Anything).
			Return(([]*shipment.Assignment)(nil), sql.ErrConnDone)

		err := serviceFail.HandleDedicatedLaneOperations(ctx, shp)
		require.Error(t, err)

		// Verify mock expectations
		mockDedicatedLaneRepoFail.AssertExpectations(t)
		mockAssignRepoFail.AssertExpectations(t)
		mockTractorRepoFail.AssertExpectations(t)
	})

	t.Run("auto assign update status failure", func(t *testing.T) {
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto_status_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     cus.ID,
			ServiceTypeID:  serviceType.ID,
			ShipmentTypeID: shipmentType.ID,
			TractorTypeID:  &tractorType.ID,
			TrailerTypeID:  &containerType.ID,
			Status:         shipment.StatusNew,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{LocationID: loc1.ID},
						{LocationID: loc2.ID},
					},
				},
			},
		}

		// Create a dedicated lane with auto_assign: true
		dl := &dedicatedlane.DedicatedLane{
			ID:                    pulid.ID("dl_test_status_fail"),
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			Name:                  "Test Auto Assign Lane Status Failure",
			Status:                domain.StatusActive,
			CustomerID:            cus.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			PrimaryWorkerID:       &worker1.ID,
			SecondaryWorkerID:     &worker2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			AutoAssign:            true,
		}

		// Create new mocks for this test
		mockAssignRepoStatus := &mocks.MockAssignmentRepository{}
		mockShipRepoStatus := &mocks.MockShipmentRepository{}
		mockTractorRepoStatus := &mocks.MockTractorRepository{}
		mockDedicatedLaneRepoStatus := &mocks.MockDedicatedLaneRepository{}

		serviceStatus := &AssignmentService{
			db:                ts.DB,
			assignmentRepo:    mockAssignRepoStatus,
			shipmentRepo:      mockShipRepoStatus,
			tractorRepo:       mockTractorRepoStatus,
			dedicatedLaneRepo: mockDedicatedLaneRepoStatus,
			l:                 log.Logger,
		}

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test_status_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Setup mocks - GetByPrimaryWorkerID and BulkAssign succeed, UpdateStatus fails
		mockDedicatedLaneRepoStatus.On("FindByShipment", ctx, mock.Anything).
			Return(dl, nil)
		mockTractorRepoStatus.On("GetByPrimaryWorkerID", ctx, mock.Anything).
			Return(mockTractor, nil)
		mockAssignRepoStatus.On("BulkAssign", ctx, mock.Anything).
			Return([]*shipment.Assignment{}, nil)
		mockShipRepoStatus.On("UpdateStatus", ctx, mock.Anything).
			Return((*shipment.Shipment)(nil), sql.ErrConnDone)

		err := serviceStatus.HandleDedicatedLaneOperations(ctx, shp)
		require.Error(t, err)

		// Verify mock expectations
		mockDedicatedLaneRepoStatus.AssertExpectations(t)
		mockAssignRepoStatus.AssertExpectations(t)
		mockShipRepoStatus.AssertExpectations(t)
		mockTractorRepoStatus.AssertExpectations(t)
	})
}
