package reconcile_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
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

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/databases"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	firstDB = "trenova-postgres"
	genOld  = "r0000000a"
	genNew  = "r0000000b"
)

var (
	pgClusters = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	pgClaims   = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
)

func clusterName(gen string) string {
	if gen == "" {
		return firstDB
	}
	return firstDB + "-" + gen
}

func pgCluster(gen string, stopped bool) *unstructured.Unstructured {
	state, ready := "off", int64(1)
	if stopped {
		state, ready = "on", 0
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1", "kind": "Cluster",
		"metadata": map[string]any{
			"name": clusterName(gen), "namespace": "trenova", "uid": clusterName(gen) + "-uid",
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
			"labels": map[string]any{
				"app.kubernetes.io/instance": "trenova", "app.kubernetes.io/component": "postgres",
			},
			"annotations": map[string]any{"cnpg.io/hibernation": state},
		},
		"spec":   map[string]any{"instances": int64(1)},
		"status": map[string]any{"phase": "Cluster in healthy state", "readyInstances": ready},
	}}
}

// databaseWorld is the loop with the real databases manager over a fake
// cluster, and one log of what reached the cluster and the chart, in order:
// "start <cluster>", "stop <cluster>", "remove <name>" and "apply <cluster>".
type databaseWorld struct {
	*backupWorld
	dyn *dynamicfake.FakeDynamicClient
	mu  sync.Mutex
	log []string
}

func (d *databaseWorld) record(event string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.log = append(d.log, event)
}

func (d *databaseWorld) events() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.log...)
}

func (d *databaseWorld) cluster(t *testing.T, gen string) *unstructured.Unstructured {
	t.Helper()
	c, err := d.dyn.Resource(pgClusters).Namespace("trenova").Get(context.Background(), clusterName(gen), metav1.GetOptions{})
	require.NoError(t, err)
	return c
}

func (d *databaseWorld) ready(t *testing.T, gen string) {
	t.Helper()
	c := d.cluster(t, gen)
	require.NoError(t, unstructured.SetNestedField(c.Object, map[string]any{
		"phase": "Cluster in healthy state", "readyInstances": int64(1),
	}, "status"))
	_, err := d.dyn.Resource(pgClusters).Namespace("trenova").Update(context.Background(), c, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func (d *databaseWorld) report(t *testing.T) instanceapi.StatusReport {
	t.Helper()
	reports := d.cp.Reports()
	require.NotEmpty(t, reports)
	return reports[len(reports)-1]
}

func (d *databaseWorld) tick(t *testing.T) {
	t.Helper()
	d.now = d.now.Add(15 * time.Second)
	d.loop.Tick(t.Context())
}

func newDatabaseWorld(t *testing.T, spec instanceapi.InstanceSpec, objs ...runtime.Object) *databaseWorld {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		pgClusters: "ClusterList", pgClaims: "PersistentVolumeClaimList",
	}, objs...)
	d := &databaseWorld{dyn: dyn}
	d.backupWorld = newBackupWorldWith(t, spec, databases.New(&databases.Params{
		Dynamic: dyn, Namespace: "trenova", Release: "trenova",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))
	dyn.PrependReactor("patch", "clusters", func(a k8stesting.Action) (bool, runtime.Object, error) {
		p := a.(k8stesting.PatchAction)
		verb := "start"
		if strings.Contains(string(p.GetPatch()), `"on"`) {
			verb = "stop"
		}
		d.record(verb + " " + p.GetName())
		return false, nil, nil
	})
	dyn.PrependReactor("delete", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		d.record("remove " + a.(k8stesting.DeleteAction).GetName())
		return false, nil, nil
	})
	d.releases.before = func(vals map[string]any) {
		pg, _ := vals["postgres"].(map[string]any)
		gen, _ := pg["generation"].(string)
		hibernate, _ := pg["hibernate"].(bool)
		d.record("apply " + clusterName(gen))
		if hibernate {
			return
		}
		c, err := dyn.Resource(pgClusters).Namespace("trenova").Get(context.Background(), clusterName(gen), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		require.NoError(t, err)
		ready, _, _ := unstructured.NestedInt64(c.Object, "status", "readyInstances")
		require.True(t, ready >= 1 && c.GetAnnotations()["cnpg.io/hibernation"] != "on",
			"the application is pointed at %s while it does not accept connections", c.GetName())
	}
	return d
}

func onDatabase(spec *instanceapi.InstanceSpec, gen string) {
	spec.Database = nil
	if gen != "" {
		spec.Database = &instanceapi.SpecDatabase{Generation: gen, TargetTime: time.Now().Add(-time.Hour).UTC()}
	}
}

func TestGoingBackToTheFirstDatabaseStartsWaitsMovesThenStops(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	onDatabase(&spec, genOld)
	w := newDatabaseWorld(t, spec, pgCluster("", true), pgCluster(genOld, false))
	w.health.set(true)
	w.tick(t)
	require.Equal(t, []string{"apply " + clusterName(genOld)}, w.events())

	onDatabase(&spec, "")
	w.cp.SetSpec(spec)
	w.tick(t)
	r := w.report(t)
	require.Contains(t, *r.Message, "going back to the first database: starting it")
	require.Equal(t, int64(1), *r.AppliedRevision, "the instance stays on the revision it runs")

	w.tick(t)
	require.Contains(t, *w.report(t).Message, "waiting for it to accept connections")
	require.Equal(t, 1, w.releases.applies)
	held, _ := w.cp.Databases()
	require.Equal(t, instanceapi.DatabaseState{
		Generation: genOld, State: instanceapi.DatabaseStateStateRunning, Live: true,
		CreatedAt: held[0].CreatedAt,
	}, held[0])
	require.Equal(t, instanceapi.DatabaseStateStateStarting, held[1].State)
	require.Equal(t, genOld, w.backups.last().Generation,
		"while held, backup health is read from the database the application runs on")

	// It accepts connections: the chart moves the application, and while the
	// rollout is not ready the database it ran on stays up.
	w.ready(t, "")
	w.health.set(false)
	w.tick(t)
	require.Equal(t, 2, w.releases.applies)
	require.Equal(t, int64(2), *w.report(t).AppliedRevision)
	require.Equal(t, "", w.backups.last().Generation, "moved: backups follow the application")
	require.Equal(t, "off", w.cluster(t, genOld).GetAnnotations()["cnpg.io/hibernation"])

	w.health.set(true)
	w.tick(t)
	require.Equal(t, []string{
		"apply " + clusterName(genOld),
		"start " + firstDB,
		"apply " + firstDB,
		"stop " + clusterName(genOld),
	}, w.events())
	held, _ = w.cp.Databases()
	require.Equal(t, "", held[0].Generation)
	require.True(t, held[0].Live)
	require.Equal(t, instanceapi.DatabaseStateStateRunning, held[0].State)
	require.Equal(t, instanceapi.DatabaseStateStateStopped, held[1].State)
}

func TestGoingBackToAnOlderRestoredGeneration(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	onDatabase(&spec, genNew)
	w := newDatabaseWorld(t, spec, pgCluster("", true), pgCluster(genOld, true), pgCluster(genNew, false))
	w.health.set(true)
	w.tick(t)

	onDatabase(&spec, genOld)
	w.cp.SetSpec(spec)
	w.tick(t)
	require.Contains(t, *w.report(t).Message, "going back to database "+genOld)
	w.ready(t, genOld)
	w.tick(t)
	require.Equal(t, []string{
		"apply " + clusterName(genNew),
		"start " + clusterName(genOld),
		"apply " + clusterName(genOld),
		"stop " + clusterName(genNew),
	}, w.events())
	require.Equal(t, "on", w.cluster(t, "").GetAnnotations()["cnpg.io/hibernation"])
}

func TestADatabaseTheClusterDoesNotHoldIsRefusedAndNothingChanges(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	w := newDatabaseWorld(t, spec, pgCluster("", false))
	w.health.set(true)
	w.tick(t)

	onDatabase(&spec, "r0000000f")
	w.cp.SetSpec(spec)
	w.tick(t)
	r := w.report(t)
	require.Contains(t, *r.Message, "the spec names database r0000000f, which this instance does not hold")
	require.Equal(t, int64(1), *r.AppliedRevision)
	require.True(t, r.Ready, "the instance's real state is reported beside the refusal")
	require.Equal(t, 1, w.releases.applies)
	require.Equal(t, []string{"apply " + firstDB}, w.events())
	want := &instanceapi.RefusedRevision{
		Revision: 2, Code: instanceapi.RefusedRevisionCodeDatabaseNotHeld, Reason: *r.Message,
	}
	require.Equal(t, want, r.Refused)
	require.Equal(t, want, w.cp.Refused(), "the control plane holds it")

	// Still standing: the next report carries it too.
	w.now = w.now.Add(time.Minute)
	reports := len(w.cp.Reports())
	w.tick(t)
	require.Len(t, w.cp.Reports(), reports+1)
	require.Equal(t, want, w.report(t).Refused)

	// The same revision becomes applicable: applied, and the refusal is gone.
	_, err := w.dyn.Resource(pgClusters).Namespace("trenova").
		Create(t.Context(), pgCluster("r0000000f", false), metav1.CreateOptions{})
	require.NoError(t, err)
	w.tick(t)
	r = w.report(t)
	require.Equal(t, int64(2), *r.AppliedRevision)
	require.Nil(t, r.Refused)
	require.Nil(t, w.cp.Refused())
}

func TestARefusalIsClearedByALaterAppliedRevision(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	onDatabase(&spec, genOld)
	w := newDatabaseWorld(t, spec, pgCluster(genOld, false))
	w.health.set(true)
	w.tick(t)

	// The first database was removed: a revision without `database` is
	// refused, not answered with a new, empty first database.
	onDatabase(&spec, "")
	w.cp.SetSpec(spec)
	w.tick(t)
	r := w.report(t)
	require.NotNil(t, r.Refused)
	require.Equal(t, instanceapi.RefusedRevisionCodeFirstDatabaseRemoved, r.Refused.Code)
	require.Equal(t, int64(2), r.Refused.Revision)
	require.Equal(t, int64(1), *r.AppliedRevision)
	require.Equal(t, 1, w.releases.applies)

	onDatabase(&spec, genOld)
	spec.Plan.Vcpu, spec.Plan.MemoryGiB = 16, 64
	w.cp.SetSpec(spec)
	w.tick(t)
	r = w.report(t)
	require.Equal(t, int64(3), *r.AppliedRevision)
	require.Nil(t, r.Refused)
	require.Nil(t, w.cp.Refused())
}

func TestAnInvalidRevisionIsRefusedAndNothingOfItRuns(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	w := newDatabaseWorld(t, spec, pgCluster("", false), pgCluster(genOld, true))
	w.health.set(true)
	w.tick(t)

	// A plan below the minimum, and a retirement in the same revision: the
	// revision is refused as a whole, so the retirement does not run either.
	spec.Plan.Vcpu = 1
	spec.RetiredDatabases = &[]string{genOld}
	w.cp.SetSpec(spec)
	w.tick(t)
	w.tick(t)
	r := w.report(t)
	require.NotNil(t, r.Refused)
	require.Equal(t, instanceapi.RefusedRevisionCodeInvalid, r.Refused.Code)
	require.Equal(t, int64(2), r.Refused.Revision)
	require.Contains(t, r.Refused.Reason, "a plan needs at least")
	require.Equal(t, *r.Message, r.Refused.Reason)
	require.True(t, r.Ready)
	require.Equal(t, int64(1), *r.AppliedRevision)
	require.Equal(t, []string{"apply " + firstDB}, w.events())

	spec.Plan.Vcpu = 4
	w.cp.SetSpec(spec)
	w.tick(t)
	require.Nil(t, w.report(t).Refused)
	require.Contains(t, w.events(), "remove "+clusterName(genOld))
}

func TestOthersAreStoppedWithNoRestoreAndTheListIsSentOnChange(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	w := newDatabaseWorld(t, spec, pgCluster("", false), pgCluster(genOld, false))
	w.health.set(true)
	w.tick(t)
	require.Equal(t, []string{"apply " + firstDB, "stop " + clusterName(genOld)}, w.events())
	held, lists := w.cp.Databases()
	require.Equal(t, 1, lists)
	require.Len(t, held, 2)
	require.Equal(t, instanceapi.DatabaseStateStateStopped, held[1].State)

	// Unchanged: the minute's report goes without it.
	w.now = w.now.Add(time.Minute)
	reports := len(w.cp.Reports())
	w.tick(t)
	require.Len(t, w.cp.Reports(), reports+1)
	require.Nil(t, w.report(t).Databases)
	_, lists = w.cp.Databases()
	require.Equal(t, 1, lists)

	// Retired: removing, refused once, then sent again; then gone.
	spec.RetiredDatabases = &[]string{genOld}
	w.cp.SetSpec(spec)
	w.cp.FailStatusNext(1)
	w.tick(t)
	_, lists = w.cp.Databases()
	require.Equal(t, 1, lists)
	w.tick(t)
	held, lists = w.cp.Databases()
	require.Equal(t, 2, lists, "a list on a refused report goes again")
	require.Len(t, held, 1, "the cluster is gone, and no volume of it is left")
	require.Contains(t, w.events(), "remove "+clusterName(genOld))
}

func TestRetiringTheLiveDatabaseIsRefusedInTheReport(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	spec.RetiredDatabases = &[]string{""}
	w := newDatabaseWorld(t, spec, pgCluster("", false))
	w.health.set(true)
	w.tick(t)
	r := w.report(t)
	require.True(t, r.Ready)
	require.Equal(t, "not removed, though the spec retires them: the first database (the instance runs on it)", *r.Message)
	require.Nil(t, r.Refused, "a declined retirement does not refuse the revision")
	require.Nil(t, w.cp.Refused())
	require.Equal(t, int64(1), *r.AppliedRevision)
	held, _ := w.cp.Databases()
	require.Len(t, held, 1, "the protected database stays listed")
	for _, e := range w.events() {
		require.False(t, strings.HasPrefix(e, "remove"), e)
	}
}

func TestSuspendedAndDecommissionedInstancesKeepTheirLifecycle(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	onDatabase(&spec, genOld)
	w := newDatabaseWorld(t, spec, pgCluster("", true), pgCluster(genOld, false))
	w.health.set(true)
	w.tick(t)

	// Suspended while going back: nothing is started; the chart points at the
	// first database, hibernated, and the one it ran on is stopped.
	spec.Lifecycle = instanceapi.Suspended
	onDatabase(&spec, "")
	w.cp.SetSpec(spec)
	w.tick(t)
	require.Equal(t, []string{
		"apply " + clusterName(genOld),
		"apply " + firstDB,
		"stop " + clusterName(genOld),
	}, w.events())

	// Decommissioned: a running database that is not live, and a retired one,
	// are left to decommissioning.
	spec = controlplanetest.Spec(uuid.New())
	onDatabase(&spec, genOld)
	w = newDatabaseWorld(t, spec, pgCluster("", false), pgCluster(genOld, false), pgCluster(genNew, true))
	w.health.set(false)
	w.tick(t)
	spec.Lifecycle = instanceapi.Decommissioned
	spec.RetiredDatabases = &[]string{genNew}
	w.cp.SetSpec(spec)
	w.health.set(true)
	w.tick(t)
	w.tick(t)
	require.Equal(t, []string{
		"apply " + clusterName(genOld),
		"apply " + clusterName(genOld),
	}, w.events(), fmt.Sprintf("nothing stopped or removed: %v", w.events()))
}
