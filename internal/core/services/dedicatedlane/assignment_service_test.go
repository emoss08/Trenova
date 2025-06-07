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
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
type MockAssignmentRepository struct {
	mock.Mock
}

func (m *MockAssignmentRepository) List(
	ctx context.Context,
	req repositories.ListAssignmentsRequest,
) (*ports.ListResult[*shipment.Assignment], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*shipment.Assignment]), args.Error(1)
}

func (m *MockAssignmentRepository) BulkAssign(
	ctx context.Context,
	req *repositories.AssignmentRequest,
) ([]*shipment.Assignment, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) SingleAssign(
	ctx context.Context,
	a *shipment.Assignment,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, a)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) Reassign(
	ctx context.Context,
	a *shipment.Assignment,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, a)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) GetByID(
	ctx context.Context,
	opts repositories.GetAssignmentByIDOptions,
) (*shipment.Assignment, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*shipment.Assignment), args.Error(1)
}

type MockShipmentRepository struct {
	mock.Mock
}

func (m *MockShipmentRepository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*shipment.Shipment), args.Error(1)
}

// Add other required methods as no-ops for the interface
func (m *MockShipmentRepository) List(
	ctx context.Context,
	opts *repositories.ListShipmentOptions,
) (*ports.ListResult[*shipment.Shipment], error) {
	return nil, nil
}

func (m *MockShipmentRepository) GetByID(
	ctx context.Context,
	opts *repositories.GetShipmentByIDOptions,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Create(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Update(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) Duplicate(
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *MockShipmentRepository) CheckForDuplicateBOLs(
	ctx context.Context,
	currentBOL string,
	orgID pulid.ID,
	buID pulid.ID,
	excludeID *pulid.ID,
) ([]repositories.DuplicateBOLsResult, error) {
	return nil, nil
}

func (m *MockShipmentRepository) CalculateShipmentTotals(
	shp *shipment.Shipment,
) (*repositories.ShipmentTotalsResponse, error) {
	return nil, nil
}

type MockTractorRepository struct {
	mock.Mock
}

func (m *MockTractorRepository) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*ports.ListResult[*tractor.Tractor], error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ports.ListResult[*tractor.Tractor]), args.Error(1)
}

func (m *MockTractorRepository) GetByID(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) GetByPrimaryWorkerID(
	ctx context.Context,
	req repositories.GetTractorByPrimaryWorkerIDRequest,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Create(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, t)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Update(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	args := m.Called(ctx, t)
	return args.Get(0).(*tractor.Tractor), args.Error(1)
}

func (m *MockTractorRepository) Assignment(
	ctx context.Context,
	req repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*repositories.AssignmentResponse), args.Error(1)
}

func TestNewAssignmentService(t *testing.T) {
	log := logger.NewLogger(testutils.NewTestConfig())

	mockAssignRepo := &MockAssignmentRepository{}
	mockShipRepo := &MockShipmentRepository{}
	mockTractorRepo := &MockTractorRepository{}

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
	mockAssignRepo := &MockAssignmentRepository{}
	mockShipRepo := &MockShipmentRepository{}
	mockTractorRepo := &MockTractorRepository{}
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
	customer := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	serviceType := ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType)
	shipmentType := ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType)
	tractorType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	containerType := ts.Fixture.MustRow("EquipmentType.container_equip_type").(*equipmenttype.EquipmentType)
	worker1 := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	worker2 := ts.Fixture.MustRow("Worker.worker_2").(*worker.Worker)

	mockAssignRepo := &MockAssignmentRepository{}
	mockShipRepo := &MockShipmentRepository{}
	mockTractorRepo := &MockTractorRepository{}
	log := logger.NewLogger(testutils.NewTestConfig())

	service := &AssignmentService{
		db:             ts.DB,
		assignmentRepo: mockAssignRepo,
		shipmentRepo:   mockShipRepo,
		tractorRepo:    mockTractorRepo,
		l:              log.Logger,
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
			CustomerID:     customer.ID,
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

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)
	})

	t.Run("dedicated lane found but auto assign disabled", func(t *testing.T) {
		// The existing dedicated lane fixture has auto_assign: false
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     customer.ID,
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

		err := service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)
	})

	t.Run("shipment with null tractor and trailer types", func(t *testing.T) {
		// Create a dedicated lane with null tractor and trailer types
		dl := ts.Fixture.MustRow("DedicatedLane.test_dedicated_lane_1").(*dedicatedlane.DedicatedLane)

		dba, err := ts.DB.DB(ctx)
		require.NoError(t, err)

		// Update the fixture to have null tractor and trailer types and disable auto_assign
		_, err = dba.NewUpdate().Model(dl).
			Set("tractor_type_id = NULL").
			Set("trailer_type_id = NULL").
			Set("auto_assign = ?", false).
			Where("id = ?", dl.ID).
			Exec(ctx)
		require.NoError(t, err)

		// Create shipment with null tractor and trailer types to match
		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_null_types"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     customer.ID,
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

		err = service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		// Reset the fixture back to its original state for subsequent tests
		_, err = dba.NewUpdate().Model(dl).
			Set("tractor_type_id = ?", tractorType.ID).
			Set("trailer_type_id = ?", containerType.ID).
			Set("auto_assign = ?", false).
			Where("id = ?", dl.ID).
			Exec(ctx)
		require.NoError(t, err)
	})

	t.Run("auto assign success", func(t *testing.T) {
		// Get the existing dedicated lane fixture and update it to enable auto_assign
		dl := ts.Fixture.MustRow("DedicatedLane.test_dedicated_lane_1").(*dedicatedlane.DedicatedLane)

		dba, err := ts.DB.DB(ctx)
		require.NoError(t, err)

		// Update the fixture to enable auto_assign
		_, err = dba.NewUpdate().Model(dl).
			Set("auto_assign = ?", true).
			Where("id = ?", dl.ID).
			Exec(ctx)
		require.NoError(t, err)

		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     customer.ID,
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

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Setup mock expectations
		mockTractorRepo.On("GetByPrimaryWorkerID", ctx, repositories.GetTractorByPrimaryWorkerIDRequest{
			WorkerID: dl.PrimaryWorkerID,
			OrgID:    shp.OrganizationID,
			BuID:     shp.BusinessUnitID,
		}).
			Return(mockTractor, nil)

		mockAssignRepo.On("BulkAssign", ctx, mock.MatchedBy(func(req *repositories.AssignmentRequest) bool {
			return req.ShipmentID == shp.ID &&
				req.PrimaryWorkerID == dl.PrimaryWorkerID &&
				req.SecondaryWorkerID != nil &&
				*req.SecondaryWorkerID == *dl.SecondaryWorkerID &&
				req.TractorID == mockTractor.ID &&
				req.OrgID == org.ID &&
				req.BuID == bu.ID
		})).
			Return([]*shipment.Assignment{}, nil)

		mockShipRepo.On("UpdateStatus", ctx, mock.MatchedBy(func(req *repositories.UpdateShipmentStatusRequest) bool {
			return req.GetOpts.ID == shp.ID &&
				req.GetOpts.OrgID == org.ID &&
				req.GetOpts.BuID == bu.ID &&
				req.Status == shipment.StatusAssigned
		})).
			Return(shp, nil)

		err = service.HandleDedicatedLaneOperations(ctx, shp)
		require.NoError(t, err)

		// Verify all mock expectations were met
		mockAssignRepo.AssertExpectations(t)
		mockShipRepo.AssertExpectations(t)
		mockTractorRepo.AssertExpectations(t)
	})

	t.Run("auto assign bulk assign failure", func(t *testing.T) {
		// Create a dedicated lane with auto_assign: true
		dl := &dedicatedlane.DedicatedLane{
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			Name:                  "Test Auto Assign Lane Failure",
			Status:                domain.StatusActive,
			CustomerID:            customer.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			PrimaryWorkerID:       worker1.ID,
			SecondaryWorkerID:     &worker2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			AutoAssign:            true,
		}

		// Insert the dedicated lane into the database
		dba, err := ts.DB.DB(ctx)
		require.NoError(t, err)
		_, err = dba.NewInsert().Model(dl).Exec(ctx)
		require.NoError(t, err)

		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     customer.ID,
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

		// Create new mocks for this test
		mockAssignRepoFail := &MockAssignmentRepository{}
		mockShipRepoFail := &MockShipmentRepository{}
		mockTractorRepoFail := &MockTractorRepository{}

		serviceFail := &AssignmentService{
			db:             ts.DB,
			assignmentRepo: mockAssignRepoFail,
			shipmentRepo:   mockShipRepoFail,
			tractorRepo:    mockTractorRepoFail,
			l:              log.Logger,
		}

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Setup mock to succeed on GetByPrimaryWorkerID but fail on BulkAssign
		mockTractorRepoFail.On("GetByPrimaryWorkerID", ctx, mock.Anything).
			Return(mockTractor, nil)
		mockAssignRepoFail.On("BulkAssign", ctx, mock.Anything).
			Return(([]*shipment.Assignment)(nil), sql.ErrConnDone)

		err = serviceFail.HandleDedicatedLaneOperations(ctx, shp)
		require.Error(t, err)

		// Verify mock expectations
		mockAssignRepoFail.AssertExpectations(t)
		mockTractorRepoFail.AssertExpectations(t)
	})

	t.Run("auto assign update status failure", func(t *testing.T) {
		// Create a dedicated lane with auto_assign: true
		dl := &dedicatedlane.DedicatedLane{
			OrganizationID:        org.ID,
			BusinessUnitID:        bu.ID,
			Name:                  "Test Auto Assign Lane Status Failure",
			Status:                domain.StatusActive,
			CustomerID:            customer.ID,
			OriginLocationID:      loc1.ID,
			DestinationLocationID: loc2.ID,
			PrimaryWorkerID:       worker1.ID,
			SecondaryWorkerID:     &worker2.ID,
			ServiceTypeID:         serviceType.ID,
			ShipmentTypeID:        shipmentType.ID,
			TractorTypeID:         &tractorType.ID,
			TrailerTypeID:         &containerType.ID,
			AutoAssign:            true,
		}

		// Insert the dedicated lane into the database
		dba, err := ts.DB.DB(ctx)
		require.NoError(t, err)
		_, err = dba.NewInsert().Model(dl).Exec(ctx)
		require.NoError(t, err)

		shp := &shipment.Shipment{
			ID:             pulid.ID("shp_test_auto_status_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			CustomerID:     customer.ID,
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

		// Create new mocks for this test
		mockAssignRepoStatus := &MockAssignmentRepository{}
		mockShipRepoStatus := &MockShipmentRepository{}
		mockTractorRepoStatus := &MockTractorRepository{}

		serviceStatus := &AssignmentService{
			db:             ts.DB,
			assignmentRepo: mockAssignRepoStatus,
			shipmentRepo:   mockShipRepoStatus,
			tractorRepo:    mockTractorRepoStatus,
			l:              log.Logger,
		}

		// Create a mock tractor to return
		mockTractor := &tractor.Tractor{
			ID:             pulid.ID("tr_test_status_fail"),
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
		}

		// Setup mocks - GetByPrimaryWorkerID and BulkAssign succeed, UpdateStatus fails
		mockTractorRepoStatus.On("GetByPrimaryWorkerID", ctx, mock.Anything).
			Return(mockTractor, nil)
		mockAssignRepoStatus.On("BulkAssign", ctx, mock.Anything).
			Return([]*shipment.Assignment{}, nil)
		mockShipRepoStatus.On("UpdateStatus", ctx, mock.Anything).
			Return((*shipment.Shipment)(nil), sql.ErrConnDone)

		err = serviceStatus.HandleDedicatedLaneOperations(ctx, shp)
		require.Error(t, err)

		// Verify mock expectations
		mockAssignRepoStatus.AssertExpectations(t)
		mockShipRepoStatus.AssertExpectations(t)
		mockTractorRepoStatus.AssertExpectations(t)
	})
}
