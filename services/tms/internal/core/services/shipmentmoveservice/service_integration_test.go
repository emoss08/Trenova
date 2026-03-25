//go:build integration

package shipmentmoveservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accessorialchargerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestUpdateStatusIntegrationRecomputesShipmentStatus(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, _, tenantInfo, fixture := newIntegrationService(t, ctx, db)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo, testutil.ShipmentGraphParams{
		BOL:          "BOL-UPD-001",
		ProNumber:    "PRO-UPD-001",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	updated, err := svc.UpdateStatus(ctx, &repositories.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     graph.Moves[0].ID,
		Status:     shipment.MoveStatusAssigned,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipment.MoveStatusAssigned, updated.Status)

	persisted, err := shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         graph.Shipment.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, shipment.StatusAssigned, persisted.Status)
	require.Len(t, persisted.Moves, 1)
	assert.Equal(t, shipment.MoveStatusAssigned, persisted.Moves[0].Status)
}

func TestBulkUpdateStatusIntegrationRollsBackOnInvalidMove(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, moveRepo, tenantInfo, fixture := newIntegrationService(t, ctx, db)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo, testutil.ShipmentGraphParams{
		BOL:          "BOL-BULK-001",
		ProNumber:    "PRO-BULK-001",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew, shipment.MoveStatusCompleted},
	})

	updated, err := svc.BulkUpdateStatus(ctx, &repositories.BulkUpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveIDs:    []pulid.ID{graph.Moves[0].ID, graph.Moves[1].ID},
		Status:     shipment.MoveStatusAssigned,
	})
	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	persistedMoves, err := moveRepo.GetMovesByShipmentID(ctx, &repositories.GetMovesByShipmentIDRequest{
		ShipmentID:        graph.Shipment.ID,
		TenantInfo:        tenantInfo,
		ExpandMoveDetails: true,
	})
	require.NoError(t, err)
	require.Len(t, persistedMoves, 2)
	assert.Equal(t, shipment.MoveStatusNew, persistedMoves[0].Status)
	assert.Equal(t, shipment.MoveStatusCompleted, persistedMoves[1].Status)
}

func TestSplitMoveIntegrationPersistsTwoLegHandoff(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, _, tenantInfo, fixture := newIntegrationService(t, ctx, db)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo, testutil.ShipmentGraphParams{
		BOL:          "BOL-SPLIT-001",
		ProNumber:    "PRO-SPLIT-001",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})
	newDestination := testutil.MustCreateLocation(
		t,
		ctx,
		db,
		fixture,
		tenantInfo,
		"FINAL",
		"Final Destination",
	)

	originalDelivery := graph.Moves[0].Stops[1]
	response, err := svc.SplitMove(ctx, &repositories.SplitMoveRequest{
		TenantInfo:            tenantInfo,
		MoveID:                graph.Moves[0].ID,
		NewDeliveryLocationID: newDestination.ID,
		SplitPickupTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: originalDelivery.EffectiveScheduledWindowEnd() + 1_800,
			ScheduledWindowEnd:   integrationInt64Ptr(originalDelivery.EffectiveScheduledWindowEnd() + 3_600),
		},
		NewDeliveryTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: originalDelivery.EffectiveScheduledWindowEnd() + 7_200,
			ScheduledWindowEnd:   integrationInt64Ptr(originalDelivery.EffectiveScheduledWindowEnd() + 9_000),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	persisted, err := shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         graph.Shipment.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, persisted.Moves, 2)
	assert.Equal(t, shipment.StatusNew, persisted.Status)

	originalMove := persisted.Moves[0]
	newMove := persisted.Moves[1]

	assert.Equal(t, graph.Moves[0].ID, originalMove.ID)
	assert.Equal(t, int64(0), originalMove.Sequence)
	require.Len(t, originalMove.Stops, 2)
	assert.Equal(t, shipment.StopTypePickup, originalMove.Stops[0].Type)
	assert.Equal(t, shipment.StopTypeSplitDelivery, originalMove.Stops[1].Type)
	assert.Equal(t, originalDelivery.LocationID, originalMove.Stops[1].LocationID)

	assert.NotEqual(t, graph.Moves[0].ID, newMove.ID)
	assert.Equal(t, int64(1), newMove.Sequence)
	assert.Nil(t, newMove.Assignment)
	require.Len(t, newMove.Stops, 2)
	assert.Equal(t, shipment.StopTypeSplitPickup, newMove.Stops[0].Type)
	assert.Equal(t, originalDelivery.LocationID, newMove.Stops[0].LocationID)
	assert.Equal(t, shipment.StopTypeDelivery, newMove.Stops[1].Type)
	assert.Equal(t, newDestination.ID, newMove.Stops[1].LocationID)
}

func newIntegrationService(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
) (
	portservices.ShipmentMoveService,
	repositories.ShipmentRepository,
	repositories.ShipmentMoveRepository,
	pagination.TenantInfo,
	*testutil.ShipmentIntegrationFixture,
) {
	t.Helper()

	conn := postgres.NewTestConnection(db)
	moveRepo := shipmentmoverepository.New(shipmentmoverepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	additionalChargeRepo := shipmentadditionalchargerepository.New(
		shipmentadditionalchargerepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		},
	)
	accessorialRepo := accessorialchargerepository.New(accessorialchargerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	commodityRepo := shipmentcommodityrepository.New(shipmentcommodityrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	formulaRepo := formulatemplaterepository.New(formulatemplaterepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{
		DB:                         conn,
		Logger:                     zap.NewNop(),
		Generator:                  testutil.TestSequenceGenerator{SingleValue: "PRO-INTEGRATION"},
		MoveRepository:             moveRepo,
		AdditionalChargeRepository: additionalChargeRepo,
		CommodityRepository:        commodityRepo,
	})
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("repositories.GetShipmentControlRequest")).
		Return(&tenant.ShipmentControl{
			AutoDelayShipments:          true,
			AutoDelayShipmentsThreshold: int16Ptr(30),
		}, nil).
		Maybe()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	formulaEngine := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	formulaSvc := formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Engine:   formulaEngine,
		Resolver: res,
		Repo:     formulaRepo,
	})
	commercial := shipmentcommercial.New(shipmentcommercial.Params{
		Formula:         formulaSvc,
		AccessorialRepo: accessorialRepo,
	})

	svc := New(Params{
		Logger:       zap.NewNop(),
		DB:           conn,
		Repo:         moveRepo,
		ShipmentRepo: shipmentRepo,
		ControlRepo:  controlRepo,
		Commercial:   commercial,
		Coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenantInfo)

	return svc, shipmentRepo, moveRepo, tenantInfo, fixture
}

func int16Ptr(value int16) *int16 {
	return &value
}

func integrationInt64Ptr(value int64) *int64 {
	return &value
}
