package databases

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	labelCluster      = "cnpg.io/cluster"
	labelInstance     = "app.kubernetes.io/instance"
	labelComponent    = "app.kubernetes.io/component"
	componentPostgres = "postgres"
	hibernation       = "cnpg.io/hibernation"
	healthyPhase      = "Cluster in healthy state"

	maxMessage = 500
	warnPeriod = time.Minute
)

var (
	clusterResource = schema.GroupVersionResource{
		Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters",
	}
	claimResource = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}

	restoredPattern = regexp.MustCompile(`^r[0-9a-f]{8}$`)
)

// Manager keeps the databases of one release: the first one, and each one a
// restore made (a generation), every one a CloudNativePG Cluster of its own.
type Manager struct {
	dynamic   dynamic.Interface
	namespace string
	release   string
	logger    *slog.Logger
	now       func() time.Time
	warned    time.Time
}

type Params struct {
	Dynamic   dynamic.Interface
	Namespace string
	Release   string
	Logger    *slog.Logger
	Now       func() time.Time
}

func New(p *Params) *Manager {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Manager{
		dynamic:   p.Dynamic,
		namespace: p.Namespace,
		release:   p.Release,
		logger:    p.Logger,
		now:       now,
	}
}

// Connected is the database the deployed release points the application at.
// Known is false until the release was read on this tick; Exists is false
// while there is no release.
type Connected struct {
	Known      bool
	Exists     bool
	Generation string
}

// Live is the database the spec runs the instance on: the generation
// `database` names, or the first one ("") without it.
func Live(spec *instanceapi.InstanceSpec) string {
	if spec.Database == nil {
		return ""
	}
	return spec.Database.Generation
}

type GateInput struct {
	Spec      *instanceapi.InstanceSpec
	Connected Connected
}

// Gate says whether the chart may now be given the spec's database. Hold
// keeps the release as it is deployed, and Message says why. Refused is set
// when the revision will never be applied as written, as opposed to waiting.
type Gate struct {
	Hold    bool
	Message string
	Refused instanceapi.RefusedRevisionCode
}

// Gate runs before every apply. A spec that moves the application to a
// database the cluster holds, stopped, starts that database and holds the
// apply until it accepts connections, so the application is never pointed at
// a database that is still starting. A spec that names a database the cluster
// does not hold, and that no restore makes, is refused.
func (m *Manager) Gate(ctx context.Context, in GateInput) Gate {
	target := Live(in.Spec)
	if r := in.Spec.Restore; r != nil && in.Spec.Database != nil && r.Generation == target {
		return Gate{}
	}
	if in.Connected.Exists && in.Connected.Generation == target {
		return Gate{}
	}
	held, err := m.clusters(ctx)
	if err != nil {
		return Gate{Hold: true, Message: clean(fmt.Sprintf(
			"the instance moves to %s once its databases can be read: %v", Name(target), err,
		))}
	}
	c, ok := held[target]
	if !ok || deleting(c) {
		if len(held) == 0 {
			return Gate{}
		}
		code := instanceapi.RefusedRevisionCodeDatabaseNotHeld
		if in.Spec.Database == nil {
			code = instanceapi.RefusedRevisionCodeFirstDatabaseRemoved
		}
		return Gate{Hold: true, Refused: code, Message: clean(fmt.Sprintf(
			"the spec names %s, which this instance does not hold and no restore makes: nothing is changed",
			Name(target),
		))}
	}
	if in.Spec.Lifecycle != instanceapi.Active || !in.Connected.Exists {
		return Gate{}
	}
	going := "going back to " + Name(target)
	if stopped(c) {
		if err = m.hibernate(ctx, c.GetName(), "off"); err != nil {
			return Gate{Hold: true, Message: clean(going + ": cannot start it: " + err.Error())}
		}
		m.logger.Info("going back to a kept database: started", "cluster", c.GetName())
		return Gate{Hold: true, Message: clean(going + ": starting it; the instance stays on " +
			Name(in.Connected.Generation) + " until it accepts connections")}
	}
	if accepting(c) {
		return Gate{}
	}
	phase, _, _ := unstructured.NestedString(c.Object, "status", "phase")
	if phase == "" {
		phase = "starting"
	}
	return Gate{
		Hold: true,
		Message: clean(going + ": waiting for it to accept connections (" + phase +
			"); the instance stays on " + Name(in.Connected.Generation)),
	}
}

// Name is how a message names a database.
func Name(generation string) string {
	if generation == "" {
		return "the first database"
	}
	return "database " + generation
}

// clusters lists the release's databases by generation. The label selector
// and the name both have to match: a Cluster of another release, or one
// whose name is not a generation of this release, is never in it.
func (m *Manager) clusters(ctx context.Context) (map[string]*unstructured.Unstructured, error) {
	list, err := m.dynamic.Resource(clusterResource).
		Namespace(m.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: m.selector(),
		})
	if err != nil {
		return nil, err
	}
	out := make(map[string]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		c := &list.Items[i]
		if gen, ok := m.generation(c.GetName()); ok && m.owned(c.GetLabels()) {
			out[gen] = c
		}
	}
	return out, nil
}

func (m *Manager) selector() string {
	return labelInstance + "=" + m.release + "," + labelComponent + "=" + componentPostgres
}

func (m *Manager) owned(labels map[string]string) bool {
	return labels[labelInstance] == m.release && labels[labelComponent] == componentPostgres
}

func (m *Manager) generation(name string) (string, bool) {
	first := values.PostgresCluster(m.release, "")
	if name == first {
		return "", true
	}
	gen, ok := strings.CutPrefix(name, first+"-")
	return gen, ok && restoredPattern.MatchString(gen)
}

func (m *Manager) hibernate(ctx context.Context, name, state string) error {
	patch := []byte(`{"metadata":{"annotations":{"` + hibernation + `":"` + state + `"}}}`)
	_, err := m.dynamic.Resource(clusterResource).Namespace(m.namespace).
		Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func (m *Manager) warn(msg string, err error) {
	at := m.now()
	if !m.warned.IsZero() && at.Sub(m.warned) < warnPeriod {
		return
	}
	m.warned = at
	m.logger.Warn(msg, "error", err)
}

func stopped(c *unstructured.Unstructured) bool {
	return c.GetAnnotations()[hibernation] == "on"
}

func deleting(obj *unstructured.Unstructured) bool {
	return obj.GetDeletionTimestamp() != nil
}

// accepting is CloudNativePG's word that the database serves: healthy, every
// instance it wants ready, and not hibernated.
func accepting(c *unstructured.Unstructured) bool {
	if stopped(c) || hibernationApplied(c) {
		return false
	}
	phase, _, _ := unstructured.NestedString(c.Object, "status", "phase")
	want, _, _ := unstructured.NestedInt64(c.Object, "spec", "instances")
	ready, _, _ := unstructured.NestedInt64(c.Object, "status", "readyInstances")
	return phase == healthyPhase && ready >= max(want, 1)
}

func hibernationApplied(c *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(c.Object, "status", "conditions")
	for _, raw := range conds {
		if cond, ok := raw.(map[string]any); ok && cond["type"] == hibernation &&
			cond["status"] == "True" {
			return true
		}
	}
	return false
}

func clean(s string) string { return health.Clean(s, maxMessage) }
