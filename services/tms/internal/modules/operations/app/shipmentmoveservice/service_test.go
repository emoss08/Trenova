package shipmentmoveservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/equipmentcontinuityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateStatus_RecomputesShipmentState(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()
	moveRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *shipmentports.UpdateMoveStatusRequest) bool {
			return req.MoveID == moveID && req.Status == shipment.MoveStatusInTransit
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusInTransit,
		}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			Status:            shipment.StatusAssigned,
			Version:           1,
			Moves: []*shipment.ShipmentMove{
				{
					ID:         moveID,
					ShipmentID: shipmentID,
					Status:     shipment.MoveStatusInTransit,
				},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StatusInTransit, entity.Status)
			return entity, nil
		}).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(nil, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentRepo,
		holdRepo:       holdRepo,
		controlRepo:    controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, shipment.MoveStatusInTransit, entity.Status)
}

func TestBulkUpdateStatus_RejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:     moveID,
			Status: shipment.MoveStatusCompleted,
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         moveRepo,
		shipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		holdRepo:     shipmentportsmock.NewMockShipmentHoldRepository(t),
		controlRepo:  tenantportsmock.NewMockShipmentControlRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.BulkUpdateStatus(t.Context(), &shipmentports.BulkUpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveIDs:    []pulid.ID{moveID},
		Status:     shipment.MoveStatusAssigned,
	})

	require.Nil(t, entity)
	require.Error(t, err)
	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
}

func TestSplitMove_RejectsNonSimpleMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			Status:     shipment.MoveStatusNew,
			ShipmentID: pulid.MustNew("shp_"),
		}, nil).
		Once()
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && req.ExpandMoveDetails && !req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID: moveID,
			Stops: []*shipment.Stop{
				{Type: shipment.StopTypePickup},
				{Type: shipment.StopTypeDelivery},
				{Type: shipment.StopTypeDelivery},
			},
			Status: shipment.MoveStatusNew,
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         moveRepo,
		shipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		holdRepo:     shipmentportsmock.NewMockShipmentHoldRepository(t),
		controlRepo:  tenantportsmock.NewMockShipmentControlRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	response, err := svc.SplitMove(t.Context(), &shipmentports.SplitMoveRequest{
		TenantInfo:            tenantInfo,
		MoveID:                moveID,
		NewDeliveryLocationID: pulid.MustNew("loc_"),
		SplitPickupTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: 5,
			ScheduledWindowEnd:   new(int64(6)),
		},
		NewDeliveryTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: 7,
			ScheduledWindowEnd:   new(int64(8)),
		},
	})

	require.Nil(t, response)
	require.Error(t, err)
}

func TestSplitMove_RecomputesShipmentState(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	bridgeLocationID := pulid.MustNew("loc_")
	newLocationID := pulid.MustNew("loc_")

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && req.ExpandMoveDetails && !req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
			Assignment: &shipment.Assignment{ID: pulid.MustNew("asn_")},
			Stops: []*shipment.Stop{
				{
					Type:                 shipment.StopTypePickup,
					ScheduleType:         shipment.StopScheduleTypeOpen,
					ScheduledWindowStart: 1,
					ScheduledWindowEnd:   new(int64(2)),
					LocationID:           pulid.MustNew("loc_"),
				},
				{
					Type:                 shipment.StopTypeDelivery,
					ScheduleType:         shipment.StopScheduleTypeOpen,
					ScheduledWindowStart: 3,
					ScheduledWindowEnd:   new(int64(4)),
					LocationID:           bridgeLocationID,
				},
			},
		}, nil).
		Once()
	moveRepo.EXPECT().
		SplitMove(mock.Anything, mock.AnythingOfType("*shipmentports.SplitMoveRequest")).
		Return(&shipmentports.SplitMoveResponse{
			OriginalMove: &shipment.ShipmentMove{
				ID:         moveID,
				ShipmentID: shipmentID,
				Status:     shipment.MoveStatusAssigned,
			},
			NewMove: &shipment.ShipmentMove{
				ID:         pulid.MustNew("sm_"),
				ShipmentID: shipmentID,
				Status:     shipment.MoveStatusNew,
			},
		}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			Status:            shipment.StatusAssigned,
			Version:           2,
			Moves: []*shipment.ShipmentMove{
				{
					ID:         moveID,
					ShipmentID: shipmentID,
					Status:     shipment.MoveStatusAssigned,
					Assignment: &shipment.Assignment{ID: pulid.MustNew("asn_")},
				},
				{ID: pulid.MustNew("sm_"), ShipmentID: shipmentID, Status: shipment.MoveStatusNew},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		Return(&shipment.Shipment{}, nil).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)

	svc := &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         moveRepo,
		shipmentRepo: shipmentRepo,
		holdRepo:     holdRepo,
		controlRepo:  controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	response, err := svc.SplitMove(t.Context(), &shipmentports.SplitMoveRequest{
		TenantInfo:            tenantInfo,
		MoveID:                moveID,
		NewDeliveryLocationID: newLocationID,
		SplitPickupTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: 5,
			ScheduledWindowEnd:   new(int64(6)),
		},
		NewDeliveryTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: 7,
			ScheduledWindowEnd:   new(int64(8)),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestUpdateStatus_RecomputesDelayedShipmentStateUsingControlThreshold(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()
	moveRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.AnythingOfType("*shipmentports.UpdateMoveStatusRequest")).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusInTransit,
		}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			Version:           1,
			Moves: []*shipment.ShipmentMove{
				{
					ID:         moveID,
					ShipmentID: shipmentID,
					Status:     shipment.MoveStatusInTransit,
					Stops: []*shipment.Stop{
						{
							Type:               shipment.StopTypePickup,
							Status:             shipment.StopStatusInTransit,
							ScheduledWindowEnd: new(int64(100)),
						},
						{
							Type:               shipment.StopTypeDelivery,
							Status:             shipment.StopStatusNew,
							ScheduledWindowEnd: new(int64(4000)),
						},
					},
				},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StatusDelayed, entity.Status)
			return entity, nil
		}).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: new(int16(1)),
		}, nil).
		Once()
	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(nil, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentRepo,
		holdRepo:       holdRepo,
		controlRepo:    controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 200 }),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
}

func TestUpdateStatus_DoesNotAutoDelayShipmentWhenToggleDisabled(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()
	moveRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.AnythingOfType("*shipmentports.UpdateMoveStatusRequest")).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusInTransit,
		}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			Version:           1,
			Moves: []*shipment.ShipmentMove{
				{
					ID:         moveID,
					ShipmentID: shipmentID,
					Status:     shipment.MoveStatusInTransit,
					Stops: []*shipment.Stop{
						{
							Type:               shipment.StopTypePickup,
							Status:             shipment.StopStatusInTransit,
							ScheduledWindowEnd: new(int64(100)),
						},
						{
							Type:               shipment.StopTypeDelivery,
							Status:             shipment.StopStatusNew,
							ScheduledWindowEnd: new(int64(4000)),
						},
					},
				},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StatusInTransit, entity.Status)
			return entity, nil
		}).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          false,
			AutoDelayShipmentsThreshold: new(int16(1)),
		}, nil).
		Once()
	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(nil, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentRepo,
		holdRepo:       holdRepo,
		controlRepo:    controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 200 }),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
}

func TestUpdateStatus_RejectsTrailerAlreadyInProgressOnAnotherMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	conflictingMoveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trac_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: moveID,
			TractorID:      &tractorID,
			TrailerID:      &trailerID,
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, moveID).
		Return(nil, nil).
		Once()
	assignmentRepo.EXPECT().
		FindInProgressByTrailerID(mock.Anything, tenantInfo, trailerID, moveID).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: conflictingMoveID,
			TrailerID:      &trailerID,
		}, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentportsmock.NewMockShipmentRepository(t),
		holdRepo:       shipmentportsmock.NewMockShipmentHoldRepository(t),
		controlRepo:    tenantportsmock.NewMockShipmentControlRepository(t),
		continuityRepo: equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Trailer is currently in progress on another move", err.Error())
}

func TestUpdateStatus_RejectsTractorAlreadyInProgressOnAnotherMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	conflictingMoveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tractorID := pulid.MustNew("trac_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
		}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: moveID,
			TractorID:      &tractorID,
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, moveID).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: conflictingMoveID,
			TractorID:      &tractorID,
		}, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentportsmock.NewMockShipmentRepository(t),
		holdRepo:       shipmentportsmock.NewMockShipmentHoldRepository(t),
		controlRepo:    tenantportsmock.NewMockShipmentControlRepository(t),
		continuityRepo: equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Tractor is currently in progress on another move", err.Error())
}

func TestUpdateStatus_AdvancesEquipmentContinuityOnCompletion(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	assignmentID := pulid.MustNew("asn_")
	tractorID := pulid.MustNew("trac_")
	trailerID := pulid.MustNew("tr_")
	deliveryLocationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusInTransit,
		}, nil).
		Once()
	moveRepo.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *shipmentports.UpdateMoveStatusRequest) bool {
			return req.MoveID == moveID && req.Status == shipment.MoveStatusCompleted
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusCompleted,
			Assignment: &shipment.Assignment{
				ID:             assignmentID,
				ShipmentMoveID: moveID,
				TractorID:      &tractorID,
				TrailerID:      &trailerID,
			},
			Stops: []*shipment.Stop{
				{Type: shipment.StopTypePickup, Sequence: 0, LocationID: pulid.MustNew("loc_")},
				{Type: shipment.StopTypeDelivery, Sequence: 1, LocationID: deliveryLocationID},
			},
		}, nil).
		Once()

	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDeliveryHold(mock.Anything, mock.MatchedBy(func(req *shipmentports.ActiveShipmentHoldRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo == tenantInfo
		})).
		Return(false, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
			return req.TenantInfo == tenantInfo &&
				req.EquipmentType == equipmentcontinuity.EquipmentTypeTractor &&
				req.EquipmentID == tractorID &&
				req.CurrentLocationID == deliveryLocationID &&
				req.SourceShipmentID == shipmentID &&
				req.SourceShipmentMoveID == moveID &&
				req.SourceAssignmentID == assignmentID
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
			return req.TenantInfo == tenantInfo &&
				req.EquipmentType == equipmentcontinuity.EquipmentTypeTrailer &&
				req.EquipmentID == trailerID &&
				req.CurrentLocationID == deliveryLocationID &&
				req.SourceShipmentID == shipmentID &&
				req.SourceShipmentMoveID == moveID &&
				req.SourceAssignmentID == assignmentID
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			Status:            shipment.StatusInTransit,
			Version:           1,
			Moves: []*shipment.ShipmentMove{
				{
					ID:         moveID,
					ShipmentID: shipmentID,
					Status:     shipment.MoveStatusCompleted,
				},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		Return(&shipment.Shipment{}, nil).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           moveRepo,
		assignmentRepo: shipmentportsmock.NewMockAssignmentRepository(t),
		shipmentRepo:   shipmentRepo,
		holdRepo:       holdRepo,
		controlRepo:    controlRepo,
		continuityRepo: continuityRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusCompleted,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, shipment.MoveStatusCompleted, entity.Status)
}

func TestUpdateStatus_RejectsDeliveryBlockingHold(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ExpandMoveDetails && req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusInTransit,
		}, nil).
		Once()

	holdRepo := shipmentportsmock.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDeliveryHold(mock.Anything, mock.MatchedBy(func(req *shipmentports.ActiveShipmentHoldRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo == tenantInfo
		})).
		Return(true, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         moveRepo,
		shipmentRepo: shipmentportsmock.NewMockShipmentRepository(t),
		holdRepo:     holdRepo,
		controlRepo:  tenantportsmock.NewMockShipmentControlRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      ratingtest.NewEngine(t),
			AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		}),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstateCoordinator(),
	}

	entity, err := svc.UpdateStatus(t.Context(), &shipmentports.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusCompleted,
	})

	require.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func shipmentstateCoordinator() *shipmentstate.Coordinator {
	return shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })
}

//go:fix inline
//go:fix inline
func ptrInt16(v int16) *int16 {
	return new(v)
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}
