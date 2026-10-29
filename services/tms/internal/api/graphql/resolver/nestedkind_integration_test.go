//go:build integration

package resolver

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/app/readscopeservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldvaluerepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const tractorDriversQuery = `query @requiresRead(kind: "worker") {
	tractors(input: { first: 10 }, includeWorkerDetails: true) {
		edges { node { id code primaryWorkerId primaryWorker { id wholeName profile { id } } } }
	}
}`

type tractorDriversResponse struct {
	Tractors struct {
		Edges []struct {
			Node struct {
				ID              string
				Code            string
				PrimaryWorkerID string
				PrimaryWorker   *struct {
					ID        string
					WholeName string
				}
			}
		}
	}
}

const withheldDriverProfile = "tractors.edges.node.primaryWorker.profile"

// A person whose role reads tractors but not workers opens the tractor list.
// The driver is a worker the tractor links to, so the screen shows the
// worker's reference view (the driver's name) and withholds the rest (the
// driver's profile), naming it in extensions.withheldByAccess. Row-level
// security gives the same answer: the read scope the person is sent narrows
// no kind they cannot read, so the database returns the driver and the field
// guard decides what of it shows. A root read of workers is still refused
// before it reaches the database.
func TestScreenRead_ATractorsDriverShowsTheWorkerReferenceView(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenant := agentreadtest.SeedTenant(t, admin, "S")
	owner := postgres.NewTestConnection(admin)
	tenantCtx := dbscope.WithTenant(t.Context(), tenant.Scope())
	driverID := writeTractorWithDriver(t, readscope.With(tenantCtx, nil), owner, tenant)

	noWorkerRead := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	enforced := &config.Config{}
	enforced.Database.RLS.Mode = databaseconfig.RLSModeEnforce
	scopes := readscopeservice.New(readscopeservice.Params{
		Config:   enforced,
		DB:       owner,
		Engine:   noWorkerRead,
		Registry: permissioncatalog.NewRegistry(),
	})
	principal := &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    tenant.Actor.UserID,
		UserID:         tenant.Actor.UserID,
		OrganizationID: tenant.Actor.OrganizationID,
		BusinessUnitID: tenant.Actor.BusinessUnitID,
	}
	rlsCtx := readscope.With(tenantCtx, scopes.Resolver(principal))

	t.Run("row-level security off shows the reference view", func(t *testing.T) {
		var response tractorDriversResponse
		raw := screenPost(
			t,
			screenClient(t, readscope.With(tenantCtx, nil), owner, noWorkerRead, tenant),
			tractorDriversQuery,
			&response,
		)

		require.Len(t, response.Tractors.Edges, 1)
		node := response.Tractors.Edges[0].Node
		assert.Equal(t, driverID.String(), node.PrimaryWorkerID)
		require.NotNil(t, node.PrimaryWorker)
		assert.Equal(t, "Hidden Driver", node.PrimaryWorker.WholeName,
			"a driver's name is in the worker reference view")
		assert.Contains(t, withheldOf(raw), withheldDriverProfile,
			"the driver's profile needs worker read")
	})

	t.Run("with worker read the whole driver shows", func(t *testing.T) {
		var response tractorDriversResponse
		raw := screenPost(t, screenClient(t, readscope.With(tenantCtx, nil), owner,
			&mocks.DenyResourcesPermissionEngine{}, tenant), tractorDriversQuery, &response)

		require.Len(t, response.Tractors.Edges, 1)
		require.NotNil(t, response.Tractors.Edges[0].Node.PrimaryWorker)
		assert.NotContains(t, withheldOf(raw), withheldDriverProfile)
	})

	t.Run("row-level security enforced shows the reference view", func(t *testing.T) {
		encoded, err := readscope.Encode(rlsCtx)
		require.NoError(t, err)
		assert.Empty(t, encoded, "a kind the person cannot read narrows nothing in the database")

		rls := dbtest.NewRLSConnection(t, admin, dsn)
		var response tractorDriversResponse
		raw := screenPost(t, screenClient(t, rlsCtx, rls, noWorkerRead, tenant),
			tractorDriversQuery, &response)

		require.Len(t, response.Tractors.Edges, 1)
		node := response.Tractors.Edges[0].Node
		assert.Equal(t, driverID.String(), node.PrimaryWorkerID)
		require.NotNil(
			t,
			node.PrimaryWorker,
			"the database returns the driver the tractor links to",
		)
		assert.Equal(t, driverID.String(), node.PrimaryWorker.ID)
		assert.Equal(t, "Hidden Driver", node.PrimaryWorker.WholeName,
			"a driver's name is in the worker reference view")
		assert.Contains(t, withheldOf(raw), withheldDriverProfile,
			"the driver's profile needs worker read, as with row-level security off")
	})

	t.Run("row-level security enforced still refuses a root read of workers", func(t *testing.T) {
		rls := dbtest.NewRLSConnection(t, admin, dsn)
		for _, query := range []string{
			`query { workers(input: { first: 10 }) { edges { node { id } } } }`,
			`query { worker(id: "` + driverID.String() + `") { id } }`,
		} {
			var response map[string]any
			err := screenClient(t, rlsCtx, rls, noWorkerRead, tenant).Post(query, &response)
			require.Error(t, err, query)
			assert.Contains(t, strings.ToLower(err.Error()), "permission", query)
		}
	})

	t.Run("a linked field outside the view that is non-null is refused", func(t *testing.T) {
		var response map[string]any
		err := screenClient(
			t,
			readscope.With(tenantCtx, nil),
			owner,
			noWorkerRead,
			tenant,
		).Post(`query {
			tractors(input: { first: 10 }, includeWorkerDetails: true) {
				edges { node { id primaryWorker { id phoneNumber } } }
			}
		}`, &response)
		require.Error(
			t,
			err,
			"a non-null field outside the view needs @requiresRead and worker read",
		)
		assert.Contains(t, err.Error(), "@requiresRead")
	})
}

func screenPost(t *testing.T, gql *client.Client, query string, out any) *client.Response {
	t.Helper()

	raw, err := gql.RawPost(query)
	require.NoError(t, err)
	if len(raw.Errors) > 0 && string(raw.Errors) != "null" {
		require.Failf(t, "the read failed", "%s", raw.Errors)
	}
	data, err := sonic.ConfigStd.Marshal(raw.Data)
	require.NoError(t, err)
	require.NoError(t, sonic.ConfigStd.Unmarshal(data, out))

	return raw
}

func withheldOf(raw *client.Response) []string {
	paths, _ := raw.Extensions["withheldByAccess"].([]any)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if name, ok := path.(string); ok {
			out = append(out, name)
		}
	}
	return out
}

func screenClient(
	t *testing.T,
	ctx context.Context,
	conn *postgres.Connection,
	engine *mocks.DenyResourcesPermissionEngine,
	tenant *agentreadtest.Tenant,
) *client.Client {
	t.Helper()

	logger := zap.NewNop()
	customFields := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger: logger,
		ValueRepo: customfieldvaluerepository.New(
			customfieldvaluerepository.Params{DB: conn, Logger: logger}),
		DefinitionRepo: customfieldrepository.New(
			customfieldrepository.Params{DB: conn, Logger: logger}),
	})
	root := FromServices(&Services{
		Core: &resolvercore.Core{L: logger, PermissionEngine: engine},
		TractorService: tractorservice.New(tractorservice.Params{
			Logger: logger,
			Repo: tractorrepository.New(tractorrepository.Params{
				DB:        conn,
				Logger:    logger,
				Locations: crossref.LocationColumns(),
				Moves:     crossref.MoveColumns(),
			}),
			CustomFieldsValuesService: customFields,
		}),
	})
	server := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: root}))
	server.AddTransport(transport.POST{})
	server.Use(gqlext.NewFieldGuardExtension(gqlext.FieldGuardParams{PermissionEngine: engine}))

	auth := &authctx.AuthContext{
		PrincipalType:  "user",
		PrincipalID:    tenant.Actor.UserID,
		UserID:         tenant.Actor.UserID,
		OrganizationID: tenant.Actor.OrganizationID,
		BusinessUnitID: tenant.Actor.BusinessUnitID,
	}
	ctx = gqlctx.WithAuthContext(ctx, auth)

	return client.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r.WithContext(ctx))
	}))
}

func writeTractorWithDriver(
	t *testing.T,
	ctx context.Context,
	conn *postgres.Connection,
	tenant *agentreadtest.Tenant,
) pulid.ID {
	t.Helper()

	logger := zap.NewNop()
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	kind, err := equipmenttyperepository.New(equipmenttyperepository.Params{
		DB:     conn,
		Logger: logger,
	}).
		Create(ctx, &equipmenttype.EquipmentType{
			BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
			Code: "TRC", Class: equipmenttype.ClassTractor, DeckType: equipmenttype.DeckTypeFlatbed,
		})
	require.NoError(t, err)
	maker, err := equipmentmanufacturerrepository.New(
		equipmentmanufacturerrepository.Params{DB: conn, Logger: logger},
	).Create(ctx, &equipmentmanufacturer.EquipmentManufacturer{
		BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive, Name: "Maker",
	})
	require.NoError(t, err)

	now := timeutils.NowUnix()
	driver, err := workerrepository.New(workerrepository.Params{DB: conn, Logger: logger}).
		Create(ctx, &worker.Worker{
			BusinessUnitID: bu, OrganizationID: org, StateID: tenant.StateID,
			Status: domaintypes.StatusActive, Type: worker.WorkerTypeEmployee,
			DriverType: worker.DriverTypeOTR, FirstName: "Hidden", LastName: "Driver",
			AddressLine1: "1 Main", City: "Dallas", PostalCode: "75001", Gender: worker.GenderMale,
			Profile: &worker.WorkerProfile{
				DOB:           now - 86400*365*30,
				LicenseNumber: "DL" + org.String()[4:12],
				LicenseExpiry: now + 86400*365,
				HireDate:      now - 86400*365,
			},
		})
	require.NoError(t, err)

	_, err = tractorrepository.New(tractorrepository.Params{
		DB:        conn,
		Logger:    logger,
		Locations: crossref.LocationColumns(),
		Moves:     crossref.MoveColumns(),
	}).
		Create(ctx, &tractor.Tractor{
			BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.EquipmentStatusAvailable,
			Code: "T1", EquipmentTypeID: kind.ID, EquipmentManufacturerID: maker.ID,
			PrimaryWorkerID: driver.ID, FuelType: domaintypes.IFTAFuelTypeDiesel,
		})
	require.NoError(t, err)

	return driver.ID
}
