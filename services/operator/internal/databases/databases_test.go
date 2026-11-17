package databases_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/emoss08/trenova/operator/internal/databases"
	"github.com/emoss08/trenova/operator/internal/fakecp"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	namespace = "trenova"
	release   = "trenova"
	first     = "trenova-postgres"
	genA      = "r0000000a"
	genB      = "r0000000b"
	genC      = "r0000000c"
	genD      = "r0000000d"
)

var (
	clusters = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	claims   = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
	now      = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

func clusterOf(gen string) string {
	if gen == "" {
		return first
	}
	return first + "-" + gen
}

func running(name, rel string, created time.Time) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1", "kind": "Cluster",
		"metadata": map[string]any{
			"name": name, "namespace": namespace, "uid": name + "-uid",
			"creationTimestamp": created.Format(time.RFC3339),
			"labels": map[string]any{
				"app.kubernetes.io/name": "trenova", "app.kubernetes.io/instance": rel,
				"app.kubernetes.io/component": "postgres",
			},
			"annotations": map[string]any{"cnpg.io/hibernation": "off"},
		},
		"spec":   map[string]any{"instances": int64(1)},
		"status": map[string]any{"phase": "Cluster in healthy state", "readyInstances": int64(1)},
	}}
}

func stoppedCluster(name, rel string, created time.Time) *unstructured.Unstructured {
	c := running(name, rel, created)
	c.SetAnnotations(map[string]string{"cnpg.io/hibernation": "on"})
	_ = unstructured.SetNestedField(c.Object, map[string]any{
		"phase": "Cluster in healthy state", "readyInstances": int64(0),
		"conditions": []any{map[string]any{"type": "cnpg.io/hibernation", "status": "True"}},
	}, "status")
	return c
}

func startingCluster(name, rel string, created time.Time) *unstructured.Unstructured {
	c := running(name, rel, created)
	_ = unstructured.SetNestedField(c.Object, map[string]any{
		"phase": "Setting up primary", "readyInstances": int64(0),
	}, "status")
	return c
}

func claim(name, cluster, rel, capacity string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "PersistentVolumeClaim",
		"metadata": map[string]any{
			"name": name, "namespace": namespace, "uid": name + "-uid",
			"labels": map[string]any{
				"cnpg.io/cluster": cluster, "app.kubernetes.io/instance": rel,
				"app.kubernetes.io/component": "postgres",
			},
		},
		"status": map[string]any{"capacity": map[string]any{"storage": capacity}},
	}}
}

func manager(t *testing.T, objs ...runtime.Object) (*databases.Manager, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		clusters: "ClusterList", claims: "PersistentVolumeClaimList",
	}, objs...)
	return databases.New(&databases.Params{
		Dynamic: dyn, Namespace: namespace, Release: release,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now },
	}), dyn
}

func spec(database string) *instanceapi.InstanceSpec {
	s := fakecp.Spec(uuid.New())
	s.Lifecycle = instanceapi.Active
	if database != "" {
		s.Database = &instanceapi.SpecDatabase{Generation: database, TargetTime: now.Add(-time.Hour)}
	}
	return &s
}

func on(gen string) databases.Connected {
	return databases.Connected{Known: true, Exists: true, Generation: gen}
}

func get(t *testing.T, dyn *dynamicfake.FakeDynamicClient, gvr schema.GroupVersionResource, name string) *unstructured.Unstructured {
	t.Helper()
	obj, err := dyn.Resource(gvr).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return obj
}

func gone(t *testing.T, dyn *dynamicfake.FakeDynamicClient, gvr schema.GroupVersionResource, name string) {
	t.Helper()
	_, err := dyn.Resource(gvr).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "%s %s should be gone", gvr.Resource, name)
}

func hibernation(t *testing.T, dyn *dynamicfake.FakeDynamicClient, gen string) string {
	t.Helper()
	return get(t, dyn, clusters, clusterOf(gen)).GetAnnotations()["cnpg.io/hibernation"]
}

func verbs(dyn *dynamicfake.FakeDynamicClient, verb string) []k8stesting.Action {
	var out []k8stesting.Action
	for _, a := range dyn.Actions() {
		if a.GetVerb() == verb {
			out = append(out, a)
		}
	}
	return out
}

func makeReady(t *testing.T, dyn *dynamicfake.FakeDynamicClient, gen string) {
	t.Helper()
	c := get(t, dyn, clusters, clusterOf(gen))
	require.NoError(t, unstructured.SetNestedField(c.Object, map[string]any{
		"phase": "Cluster in healthy state", "readyInstances": int64(1),
	}, "status"))
	_, err := dyn.Resource(clusters).Namespace(namespace).Update(context.Background(), c, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func TestGoingBackToTheFirstDatabaseStartsItAndHoldsUntilItAcceptsConnections(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		stoppedCluster(first, release, now.Add(-60*24*time.Hour)),
		running(clusterOf(genA), release, now.Add(-24*time.Hour)),
	)
	in := databases.GateInput{Spec: spec(""), Connected: on(genA)}

	g := m.Gate(ctx, in)
	require.True(t, g.Hold)
	require.Contains(t, g.Message, "going back to the first database: starting it")
	require.Contains(t, g.Message, "stays on database "+genA)
	require.Equal(t, "off", hibernation(t, dyn, ""))

	g = m.Gate(ctx, in)
	require.True(t, g.Hold, "hibernation is still applied: not accepting connections")
	require.Contains(t, g.Message, "waiting for it to accept connections")

	// The one being woken is starting; the one the application runs on is
	// live and running.
	res := m.Settle(ctx, databases.SettleInput{Spec: in.Spec, Connected: on(genA)})
	require.Equal(t, []instanceapi.DatabaseState{
		{Generation: genA, State: instanceapi.DatabaseStateStateRunning, Live: true, CreatedAt: new(now.Add(-24 * time.Hour))},
		{Generation: "", State: instanceapi.DatabaseStateStateStarting, CreatedAt: new(now.Add(-60 * 24 * time.Hour))},
	}, *res.Databases)

	makeReady(t, dyn, "")
	require.Equal(t, databases.Gate{}, m.Gate(ctx, in))
	require.Equal(t, "off", hibernation(t, dyn, genA), "the database it runs on is never stopped by the gate")
	require.Len(t, verbs(dyn, "patch"), 1)
}

func TestGoingBackToAnOlderRestoredGenerationThenStopsTheOneItRanOn(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		stoppedCluster(first, release, now.Add(-60*24*time.Hour)),
		stoppedCluster(clusterOf(genA), release, now.Add(-2*24*time.Hour)),
		running(clusterOf(genB), release, now.Add(-24*time.Hour)),
	)
	s := spec(genA)

	require.True(t, m.Gate(ctx, databases.GateInput{Spec: s, Connected: on(genB)}).Hold)
	require.Equal(t, "off", hibernation(t, dyn, genA))
	require.Equal(t, "on", hibernation(t, dyn, ""), "only the named database is started")
	makeReady(t, dyn, genA)
	require.False(t, m.Gate(ctx, databases.GateInput{Spec: s, Connected: on(genB)}).Hold)

	// The chart moved the application; until the instance is ready on the
	// revision, the database it ran on stays up.
	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(genA), Applied: true, Ready: false})
	require.Equal(t, "off", hibernation(t, dyn, genB))
	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(genB), Applied: false, Ready: true})
	require.Equal(t, "off", hibernation(t, dyn, genB))

	res := m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(genA), Applied: true, Ready: true})
	require.Equal(t, "on", hibernation(t, dyn, genB))
	require.Equal(t, "off", hibernation(t, dyn, genA))
	require.Equal(t, instanceapi.DatabaseState{
		Generation: genA, State: instanceapi.DatabaseStateStateRunning, Live: true,
		CreatedAt: new(now.Add(-2 * 24 * time.Hour)),
	}, (*res.Databases)[0])
}

func TestANamedGenerationTheClusterDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t, running(first, release, now))

	g := m.Gate(ctx, databases.GateInput{Spec: spec(genC), Connected: on("")})
	require.True(t, g.Hold)
	require.Contains(t, g.Message, "the spec names database "+genC+", which this instance does not hold")
	require.Empty(t, verbs(dyn, "patch"))
	require.Empty(t, verbs(dyn, "delete"))

	// Named by a restore: the chart makes it.
	s := spec(genC)
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: genC, TargetTime: now}
	require.Equal(t, databases.Gate{}, m.Gate(ctx, databases.GateInput{Spec: s, Connected: on("")}))

	// The first database, retired, is not made again empty.
	m, _ = manager(t, running(clusterOf(genA), release, now))
	g = m.Gate(ctx, databases.GateInput{Spec: spec(""), Connected: on(genA)})
	require.True(t, g.Hold)
	require.Contains(t, g.Message, "the first database, which this instance does not hold")
}

func TestTheDatabaseItRunsOnNeedsNoRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t)
	require.Equal(t, databases.Gate{}, m.Gate(ctx, databases.GateInput{Spec: spec(genA), Connected: on(genA)}))
	require.Empty(t, dyn.Actions())

	// A first install: the cluster holds none of the instance's databases.
	require.Equal(t, databases.Gate{}, m.Gate(ctx, databases.GateInput{
		Spec: spec(""), Connected: databases.Connected{Known: true},
	}))
}

func TestAFailedReadHoldsAMoveAndLeavesTheListOut(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t, running(first, release, now))
	dyn.PrependReactor("list", "clusters", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the API server is away")
	})
	g := m.Gate(ctx, databases.GateInput{Spec: spec(genA), Connected: on("")})
	require.True(t, g.Hold)
	require.Contains(t, g.Message, "once its databases can be read")
	require.Equal(t, databases.Result{}, m.Settle(ctx, databases.SettleInput{
		Spec: spec(""), Connected: on(""), Applied: true, Ready: true,
	}))
}

func TestGoingBackWaitsForNothingWhileTheInstanceShouldNotRun(t *testing.T) {
	t.Parallel()
	for _, lc := range []instanceapi.InstanceSpecLifecycle{instanceapi.Suspended, instanceapi.Decommissioned} {
		m, dyn := manager(t,
			stoppedCluster(first, release, now.Add(-time.Hour)),
			running(clusterOf(genA), release, now),
		)
		s := spec("")
		s.Lifecycle = lc
		require.Equal(t, databases.Gate{}, m.Gate(context.Background(), databases.GateInput{Spec: s, Connected: on(genA)}),
			"%s: the chart's lifecycle values decide whether it runs", lc)
		require.Empty(t, verbs(dyn, "patch"), "%s: nothing is started", lc)
	}
}

func TestEveryOtherDatabaseIsStoppedOnEveryReconcileButTheRestoreCandidate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		running(first, release, now.Add(-60*24*time.Hour)),
		running(clusterOf(genA), release, now.Add(-24*time.Hour)),
		startingCluster(clusterOf(genB), release, now.Add(-time.Hour)),
		running("other-trenova-postgres", "other", now),
	)
	s := spec("")
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: genB, TargetTime: now}

	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: false, Ready: true})
	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: false})
	require.Empty(t, verbs(dyn, "patch"), "nothing is stopped before the instance is ready on the revision")

	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: true})
	require.Equal(t, "on", hibernation(t, dyn, genA))
	require.Equal(t, "off", hibernation(t, dyn, genB), "the restore candidate is never stopped")
	require.Equal(t, "off", hibernation(t, dyn, ""))
	require.Equal(t, "off", get(t, dyn, clusters, "other-trenova-postgres").GetAnnotations()["cnpg.io/hibernation"])

	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: true})
	require.Len(t, verbs(dyn, "patch"), 1, "a stopped database is not patched again")
}

func TestRetiringRemovesExactlyTheNamedGeneration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		running(first, release, now.Add(-60*24*time.Hour)),
		claim(first+"-1", first, release, "10Gi"),
		stoppedCluster(clusterOf(genA), release, now.Add(-10*24*time.Hour)),
		claim(clusterOf(genA)+"-1", clusterOf(genA), release, "10Gi"),
		claim(clusterOf(genA)+"-wal", clusterOf(genA), release, "2Gi"),
		stoppedCluster(clusterOf(genB), release, now.Add(-5*24*time.Hour)),
		claim(clusterOf(genB)+"-1", clusterOf(genB), release, "10Gi"),
		stoppedCluster("other-trenova-postgres-"+genA, "other", now),
		claim("other-trenova-postgres-"+genA+"-1", "other-trenova-postgres-"+genA, "other", "10Gi"),
		claim("decoy-1", clusterOf(genA), "other", "10Gi"),
	)
	s := spec("")
	s.RetiredDatabases = &[]string{genA, genA}
	settle := func() databases.Result {
		return m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: true})
	}

	res := settle()
	require.Empty(t, res.Message)
	gone(t, dyn, clusters, clusterOf(genA))
	deletes := verbs(dyn, "delete")
	require.Len(t, deletes, 1)
	opts := deletes[0].(k8stesting.DeleteActionImpl).DeleteOptions
	require.Equal(t, metav1.DeletePropagationForeground, *opts.PropagationPolicy)
	require.Equal(t, clusterOf(genA)+"-uid", string(*opts.Preconditions.UID))
	require.Contains(t, *res.Databases, instanceapi.DatabaseState{
		Generation: genA, State: instanceapi.DatabaseStateStateRemoving, SizeBytes: new(int64(12 << 30)),
		CreatedAt: new(now.Add(-10 * 24 * time.Hour)),
	})

	// The cluster is gone; its volumes the garbage collector has not taken
	// yet are removed by name and owner, and nothing else is.
	res = settle()
	gone(t, dyn, claims, clusterOf(genA)+"-1")
	gone(t, dyn, claims, clusterOf(genA)+"-wal")
	require.Contains(t, *res.Databases, instanceapi.DatabaseState{
		Generation: genA, State: instanceapi.DatabaseStateStateRemoving, SizeBytes: new(int64(12 << 30)),
	})

	res = settle()
	for _, d := range *res.Databases {
		require.NotEqual(t, genA, d.Generation, "a removed database leaves the list")
	}
	deleted := len(verbs(dyn, "delete"))
	settle()
	require.Len(t, verbs(dyn, "delete"), deleted, "retiring one that is gone is done")

	get(t, dyn, clusters, first)
	get(t, dyn, claims, first+"-1")
	get(t, dyn, clusters, clusterOf(genB))
	get(t, dyn, claims, clusterOf(genB)+"-1")
	get(t, dyn, clusters, "other-trenova-postgres-"+genA)
	get(t, dyn, claims, "other-trenova-postgres-"+genA+"-1")
	get(t, dyn, claims, "decoy-1")
}

func TestRetiringTheLiveTheCandidateOrTheSourceIsRefused(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		stoppedCluster(first, release, now.Add(-60*24*time.Hour)),
		running(clusterOf(genA), release, now.Add(-24*time.Hour)),
		startingCluster(clusterOf(genB), release, now.Add(-time.Hour)),
		running(clusterOf(genC), release, now.Add(-2*time.Hour)),
	)
	s := spec(genA)
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: genB, SourceGeneration: "", TargetTime: now}
	s.RetiredDatabases = &[]string{"", genA, genB, genC}

	res := m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(genC), Applied: false, Ready: true})
	require.Empty(t, verbs(dyn, "delete"))
	require.Contains(t, res.Message, "not removed, though the spec retires them")
	require.Contains(t, res.Message, "the first database (a restore reads from it)")
	require.Contains(t, res.Message, "database "+genA+" (the instance runs on it)")
	require.Contains(t, res.Message, "database "+genB+" (it is being restored)")
	require.Contains(t, res.Message, "database "+genC+" (the instance is still connected to it)")
	for _, gen := range []string{"", genA, genB, genC} {
		get(t, dyn, clusters, clusterOf(gen))
	}
}

func TestADecommissionedInstanceIsLeftToDecommissioning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t,
		running(first, release, now.Add(-60*24*time.Hour)),
		running(clusterOf(genA), release, now.Add(-24*time.Hour)),
		stoppedCluster(clusterOf(genB), release, now.Add(-2*24*time.Hour)),
	)
	s := spec("")
	s.Lifecycle = instanceapi.Decommissioned
	s.RetiredDatabases = &[]string{genB}
	res := m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: true})
	require.Empty(t, verbs(dyn, "patch"))
	require.Empty(t, verbs(dyn, "delete"))
	require.Len(t, *res.Databases, 3, "the list is still reported")

	// Suspended: nothing runs, and the others are stopped all the same.
	s.Lifecycle = instanceapi.Suspended
	m.Settle(ctx, databases.SettleInput{Spec: s, Connected: on(""), Applied: true, Ready: true})
	require.Equal(t, "on", hibernation(t, dyn, genA))
	gone(t, dyn, clusters, clusterOf(genB))
}

func TestTheListHasEachDatabaseLiveFirst(t *testing.T) {
	t.Parallel()
	deleted := running(clusterOf(genC), release, now.Add(-3*time.Hour))
	deleted.SetDeletionTimestamp(&metav1.Time{Time: now})
	deleted.SetFinalizers([]string{"foregroundDeletion"})
	m, _ := manager(t,
		running(first, release, now.Add(-60*24*time.Hour)),
		claim(first+"-1", first, release, "10Gi"),
		stoppedCluster(clusterOf(genA), release, now.Add(-24*time.Hour)),
		startingCluster(clusterOf(genB), release, now.Add(-time.Hour)),
		deleted,
		claim(clusterOf(genD)+"-1", clusterOf(genD), release, "5Gi"),
		running("other-trenova-postgres", "other", now),
	)
	s := spec("")
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: genB, TargetTime: now}
	res := m.Settle(context.Background(), databases.SettleInput{Spec: s, Connected: on(""), Applied: false, Ready: false})
	require.Empty(t, res.Message)
	require.Equal(t, []instanceapi.DatabaseState{
		{
			Generation: "", State: instanceapi.DatabaseStateStateRunning, Live: true,
			SizeBytes: new(int64(10 << 30)), CreatedAt: new(now.Add(-60 * 24 * time.Hour)),
		},
		{Generation: genB, State: instanceapi.DatabaseStateStateRestoring, CreatedAt: new(now.Add(-time.Hour))},
		{Generation: genC, State: instanceapi.DatabaseStateStateRemoving, CreatedAt: new(now.Add(-3 * time.Hour))},
		{Generation: genA, State: instanceapi.DatabaseStateStateStopped, CreatedAt: new(now.Add(-24 * time.Hour))},
		{Generation: genD, State: instanceapi.DatabaseStateStateRemoving, SizeBytes: new(int64(5 << 30))},
	}, *res.Databases)

	// No release yet: none is live.
	res = m.Settle(context.Background(), databases.SettleInput{Spec: s, Connected: databases.Connected{Known: true}})
	for _, d := range *res.Databases {
		require.False(t, d.Live)
	}
}

func TestOnlyADatabaseThatShouldRunIsStarting(t *testing.T) {
	t.Parallel()
	crashed := startingCluster(clusterOf(genA), release, now.Add(-time.Hour))
	m, _ := manager(t,
		startingCluster(first, release, now.Add(-60*24*time.Hour)),
		crashed,
		stoppedCluster(clusterOf(genB), release, now.Add(-2*time.Hour)),
	)
	res := m.Settle(context.Background(), databases.SettleInput{Spec: spec(""), Connected: on("")})
	states := map[string]instanceapi.DatabaseStateState{}
	for _, d := range *res.Databases {
		states[d.Generation] = d.State
	}
	require.Equal(t, map[string]instanceapi.DatabaseStateState{
		"":   instanceapi.DatabaseStateStateStarting, // live, restarting
		genA: instanceapi.DatabaseStateStateStopped,  // not meant to run: stopped once the instance is ready
		genB: instanceapi.DatabaseStateStateStopped,  // kept by intent
	}, states)

	// Suspended: the live one is hibernated by intent, so stopped.
	m, _ = manager(t, stoppedCluster(first, release, now))
	s := spec("")
	s.Lifecycle = instanceapi.Suspended
	res = m.Settle(context.Background(), databases.SettleInput{Spec: s, Connected: on("")})
	require.Equal(t, instanceapi.DatabaseStateStateStopped, (*res.Databases)[0].State)
}
