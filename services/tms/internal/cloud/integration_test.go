//go:build integration

package cloud_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/interceptors"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/emoss08/trenova/internal/testutil/heavyslotstest"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudcommandrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudtenantrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpeoplerepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/aiusagemeter"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentmeter"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationmeter"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/passwordresetrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/supportaccessrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/tenantbootstraprepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/app/firstadmininviteservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedusagerepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmeter"
	"github.com/emoss08/trenova/internal/testutil/metricstest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type measuredStatements struct{}

type statementCounter struct {
	mu         sync.Mutex
	statements []string
}

func (c *statementCounter) BeforeQuery(ctx context.Context, e *keel.QueryEvent) context.Context {
	if ctx.Value(measuredStatements{}) == nil {
		return ctx
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements = append(c.statements, e.Op+" "+e.Table)
	return ctx
}

func (c *statementCounter) measure(ctx context.Context, run func(context.Context)) []string {
	c.mu.Lock()
	c.statements = nil
	c.mu.Unlock()
	run(context.WithValue(ctx, measuredStatements{}, true))
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.statements)
}

func (c *statementCounter) AfterQuery(context.Context, *keel.QueryEvent) {}

type memorySessions struct {
	sessionports.SessionRepository
	mu   sync.Mutex
	byID map[pulid.ID]*session.Session
}

func (m *memorySessions) add(user, org pulid.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := pulid.MustNew("ses_")
	m.byID[id] = &session.Session{ID: id, UserID: user, OrganizationID: org}
}

func (m *memorySessions) ListForUser(_ context.Context, user pulid.ID) ([]*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*session.Session, 0)
	for _, s := range m.byID {
		if s.UserID == user {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memorySessions) Delete(_ context.Context, id pulid.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
	return nil
}

func (m *memorySessions) DeleteAllForUser(_ context.Context, user pulid.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.byID {
		if s.UserID == user {
			delete(m.byID, id)
		}
	}
	return nil
}

func (m *memorySessions) of(org pulid.ID) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.byID {
		if s.OrganizationID == org {
			n++
		}
	}
	return n
}

type startedWorkflows struct {
	kernelports.WorkflowStarter
	mu  sync.Mutex
	ids []string
}

func (w *startedWorkflows) StartWorkflow(
	_ context.Context,
	opts client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ids = append(w.ids, opts.ID)
	return nil, nil
}

type memoryObjects struct {
	kernelports.Client
	mu   sync.Mutex
	keys map[string]bool
}

func (o *memoryObjects) DeletePrefix(_ context.Context, prefix string) (int64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var n int64
	for key := range o.keys {
		if strings.HasPrefix(key, prefix) {
			delete(o.keys, key)
			n++
		}
	}
	return n, nil
}

func (o *memoryObjects) count(prefix string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := 0
	for key := range o.keys {
		if strings.HasPrefix(key, prefix) {
			n++
		}
	}
	return n
}

type purgedKeys struct {
	fragments []string
}

func (k *purgedKeys) DeleteMatching(_ context.Context, fragment string, _ func()) (int64, error) {
	k.fragments = append(k.fragments, fragment)
	return 1, nil
}

type purgedSearch struct {
	organizations []pulid.ID
}

func (s *purgedSearch) DeleteOrganizationDocuments(_ context.Context, org pulid.ID) (int, error) {
	s.organizations = append(s.organizations, org)
	return 2, nil
}

type rig struct {
	ctx       context.Context
	admin     *keel.DB
	conn      *postgres.Connection
	svc       *tenantlifecycle.Service
	features  *managedfeaturesservice.Service
	usage     *managedusageservice.Service
	purge     cloudtenantports.Purge
	tenants   cloudtenantports.Repository
	sessions  *memorySessions
	workflows *startedWorkflows
	objects   *memoryObjects
	keys      *purgedKeys
	search    *purgedSearch
	counter   *statementCounter
}

func randomSecret(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 24)
	_, err := rand.Read(raw)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func enforcedConnection(
	t *testing.T,
	admin *keel.DB,
	dsn string,
	counter *statementCounter,
) (*postgres.Connection, *databaseconfig.Config) {
	t.Helper()
	suffix := strings.ToLower(pulid.MustNew("r").String())
	appRole, systemRole, password := "cloud_app_"+suffix, "cloud_sys_"+suffix, randomSecret(t)
	_, err := postgres.ProvisionRLSRoles(t.Context(), admin, postgres.ProvisionRLSRolesParams{
		App:    postgres.RLSLoginRole{User: appRole, Password: password},
		System: postgres.RLSLoginRole{User: systemRole, Password: password},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, role := range []string{appRole, systemRole} {
			_, _ = keel.Exec(
				context.Background(),
				admin,
				fmt.Sprintf("DROP ROLE IF EXISTS %q", role),
			)
		}
	})

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	key := make([]byte, 48)
	_, err = rand.Read(key)
	require.NoError(t, err)
	dbCfg := &databaseconfig.Config{
		Host:     parsed.Hostname(),
		Port:     port,
		Name:     strings.TrimPrefix(parsed.Path, "/"),
		User:     appRole,
		Password: password,
		SSLMode:  "disable",
		RLS: databaseconfig.RLSConfig{
			Mode:       databaseconfig.RLSModeEnforce,
			ScopeKeyID: "kcloud",
			ScopeKey:   base64.StdEncoding.EncodeToString(key),
		},
		System: databaseconfig.Role{User: systemRole, Password: password},
	}
	_, err = postgres.ProvisionRLS(t.Context(), admin, dbCfg)
	require.NoError(t, err)

	lc := fxtest.NewLifecycle(t)
	conn, err := postgres.NewConnection(postgres.ConnectionParams{
		Lifecycle:       lc,
		AppConfig:       &appconfig.Config{Name: "trenova-cloud-test"},
		DatabaseConfig:  dbCfg,
		ReportingConfig: &reportingconfig.Config{},
		Logger:          zaptest.NewLogger(t),
		Metrics:         metricstest.Disabled(),
		Observers:       []keel.Observer{counter},
	})
	require.NoError(t, err)
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)
	return conn, dbCfg
}

func newRig(t *testing.T) *rig {
	t.Helper()
	return newRigOn(t, false)
}

func newRigOn(t *testing.T, seeded bool) *rig {
	t.Helper()
	setup := seedtest.SetupTestDBWithDSN
	if seeded {
		setup = seededtest.SetupTestDBWithDSN
	}
	ctx, admin, dsn, cleanup := setup(t)
	t.Cleanup(cleanup)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	t.Cleanup(cancel)

	r := &rig{
		ctx:       ctx,
		admin:     admin,
		counter:   &statementCounter{},
		sessions:  &memorySessions{byID: map[pulid.ID]*session.Session{}},
		workflows: &startedWorkflows{},
		objects:   &memoryObjects{keys: map[string]bool{}},
		keys:      &purgedKeys{},
		search:    &purgedSearch{},
	}
	r.conn, _ = enforcedConnection(t, admin, dsn, r.counter)
	logger := zap.NewNop()
	clock := func() int64 { return time.Now().Unix() }
	scopes := managedscope.NewResolver(true)
	cfg := &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared}

	r.features = managedfeaturesservice.New(managedfeaturesservice.Params{
		Repo: managedfeaturesrepository.New(
			managedfeaturesrepository.Params{DB: r.conn, Logger: logger},
		),
		Scopes: scopes,
		Clock:  clock,
		Logger: logger,
	})
	r.usage = managedusageservice.New(managedusageservice.Params{
		Repo: managedusagerepository.New(
			managedusagerepository.Params{DB: r.conn, Logger: logger},
		),
		Scopes: scopes,
		Counters: []managedusageports.HourlyCounter{
			shipmentmeter.New(shipmentmeter.Params{DB: r.conn}).Counter,
			aiusagemeter.New(aiusagemeter.Params{DB: r.conn}).Pages,
		},
		Levels: []managedusageports.LevelReader{
			documentmeter.New(documentmeter.Params{DB: r.conn}).Reader,
			organizationmeter.New(organizationmeter.Params{DB: r.conn}).Reader,
		},
		Clock:  clock,
		Logger: logger,
	})
	usageReceiver := managedusageservice.NewReceiver(managedusageservice.ReceiverParams{
		Usage: r.usage, Logger: logger,
	})
	quota := quotaservice.NewManaged(r.usage, scopes, clock)
	users := userrepository.New(userrepository.Params{
		DB: r.conn, Logger: logger, Quota: quota, Managed: cfg,
	})
	audit := &securityaudittest.Recorder{}
	invites := firstadmininviteservice.New(firstadmininviteservice.Params{
		DB:    r.conn,
		Users: users,
		Tokens: passwordresetrepository.New(
			passwordresetrepository.Params{DB: r.conn, Logger: logger},
		),
		SecurityConfig: &securityconfig.Config{},
		Auditor:        audit,
		Logger:         logger,
	})
	r.tenants = cloudtenantrepository.New(cloudtenantrepository.Params{DB: r.conn})
	people := tenantpeoplerepository.New(tenantpeoplerepository.Params{DB: r.conn})
	r.purge = tenantpurgerepository.New(tenantpurgerepository.Params{DB: r.conn, Logger: logger})
	r.svc = tenantlifecycle.New(tenantlifecycle.Params{
		Config:         cfg,
		DB:             r.conn,
		Tenants:        r.tenants,
		Administrators: people.Administrators,
		Members:        people.Members,
		Bootstrap: tenantbootstraprepository.New(tenantbootstraprepository.Params{
			DB: r.conn, Logger: logger, Quota: quota,
		}),
		Features: managedfeaturesservice.NewReceiver(managedfeaturesservice.ReceiverParams{
			Features: r.features, Logger: logger,
		}),
		Usage:     usageReceiver,
		Readings:  usageReceiver,
		Invites:   invites,
		Sessions:  r.sessions,
		Auditor:   audit,
		Workflows: r.workflows,
		Commands: cloudcommandrepository.New(cloudcommandrepository.Params{
			DB: r.conn, Clock: clock,
		}),
		Grants: supportaccessrepository.NewGrantRepository(supportaccessrepository.Params{
			Entitlements: entitlementtest.Instance(), DB: r.conn, Logger: logger,
		}),
		Buckets: newBucket(),
		Secrets: encryptionservice.NewSecretCipher(encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("lifecycle-test-key-not-a-real-secret"),
		)),
		Scopes: scopes,
		Clock:  clock,
		Logger: logger,
	})
	return r
}

const (
	instanceA = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	instanceB = "6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21"
)

func entry(
	id, email string,
	revision int64,
	features ...platformcatalog.FeatureKey,
) *cloudtenant.Desired {
	keys := make([]string, 0, len(features))
	for _, f := range features {
		keys = append(keys, string(f))
	}
	return &cloudtenant.Desired{
		InstanceID:       id,
		Revision:         revision,
		Lifecycle:        cloudtenant.LifecycleActive,
		OrganizationName: "Tenant " + id[:4],
		Plan: cloudtenant.Plan{
			Key:                 "shared",
			Name:                "Shared",
			BackupRetentionDays: 14,
			Support:             "email",
		},
		Addons:     []string{},
		Features:   keys,
		AIProvider: cloudtenant.AIProviderCustomer,
		Allowances: []cloudtenant.Allowance{},
		FirstAdministrator: cloudtenant.FirstAdministrator{
			Name:  "Admin " + id[:4],
			Email: email,
			Invite: &cloudtenant.Invite{
				TokenSHA256: tokenOf(id),
				ExpiresAt:   time.Now().Add(72 * time.Hour).UTC(),
			},
		},
		Account: cloudtenant.Account{
			PlanURL:         "https://console.test/instances/" + id + "/plan",
			FinalBackupDays: 30,
		},
		Commands: []cloudtenant.Command{},
	}
}

func tokenOf(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func (r *rig) apply(t *testing.T, e *cloudtenant.Desired) *cloudtenant.Status {
	t.Helper()
	status, err := r.svc.Apply(r.ctx, e.InstanceID, e, nil)
	require.NoError(t, err)
	return status
}

func (r *rig) tenant(t *testing.T, id string) *cloudtenant.Tenant {
	t.Helper()
	held, err := r.tenants.Get(r.ctx, id)
	require.NoError(t, err)
	return held
}

func scopeOf(t *cloudtenant.Tenant) dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: t.OrganizationID, BusinessUnitID: t.BusinessUnitID}
}

func (r *rig) remaining(t *testing.T, held *cloudtenant.Tenant) map[string]int64 {
	t.Helper()
	counts, err := r.purge.Remaining(r.ctx, scopeOf(held))
	require.NoError(t, err)
	out := make(map[string]int64, len(counts))
	for _, c := range counts {
		out[c.Table] = c.Rows
	}
	return out
}

func TestTwoConcurrentAppliesMakeOneTenant(t *testing.T) {
	r := newRig(t)
	e := entry(instanceA, "first@a.test", 1, platformcatalog.FeatureAccounting)

	var wg sync.WaitGroup
	statuses := make([]*cloudtenant.Status, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Go(func() {
			statuses[i], errs[i] = r.svc.Apply(r.ctx, instanceA, e, nil)
		})
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, statuses[0].State, statuses[1].State)

	orgs, err := keel.QueryValue[int64](r.ctx, r.admin,
		"SELECT count(*) FROM organizations WHERE name = $1", e.OrganizationName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), orgs)
	users, err := keel.QueryValue[int64](r.ctx, r.admin,
		"SELECT count(*) FROM users WHERE email_address = $1", "first@a.test")
	require.NoError(t, err)
	assert.Equal(t, int64(1), users)
	invites, err := keel.QueryValue[int64](
		r.ctx,
		r.admin,
		"SELECT count(*) FROM password_reset_tokens WHERE token_hash = $1 AND used_at IS NULL",
		tokenOf(instanceA),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), invites, "the first administrator's invite is installed once")

	status := r.apply(t, e)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	require.NotNil(t, status.Administrator)
	assert.False(t, status.Administrator.SetUp)
	assert.Equal(t, tokenOf(instanceA), status.Administrator.InviteTokenSHA256)
}

func TestASecondTenantSeesNoneOfTheFirstsRowsAndHasItsOwnPlan(t *testing.T) {
	r := newRig(t)
	r.apply(t, entry(instanceA, "a@a.test", 1, platformcatalog.FeatureAccounting))
	r.apply(t, entry(instanceB, "b@b.test", 1))
	a, b := r.tenant(t, instanceA), r.tenant(t, instanceB)

	countUsers := func(scope dbscope.Tenant, org pulid.ID) int64 {
		var n int64
		err := r.conn.Transact(
			dbscope.WithTenant(r.ctx, scope),
			kernelports.TxOptions{ReadOnly: true},
			func(ctx context.Context) error {
				rows, qerr := keel.QueryAll(ctx, r.conn.Keel(ctx), pgx.RowTo[int64],
					"SELECT count(*) FROM users WHERE current_organization_id = $1", org.String())
				if qerr == nil {
					n = rows[0]
				}
				return qerr
			},
		)
		require.NoError(t, err)
		return n
	}
	assert.Equal(t, int64(2), countUsers(scopeOf(a), a.OrganizationID),
		"the first administrator and the tenant's own system actor")
	assert.Equal(t, int64(0), countUsers(scopeOf(b), a.OrganizationID),
		"row-level security hides the first tenant's people from the second")

	enabledFor := func(held *cloudtenant.Tenant) bool {
		on, err := r.features.Enabled(
			dbscope.WithTenant(r.ctx, scopeOf(held)),
			platformcatalog.FeatureAccounting,
		)
		require.NoError(t, err)
		return on
	}
	assert.True(t, enabledFor(a))
	assert.False(t, enabledFor(b))
}

func TestATakenAddressRefusesTheEntryWithoutNamingTheHolder(t *testing.T) {
	r := newRig(t)
	r.apply(t, entry(instanceA, "shared@carrier.test", 1))

	status := r.apply(t, entry(instanceB, "Shared@Carrier.test", 1))
	assert.Equal(t, cloudtenant.StateApplying, status.State)
	require.NotNil(t, status.Refused)
	assert.Equal(t, cloudtenant.RefusalInvalid, status.Refused.Code)
	assert.NotContains(t, strings.ToLower(status.Refused.Reason), "shared@carrier.test")
	assert.NotContains(t, status.Refused.Reason, "Tenant "+instanceA[:4])
	assert.False(t, r.tenant(t, instanceB).Made())
}

func TestSuspensionRefusesEveryRequestAndEndsSessionsWhileAnotherTenantIsServed(t *testing.T) {
	r := newRig(t)
	r.apply(t, entry(instanceA, "a@a.test", 1))
	r.apply(t, entry(instanceB, "b@b.test", 1))
	a, b := r.tenant(t, instanceA), r.tenant(t, instanceB)
	r.sessions.add(a.AdminUserID, a.OrganizationID)
	r.sessions.add(b.AdminUserID, b.OrganizationID)

	suspended := entry(instanceA, "a@a.test", 2)
	suspended.Lifecycle = cloudtenant.LifecycleSuspended
	assert.Equal(t, cloudtenant.StateSuspended, r.apply(t, suspended).State)
	assert.Contains(t, r.workflows.ids, tenantjobscontract.SessionsWorkflowID(instanceA, 2))

	_, err := r.svc.RevokeSessions(
		r.ctx,
		&tenantlifecycle.RevocationRequest{Tenant: scopeOf(a)},
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, 0, r.sessions.of(a.OrganizationID))
	assert.Equal(t, 1, r.sessions.of(b.OrganizationID))

	admission := &planservice.Admission{
		Plans: planservice.NewUnlimited(), Features: r.features, Standing: r.features,
	}
	admit := func(held *cloudtenant.Tenant, write bool) error {
		return admission.Admit(
			dbscope.WithTenant(r.ctx, scopeOf(held)),
			&planservice.AdmissionRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: held.OrganizationID,
					BuID:  held.BusinessUnitID,
				},
				Write: write,
			},
		)
	}
	for _, write := range []bool{false, true} {
		err = admit(a, write)
		require.Error(t, err)
		assert.True(t, errortypes.IsSuspendedError(err))
		require.NoError(t, admit(b, write))
	}

	assert.Equal(t, cloudtenant.StateActive, r.apply(t, entry(instanceA, "a@a.test", 3)).State)
	require.NoError(t, admit(a, true), "resuming restores the tenant")
}

type missingOrganizations struct{}

func (missingOrganizations) GetByID(context.Context, pulid.ID) (*tenant.Organization, error) {
	return nil, errortypes.NewNotFoundError("not cached")
}

func (r *rig) organizationNamed(t *testing.T, name string) dbscope.Tenant {
	t.Helper()
	type row struct {
		ID pulid.ID `db:"id"`
		BU pulid.ID `db:"business_unit_id"`
	}
	rows, err := keel.QueryAll(r.ctx, r.admin, pgx.RowToStructByName[row],
		"SELECT id, business_unit_id FROM organizations WHERE name = $1", name)
	require.NoError(t, err)
	require.Len(t, rows, 1, name)
	return dbscope.Tenant{OrganizationID: rows[0].ID, BusinessUnitID: rows[0].BU}
}

func (r *rig) purgeAll(t *testing.T, tenant dbscope.Tenant, unit bool) {
	t.Helper()
	for range 200 {
		result, err := r.purge.PurgeRows(r.ctx, &cloudtenantports.PurgeRowsRequest{
			Tenant: tenant, BatchSize: 2000,
		})
		require.NoError(t, err)
		if result.Complete || result.Deleted == 0 {
			break
		}
	}
	deleted, err := r.purge.DeleteOrganization(r.ctx, tenant)
	require.NoError(t, err)
	require.True(t, deleted.Deleted, "%s is gone: %s", tenant.OrganizationID, deleted.BlockedBy)
	if unit {
		_, err = r.purge.DeleteBusinessUnit(r.ctx, tenant)
		require.NoError(t, err)
	}
}

func (r *rig) naming(t *testing.T, org pulid.ID) map[string]int64 {
	t.Helper()
	columns, err := keel.QueryAll(r.ctx, r.admin, pgx.RowTo[string], `
SELECT format('SELECT %L::text AS name, count(*)::bigint AS n FROM %I WHERE %I = $1', c.relname || '.' || a.attname, c.relname, a.attname)
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
  AND a.attname LIKE '%organization_id'
  AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
ORDER BY 1`)
	require.NoError(t, err)
	type count struct {
		Name string `db:"name"`
		N    int64  `db:"n"`
	}
	out := map[string]int64{}
	for _, query := range columns {
		rows, qerr := keel.QueryAll(
			r.ctx,
			r.admin,
			pgx.RowToStructByName[count],
			query,
			org.String(),
		)
		require.NoError(t, qerr, query)
		for _, c := range rows {
			if c.N > 0 {
				out[c.Name] = c.N
			}
		}
	}
	return out
}

func TestRemovalDeletesEveryRowAndObjectOfTheTenantAndNothingOfAnother(t *testing.T) {
	r := newRigOn(t, true)
	r.purgeAll(t, r.organizationNamed(t, "Trenova Transportation"), false)
	full := r.organizationNamed(t, "Trenova Logistics")
	r.apply(t, entry(instanceA, "a@a.test", 1, platformcatalog.FeatureAccounting))
	r.apply(t, entry(instanceB, "b@b.test", 1, platformcatalog.FeatureAccounting))
	a, b := r.tenant(t, instanceA), r.tenant(t, instanceB)
	made := scopeOf(a)
	a.OrganizationID, a.BusinessUnitID = full.OrganizationID, full.BusinessUnitID
	require.NoError(t, r.tenants.Save(r.ctx, a))
	r.purgeAll(t, made, true)
	a = r.tenant(t, instanceA)
	assert.Positive(t, r.remaining(t, a)["shipments"], "tenant A holds the whole seeded company")
	for _, family := range objectkey.Families() {
		r.objects.keys[family.Key(a.OrganizationID, "x", "1.pdf")] = true
		r.objects.keys[family.Key(b.OrganizationID, "x", "1.pdf")] = true
	}

	for _, held := range []*cloudtenant.Tenant{a, b} {
		_, err := keel.Exec(r.ctx, r.admin, `
INSERT INTO auth_events (id, organization_id, business_unit_id, provider, outcome)
VALUES ($1, $2, $3, 'password', 'success')`,
			pulid.MustNew("aue_").String(), held.OrganizationID.String(), held.BusinessUnitID.String())
		require.NoError(t, err)
	}

	beforeA, beforeB := r.remaining(t, a), r.remaining(t, b)
	assert.Positive(t, beforeA["user_organization_memberships"])
	assert.Positive(t, beforeA["auth_events"], "tenant A has a sign-in history to remove")
	assert.Equal(t, int64(1), beforeA["organizations"])
	namingB := r.naming(t, b.OrganizationID)

	a.LastArchiveCommandID = "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	a.LastArchivePurpose = cloudtenant.ArchivePurposeFinal
	require.NoError(t, r.tenants.Save(r.ctx, a))
	removed := entry(instanceA, "a@a.test", 2, platformcatalog.FeatureAccounting)
	removed.Lifecycle = cloudtenant.LifecycleRemoved
	assert.Equal(t, cloudtenant.StateApplying, r.apply(t, removed).State)
	require.Contains(t, r.workflows.ids, tenantjobscontract.RemovalWorkflowID(instanceA))

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetTestTimeout(2 * time.Minute)
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
		interceptors.NewTenantScopeInterceptor(),
		interceptors.NewEntitlementInterceptor(refuseEveryPlan{}, nil, zap.NewNop()),
	}})
	env.RegisterActivity(tenantjobs.NewActivities(tenantjobs.ActivitiesParams{
		Lifecycle: r.svc,
		Purge:     r.purge,
		Keys:      r.keys,
		Search:    r.search,
		Storage:   r.objects,
		Slots:     heavyslotstest.Unbounded(),
		Logger:    zap.NewNop(),
	}))
	env.ExecuteWorkflow(tenantjobs.CloudTenantRemovalWorkflow, &tenantjobscontract.TenantPayload{
		InstanceID:     a.InstanceID,
		OrganizationID: a.OrganizationID,
		BusinessUnitID: a.BusinessUnitID,
		Revision:       2,
	})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	for table, n := range r.remaining(t, a) {
		assert.Zerof(t, n, "%s still holds %d rows of the removed tenant", table, n)
	}
	assert.Equal(t, beforeB, r.remaining(t, b), "nothing of the other tenant is touched")
	assert.Empty(t, r.naming(t, a.OrganizationID),
		"no row anywhere names the removed organization")
	assert.Equal(t, namingB, r.naming(t, b.OrganizationID))
	survivor, err := organizationrepository.New(organizationrepository.Params{
		DB: r.conn, Logger: zap.NewNop(), Cache: missingOrganizations{},
	}).GetByID(dbscope.WithValidTenant(r.ctx, scopeOf(b)), tenantports.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{OrgID: b.OrganizationID, BuID: b.BusinessUnitID},
	})
	require.NoError(t, err, "the other tenant's organization still reads through its repository")
	assert.Equal(t, b.OrganizationID, survivor.ID)
	for key := range r.objects.keys {
		assert.NotContains(t, key, a.OrganizationID.String(), "no object of the removed tenant is left")
		assert.NotContains(t, key, a.BusinessUnitID.String(), "no object of the removed tenant is left")
	}
	for _, family := range objectkey.Families() {
		assert.Equal(t, 1, r.objects.count(family.Prefix(b.OrganizationID)), family.Name)
	}
	assert.Equal(t, []pulid.ID{a.OrganizationID}, r.search.organizations)
	assert.ElementsMatch(
		t,
		[]string{a.OrganizationID.String(), a.BusinessUnitID.String()},
		r.keys.fragments,
	)

	status := r.apply(t, removed)
	assert.Equal(t, cloudtenant.StateRemoved, status.State)
	require.NotNil(t, status.AppliedRevision)
	assert.Equal(t, int64(2), *status.AppliedRevision)
	events, err := keel.QueryValue[int64](
		r.ctx,
		r.admin,
		"SELECT count(*) FROM cloud_tenant_events WHERE instance_id = $1 AND kind = 'removed'",
		instanceA,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), events, "the removal is kept in the ledger after the tenant is gone")
	stateRows, err := keel.QueryValue[int64](
		r.ctx,
		r.admin,
		"SELECT count(*) FROM managed_feature_states WHERE scope_key = $1",
		a.OrganizationID.String(),
	)
	require.NoError(t, err)
	assert.Zero(t, stateRows, "the tenant's managed scope is removed with it")
	assert.True(t, maps.Equal(beforeB, r.remaining(t, b)))
}

func TestReadingsForEveryTenantAreABoundedNumberOfStatements(t *testing.T) {
	r := newRig(t)
	statementsFor := func(tenants int) []string {
		ids := make([]pulid.ID, 0, tenants)
		for i := range tenants {
			id := fmt.Sprintf("%08x-8c1f-4a39-9a43-5b0d7c4e2f10", i+1)
			if _, err := r.tenants.Get(r.ctx, id); err != nil {
				r.apply(t, entry(id, fmt.Sprintf("admin%d@reads.test", i), 1))
			}
			ids = append(ids, r.tenant(t, id).OrganizationID)
		}
		hour := time.Now().Unix() - time.Now().Unix()%3600
		return r.counter.measure(r.ctx, func(ctx context.Context) {
			readings, err := r.usage.ReadingsForOrganizations(ctx, ids, hour-3600)
			require.NoError(t, err)
			require.Len(t, readings, tenants)
		})
	}

	expected := []string{
		"select ai_usage_records",
		"select shipments",
		"select document_storage_totals",
		"select organizations",
		"select organizations",
	}
	two := statementsFor(2)
	six := statementsFor(6)
	assert.Equal(t, expected, two, "the readings for two tenants ran %v", two)
	assert.Equal(t, expected, six,
		"the readings cost the same however many tenants they cover; six tenants ran %v", six)
}

type refuseEveryPlan struct{}

func (refuseEveryPlan) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, nil
}

func (refuseEveryPlan) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, nil
}
