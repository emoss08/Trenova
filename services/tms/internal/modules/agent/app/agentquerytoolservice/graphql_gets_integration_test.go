//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierintelrepository"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/telematicsrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workercredentialrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdrugalcoholrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/telematicsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// convertedGets names each converted get by the converted list whose rows
// it reads one of.
var convertedGets = map[string]string{
	"customer_list":           "customer_get",
	"location_list":           "location_get",
	"carrier_list":            "carrier_get",
	"equipment_list_tractors": "equipment_get_tractor",
	"equipment_list_trailers": "equipment_get_trailer",
	"worker_list":             "worker_get",
}

// getCase is one converted get read against real repositories: how to
// write its record, and what its card withholds at the default data access.
type getCase struct {
	*referenceCase
	// internal are the fields an agent at Internal is never shown, which a
	// get names in withheldByAccess.
	internal []string
	// newly are the fields the repository tool returned ungated that its
	// kind keeps Restricted, which the get now withholds at Internal.
	newly []string
	// key is the field the record carries its own id in (a driver's hours
	// carry the worker's).
	key string
	// absent marks a get whose record of another tenant reads as nothing on
	// file, as a driver no ELD reported for does, rather than as an error.
	absent bool
	// ungated marks a record of no kind, which no role ceiling withholds.
	ungated bool
}

func TestConvertedGets_ReadOneRecordOfTheTenantUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	lists, services := convertedListCases(conn)
	events := carrierintelrepository.NewEventRepository(carrierintelrepository.Params{
		DB: conn, Logger: logger,
	})
	services.CarrierIntelService = carrierintelservice.New(carrierintelservice.Params{
		Logger: logger, Config: &config.Config{}, EventRepo: events,
	})
	workers := workerrepository.New(workerrepository.Params{DB: conn, Logger: logger})
	credentials := workercredentialrepository.New(
		workercredentialrepository.Params{DB: conn, Logger: logger})
	services.WorkerCredentialService = workercredentialservice.New(workercredentialservice.Params{
		Logger: logger, Repo: credentials, WorkerRepo: workers,
	})
	draws := workerdrugalcoholrepository.New(
		workerdrugalcoholrepository.Params{DB: conn, Logger: logger})
	services.WorkerDrugAlcoholService = workerdrugalcoholservice.New(workerdrugalcoholservice.Params{
		Logger: logger, Repo: draws, WorkerRepo: workers,
	})
	hours := telematicsrepository.New(telematicsrepository.Params{DB: conn, Logger: logger})
	services.TelematicsService = telematicsservice.New(telematicsservice.Params{
		Repo: hours, Logger: logger,
	})

	gets := make(map[string]*getCase, len(convertedGets)+4)
	for idx := range lists {
		if name, ok := convertedGets[lists[idx].tool]; ok {
			gets[name] = &getCase{referenceCase: &lists[idx], key: "id"}
		}
	}
	gets["worker_get"].newly = []string{
		"leaveType", "profile.nextTrainingDue", "profile.safetyRating",
	}
	gets["worker_get"].internal = []string{
		"email", "phoneNumber", "city", "postalCode", "emergencyContactName",
		"emergencyContactPhone", "leaveType", "state", "manager", "profile.hireDate",
		"profile.terminationDate", "profile.disqualificationReason", "profile.nextTrainingDue",
		"profile.safetyRating", "profile.licenseNumber", "profile.licenseState",
	}
	gets["carrier_get_intelligence_event"] = &getCase{key: "id", referenceCase: &referenceCase{
		tool: "carrier_get_intelligence_event",
		write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
			bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
			rows, err := events.InsertIgnoreDuplicates(ctx, []*carrierintel.CarrierIntelEvent{{
				ID: pulid.MustNew("cievt_"), BusinessUnitID: bu, OrganizationID: org,
				SubjectType: carrierintel.SubjectTypeCarrier, SubjectID: "DOT" + label,
				DOTNumber: "100" + label[1:], SubjectName: "Carrier " + label,
				Provider: integration.TypeCarrierOK, Source: carrierintel.EventSourceSnapshotDiff,
				Category: carrierintel.SectionInsurance, Severity: carrierintel.SeverityHigh,
				PriorValue: "Active", CurrentValue: "Lapsed", Summary: "Insurance lapsed",
				DetectedAt: timeutils.NowUnix(), Fingerprint: "fp" + label,
				Status: carrierintel.EventStatusOpen,
			}})
			if err != nil {
				return pulid.Nil, err
			}
			require.Len(t, rows, 1)

			return rows[0].ID, nil
		},
	}}
	driver := func(ctx context.Context, tenant *agentreadtest.Tenant, label string) (pulid.ID, error) {
		row, err := workers.Create(ctx, testWorker(tenant, "G"+label, false, pulid.Nil))
		return created(row, err)
	}
	gets["worker_get_credential"] = &getCase{
		key: "id", internal: []string{"notes", "archiveReason"}, newly: []string{"notes"},
		referenceCase: &referenceCase{
			tool: "worker_get_credential",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
				bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
				workerID, err := driver(ctx, tenant, label)
				if err != nil {
					return pulid.Nil, err
				}
				kind, err := credentials.CreateType(ctx, credentialType(bu, org, "G"+label))
				if err != nil {
					return pulid.Nil, err
				}
				expires := timeutils.NowUnix() + 86400*90
				row, err := credentials.Create(ctx, &workerports.CreateWorkerCredentialRequest{
					Entity: &worker.WorkerCredential{
						BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
						CredentialTypeID: kind.ID, Status: worker.CredentialStatusActive,
						Number: "N-" + label, IssuingAuthority: "TX", ExpiresAt: &expires,
						Notes: "Renewal booked " + label,
					},
				})
				return created(row, err)
			},
		},
	}
	gets["worker_get_dot_random_draw"] = &getCase{
		key: "id",
		referenceCase: &referenceCase{
			tool: "worker_get_dot_random_draw",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
				bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
				workerID, err := driver(ctx, tenant, label)
				if err != nil {
					return pulid.Nil, err
				}
				pool, err := draws.CreatePool(ctx, &worker.DOTRandomPool{
					BusinessUnitID: bu, OrganizationID: org, Code: "P" + label,
					Name: "Pool " + label, Status: domaintypes.StatusActive,
					Period: worker.RandomPeriodQuarterly, DrugRatePercent: 50,
					AlcoholRatePercent: 10, IncludedDriverTypes: []string{},
				})
				if err != nil {
					return pulid.Nil, err
				}
				now := timeutils.NowUnix()
				row, err := draws.CreateDrawWithEntries(ctx, &worker.DOTRandomDraw{
					ID: pulid.MustNew("drdraw_"), BusinessUnitID: bu, OrganizationID: org,
					PoolID: pool.ID, PeriodKey: "2026-Q" + label, PeriodStart: now - 86400*30,
					PeriodEnd: now + 86400*60, Status: worker.RandomDrawStatusFinal,
					PoolSize: 1, DrugTarget: 1, DrugSelected: 1, Seed: "seed" + label,
					Method: "sha256-rank", DrawnAt: now, FinalizedAt: &now,
				}, []*worker.DOTRandomDrawEntry{{
					ID: pulid.MustNew("drde_"), BusinessUnitID: bu, OrganizationID: org,
					WorkerID: workerID, Substance: worker.DOTSubstanceDrug, Rank: 1,
					Status: worker.RandomEntrySelected,
				}})
				return created(row, err)
			},
		},
	}
	gets["worker_get_hos"] = &getCase{
		key: "workerId", absent: true, ungated: true,
		referenceCase: &referenceCase{
			tool: "worker_get_hos",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
				workerID, err := driver(ctx, tenant, label)
				if err != nil {
					return pulid.Nil, err
				}
				now := timeutils.NowUnix()
				err = hours.UpsertWorkerHOSStates(ctx, []*telematics.WorkerHOSState{{
					OrganizationID: tenant.Actor.OrganizationID,
					BusinessUnitID: tenant.Actor.BusinessUnitID, WorkerID: workerID,
					Provider: "Samsara", ProviderDriverID: "drv-" + label,
					DutyStatus: telematics.DutyStatusDriving, DriveRemainingMs: 90 * 60_000,
					ShiftRemainingMs: 5 * 3_600_000, CycleRemainingMs: 40 * 3_600_000,
					BreakRemainingMs: 3 * 3_600_000, RecordedAt: now - 600, ReceivedAt: now,
				}})
				return workerID, err
			},
		},
	}
	require.Len(t, gets, len(convertedGets)+4)
	require.ElementsMatch(t, agentquerytoolservice.ConvertedGetNames(), keysOf(gets),
		"every converted get is read on Postgres")

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: services})

	env := &convertedListEnv{
		engine: engine,
		reader: reader,
		public: public,
		mine:   agentreadtest.SeedTenant(t, admin, "A"),
		theirs: agentreadtest.SeedTenant(t, admin, "B"),
	}
	for name, c := range gets {
		t.Run(name, func(t *testing.T) {
			get, err := agentquerytoolservice.ConvertedGetTool(name, env.reader, env.engine)
			require.NoError(t, err)
			ours := writeReferenceRows(t, c.referenceCase, env.mine, "A")
			other := writeReferenceRows(t, c.referenceCase, env.theirs, "B")
			ours.ctx = env.within(ours.ctx, env.mine)
			actor := env.mine.Actor
			param := get.ParamSchema()["required"].([]string)[0]
			atInternal := func(id string) map[string]any {
				data, queryErr := get.Query(ours.ctx, &agentports.QueryToolParams{
					OrganizationID:    actor.OrganizationID,
					BusinessUnitID:    actor.BusinessUnitID,
					Actor:             actor,
					Timezone:          "America/Chicago",
					Params:            map[string]any{param: id},
					DataAccessCeiling: permission.SensitivityInternal,
					RunID:             pulid.MustNew("ar_"),
				})
				require.NoError(t, queryErr)
				return asMap(t, data)
			}

			t.Run("our record, as the card reads it", func(t *testing.T) {
				for _, id := range ours.rows {
					record := callTool(t, ours.ctx, get, actor, map[string]any{param: id})
					assert.Equal(t, id, record[c.key])
				}
			})

			t.Run("at the default agent data access only what stays Restricted is withheld", func(t *testing.T) {
				record := atInternal(ours.rows[0])
				withheld, _ := record["withheldByAccess"].([]any)
				names := make([]string, 0, len(withheld))
				for _, field := range withheld {
					names = append(names, field.(string))
				}
				assert.Subset(t, c.internal, names, "nothing Internal is withheld")
				assert.Subset(t, names, c.newly, "what stays Restricted is withheld")
				for _, field := range names {
					assert.NotContains(t, record, field)
				}
			})

			t.Run("another tenant's record is not ours to read", func(t *testing.T) {
				data, queryErr := get.Query(ours.ctx, &agentports.QueryToolParams{
					OrganizationID: actor.OrganizationID,
					BusinessUnitID: actor.BusinessUnitID,
					Actor:          actor,
					Timezone:       "America/Chicago",
					Params:         map[string]any{param: other.rows[0]},
					RunID:          pulid.MustNew("ar_"),
				})
				if !c.absent {
					require.Error(t, queryErr)
					return
				}
				require.NoError(t, queryErr)
				record := asMap(t, data)
				assert.Equal(t, "Unknown", record["dutyStatus"],
					"another tenant's driver reads as no hours on file")
				assert.NotContains(t, record, "driveRemainingMinutes")
			})

			t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
				hidden, hiddenErr := agentquerytoolservice.ConvertedGetTool(name, env.public, &publicRole{})
				require.NoError(t, hiddenErr)
				record := callTool(t, ours.ctx, hidden, actor, map[string]any{param: ours.rows[0]})
				withheld, _ := record["withheldByAccess"].([]any)
				if c.ungated {
					assert.Empty(t, withheld, "a record of no kind is gated by no ceiling")
					return
				}
				require.NotEmpty(t, withheld)
				assert.Equal(t, ours.rows[0], record[c.key])
				for _, field := range withheld {
					assert.NotContains(t, record, field)
				}
			})
		})
	}
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}

	return out
}

// checkConvertedGet holds one converted get to its tenant's records under
// row-level security, with nothing withheld at the default data access and
// what a Public role may not see withheld. idOf finds the record's id in
// what the card returns.
func checkConvertedGet(
	t *testing.T,
	name string,
	c *referenceCase,
	env *convertedListEnv,
	idOf func(map[string]any) any,
) {
	t.Helper()

	get, err := agentquerytoolservice.ConvertedGetTool(name, env.reader, env.engine)
	require.NoError(t, err)
	ours := writeReferenceRows(t, c, env.mine, "A")
	other := writeReferenceRows(t, c, env.theirs, "B")
	ours.ctx = env.within(ours.ctx, env.mine)
	actor := env.mine.Actor
	param := get.ParamSchema()["required"].([]string)[0]

	t.Run("our record, as the card reads it", func(t *testing.T) {
		for _, id := range ours.rows {
			record := callTool(t, ours.ctx, get, actor, map[string]any{param: id})
			assert.Equal(t, id, idOf(record))
			assert.NotContains(t, record, "withheldByAccess")
		}
	})

	t.Run("nothing is withheld at the default agent data access", func(t *testing.T) {
		data, err := get.Query(ours.ctx, &agentports.QueryToolParams{
			OrganizationID:    actor.OrganizationID,
			BusinessUnitID:    actor.BusinessUnitID,
			Actor:             actor,
			Timezone:          "America/Chicago",
			Params:            map[string]any{param: ours.rows[0]},
			DataAccessCeiling: permission.SensitivityInternal,
			RunID:             pulid.MustNew("ar_"),
		})
		require.NoError(t, err)
		assert.NotContains(t, asMap(t, data), "withheldByAccess")
	})

	t.Run("another tenant's record is not found", func(t *testing.T) {
		_, err := get.Query(ours.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			Actor:          actor,
			Timezone:       "America/Chicago",
			Params:         map[string]any{param: other.rows[0]},
			RunID:          pulid.MustNew("ar_"),
		})
		require.Error(t, err)
	})

	t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
		hidden, err := agentquerytoolservice.ConvertedGetTool(name, env.public, &publicRole{})
		require.NoError(t, err)
		record := callTool(t, ours.ctx, hidden, actor, map[string]any{param: ours.rows[0]})
		withheld, _ := record["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		for _, field := range withheld {
			assert.NotContains(t, record, field)
		}
	})
}
