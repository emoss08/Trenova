//go:build integration

package shipmenthandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/app/readscopeservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	admin          *keel.DB
	dsn            string
}

// seedDispatchedLoad finds a seeded shipment move, assigns it a seeded driver
// with a home address and an emergency contact, and returns the shipment.
func seedDispatchedLoad(t *testing.T) (context.Context, *postgres.Connection, dispatchedLoad) {
	t.Helper()

	ctx, db, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().
		Where(sm.Status.Ne(shipment.MoveStatusCanceled)).
		Order(sm.OrganizationID.Asc(), sm.BusinessUnitID.Asc(), sm.ShipmentID.Asc()).
		First(dbscope.WithSystem(ctx, "find a seeded shipment move for the test"), db)
	require.NoError(t, err)

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: move.OrganizationID,
		BusinessUnitID: move.BusinessUnitID,
	})
	w := workerdb.Workers
	driver, err := w.Order(w.ID.Asc()).First(tenantCtx, db)
	require.NoError(t, err)

	_, err = w.Where(w.ID.Eq(driver.ID)).Update(tenantCtx, db,
		w.AddressLine1.To(driverHomeAddress),
		w.EmergencyContactPhone.To(driverEmergencyPhone),
	)
	require.NoError(t, err)

	a := shipmentdb.Assignments
	_, err = a.Where(a.ShipmentMoveID.Eq(move.ID)).Delete(tenantCtx, db)
	require.NoError(t, err)
	require.NoError(t, a.Create(tenantCtx, db, &shipment.Assignment{
		ShipmentMoveID:  move.ID,
		PrimaryWorkerID: &driver.ID,
		Status:          shipment.AssignmentStatusNew,
	}))

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
		DB:     conn,
		Logger: zap.NewNop(),
		MoveRepository: shipmentmoverepository.New(
			shipmentmoverepository.Params{DB: conn, Logger: zap.NewNop()},
		),
	})
	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Get(mock.Anything, mock.Anything).
		RunAndReturn(func(requestCtx context.Context, req *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			require.True(t, req.ExpandShipmentDetails)
			return repo.GetByID(requestCtx, req)
		})

	handler := buildShipmentHandler(t, handlerDeps{
		service:     service,
		permissions: engine,
	})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, load.userID, load.businessUnitID, load.organizationID)
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
	rls := dbtest.NewRLSConnection(t, load.admin, load.dsn)
	scopes := readscopeservice.New(readscopeservice.Params{
		Config:   enforced,
		DB:       rls,
		Engine:   shipmentOnly,
		Registry: permissioncatalog.NewRegistry(),
	})

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

	whole, header := serveExpandedShipment(
		t,
		ctx,
		rls,
		load,
		&mocks.DenyResourcesPermissionEngine{},
		true,
		readscopeservice.New(readscopeservice.Params{
			Config:   enforced,
			DB:       rls,
			Engine:   &mocks.DenyResourcesPermissionEngine{},
			Registry: permissioncatalog.NewRegistry(),
		}),
	)
	assert.Equal(t, driverHomeAddress, primaryWorkerOf(t, whole)["addressLine1"])
	assert.NotContains(t, header.Get(responseguard.WithheldHeader), "primaryWorker")
}
