//go:build integration

package shipmenthandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/app/readscopeservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	driverHomeAddress    = "12 Driver Home Lane"
	driverEmergencyPhone = "614-555-0199"
)

type dispatchedLoad struct {
	shipmentID     pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	userID         pulid.ID
	workerFirst    string
	admin          *bun.DB
	dsn            string
}

// seedDispatchedLoad finds a seeded shipment move, assigns it a seeded driver
// with a home address and an emergency contact, and returns the shipment.
func seedDispatchedLoad(t *testing.T) (context.Context, *postgres.Connection, dispatchedLoad) {
	t.Helper()

	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true})
	require.NoError(t, err)

	var move struct {
		MoveID         pulid.ID `bun:"move_id"`
		ShipmentID     pulid.ID `bun:"shipment_id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewRaw(`
		SELECT sm.id AS move_id, sm.shipment_id, sm.organization_id, sm.business_unit_id
		FROM shipment_moves AS sm
		WHERE sm.status != 'Canceled'
		ORDER BY sm.organization_id, sm.business_unit_id, sm.shipment_id
		LIMIT 1`).Scan(ctx, &move))
	require.False(t, move.MoveID.IsNil())

	var driver struct {
		ID        pulid.ID `bun:"id"`
		FirstName string   `bun:"first_name"`
	}
	require.NoError(t, db.NewRaw(
		"SELECT id, first_name FROM workers WHERE organization_id = ? AND business_unit_id = ? ORDER BY id LIMIT 1",
		move.OrganizationID, move.BusinessUnitID,
	).Scan(ctx, &driver))
	require.False(t, driver.ID.IsNil())

	_, err = db.NewRaw(
		"UPDATE workers SET address_line1 = ?, emergency_contact_phone = ? WHERE id = ? AND organization_id = ?",
		driverHomeAddress, driverEmergencyPhone, driver.ID, move.OrganizationID,
	).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(
		"DELETE FROM assignments WHERE shipment_move_id = ? AND organization_id = ?",
		move.MoveID, move.OrganizationID,
	).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(`
		INSERT INTO assignments (id, business_unit_id, organization_id, shipment_move_id, primary_worker_id, status)
		VALUES (?, ?, ?, ?, ?, 'New')`,
		pulid.MustNew("a_"), move.BusinessUnitID, move.OrganizationID, move.MoveID, driver.ID,
	).Exec(ctx)
	require.NoError(t, err)

	return ctx, postgres.NewTestConnection(db), dispatchedLoad{
		shipmentID:     move.ShipmentID,
		organizationID: move.OrganizationID,
		businessUnitID: move.BusinessUnitID,
		userID:         pulid.MustNew("usr_"),
		workerFirst:    driver.FirstName,
		admin:          db,
		dsn:            dsn,
	}
}

// getExpandedShipment serves GET /shipments/:id?expandShipmentDetails=true
// through the shipment handler, reading the shipment from Postgres, as a
// caller the engine denies the given resources.
func getExpandedShipment(
	t *testing.T,
	ctx context.Context,
	conn *postgres.Connection,
	load dispatchedLoad,
	engine permissionports.PermissionEngine,
	guarded bool,
) (map[string]any, http.Header) {
	t.Helper()

	return serveExpandedShipment(t, ctx, conn, load, engine, guarded, nil)
}

// serveExpandedShipment is getExpandedShipment with the request's own
// context, and so its tenant and read scope, when scopes is set: the
// shipment is read as the request reads it.
func serveExpandedShipment(
	t *testing.T,
	ctx context.Context,
	conn *postgres.Connection,
	load dispatchedLoad,
	engine permissionports.PermissionEngine,
	guarded bool,
	scopes permissionports.ReadScopes,
) (map[string]any, http.Header) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := shipmentrepository.New(shipmentrepository.Params{
		DB:             conn,
		Logger:         zap.NewNop(),
		MoveRepository: shipmentmoverepository.New(shipmentmoverepository.Params{DB: conn, Logger: zap.NewNop()}),
	})
	service := shipmentportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Get(mock.Anything, mock.Anything).
		RunAndReturn(func(requestCtx context.Context, req *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			require.True(t, req.ExpandShipmentDetails)
			if scopes != nil {
				return repo.GetByID(requestCtx, req)
			}
			return repo.GetByID(ctx, req)
		})

	handler := buildShipmentHandler(t, handlerDeps{
		service:     service,
		permissions: engine,
	})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, load.userID, load.businessUnitID, load.organizationID)
		if scopes != nil {
			c.Request = c.Request.WithContext(dbscope.WithTenant(c.Request.Context(), dbscope.Tenant{
				OrganizationID: load.organizationID,
				BusinessUnitID: load.businessUnitID,
				UserID:         load.userID,
			}))
		}
		c.Next()
	})
	if scopes != nil {
		router.Use(middleware.NewReadScopeMiddleware(middleware.ReadScopeMiddlewareParams{
			Scopes: scopes,
		}).Install())
	}
	if guarded {
		router.Use(responseguard.NewGuard(responseguard.GuardParams{
			PermissionEngine: engine,
			ReferenceViews:   refview.NewPort(refview.Default(), engine),
		}).Install())
	}
	handler.RegisterRoutes(router.Group("/api/v1"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(
		ctx, http.MethodGet,
		"/api/v1/shipments/"+load.shipmentID.String()+"/?expandShipmentDetails=true", nil,
	))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := map[string]any{}
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &body))

	return body, rec.Header()
}

func primaryWorkerOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	moves, ok := body["moves"].([]any)
	require.True(t, ok, "the expanded shipment carries its moves")
	for _, raw := range moves {
		move, _ := raw.(map[string]any)
		assignment, _ := move["assignment"].(map[string]any)
		if worker, found := assignment["primaryWorker"].(map[string]any); found {
			return worker
		}
	}
	require.Fail(t, "no move carries an assigned driver")

	return nil
}

func TestGetShipment_WithholdsTheDriversPersonalDetailsFromAShipmentReader(t *testing.T) {
	ctx, conn, load := seedDispatchedLoad(t)
	shipmentOnly := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}

	leaked, _ := getExpandedShipment(t, ctx, conn, load, shipmentOnly, false)
	unguarded := primaryWorkerOf(t, leaked)
	require.Equal(t, driverHomeAddress, unguarded["addressLine1"],
		"the route hydrates the driver's whole row; without the guard it leaves as is")
	require.Equal(t, driverEmergencyPhone, unguarded["emergencyContactPhone"])

	body, header := getExpandedShipment(t, ctx, conn, load, shipmentOnly, true)
	driver := primaryWorkerOf(t, body)
	assert.Equal(t, load.workerFirst, driver["firstName"], "the driver's name is in the view")
	assert.NotEmpty(t, driver["id"])
	for _, field := range []string{
		"addressLine1", "city", "postalCode", "phoneNumber",
		"emergencyContactName", "emergencyContactPhone",
	} {
		assert.Nilf(t, driver[field], "%s of a driver reached through a shipment", field)
	}
	assert.Contains(t, header.Get(responseguard.WithheldHeader), "moves.assignment.primaryWorker")
	assert.NotEmpty(t, body["proNumber"], "the shipment itself is the route's own record")
}

func TestGetShipment_ShowsTheWholeDriverToAWorkerReader(t *testing.T) {
	ctx, conn, load := seedDispatchedLoad(t)

	body, header := getExpandedShipment(t, ctx, conn, load, &mocks.AllowAllPermissionEngine{}, true)
	driver := primaryWorkerOf(t, body)

	assert.Equal(t, driverHomeAddress, driver["addressLine1"])
	assert.Equal(t, driverEmergencyPhone, driver["emergencyContactPhone"])
	assert.NotContains(t, header.Get(responseguard.WithheldHeader), "primaryWorker")
}

// With row-level security enforced the driver a shipment links to is still
// read from the database for a person who cannot read workers, and the
// response guard shows its reference view, exactly as with row-level security
// off. The read scope the person is sent narrows no kind they cannot read.
func TestGetShipment_UnderRowLevelSecurityShowsTheDriversReferenceView(t *testing.T) {
	ctx, _, load := seedDispatchedLoad(t)
	shipmentOnly := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	enforced := &config.Config{}
	enforced.Database.RLS.Mode = config.RLSModeEnforce
	scopes := readscopeservice.New(readscopeservice.Params{
		Config:   enforced,
		Engine:   shipmentOnly,
		Registry: permission.NewRegistry(),
	})
	rls := dbtest.NewRLSConnection(t, load.admin, load.dsn)

	body, header := serveExpandedShipment(t, ctx, rls, load, shipmentOnly, true, scopes)
	driver := primaryWorkerOf(t, body)
	assert.Equal(t, load.workerFirst, driver["firstName"], "the driver's name is in the view")
	assert.NotEmpty(t, driver["id"])
	for _, field := range []string{
		"addressLine1", "city", "postalCode", "phoneNumber",
		"emergencyContactName", "emergencyContactPhone",
	} {
		assert.Nilf(t, driver[field], "%s of a driver reached through a shipment", field)
	}
	assert.Contains(t, header.Get(responseguard.WithheldHeader), "moves.assignment.primaryWorker")
	assert.NotEmpty(t, body["proNumber"])

	whole, header := serveExpandedShipment(t, ctx, rls, load, &mocks.DenyResourcesPermissionEngine{}, true,
		readscopeservice.New(readscopeservice.Params{
			Config:   enforced,
			Engine:   &mocks.DenyResourcesPermissionEngine{},
			Registry: permission.NewRegistry(),
		}))
	assert.Equal(t, driverHomeAddress, primaryWorkerOf(t, whole)["addressLine1"])
	assert.NotContains(t, header.Get(responseguard.WithheldHeader), "primaryWorker")
}
