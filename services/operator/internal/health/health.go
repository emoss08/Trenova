package health

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	componentRelease  = "release"
	componentPostgres = "postgres"
	labelName         = "app.kubernetes.io/name"
	labelInstance     = "app.kubernetes.io/instance"
	labelComponent    = "app.kubernetes.io/component"
	maxMessage        = 200
	healthyPhase      = "Cluster in healthy state"
)

var clusterResource = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
}

type Component struct {
	Name    string
	Ready   bool
	Message string
}

type Checker struct {
	kube      kubernetes.Interface
	dynamic   dynamic.Interface
	namespace string
	release   string
	chart     string
}

type CheckerParams struct {
	Kube      kubernetes.Interface
	Dynamic   dynamic.Interface
	Namespace string
	Release   string
	Chart     string
}

func NewChecker(p *CheckerParams) *Checker {
	return &Checker{
		kube:      p.Kube,
		dynamic:   p.Dynamic,
		namespace: p.Namespace,
		release:   p.Release,
		chart:     p.Chart,
	}
}

type CheckParams struct {
	ReleaseJob string
	Stopped    bool
	Hibernated bool
	Generation string
}

func (c *Checker) Check(ctx context.Context, p CheckParams) ([]Component, error) {
	parts, err := c.Parts(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Component, 0, len(parts))
	var seen *observed
	for i := range parts {
		f := &parts[i]
		if !f.Ready && f.Pods != nil {
			if seen == nil {
				o := c.observe(ctx)
				seen = &o
			}
			f.Component = explain(f.Component, seen.cause(f.Pods))
		}
		out = append(out, f.Component)
	}
	return out, nil
}

// Part is a component, the copies it should run, the image it runs and the
// selector of the pods that make it up: what a status report and a telemetry
// sample both read, so the two name the same components.
type Part struct {
	Component
	Replicas      *int32
	ReadyReplicas *int32
	Image         string
	Pods          labels.Selector
}

func (c *Checker) Parts(ctx context.Context, p CheckParams) ([]Part, error) {
	selector := fmt.Sprintf("%s=%s,%s=%s", labelName, c.chart, labelInstance, c.release)
	list := metav1.ListOptions{LabelSelector: selector}
	found := []Part{}

	deployments, err := c.kube.AppsV1().Deployments(c.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for i := range deployments.Items {
		d := &deployments.Items[i]
		found = append(found, Part{
			Component:     Deployment(d),
			Replicas:      wanted(d.Spec.Replicas),
			ReadyReplicas: &d.Status.ReadyReplicas,
			Image:         firstImage(&d.Spec.Template.Spec),
			Pods:          podSelector(d.Spec.Selector),
		})
	}
	sets, err := c.kube.AppsV1().StatefulSets(c.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	for i := range sets.Items {
		s := &sets.Items[i]
		found = append(found, Part{
			Component:     StatefulSet(s),
			Replicas:      wanted(s.Spec.Replicas),
			ReadyReplicas: &s.Status.ReadyReplicas,
			Image:         firstImage(&s.Spec.Template.Spec),
			Pods:          podSelector(s.Spec.Selector),
		})
	}
	found = append(found, c.postgres(ctx, p.Hibernated, p.Generation))
	if !p.Stopped {
		job, jerr := c.releaseJob(ctx, &list, p.ReleaseJob)
		if jerr != nil {
			return nil, jerr
		}
		found = append(found, job)
	}
	slices.SortFunc(found, func(a, b Part) int { return strings.Compare(a.Name, b.Name) })
	return found, nil
}

func wanted(replicas *int32) *int32 {
	want := int32(1)
	if replicas != nil {
		want = *replicas
	}
	return &want
}

func firstImage(spec *corev1.PodSpec) string {
	if len(spec.Containers) == 0 {
		return ""
	}
	return spec.Containers[0].Image
}

func podSelector(s *metav1.LabelSelector) labels.Selector {
	sel, err := metav1.LabelSelectorAsSelector(s)
	if err != nil {
		return nil
	}
	return sel
}

func component(m *metav1.ObjectMeta) string {
	if n := m.Labels[labelComponent]; n != "" {
		return n
	}
	return m.Name
}

func Deployment(d *appsv1.Deployment) Component {
	want := int32(1)
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	c := Component{Name: component(&d.ObjectMeta)}
	switch {
	case want == 0 && d.Status.Replicas == 0:
		c.Ready, c.Message = true, "stopped"
	case d.Status.ObservedGeneration < d.Generation:
		c.Message = "rolling out"
	case d.Status.UpdatedReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas updated", d.Status.UpdatedReplicas, want)
	case d.Status.AvailableReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas available", d.Status.AvailableReplicas, want)
	case d.Status.Replicas > want:
		c.Message = "old replicas terminating"
	default:
		c.Ready = true
	}
	return c
}

func StatefulSet(s *appsv1.StatefulSet) Component {
	want := int32(1)
	if s.Spec.Replicas != nil {
		want = *s.Spec.Replicas
	}
	c := Component{Name: component(&s.ObjectMeta)}
	switch {
	case want == 0 && s.Status.Replicas == 0:
		c.Ready, c.Message = true, "stopped"
	case s.Status.ObservedGeneration < s.Generation:
		c.Message = "rolling out"
	case s.Status.UpdateRevision != "" && s.Status.CurrentRevision != s.Status.UpdateRevision:
		c.Message = "updating"
	case s.Status.ReadyReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas ready", s.Status.ReadyReplicas, want)
	default:
		c.Ready = true
	}
	return c
}

func (c *Checker) postgres(ctx context.Context, hibernated bool, generation string) Part {
	name := c.release + "-postgres"
	if c.release != c.chart && !strings.Contains(c.release, c.chart) {
		name = c.release + "-" + c.chart + "-postgres"
	}
	if generation != "" {
		name += "-" + generation
	}
	part := Part{Pods: labels.SelectorFromSet(labels.Set{labelCNPGCluster: name})}
	obj, err := c.dynamic.Resource(clusterResource).
		Namespace(c.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		part.Component = Component{
			Name:    componentPostgres,
			Message: trim("cluster " + name + ": " + err.Error()),
		}
		return part
	}
	part.Component = cluster(obj, hibernated)
	if want, ok, _ := unstructured.NestedInt64(obj.Object, "spec", "instances"); ok {
		w := replicaCount(want)
		if hibernated {
			w = 0
		}
		part.Replicas = &w
	}
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyInstances")
	r := replicaCount(ready)
	part.ReadyReplicas = &r
	part.Image, _, _ = unstructured.NestedString(obj.Object, "status", "image")
	if part.Image == "" {
		part.Image, _, _ = unstructured.NestedString(obj.Object, "spec", "imageName")
	}
	return part
}

func replicaCount(n int64) int32 {
	return int32(min(max(n, 0), math.MaxInt32))
}

func cluster(obj *unstructured.Unstructured, hibernated bool) Component {
	c := Component{Name: componentPostgres}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	want, _, _ := unstructured.NestedInt64(obj.Object, "spec", "instances")
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyInstances")
	if hibernated {
		if hibernationApplied(obj) {
			c.Ready, c.Message = true, "hibernated"
		} else {
			c.Message = "hibernating"
		}
		return c
	}
	switch {
	case phase == healthyPhase && ready >= want:
		c.Ready = true
	case phase != "":
		c.Message = trim(fmt.Sprintf("%s (%d of %d instances ready)", phase, ready, want))
	default:
		c.Message = "starting"
	}
	return c
}

func hibernationApplied(obj *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conds {
		cond, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if cond["type"] == "cnpg.io/hibernation" && cond["status"] == "True" {
			return true
		}
	}
	return false
}

func (c *Checker) releaseJob(
	ctx context.Context,
	list *metav1.ListOptions,
	name string,
) (Part, error) {
	comp := Component{Name: componentRelease}
	if name == "" {
		comp.Message = "the deployed release names no release Job"
		return Part{Component: comp}, nil
	}
	jobs, err := c.kube.BatchV1().Jobs(c.namespace).List(ctx, *list)
	if err != nil {
		return Part{Component: comp}, fmt.Errorf("list jobs: %w", err)
	}
	for i := range jobs.Items {
		j := &jobs.Items[i]
		if j.Labels[labelComponent] != componentRelease || j.Name != name {
			continue
		}
		return Part{
			Component: job(j),
			Image:     firstImage(&j.Spec.Template.Spec),
			Pods:      labels.SelectorFromSet(labels.Set{labelJobName: j.Name}),
		}, nil
	}
	comp.Message = "waiting for the release Job " + name
	return Part{Component: comp}, nil
}

func job(j *batchv1.Job) Component {
	c := Component{Name: componentRelease}
	for _, cond := range j.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.Type == batchv1.JobComplete {
			c.Ready = true
			return c
		}
		if cond.Type == batchv1.JobFailed {
			c.Message = trim("failed: " + cond.Message)
			return c
		}
	}
	c.Message = "running"
	return c
}

// trim keeps a component message within the instance API's limit.
func trim(s string) string { return Clean(s, maxMessage) }

// Clean makes s acceptable to the instance API, which refuses control
// characters in a message: every run of control characters and spaces
// becomes one space (Helm's and Kubernetes' errors are often several lines),
// and the result is cut between characters to at most maxLen.
func Clean(s string, maxLen int) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	out := []rune(b.String())
	if len(out) <= maxLen {
		return string(out)
	}
	return string(out[:maxLen-1]) + "…"
}
