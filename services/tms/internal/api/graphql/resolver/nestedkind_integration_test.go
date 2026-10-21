//go:build integration

package resolver

import (
	"context"
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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

const tractorDriversQuery = `query {
	tractors(input: { first: 10 }, includeWorkerDetails: true) {
		edges { node { id code primaryWorkerId primaryWorker { id firstName lastName } } }
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
					FirstName string
					LastName  string
				}
			}
		}
	}
}

// A person whose role reads tractors but not workers opens the tractor
// list. The driver is a worker, so the screen names no driver to them: the
// driver comes back as its id alone, whether or not row-level security is
// installed (self-hosted deployments run with it off).
func TestScreenRead_ATractorsDriversFollowWorkerRead(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenant := agentreadtest.SeedTenant(t, admin, "S")
	owner := postgres.NewTestConnection(admin)
	tenantCtx := dbscope.WithTenant(t.Context(), tenant.Scope())
	driverID := writeTractorWithDriver(t, readscope.With(tenantCtx, nil), owner, tenant)

	engine := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	noWorkerRead := func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"worker": {Scope: readscope.ScopeNone}}, nil
	}

	cases := []struct {
		name  string
		conn  *postgres.Connection
		grant readscope.Resolver
	}{
		{name: "row-level security off", conn: owner},
		{name: "row-level security enforced", conn: dbtest.NewRLSConnection(t, admin, dsn), grant: noWorkerRead},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := readscope.With(tenantCtx, c.grant)
			var response tractorDriversResponse
			screenClient(t, ctx, c.conn, engine, tenant).MustPost(tractorDriversQuery, &response)

			require.Len(t, response.Tractors.Edges, 1)
			node := response.Tractors.Edges[0].Node
			assert.Equal(t, driverID.String(), node.PrimaryWorkerID, "the tractor still names its driver's id")
			if node.PrimaryWorker != nil {
				assert.Empty(t, node.PrimaryWorker.FirstName, "a driver's name needs worker read")
				assert.Empty(t, node.PrimaryWorker.LastName, "a driver's name needs worker read")
			}
		})
	}
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
		Core: &base.Core{L: logger, PermissionEngine: engine},
		TractorService: tractorservice.New(tractorservice.Params{
			Logger:                    logger,
			Repo:                      tractorrepository.New(tractorrepository.Params{DB: conn, Logger: logger}),
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
	kind, err := equipmenttyperepository.New(equipmenttyperepository.Params{DB: conn, Logger: logger}).
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

	_, err = tractorrepository.New(tractorrepository.Params{DB: conn, Logger: logger}).
		Create(ctx, &tractor.Tractor{
			BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.EquipmentStatusAvailable,
			Code: "T1", EquipmentTypeID: kind.ID, EquipmentManufacturerID: maker.ID,
			PrimaryWorkerID: driver.ID, FuelType: domaintypes.IFTAFuelTypeDiesel,
		})
	require.NoError(t, err)

	return driver.ID
}
