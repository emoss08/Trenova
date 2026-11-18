package databases

import (
	"context"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const maxListed = 20

type SettleInput struct {
	Spec      *instanceapi.InstanceSpec
	Connected Connected
	Applied   bool
	Ready     bool
}

// Result is what the status report carries: the databases, nil when they
// could not be read, and a message when the spec asked for something the
// operator refused.
type Result struct {
	Databases *[]instanceapi.DatabaseState
	Message   string
}

// Settle runs after every converge. Once the desired revision is applied and
// ready, every database but the live one and the one being restored is
// stopped; the databases the spec retires are removed, never the live one,
// the one the application is connected to, the one being restored or the one
// a restore reads from. A decommissioned instance is left to what
// decommissioning does: nothing is stopped or removed.
func (m *Manager) Settle(ctx context.Context, in SettleInput) Result {
	held, err := m.clusters(ctx)
	if err != nil {
		m.warn("cannot list the instance's databases; the report leaves them out", err)
		return Result{}
	}
	claims, claimsErr := m.claims(ctx)
	if claimsErr != nil {
		m.warn("cannot list the instance's database volumes; sizes are left out", claimsErr)
	}
	s := &settling{Manager: m, in: in, held: held, claims: claims, removing: map[string]bool{}}
	var out Result
	if in.Connected.Known && in.Spec.Lifecycle != instanceapi.Decommissioned {
		if in.Applied && in.Ready {
			s.stopOthers(ctx)
		}
		out.Message = s.retire(ctx, claimsErr == nil)
	}
	list := s.states()
	out.Databases = &list
	return out
}

type settling struct {
	*Manager
	in       SettleInput
	held     map[string]*unstructured.Unstructured
	claims   map[string][]unstructured.Unstructured
	removing map[string]bool
}

func (s *settling) stopOthers(ctx context.Context) {
	target := Live(s.in.Spec)
	for gen, c := range s.held {
		if gen == target || s.restoring(gen) || s.connected(gen) || deleting(c) || stopped(c) {
			continue
		}
		if err := s.hibernate(ctx, c.GetName(), "on"); err != nil {
			s.logger.Warn("cannot stop a database the instance does not run on", "cluster", c.GetName(), "error", err)
			continue
		}
		c.SetAnnotations(merged(c.GetAnnotations(), hibernation, "on"))
		s.logger.Info("a database the instance does not run on is stopped and kept", "cluster", c.GetName())
	}
}

func (s *settling) retire(ctx context.Context, claimsRead bool) string {
	if s.in.Spec.RetiredDatabases == nil {
		return ""
	}
	var refused []string
	seen := map[string]bool{}
	for _, gen := range *s.in.Spec.RetiredDatabases {
		if seen[gen] {
			continue
		}
		seen[gen] = true
		if why := s.kept(gen); why != "" {
			refused = append(refused, Name(gen)+" ("+why+")")
			continue
		}
		if c, ok := s.held[gen]; ok {
			s.removing[gen] = true
			if !deleting(c) {
				s.remove(ctx, c, clusterResource)
			}
			continue
		}
		if !claimsRead {
			continue
		}
		claims := s.claims[values.PostgresCluster(s.release, gen)]
		for i := range claims {
			s.removing[gen] = true
			if !deleting(&claims[i]) {
				s.remove(ctx, &claims[i], claimResource)
			}
		}
	}
	if len(refused) == 0 {
		return ""
	}
	return clean("not removed, though the spec retires them: " + strings.Join(refused, ", "))
}

func (s *settling) kept(gen string) string {
	switch {
	case gen != "" && !restoredPattern.MatchString(gen):
		return "not a database of this instance"
	case gen == Live(s.in.Spec):
		return "the instance runs on it"
	case s.connected(gen):
		return "the instance is still connected to it"
	case s.restoring(gen):
		return "it is being restored"
	case s.in.Spec.Restore != nil && gen == s.in.Spec.Restore.SourceGeneration:
		return "a restore reads from it"
	}
	return ""
}

// remove deletes exactly the object that was listed: the UID precondition
// fails if anything else has taken its name since. A Cluster goes in the
// foreground, so it stays visible, being removed, until its instances,
// volumes, Services and Secrets (all owned by it) are gone.
func (s *settling) remove(ctx context.Context, obj *unstructured.Unstructured, gvr schema.GroupVersionResource) {
	uid := obj.GetUID()
	foreground := metav1.DeletePropagationForeground
	opts := metav1.DeleteOptions{PropagationPolicy: &foreground}
	if uid != "" {
		opts.Preconditions = &metav1.Preconditions{UID: &uid}
	}
	err := s.dynamic.Resource(gvr).Namespace(s.namespace).Delete(ctx, obj.GetName(), opts)
	if err != nil && !apierrors.IsNotFound(err) {
		s.logger.Warn("cannot remove a retired database", "kind", gvr.Resource, "name", obj.GetName(), "error", err)
		return
	}
	s.logger.Info("a retired database is being removed", "kind", gvr.Resource, "name", obj.GetName())
}

func (s *settling) restoring(gen string) bool {
	return s.in.Spec.Restore != nil && gen == s.in.Spec.Restore.Generation
}

func (s *settling) connected(gen string) bool {
	return s.in.Connected.Exists && gen == s.in.Connected.Generation
}

func (s *settling) states() []instanceapi.DatabaseState {
	out := make([]instanceapi.DatabaseState, 0, len(s.held))
	listed := map[string]bool{}
	for gen, c := range s.held {
		listed[gen] = true
		created := c.GetCreationTimestamp().UTC()
		st := instanceapi.DatabaseState{
			Generation: gen,
			State:      s.state(gen, c),
			Live:       s.connected(gen),
			SizeBytes:  s.size(c.GetName()),
		}
		if !created.IsZero() {
			st.CreatedAt = &created
		}
		out = append(out, st)
	}
	for name := range s.claims {
		gen, ok := s.generation(name)
		if !ok || listed[gen] {
			continue
		}
		out = append(out, instanceapi.DatabaseState{
			Generation: gen,
			State:      instanceapi.DatabaseStateStateRemoving,
			SizeBytes:  s.size(name),
		})
	}
	sort.Slice(out, func(i, j int) bool { return before(out[i], out[j]) })
	if len(out) > maxListed {
		out = out[:maxListed]
	}
	return out
}

func before(a, b instanceapi.DatabaseState) bool {
	if a.Live != b.Live {
		return a.Live
	}
	at, bt := stamp(a.CreatedAt), stamp(b.CreatedAt)
	if !at.Equal(bt) {
		return at.After(bt)
	}
	return a.Generation < b.Generation
}

func stamp(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func (s *settling) state(gen string, c *unstructured.Unstructured) instanceapi.DatabaseStateState {
	switch {
	case s.removing[gen] || deleting(c):
		return instanceapi.DatabaseStateStateRemoving
	case stopped(c):
		return instanceapi.DatabaseStateStateStopped
	case accepting(c):
		return instanceapi.DatabaseStateStateRunning
	case s.restoring(gen) && !s.connected(gen):
		return instanceapi.DatabaseStateStateRestoring
	case gen == Live(s.in.Spec) || s.connected(gen):
		return instanceapi.DatabaseStateStateStarting
	}
	return instanceapi.DatabaseStateStateStopped
}

// size is the capacity of the cluster's volumes, when every one of them
// reports it.
func (s *settling) size(cluster string) *int64 {
	claims := s.claims[cluster]
	if len(claims) == 0 {
		return nil
	}
	var total int64
	for i := range claims {
		raw, _, _ := unstructured.NestedString(claims[i].Object, "status", "capacity", "storage")
		q, err := resource.ParseQuantity(raw)
		if raw == "" || err != nil || q.Value() <= 0 {
			return nil
		}
		total += q.Value()
	}
	return &total
}

// claims lists the release's database volumes by the Cluster they belong to.
// CloudNativePG labels each with its cluster, and the chart's inherited
// metadata with the release and the component.
func (m *Manager) claims(ctx context.Context) (map[string][]unstructured.Unstructured, error) {
	list, err := m.dynamic.Resource(claimResource).Namespace(m.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: m.selector(),
	})
	if err != nil {
		return nil, err
	}
	out := map[string][]unstructured.Unstructured{}
	for i := range list.Items {
		claim := list.Items[i]
		name := claim.GetLabels()[labelCluster]
		if _, ok := m.generation(name); ok && m.owned(claim.GetLabels()) {
			out[name] = append(out[name], claim)
		}
	}
	for name := range out {
		slices.SortFunc(out[name], func(a, b unstructured.Unstructured) int {
			return strings.Compare(a.GetName(), b.GetName())
		})
	}
	return out, nil
}

func merged(m map[string]string, k, v string) map[string]string {
	out := maps.Clone(m)
	if out == nil {
		out = map[string]string{}
	}
	out[k] = v
	return out
}
