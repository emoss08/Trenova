package reconcile_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type fakeSupportKeys struct {
	mu      sync.Mutex
	desired []*instanceapi.SpecSupport
	err     error
}

func (f *fakeSupportKeys) Sync(_ context.Context, desired *instanceapi.SpecSupport) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.desired = append(f.desired, desired)
	return f.err
}

func (f *fakeSupportKeys) calls() []*instanceapi.SpecSupport {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*instanceapi.SpecSupport(nil), f.desired...)
}

func (f *fakeSupportKeys) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

type fakeSupportGrants struct {
	mu     sync.Mutex
	grants []tms.SupportGrant
	err    error
	reads  int
}

func (f *fakeSupportGrants) SupportGrants(context.Context) ([]tms.SupportGrant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	return append([]tms.SupportGrant{}, f.grants...), nil
}

func (f *fakeSupportGrants) hold(grants []tms.SupportGrant, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.grants, f.err = grants, err
}

func (f *fakeSupportGrants) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

const acmeOrg = "org_01JDZ8K3M4N5P6Q7R8S9T0V1W2"

func liveGrant(id, scope string) tms.SupportGrant {
	return tms.SupportGrant{
		ID:               id,
		OrganizationID:   acmeOrg,
		OrganizationName: "Acme Freight",
		StaffEmail:       "dana@trenova.app",
		Scope:            scope,
		ExpiresAt:        time.Date(2027, 1, 15, 8, 0, 0, 0, time.UTC),
	}
}

func specSupport(kids ...string) *instanceapi.SpecSupport {
	keys := make([]instanceapi.SupportKey, 0, len(kids))
	for _, kid := range kids {
		keys = append(keys, instanceapi.SupportKey{
			Kid: kid, Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig",
			X: "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
		})
	}
	return &instanceapi.SpecSupport{
		InstanceId:  "0b7d3f4e-8f7a-4a52-9d61-2f3c8e1a5b90",
		StaffDomain: "trenova.app",
		Keys:        keys,
	}
}

func TestTheSpecsSupportKeysReachTheTMSOnceReady(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, controlplanetest.Spec(uuid.New()), nil)
	w.cp.SetSupport(specSupport("cp-1"))

	w.loop.Tick(t.Context())
	require.Empty(t, w.keys.calls(), "not before the TMS is ready")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	calls := w.keys.calls()
	require.Len(t, calls, 1)
	require.NotNil(t, calls[0])
	assert.Equal(t, "cp-1", calls[0].Keys[0].Kid)
	assert.Equal(t, "trenova.app", calls[0].StaffDomain)
}

func TestARotatedKeySetReachesTheTMSWithoutANewRevision(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, controlplanetest.Spec(uuid.New()), nil)
	w.health.set(true)
	w.cp.SetSupport(specSupport("cp-1"))
	w.loop.Tick(t.Context())
	revision := w.cp.Revision()

	w.cp.SetSupport(specSupport("cp-1", "cp-2"))
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(t, revision, w.cp.Revision())
	calls := w.keys.calls()
	require.Len(t, calls, 2)
	assert.Len(t, calls[1].Keys, 2, "the spec's ETag changed with the keys, so the new set was read")
}

func TestNoSupportKeysAreSentWithoutSupportInTheSpec(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, controlplanetest.Spec(uuid.New()), nil)
	w.health.set(true)

	w.loop.Tick(t.Context())

	assert.Empty(t, w.keys.calls())
	assert.Nil(t, w.lastReport(t).Message)
}

func TestNoSupportKeysAreSentWhileSuspended(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	spec.Lifecycle = instanceapi.Suspended
	w := newWorldWith(t, spec, nil)
	w.cp.SetSupport(specSupport("cp-1"))
	w.health.set(true)

	w.loop.Tick(t.Context())

	assert.Empty(t, w.keys.calls())
}

func TestASupportKeyFailureIsReportedAndRetriedWithoutStoppingTheRest(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "ai.desk"), nil)
	w.cp.SetSupport(specSupport("cp-1"))
	w.health.set(true)
	w.keys.fail(errors.New("key ids cp-1: the TMS answered 503"))

	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed key delivery never holds the report back")
	require.NotNil(t, r.Message)
	assert.True(t, strings.HasPrefix(*r.Message, "support access keys not delivered"))
	assert.Contains(t, *r.Message, "cp-1")
	assert.Equal(t, [][]string{{"ai.desk"}}, w.features.calls(), "nor the feature list")
	assert.Equal(t, 1, w.usage.count(), "nor the usage relay")

	w.keys.fail(nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.keys.calls(), 2, "retried on the next tick")
	assert.Nil(t, w.lastReport(t).Message, "a delivered set clears the message at once")
}

func TestReportsTheInstancesLiveSupportGrants(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ends := time.Date(2027, 1, 15, 8, 0, 0, 0, time.UTC)
	w.grants.hold([]tms.SupportGrant{liveGrant("sag_01", "read_only")}, nil)

	w.loop.Tick(t.Context())
	require.Nil(t, w.lastReport(t).SupportGrants, "not read before the TMS is ready")
	require.Zero(t, w.grants.count())

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.NotNil(t, r.SupportGrants)
	require.Len(t, *r.SupportGrants, 1)
	g := (*r.SupportGrants)[0]
	assert.Equal(t, "sag_01", g.Id)
	assert.Equal(t, acmeOrg, g.OrganizationId)
	assert.Equal(t, "Acme Freight", g.OrganizationName)
	assert.Equal(t, "dana@trenova.app", g.StaffEmail)
	assert.Equal(t, instanceapi.SupportGrantScope("read_only"), g.Scope)
	assert.True(t, g.ExpiresAt.Equal(ends))

	held, ok := w.cp.SupportGrants()
	require.True(t, ok)
	require.Len(t, held, 1)

	sent := len(w.cp.Reports())
	w.grants.hold([]tms.SupportGrant{}, nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), sent+1, "a changed list is reported at once")
	r = w.lastReport(t)
	require.NotNil(t, r.SupportGrants, "no live grant is reported as an empty list")
	assert.Empty(t, *r.SupportGrants)
	held, ok = w.cp.SupportGrants()
	require.True(t, ok)
	assert.Empty(t, held, "the control plane replaced what it held")
}

func TestAFailedGrantReadLeavesTheGrantsOutOfTheReport(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.grants.hold([]tms.SupportGrant{liveGrant("sag_01", "full")}, nil)
	w.loop.Tick(t.Context())
	require.NotNil(t, w.lastReport(t).SupportGrants)

	w.grants.hold(nil, errors.New("the TMS answered 503"))
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())

	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed read never holds the report back")
	assert.Nil(t, r.Message)
	assert.Nil(t, r.SupportGrants)
	held, ok := w.cp.SupportGrants()
	require.True(t, ok)
	assert.Len(t, held, 1, "left out, the control plane keeps the last list")
}

func TestAGrantTheContractCannotCarryIsLeftOutAndTheListIsBounded(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	unknownScope := liveGrant("sag_scope", "admin")
	badOrg := liveGrant("sag_org", "full")
	badOrg.OrganizationID = "org_01"
	noName := liveGrant("sag_name", "full")
	noName.OrganizationName = "  "
	grants := []tms.SupportGrant{unknownScope, badOrg, noName, liveGrant("sag_dup", "full")}
	for i := range 101 {
		g := liveGrant(fmt.Sprintf("sag_%03d", i), "full")
		g.OrganizationName = strings.Repeat("Á", 250)
		g.StaffEmail = "Dana@Trenova.app"
		grants = append(grants, g)
	}
	grants = append(grants, liveGrant("sag_dup", "read_only"))
	w.grants.hold(grants, nil)

	w.loop.Tick(t.Context())

	r := w.lastReport(t)
	require.NotNil(t, r.SupportGrants)
	require.Len(t, *r.SupportGrants, 100, "at most 100, the bad ones never taking a place")
	ids := map[string]bool{}
	for _, g := range *r.SupportGrants {
		assert.False(t, ids[g.Id], "each grant is listed once")
		ids[g.Id] = true
		assert.NotContains(t, []string{"sag_scope", "sag_org", "sag_name"}, g.Id)
		assert.LessOrEqual(t, len([]rune(g.OrganizationName)), 200)
		assert.NotEmpty(t, g.OrganizationName)
	}
	assert.Equal(t, "dana@trenova.app", (*r.SupportGrants)[1].StaffEmail)
}
