package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/calculator"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/internal/pkg/seqgen/adapters"
	"github.com/emoss08/trenova/internal/pkg/statemachine"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentRepository(t *testing.T) {
	// Load test fixtures
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	testShipment := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)
	inTransitShipment := ts.Fixture.MustRow("Shipment.in_transit_shipment").(*shipment.Shipment)
	// completedShipment := ts.Fixture.MustRow("Shipment.completed_shipment").(*shipment.Shipment)
	serviceType := ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType)
	shipmentType := ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType)
	customerFixture := ts.Fixture.MustRow("Customer.honeywell_customer").(*customer.Customer)
	tractorEquipType := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	trailerEquipType := ts.Fixture.MustRow("EquipmentType.trailer_equip_type").(*equipmenttype.EquipmentType)
	location1 := ts.Fixture.MustRow("Location.test_location").(*location.Location)
	location2 := ts.Fixture.MustRow("Location.test_location_2").(*location.Location)
	testUser := ts.Fixture.MustRow("User.test_user").(*user.User)
	testCommodity := ts.Fixture.MustRow("Commodity.test_commodity").(*commodity.Commodity)

	// Setup dependencies
	log := testutils.NewTestLogger(t)
	repo := setupShipmentRepository(log)

	ctx := context.Background()

	t.Run("Repository Setup", func(t *testing.T) {
		require.NotNil(t, repo, "Repository should be initialized")
	})

	// Test List operations
	t.Run("List", func(t *testing.T) {
		t.Run("Basic List", func(t *testing.T) {
			opts := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  10,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "List should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			assert.LessOrEqual(t, len(result.Items), 10, "Items should not exceed limit")
		})

		t.Run("List with Query Filter", func(t *testing.T) {
			opts := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  10,
					Offset: 0,
					Query:  testShipment.ProNumber[:3], // Search by partial pro number
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "List with query should not return error")
			require.NotNil(t, result, "Result should not be nil")
		})

		t.Run("List with Status Filter", func(t *testing.T) {
			t.Skip(
				"This test is no longer valid as we are using the querybuilder to filter by status",
			)
			opts := &repoports.ListShipmentOptions{
				ShipmentOptions: repoports.ShipmentOptions{
					Status: string(shipment.StatusNew),
				},
				Filter: &ports.QueryOptions{
					Limit:  10,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "List with status filter should not return error")
			require.NotNil(t, result, "Result should not be nil")
			for _, item := range result.Items {
				assert.Equal(t, shipment.StatusNew, item.Status, "All items should have New status")
			}
		})

		t.Run("List with Nested Field Filters", func(t *testing.T) {
			t.Run("Filter by Customer Name", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						FieldFilters: []ports.FieldFilter{
							{
								Field:    "customer.name",
								Operator: ports.OpContains,
								Value:    "Honeywell",
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(t, err, "List with customer name filter should not return error")
				require.NotNil(t, result, "Result should not be nil")

				// Verify that results contain the expected customer
				for _, item := range result.Items {
					if item.Customer != nil {
						assert.Contains(
							t,
							item.Customer.Name,
							"Honeywell",
							"Customer name should contain 'Honeywell'",
						)
					}
				}
			})

			t.Run("Filter by Origin Location Name", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						FieldFilters: []ports.FieldFilter{
							{
								Field:    "originLocation.name",
								Operator: ports.OpEqual,
								Value:    location1.Name,
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(t, err, "List with origin location filter should not return error")
				require.NotNil(t, result, "Result should not be nil")

				// This tests that the query executes without error
				// The specific results will depend on test data
				assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			})

			t.Run("Filter by Destination Location Name", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						FieldFilters: []ports.FieldFilter{
							{
								Field:    "destinationLocation.name",
								Operator: ports.OpStartsWith,
								Value:    location2.Name[:3],
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(
					t,
					err,
					"List with destination location filter should not return error",
				)
				require.NotNil(t, result, "Result should not be nil")
				assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			})

			t.Run("Sort by Customer Name", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						Sort: []ports.SortField{
							{
								Field:     "customer.name",
								Direction: ports.SortAsc,
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(t, err, "List with customer name sort should not return error")
				require.NotNil(t, result, "Result should not be nil")
				assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			})

			t.Run("Complex Nested Field Query", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						FieldFilters: []ports.FieldFilter{
							{
								Field:    "status",
								Operator: ports.OpEqual,
								Value:    string(shipment.StatusNew),
							},
							{
								Field:    "customer.name",
								Operator: ports.OpContains,
								Value:    "Honeywell",
							},
						},
						Sort: []ports.SortField{
							{
								Field:     "originLocation.name",
								Direction: ports.SortAsc,
							},
							{
								Field:     "customer.name",
								Direction: ports.SortDesc,
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(t, err, "Complex nested field query should not return error")
				require.NotNil(t, result, "Result should not be nil")
				assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			})

			t.Run("Filter by Origin Date Range", func(t *testing.T) {
				opts := &repoports.ListShipmentOptions{
					ShipmentOptions: repoports.ShipmentOptions{
						ExpandShipmentDetails: true,
					},
					Filter: &ports.QueryOptions{
						Limit:  10,
						Offset: 0,
						FieldFilters: []ports.FieldFilter{
							{
								Field:    "originDate",
								Operator: ports.OpDateRange,
								Value: map[string]any{
									"start": "2024-01-01",
									"end":   "2024-12-31",
								},
							},
						},
						TenantOpts: &ports.TenantOptions{
							OrgID: org.ID,
							BuID:  bu.ID,
						},
					},
				}

				result, err := repo.List(ctx, opts)
				require.NoError(t, err, "List with origin date filter should not return error")
				require.NotNil(t, result, "Result should not be nil")
				assert.GreaterOrEqual(t, result.Total, 0, "Total should be non-negative")
			})
		})

		t.Run("List with Expanded Details", func(t *testing.T) {
			opts := &repoports.ListShipmentOptions{
				ShipmentOptions: repoports.ShipmentOptions{
					ExpandShipmentDetails: true,
				},
				Filter: &ports.QueryOptions{
					Limit:  5,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "List with expanded details should not return error")
			require.NotNil(t, result, "Result should not be nil")

			if len(result.Items) > 0 {
				shipmentItem := result.Items[0]
				assert.NotNil(t, shipmentItem.Customer, "Customer should be loaded")
				assert.NotNil(t, shipmentItem.ServiceType, "ServiceType should be loaded")
				assert.NotNil(t, shipmentItem.ShipmentType, "ShipmentType should be loaded")
				if len(shipmentItem.Moves) > 0 {
					assert.NotNil(t, shipmentItem.Moves[0].Stops, "Stops should be loaded")
				}
			}
		})

		t.Run("List with Pagination", func(t *testing.T) {
			// Test first page
			opts1 := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  2,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result1, err := repo.List(ctx, opts1)
			require.NoError(t, err, "First page should not return error")
			require.NotNil(t, result1, "First page result should not be nil")

			// Test second page
			opts2 := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  2,
					Offset: 2,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result2, err := repo.List(ctx, opts2)
			require.NoError(t, err, "Second page should not return error")
			require.NotNil(t, result2, "Second page result should not be nil")

			// Ensure totals match and pages are different
			assert.Equal(t, result1.Total, result2.Total, "Total should be consistent across pages")
			if len(result1.Items) > 0 && len(result2.Items) > 0 {
				assert.NotEqual(
					t,
					result1.Items[0].ID,
					result2.Items[0].ID,
					"Pages should contain different items",
				)
			}
		})
	})

	// Test GetByID operations
	t.Run("GetByID", func(t *testing.T) {
		t.Run("Valid ID", func(t *testing.T) {
			opts := &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: org.ID,
				BuID:  bu.ID,
			}

			result, err := repo.GetByID(ctx, opts)
			require.NoError(t, err, "GetByID should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, testShipment.ID, result.ID, "ID should match")
			assert.Equal(t, org.ID, result.OrganizationID, "OrganizationID should match")
			assert.Equal(t, bu.ID, result.BusinessUnitID, "BusinessUnitID should match")
		})

		t.Run("Invalid ID", func(t *testing.T) {
			opts := &repoports.GetShipmentByIDOptions{
				ID:    pulid.MustNew("shp_"),
				OrgID: org.ID,
				BuID:  bu.ID,
			}

			result, err := repo.GetByID(ctx, opts)
			require.Error(t, err, "GetByID with invalid ID should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("Wrong Organization", func(t *testing.T) {
			opts := &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: pulid.MustNew("org_"),
				BuID:  bu.ID,
			}

			result, err := repo.GetByID(ctx, opts)
			require.Error(t, err, "GetByID with wrong org should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("With Expanded Details", func(t *testing.T) {
			opts := &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: org.ID,
				BuID:  bu.ID,
				ShipmentOptions: repoports.ShipmentOptions{
					ExpandShipmentDetails: true,
				},
			}

			result, err := repo.GetByID(ctx, opts)
			require.NoError(t, err, "GetByID with details should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.NotNil(t, result.Customer, "Customer should be loaded")
			assert.NotNil(t, result.ServiceType, "ServiceType should be loaded")
			assert.NotNil(t, result.ShipmentType, "ShipmentType should be loaded")
		})
	})

	// Test Create operations
	t.Run("Create", func(t *testing.T) {
		t.Run("Valid Shipment", func(t *testing.T) {
			newShipment := &shipment.Shipment{
				ServiceTypeID:  serviceType.ID,
				ShipmentTypeID: shipmentType.ID,
				TrailerTypeID:  &trailerEquipType.ID,
				TractorTypeID:  &tractorEquipType.ID,
				CustomerID:     customerFixture.ID,
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
				BOL:            "TEST-BOL-001",
				Weight:         intutils.SafeInt64PtrNonNil(1000),
				Pieces:         intutils.SafeInt64PtrNonNil(10),
				Moves: []*shipment.ShipmentMove{
					{
						Status:   shipment.MoveStatusNew,
						Sequence: 0,
						Stops: []*shipment.Stop{
							{
								Status:           shipment.StopStatusNew,
								Sequence:         0,
								Type:             shipment.StopTypePickup,
								LocationID:       location1.ID,
								PlannedArrival:   timeutils.NowUnix(),
								PlannedDeparture: timeutils.NowUnix() + 3600,
								Weight:           func() *int { v := 1000; return &v }(),
								Pieces:           func() *int { v := 10; return &v }(),
							},
							{
								Status:           shipment.StopStatusNew,
								Sequence:         1,
								Type:             shipment.StopTypeDelivery,
								LocationID:       location2.ID,
								PlannedArrival:   timeutils.NowUnix() + 7200,
								PlannedDeparture: timeutils.NowUnix() + 10800,
								Weight:           func() *int { v := 1000; return &v }(),
								Pieces:           func() *int { v := 10; return &v }(),
							},
						},
					},
				},
			}

			result, err := repo.Create(ctx, newShipment, testUser.ID)
			require.NoError(t, err, "Create should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.NotEmpty(t, result.ID, "ID should be generated")
			assert.NotEmpty(t, result.ProNumber, "ProNumber should be generated")
			assert.Equal(t, shipment.StatusNew, result.Status, "Status should be New")
			assert.NotEmpty(t, result.Moves, "Moves should be created")
			assert.NotEmpty(t, result.Moves[0].Stops, "Stops should be created")
		})

		t.Run("Invalid ShipmentType", func(t *testing.T) {
			newShipment := &shipment.Shipment{
				ServiceTypeID:  serviceType.ID,
				ShipmentTypeID: pulid.MustNew("smt_"),
				CustomerID:     customerFixture.ID,
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
			}

			result, err := repo.Create(ctx, newShipment, testUser.ID)
			require.Error(t, err, "Create with invalid shipment type should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("Missing Required Fields", func(t *testing.T) {
			newShipment := &shipment.Shipment{
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
			}

			result, err := repo.Create(ctx, newShipment, testUser.ID)
			require.Error(t, err, "Create with missing fields should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("With Commodities", func(t *testing.T) {
			newShipment := &shipment.Shipment{
				ServiceTypeID:  serviceType.ID,
				ShipmentTypeID: shipmentType.ID,
				CustomerID:     customerFixture.ID,
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
				BOL:            "TEST-BOL-002",
				Commodities: []*shipment.ShipmentCommodity{
					{
						CommodityID: testCommodity.ID,
						Weight:      500,
						Pieces:      5,
					},
				},
			}

			result, err := repo.Create(ctx, newShipment, testUser.ID)
			require.NoError(t, err, "Create with commodities should not return error")
			require.NotNil(t, result, "Result should not be nil")
		})
	})

	// Test Update operations
	t.Run("Update", func(t *testing.T) {
		t.Run("Valid Update", func(t *testing.T) {
			// Get fresh copy to avoid version conflicts
			fresh, err := repo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: org.ID,
				BuID:  bu.ID,
			})
			require.NoError(t, err, "Should get fresh shipment")

			originalVersion := fresh.Version
			fresh.BOL = "UPDATED-BOL-001"
			fresh.Weight = intutils.SafeInt64PtrNonNil(2000)

			result, err := repo.Update(ctx, fresh, testUser.ID)
			require.NoError(t, err, "Update should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, "UPDATED-BOL-001", result.BOL, "BOL should be updated")
			assert.Equal(t, originalVersion+1, result.Version, "Version should be incremented")
		})

		t.Run("Version Conflict", func(t *testing.T) {
			// Get fresh copy and modify version to simulate conflict
			fresh, err := repo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: org.ID,
				BuID:  bu.ID,
			})
			require.NoError(t, err, "Should get fresh shipment")

			fresh.Version = 0 // Set to old version
			fresh.BOL = "CONFLICT-BOL"

			result, err := repo.Update(ctx, fresh, testUser.ID)
			require.Error(t, err, "Update with version conflict should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("Invalid ShipmentType Update", func(t *testing.T) {
			fresh, err := repo.GetByID(ctx, &repoports.GetShipmentByIDOptions{
				ID:    testShipment.ID,
				OrgID: org.ID,
				BuID:  bu.ID,
			})
			require.NoError(t, err, "Should get fresh shipment")

			fresh.ShipmentTypeID = pulid.MustNew("smt_")

			result, err := repo.Update(ctx, fresh, testUser.ID)
			require.Error(t, err, "Update with invalid shipment type should return error")
			require.Nil(t, result, "Result should be nil")
		})
	})

	// Test UpdateStatus operations
	t.Run("UpdateStatus", func(t *testing.T) {
		t.Run("Valid Status Update", func(t *testing.T) {
			opts := &repoports.UpdateShipmentStatusRequest{
				GetOpts: &repoports.GetShipmentByIDOptions{
					ID:    inTransitShipment.ID,
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Status: shipment.StatusCompleted,
			}

			result, err := repo.UpdateStatus(ctx, opts)
			require.NoError(t, err, "UpdateStatus should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, shipment.StatusCompleted, result.Status, "Status should be updated")
		})

		t.Run("Invalid Shipment ID", func(t *testing.T) {
			opts := &repoports.UpdateShipmentStatusRequest{
				GetOpts: &repoports.GetShipmentByIDOptions{
					ID:    pulid.MustNew("shp_"),
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Status: shipment.StatusCompleted,
			}

			result, err := repo.UpdateStatus(ctx, opts)
			require.Error(t, err, "UpdateStatus with invalid ID should return error")
			require.Nil(t, result, "Result should be nil")
		})
	})

	// Test Cancel operations
	t.Run("Cancel", func(t *testing.T) {
		t.Run("Valid Cancellation", func(t *testing.T) {
			// Create a new shipment to cancel
			newShipment := &shipment.Shipment{
				ServiceTypeID:  serviceType.ID,
				ShipmentTypeID: shipmentType.ID,
				CustomerID:     customerFixture.ID,
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
				BOL:            "CANCEL-TEST-001",
				Moves: []*shipment.ShipmentMove{
					{
						Status:   shipment.MoveStatusNew,
						Sequence: 0,
						Stops: []*shipment.Stop{
							{
								Status:           shipment.StopStatusNew,
								Sequence:         0,
								Type:             shipment.StopTypePickup,
								LocationID:       location1.ID,
								PlannedArrival:   timeutils.NowUnix(),
								PlannedDeparture: timeutils.NowUnix() + 3600, // Add 1 hour
							},
						},
					},
				},
			}

			created, err := repo.Create(ctx, newShipment, testUser.ID)
			require.NoError(t, err, "Should create shipment to cancel")

			now := timeutils.NowUnix()
			req := &repoports.CancelShipmentRequest{
				ShipmentID:   created.ID,
				OrgID:        org.ID,
				BuID:         bu.ID,
				CanceledByID: testUser.ID,
				CanceledAt:   now,
				CancelReason: "Test cancellation",
			}

			result, err := repo.Cancel(ctx, req)
			require.NoError(t, err, "Cancel should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, shipment.StatusCanceled, result.Status, "Status should be Canceled")
			assert.Equal(t, "Test cancellation", result.CancelReason, "Cancel reason should match")
			assert.Equal(t, &now, result.CanceledAt, "CanceledAt should match")
			assert.Equal(t, &testUser.ID, result.CanceledByID, "CanceledByID should match")
		})

		t.Run("Invalid Shipment ID", func(t *testing.T) {
			now := timeutils.NowUnix()
			req := &repoports.CancelShipmentRequest{
				ShipmentID:   pulid.MustNew("shp_"),
				OrgID:        org.ID,
				BuID:         bu.ID,
				CanceledByID: testUser.ID,
				CanceledAt:   now,
				CancelReason: "Test cancellation",
			}

			result, err := repo.Cancel(ctx, req)
			require.Error(t, err, "Cancel with invalid ID should return error")
			require.Nil(t, result, "Result should be nil")
		})

		t.Run("Wrong Organization", func(t *testing.T) {
			now := timeutils.NowUnix()
			req := &repoports.CancelShipmentRequest{
				ShipmentID:   testShipment.ID,
				OrgID:        pulid.MustNew("org_"),
				BuID:         bu.ID,
				CanceledByID: testUser.ID,
				CanceledAt:   now,
				CancelReason: "Test cancellation",
			}

			result, err := repo.Cancel(ctx, req)
			require.Error(t, err, "Cancel with wrong org should return error")
			require.Nil(t, result, "Result should be nil")
		})
	})

	// Test CheckForDuplicateBOLs operations
	t.Run("CheckForDuplicateBOLs", func(t *testing.T) {
		t.Run("No Duplicates", func(t *testing.T) {
			result, err := repo.CheckForDuplicateBOLs(ctx, "UNIQUE-BOL-123", org.ID, bu.ID, nil)
			require.NoError(t, err, "CheckForDuplicateBOLs should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Empty(t, result, "Should find no duplicates")
		})

		t.Run("Empty BOL", func(t *testing.T) {
			result, err := repo.CheckForDuplicateBOLs(ctx, "", org.ID, bu.ID, nil)
			require.NoError(t, err, "CheckForDuplicateBOLs with empty BOL should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.Empty(t, result, "Should find no duplicates for empty BOL")
		})

		t.Run("With Exclusion", func(t *testing.T) {
			// Create shipment with specific BOL
			newShipment := &shipment.Shipment{
				ServiceTypeID:  serviceType.ID,
				ShipmentTypeID: shipmentType.ID,
				CustomerID:     customerFixture.ID,
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				Status:         shipment.StatusNew,
				BOL:            "DUPLICATE-BOL-TEST",
			}

			created, err := repo.Create(ctx, newShipment, testUser.ID)
			require.NoError(t, err, "Should create shipment with BOL")

			// Check for duplicates excluding the created shipment
			result, err := repo.CheckForDuplicateBOLs(
				ctx,
				"DUPLICATE-BOL-TEST",
				org.ID,
				bu.ID,
				&created.ID,
			)
			require.NoError(t, err, "CheckForDuplicateBOLs with exclusion should not return error")
			assert.Empty(t, result, "Should find no duplicates when excluding self")

			// Check for duplicates without exclusion
			result2, err := repo.CheckForDuplicateBOLs(
				ctx,
				"DUPLICATE-BOL-TEST",
				org.ID,
				bu.ID,
				nil,
			)
			require.NoError(
				t,
				err,
				"CheckForDuplicateBOLs without exclusion should not return error",
			)
			assert.Len(t, result2, 1, "Should find one duplicate when not excluding")
			assert.Equal(t, created.ID, result2[0].ID, "Duplicate should match created shipment")
		})

		t.Run("Wrong Organization", func(t *testing.T) {
			result, err := repo.CheckForDuplicateBOLs(
				ctx,
				testShipment.BOL,
				pulid.MustNew("org_"),
				bu.ID,
				nil,
			)
			require.NoError(t, err, "CheckForDuplicateBOLs with wrong org should not return error")
			assert.Empty(t, result, "Should find no duplicates in wrong organization")
		})
	})

	// Test CalculateShipmentTotals operations
	t.Run("CalculateShipmentTotals", func(t *testing.T) {
		t.Run("Basic Calculation", func(t *testing.T) {
			testShipment := &shipment.Shipment{
				FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromFloat(1000.00)),
				OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromFloat(100.00)),
				AdditionalCharges: []*shipment.AdditionalCharge{
					{
						Amount: decimal.NewFromFloat(50.00),
					},
				},
			}

			result, err := repo.CalculateShipmentTotals(ctx, testShipment, testUser.ID)
			require.NoError(t, err, "CalculateShipmentTotals should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.True(
				t,
				result.BaseCharge.GreaterThanOrEqual(decimal.Zero),
				"BaseCharge should be non-negative",
			)
			assert.True(
				t,
				result.OtherChargeAmount.GreaterThanOrEqual(decimal.Zero),
				"OtherChargeAmount should be non-negative",
			)
			assert.True(
				t,
				result.TotalChargeAmount.GreaterThanOrEqual(decimal.Zero),
				"TotalChargeAmount should be non-negative",
			)
		})

		t.Run("Zero Amounts", func(t *testing.T) {
			testShipment := &shipment.Shipment{
				FreightChargeAmount: decimal.NewNullDecimal(decimal.Zero),
				OtherChargeAmount:   decimal.NewNullDecimal(decimal.Zero),
			}

			result, err := repo.CalculateShipmentTotals(t.Context(), testShipment, testUser.ID)
			require.NoError(
				t,
				err,
				"CalculateShipmentTotals with zero amounts should not return error",
			)
			require.NotNil(t, result, "Result should not be nil")
			assert.True(t, result.BaseCharge.Equal(decimal.Zero), "BaseCharge should be zero")
			assert.True(
				t,
				result.OtherChargeAmount.Equal(decimal.Zero),
				"OtherChargeAmount should be zero",
			)
			assert.True(
				t,
				result.TotalChargeAmount.Equal(decimal.Zero),
				"TotalChargeAmount should be zero",
			)
		})

		t.Run("Null Values", func(t *testing.T) {
			testShipment := &shipment.Shipment{
				// All null values
			}

			result, err := repo.CalculateShipmentTotals(t.Context(), testShipment, testUser.ID)
			require.NoError(
				t,
				err,
				"CalculateShipmentTotals with null values should not return error",
			)
			require.NotNil(t, result, "Result should not be nil")
		})
	})

	// Test Edge Cases and Error Conditions
	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Database Connection Error", func(t *testing.T) {
			// This would require mocking the database connection
			// For now, we'll test with valid connections
			t.Skip("Database connection error testing requires mocking")
		})

		t.Run("Large Pagination", func(t *testing.T) {
			opts := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  1000,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "Large pagination should not return error")
			require.NotNil(t, result, "Result should not be nil")
			assert.LessOrEqual(t, len(result.Items), 1000, "Items should not exceed limit")
		})

		t.Run("Very Long Query String", func(t *testing.T) {
			longQuery := string(make([]byte, 1000))
			for i := range longQuery {
				longQuery = longQuery[:i] + "a" + longQuery[i+1:]
			}

			opts := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  10,
					Offset: 0,
					Query:  longQuery,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			result, err := repo.List(ctx, opts)
			require.NoError(t, err, "Very long query should not return error")
			require.NotNil(t, result, "Result should not be nil")
		})
	})

	// Test Concurrent Operations
	t.Run("Concurrent Operations", func(t *testing.T) {
		t.Run("Concurrent List Operations", func(t *testing.T) {
			opts := &repoports.ListShipmentOptions{
				Filter: &ports.QueryOptions{
					Limit:  5,
					Offset: 0,
					TenantOpts: &ports.TenantOptions{
						OrgID: org.ID,
						BuID:  bu.ID,
					},
				},
			}

			// Run multiple concurrent list operations
			done := make(chan bool)
			for range 5 {
				go func() {
					defer func() { done <- true }()
					result, err := repo.List(ctx, opts)
					assert.NoError(t, err, "Concurrent list should not return error")
					assert.NotNil(t, result, "Concurrent list result should not be nil")
				}()
			}

			// Wait for all goroutines to complete
			for range 5 {
				select {
				case <-done:
					// Success
				case <-time.After(10 * time.Second):
					t.Fatal("Concurrent operations timed out")
				}
			}
		})
	})
}

// setupShipmentRepository creates a shipment repository with all dependencies
func setupShipmentRepository(log *logger.Logger) repoports.ShipmentRepository {
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{
		Store:    seqgen.NewSequenceStore(ts.DB, log),
		Provider: adapters.NewProNumberFormatProvider(),
		Logger:   log,
	})

	proNumberRepo := repositories.NewProNumberRepository(repositories.ProNumberRepositoryParams{
		Logger:    log,
		Generator: generator,
	})

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

	shipmentCommodityRepo := repositories.NewShipmentCommodityRepository(
		repositories.ShipmentCommodityRepositoryParams{
			Logger: log,
			DB:     ts.DB,
		},
	)

	manager := statemachine.NewManager(statemachine.ManagerParams{
		Logger: log,
	})

	calc := calculator.NewShipmentCalculator(calculator.ShipmentCalculatorParams{
		Logger:              log,
		StateMachineManager: manager,
	})

	additionalChargeRepo := repositories.NewAdditionalChargeRepository(
		repositories.AdditionalChargeRepositoryParams{
			Logger: log,
			DB:     ts.DB,
		},
	)

	return repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger:                      log,
		DB:                          ts.DB,
		ProNumberRepo:               proNumberRepo,
		ShipmentMoveRepository:      moveRepo,
		ShipmentCommodityRepository: shipmentCommodityRepo,
		Calculator:                  calc,
		AdditionalChargeRepository:  additionalChargeRepo,
	})
}
