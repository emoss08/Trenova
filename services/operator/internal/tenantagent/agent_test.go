package tenantagent_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/dirstore"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/escrow/escrowfile"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/supportkeys"
	"github.com/emoss08/trenova/operator/internal/systememail"
	"github.com/emoss08/trenova/operator/internal/tenantagent"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
	"github.com/emoss08/trenova/operator/internal/usage"
)

const token = "0123456789abcdef0123456789abcdef"

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type rig struct {
	cp      *controlplanetest.Server
	tms     *tmstest.Server
	agent   *tenantagent.Agent
	clock   *clock
	secrets string
	state   string
	logs    *syncBuffer
	creds   *credential.Manager
	store   dirstore.Store
	build   func() *tenantagent.Agent
}

func newRig(t *testing.T) *rig {
	t.Helper()
	ctx := t.Context()
	logs := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cp := controlplanetest.NewShared(t, uuid.New())
	state := t.TempDir()
	bundle, err := sonic.Marshal(cp.Bundle(7 * 24 * time.Hour))
	require.NoError(t, err)
	enrolment := filepath.Join(state, "enrolment.json")
	require.NoError(t, os.WriteFile(enrolment, bundle, 0o600))

	store, err := dirstore.New(
		dirstore.Params{Dir: filepath.Join(state, "identity"), Enrolment: enrolment},
	)
	require.NoError(t, err)
	transport := &credential.Transport{}
	lazy := controlplane.NewLazy(&http.Client{Transport: transport, Timeout: 10 * time.Second})
	creds := credential.NewManager(&credential.ManagerParams{
		Store: store, Transport: transport, Renewer: lazy, Logger: logger,
	})
	lazy.Locate(func() string { return creds.Current().APIURL })
	require.NoError(t, creds.Start(ctx))

	fakeTMS := tmstest.New(t, token)
	tokenFile := filepath.Join(state, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte(token+"\n"), 0o600))
	client, err := tms.New(tms.Params{URL: fakeTMS.URL, Tokens: tms.NewFileTokens(tokenFile)})
	require.NoError(t, err)

	secrets := filepath.Join(state, "escrow")
	escrowed, err := escrowfile.New(
		&escrowfile.Params{Dir: secrets, ControlPlane: lazy, Logger: logger},
	)
	require.NoError(t, err)
	events, err := emailevents.New(
		&emailevents.Params{ControlPlane: lazy, TMS: client, Logger: logger},
	)
	require.NoError(t, err)

	c := &clock{now: time.Date(2026, 11, 20, 14, 10, 0, 0, time.UTC)}
	build := func() *tenantagent.Agent {
		agent, berr := tenantagent.New(tenantagent.Params{
			ControlPlane: lazy,
			TMS:          client,
			Credentials:  creds,
			Escrow:       escrowed,
			SystemEmail: systememail.New(
				&systememail.Params{TMS: client, ControlPlane: lazy, Logger: logger},
			),
			Usage: usage.NewShared(
				usage.SharedParams{ControlPlane: lazy, TMS: client, Now: c.Now},
			),
			EmailEvents: events,
			SupportKeys: supportkeys.New(client, logger),
			Version:     "0.15.0",
			Logger:      logger,
			Now:         c.Now,
		})
		require.NoError(t, berr)
		return agent
	}
	return &rig{
		cp:      cp,
		tms:     fakeTMS,
		agent:   build(),
		clock:   c,
		secrets: secrets,
		state:   state,
		logs:    logs,
		creds:   creds,
		store:   store,
		build:   build,
	}
}

func tenant(
	id uuid.UUID,
	revision int64,
	lifecycle instanceapi.SharedTenantLifecycle,
) instanceapi.SharedTenant {
	appMeter := "shipments.total"
	return instanceapi.SharedTenant{
		InstanceId:       id,
		Revision:         revision,
		Lifecycle:        lifecycle,
		OrganizationName: "Acme Freight",
		Plan: instanceapi.SharedTenantPlan{
			Key: "shared", Name: "Shared", BackupRetentionDays: 14,
			Support: instanceapi.SharedTenantSupportEmail,
		},
		Addons:     []string{},
		Features:   []string{"tms.accounting"},
		AiProvider: instanceapi.SharedTenantAiCustomer,
		Allowances: []instanceapi.SpecAllowance{{
			Meter: "shipments", AppMeter: &appMeter, Unit: instanceapi.SpecAllowanceUnit("count"),
			Quantity: 100,
		}},
		FirstAdministrator: instanceapi.SpecFirstAdministrator{
			Name: "Dana Whitfield", Email: "dana@acme.example", OrganizationName: "Acme Freight",
		},
		Account: instanceapi.SpecAccount{
			PlanUrl:         "https://console.test/plan",
			FinalBackupDays: 30,
		},
		Commands: []instanceapi.SharedTenantCommand{},
	}
}

func (r *rig) iterate(t *testing.T) {
	t.Helper()
	r.clock.Advance(15 * time.Second)
	require.NoError(t, r.agent.Iterate(t.Context()))
}

func (r *rig) reported(t *testing.T, id uuid.UUID) instanceapi.SharedTenantStatus {
	t.Helper()
	report, ok := r.cp.LastSharedReport()
	require.True(t, ok)
	for _, s := range report.Tenants {
		if s.InstanceId == id {
			return s
		}
	}
	t.Fatalf("the last report does not name %s", id)
	return instanceapi.SharedTenantStatus{}
}

func TestATenantIsMadeChangedSuspendedResumedAndRemoved(t *testing.T) {
	r := newRig(t)
	id := uuid.New()

	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 1, instanceapi.SharedTenantActive)})
	r.iterate(t)
	status := r.reported(t, id)
	assert.Equal(t, instanceapi.TenantActive, status.State)
	require.NotNil(t, status.AppliedRevision)
	assert.Equal(t, int64(1), *status.AppliedRevision)
	report, _ := r.cp.LastSharedReport()
	assert.True(t, report.Ready)
	require.NotNil(t, report.Version)
	assert.Equal(t, "0.15.0", *report.Version)
	for _, key := range escrow.Keys {
		info, err := os.Stat(filepath.Join(r.secrets, key))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}

	changed := tenant(id, 2, instanceapi.SharedTenantActive)
	changed.Features = []string{"tms.accounting", "ai.desk"}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{changed})
	r.iterate(t)
	assert.Equal(t, int64(2), *r.reported(t, id).AppliedRevision)
	assert.Contains(t, string(r.tms.TenantEntry(id.String())), "ai.desk")

	r.cp.SetSharedTenants(
		[]instanceapi.SharedTenant{tenant(id, 3, instanceapi.SharedTenantSuspended)},
	)
	r.iterate(t)
	assert.Equal(t, instanceapi.TenantSuspended, r.reported(t, id).State)

	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 4, instanceapi.SharedTenantActive)})
	r.iterate(t)
	assert.Equal(t, instanceapi.TenantActive, r.reported(t, id).State)

	r.cp.SetSharedTenants(
		[]instanceapi.SharedTenant{tenant(id, 5, instanceapi.SharedTenantRemoved)},
	)
	r.iterate(t)
	status = r.reported(t, id)
	assert.Equal(t, instanceapi.TenantRemoved, status.State)
	assert.Equal(t, int64(5), *status.AppliedRevision)
	assert.Equal(t, 5, r.tms.TenantPuts(id.String()))

	r.cp.SetSharedTenants([]instanceapi.SharedTenant{})
	r.clock.Advance(2 * time.Minute)
	r.iterate(t)
	report, _ = r.cp.LastSharedReport()
	assert.Empty(t, report.Tenants, "a tenant the control plane no longer lists is not reported")
}

func TestTwoIterationsDoNotApplyTwiceAndAnUnchangedListIsNotRead(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 1, instanceapi.SharedTenantActive)})

	r.iterate(t)
	r.iterate(t)
	r.iterate(t)

	assert.Equal(t, 1, r.tms.TenantPuts(id.String()))
	listed, unchanged := r.cp.SharedListings()
	assert.Equal(t, 3, listed)
	assert.Equal(t, 2, unchanged, "the ETag answers 304 while nothing changed")
	assert.Len(t, r.cp.SharedReports(), 1, "an unchanged report is not sent again within a minute")

	r.clock.Advance(time.Minute)
	r.iterate(t)
	assert.Len(t, r.cp.SharedReports(), 2, "the report is sent at least every minute")
}

func TestARefusalFromTheTMSIsReportedToTheControlPlane(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	r.tms.RefuseTenant(
		id.String(),
		"the first administrator's email address cannot be used on this server",
	)
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 1, instanceapi.SharedTenantActive)})

	r.iterate(t)
	r.iterate(t)

	status := r.reported(t, id)
	assert.Equal(t, instanceapi.TenantApplying, status.State)
	require.NotNil(t, status.Refused)
	assert.Equal(t, int64(1), status.Refused.Revision)
	assert.Equal(t, instanceapi.SharedRefusedInvalid, status.Refused.Code)
	assert.Contains(t, status.Refused.Reason, "cannot be used")
	assert.Equal(t, 1, r.tms.TenantPuts(id.String()), "a refused revision is not sent again")
}

func TestUsageIsReportedWithAMissedHourAndEachAnswerWrittenBack(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 1, instanceapi.SharedTenantActive)})
	hour := r.clock.Now().Truncate(time.Hour)
	r.tms.SetTenantReadings(id.String(), []tms.TenantReading{
		{Meter: "shipments.total", Hour: hour.Add(-2 * time.Hour), Quantity: 4},
		{Meter: "shipments.total", Hour: hour.Add(-time.Hour), Quantity: 6},
		{Meter: "not.sold", Hour: hour.Add(-time.Hour), Quantity: 9},
	})
	r.cp.SetSharedUsageAnswer(instanceapi.SharedTenantUsage{
		InstanceId: id,
		Overage:    true,
		Meters: []instanceapi.MeterUsage{{
			Meter: "shipments", Unit: instanceapi.MeterUsageUnit("count"), Quantity: 10,
			Allowance: 100, Stopped: false,
		}},
	})
	r.cp.FailSharedUsage(1)

	err := r.agent.Iterate(t.Context())
	require.Error(t, err, "the control plane was down for the usage report")
	assert.Empty(t, r.tms.TenantUsage())

	r.iterate(t)
	reports := r.cp.SharedUsageReports()
	require.Len(t, reports, 1)
	assert.Len(
		t,
		reports[0].Readings,
		2,
		"both hours are read again, and a meter the plan does not sell is dropped",
	)
	for _, rd := range reports[0].Readings {
		assert.Equal(t, id, rd.InstanceId)
		assert.Equal(t, "shipments", rd.Meter)
	}
	written := r.tms.TenantUsage()
	require.Len(t, written, 1)
	require.Len(t, written[0], 1)
	assert.Equal(t, id.String(), written[0][0].InstanceID)
	assert.True(t, written[0][0].Overage)
	require.Len(t, written[0][0].Meters, 1)
	assert.Equal(t, "shipments.total", written[0][0].Meters[0].Meter)

	r.iterate(t)
	assert.Len(t, r.cp.SharedUsageReports(), 1, "usage is reported again only after its interval")
}

func TestTheAgentKeepsReportingWhileTheTMSIsDown(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 1, instanceapi.SharedTenantActive)})
	r.tms.FailTenants(1)

	err := r.agent.Iterate(t.Context())
	require.Error(t, err)
	report, ok := r.cp.LastSharedReport()
	require.True(t, ok)
	assert.False(t, report.Ready)
	require.NotNil(t, report.Message)
	assert.Contains(t, *report.Message, "managed listener")
	require.Error(t, r.agent.Healthy())

	r.iterate(t)
	assert.Equal(t, instanceapi.TenantActive, r.reported(t, id).State)
	require.NoError(t, r.agent.Healthy())
}

func TestTheAgentHoldsItsCredentialInItsDirectory(t *testing.T) {
	r := newRig(t)
	loaded, ok, err := r.store.Load(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, credential.KindSharedServer, loaded.Kind)
	assert.Equal(t, current(r.creds).Fingerprint(), loaded.Fingerprint())
	assert.NotEqual(
		t,
		loaded.EnrolledFrom,
		loaded.Fingerprint(),
		"the agent renewed onto its own key",
	)
	assert.True(t, strings.HasPrefix(loaded.APIURL, "https://"))
}

func current(m *credential.Manager) *credential.Identity {
	ident := m.Current()
	return &ident
}
